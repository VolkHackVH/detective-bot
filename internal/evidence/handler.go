package evidence

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const interactionTimeout = 2 * time.Second

type Handler struct {
	repo            *Repository
	intakeChannelID string
	reviewChannelID string

	reviewerRoleIDs []string
	reviewerRoleSet map[string]struct{}
}

func NewHandler(
	repo *Repository,
	intakeChannelID string,
	reviewChannelID string,
	reviewerRoleIDs []string,
) *Handler {
	roleIDs := make([]string, 0, len(reviewerRoleIDs))
	roleSet := make(map[string]struct{}, len(reviewerRoleIDs))

	for _, roleID := range reviewerRoleIDs {
		roleID = strings.TrimSpace(roleID)
		if roleID == "" {
			continue
		}

		if _, exists := roleSet[roleID]; exists {
			continue
		}

		roleIDs = append(roleIDs, roleID)
		roleSet[roleID] = struct{}{}
	}

	return &Handler{
		repo:            repo,
		intakeChannelID: intakeChannelID,
		reviewChannelID: reviewChannelID,
		reviewerRoleIDs: roleIDs,
		reviewerRoleSet: roleSet,
	}
}

func (h *Handler) OnMessageCreate(
	session *discordgo.Session,
	message *discordgo.MessageCreate,
) {
	if message.Author == nil ||
		message.Author.Bot ||
		message.WebhookID != "" ||
		message.GuildID == "" ||
		message.ChannelID != h.intakeChannelID ||
		strings.TrimSpace(message.Content) != createPanelCommand {
		return
	}

	permissions, err := session.UserChannelPermissions(
		message.Author.ID,
		message.ChannelID,
	)
	if err != nil {
		slog.Error(
			"cannot check user permissions",
			"user_id", message.Author.ID,
			"error", err,
		)
		return
	}

	if permissions&discordgo.PermissionAdministrator == 0 {
		_, err = session.ChannelMessageSendReply(
			message.ChannelID,
			"Создавать панель подачи улик может только администратор.",
			message.Reference(),
		)
		if err != nil {
			slog.Error("cannot send permission error", "error", err)
		}
		return
	}

	if _, err = session.ChannelMessageSendComplex(
		message.ChannelID,
		evidencePanelMessage(),
	); err != nil {
		slog.Error("cannot send evidence panel", "error", err)
	}
}

func (h *Handler) OnInteraction(
	session *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) {
	switch interaction.Type {
	case discordgo.InteractionMessageComponent:
		h.handleMessageComponent(session, interaction)
	case discordgo.InteractionModalSubmit:
		h.handleModalSubmit(session, interaction)
	}
}

func (h *Handler) handleMessageComponent(
	session *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) {
	customID := interaction.MessageComponentData().CustomID

	switch {
	case customID == createEvidenceButtonID:
		if err := session.InteractionRespond(
			interaction.Interaction,
			evidenceModal(),
		); err != nil {
			slog.Error("cannot show evidence modal", "error", err)
		}
	case strings.HasPrefix(customID, "evidence:claim:"):
		h.handleClaim(session, interaction)
	case strings.HasPrefix(customID, "evidence:accept:"):
		h.handleResolve(session, interaction, StatusAccepted)
	case strings.HasPrefix(customID, "evidence:reject:"):
		h.handleResolve(session, interaction, StatusRejected)
	}
}

func (h *Handler) reviewerMentions() string {
	mentions := make([]string, 0, len(h.reviewerRoleIDs))

	for _, roleID := range h.reviewerRoleIDs {
		mentions = append(mentions, "<@&"+roleID+">")
	}

	return strings.Join(mentions, " ")
}

func (h *Handler) handleModalSubmit(
	session *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) {
	if interaction.ModalSubmitData().CustomID != submitEvidenceModalID {
		return
	}

	values := modalValues(interaction.ModalSubmitData())
	if err := validateEvidence(values); err != nil {
		respondEphemeral(session, interaction, err.Error())
		return
	}

	authorID := interactionUserID(interaction)
	if authorID == "" {
		respondEphemeral(session, interaction, "Не удалось определить отправителя.")
		return
	}
	if err := deferEphemeral(session, interaction); err != nil {
		slog.Error("cannot defer evidence modal response", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	item, err := h.repo.Create(ctx, CreateEvidence{
		AuthorDiscordID: authorID,
		GuildID:         interaction.GuildID,
		NicknameStatic:  values["nickname_static"],
		ProofURL:        values["proof_url"],
		Timecodes:       values["timecodes"],
		FactionFamily:   values["faction_family"],
	})
	cancel()
	if err != nil {
		slog.Error("cannot create evidence", "error", err)
		editDeferredResponse(session, interaction, "Не удалось сохранить улику.")
		return
	}

	reviewMessage, err := session.ChannelMessageSendComplex(
		h.reviewChannelID,
		&discordgo.MessageSend{
			Content: h.reviewerMentions(),

			AllowedMentions: &discordgo.MessageAllowedMentions{
				Roles: h.reviewerRoleIDs,
			},

			Embeds: []*discordgo.MessageEmbed{
				evidenceEmbed(item),
			},

			Components: evidenceButtons(item),
		},
	)
	if err != nil {
		slog.Error("cannot publish evidence", "evidence_id", item.ID, "error", err)
		editDeferredResponse(
			session,
			interaction,
			fmt.Sprintf("Улика №%d сохранена, но не опубликована. Обратитесь к администратору.", item.ID),
		)
		return
	}

	ctx, cancel = context.WithTimeout(context.Background(), interactionTimeout)
	if err = h.repo.SetReviewMessage(
		ctx,
		item.ID,
		reviewMessage.ChannelID,
		reviewMessage.ID,
	); err != nil {
		slog.Error(
			"cannot save review message",
			"evidence_id", item.ID,
			"error", err,
		)
	}
	cancel()

	editDeferredResponse(
		session,
		interaction,
		fmt.Sprintf("Улика №%d успешно отправлена.", item.ID),
	)
}

func (h *Handler) handleClaim(
	session *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) {
	if !h.isReviewer(interaction.Member) {
		respondEphemeral(session, interaction, "У вас нет роли проверяющего.")
		return
	}

	evidenceID, err := parseEvidenceID(interaction.MessageComponentData().CustomID)
	if err != nil {
		respondEphemeral(session, interaction, "Некорректный номер улики.")
		return
	}

	reviewerID := interactionUserID(interaction)
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()

	item, claimed, err := h.repo.Claim(ctx, evidenceID, reviewerID)
	if err != nil {
		slog.Error("cannot claim evidence", "evidence_id", evidenceID, "error", err)
		respondEphemeral(session, interaction, "Не удалось взять улику в работу.")
		return
	}
	if !claimed {
		respondEphemeral(session, interaction, "Эту улику уже взял другой проверяющий.")
		return
	}

	if err = updateInteractionMessage(session, interaction, item); err != nil {
		slog.Error("cannot update evidence message", "evidence_id", evidenceID, "error", err)
		return
	}

	h.sendDM(
		session,
		item.AuthorDiscordID,
		fmt.Sprintf("🔎 Ваша улика `№%d` находится на рассмотрении.", item.ID),
	)
}

func (h *Handler) handleResolve(
	session *discordgo.Session,
	interaction *discordgo.InteractionCreate,
	finalStatus Status,
) {
	if !h.isReviewer(interaction.Member) {
		respondEphemeral(
			session,
			interaction,
			"У вас нет роли проверяющего.",
		)
		return
	}

	evidenceID, err := parseEvidenceID(interaction.MessageComponentData().CustomID)
	if err != nil {
		respondEphemeral(session, interaction, "Некорректный номер улики.")
		return
	}

	reviewerID := interactionUserID(interaction)
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()

	item, resolved, err := h.repo.Resolve(ctx, evidenceID, reviewerID, finalStatus)
	if err != nil {
		slog.Error("cannot resolve evidence", "evidence_id", evidenceID, "error", err)
		respondEphemeral(session, interaction, "Не удалось изменить статус улики.")
		return
	}
	if !resolved {
		respondEphemeral(
			session,
			interaction,
			"Закрыть улику может только проверяющий, который взял её в работу.",
		)
		return
	}

	if err = updateInteractionMessage(session, interaction, item); err != nil {
		slog.Error("cannot update evidence message", "evidence_id", evidenceID, "error", err)
		return
	}

	result := "(✅ **одобрена**)"
	if finalStatus == StatusRejected {
		result = "(❌ **отклонена**)"
	}

	h.sendDM(
		session,
		item.AuthorDiscordID,
		fmt.Sprintf("Ваша улика `№%d` %s.", item.ID, result),
	)
}

func (h *Handler) isReviewer(member *discordgo.Member) bool {
	if member == nil {
		return false
	}

	for _, memberRoleID := range member.Roles {
		if _, allowed := h.reviewerRoleSet[memberRoleID]; allowed {
			return true
		}
	}

	return false
}

func (h *Handler) sendDM(
	session *discordgo.Session,
	userID string,
	content string,
) {
	channel, err := session.UserChannelCreate(userID)
	if err != nil {
		slog.Warn("cannot create DM channel", "user_id", userID, "error", err)
		return
	}

	if _, err = session.ChannelMessageSend(channel.ID, content); err != nil {
		slog.Warn("cannot send DM", "user_id", userID, "error", err)
	}
}

func modalValues(data discordgo.ModalSubmitInteractionData) map[string]string {
	values := make(map[string]string)

	for _, component := range data.Components {
		row, ok := component.(*discordgo.ActionsRow)
		if !ok {
			continue
		}

		for _, child := range row.Components {
			input, ok := child.(*discordgo.TextInput)
			if !ok {
				continue
			}
			values[input.CustomID] = strings.TrimSpace(input.Value)
		}
	}

	return values
}

func validateEvidence(values map[string]string) error {
	required := []string{
		"nickname_static",
		"proof_url",
		"timecodes",
		"faction_family",
	}
	for _, field := range required {
		if values[field] == "" {
			return errors.New("необходимо заполнить все поля")
		}
	}

	proofURL, err := url.ParseRequestURI(values["proof_url"])
	if err != nil || (proofURL.Scheme != "http" && proofURL.Scheme != "https") {
		return errors.New("поле «Доказательство» должно содержать HTTP/HTTPS-ссылку")
	}

	return nil
}

func parseEvidenceID(customID string) (int64, error) {
	parts := strings.Split(customID, ":")
	if len(parts) != 3 || parts[0] != "evidence" {
		return 0, errors.New("invalid evidence custom id")
	}

	return strconv.ParseInt(parts[2], 10, 64)
}

func interactionUserID(interaction *discordgo.InteractionCreate) string {
	if interaction.Member != nil && interaction.Member.User != nil {
		return interaction.Member.User.ID
	}
	if interaction.User != nil {
		return interaction.User.ID
	}
	return ""
}

func respondEphemeral(
	session *discordgo.Session,
	interaction *discordgo.InteractionCreate,
	content string,
) {
	if err := session.InteractionRespond(
		interaction.Interaction,
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		},
	); err != nil {
		slog.Error("cannot respond to interaction", "error", err)
	}
}

func deferEphemeral(
	session *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) error {
	return session.InteractionRespond(
		interaction.Interaction,
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		},
	)
}

func editDeferredResponse(
	session *discordgo.Session,
	interaction *discordgo.InteractionCreate,
	content string,
) {
	if _, err := session.InteractionResponseEdit(
		interaction.Interaction,
		&discordgo.WebhookEdit{Content: &content},
	); err != nil {
		slog.Error("cannot edit deferred interaction response", "error", err)
	}
}

func updateInteractionMessage(
	session *discordgo.Session,
	interaction *discordgo.InteractionCreate,
	item *Evidence,
) error {
	return session.InteractionRespond(
		interaction.Interaction,
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{evidenceEmbed(item)},
				Components: evidenceButtons(item),
			},
		},
	)
}

func evidenceButtons(item *Evidence) []discordgo.MessageComponent {
	claimDisabled := item.Status != StatusSubmitted
	decisionDisabled := item.Status != StatusInReview

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Взять в работу",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("evidence:claim:%d", item.ID),
					Disabled: claimDisabled,
				},
				discordgo.Button{
					Label:    "Принято",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("evidence:accept:%d", item.ID),
					Disabled: decisionDisabled,
				},
				discordgo.Button{
					Label:    "Не принято",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("evidence:reject:%d", item.ID),
					Disabled: decisionDisabled,
				},
			},
		},
	}
}

func evidenceEmbed(item *Evidence) *discordgo.MessageEmbed {
	status := "🕵️ Ожидает проверяющего"
	color := 0xF1C40F

	switch item.Status {
	case StatusInReview:
		status = "🔎 На рассмотрении"
		color = 0x3498DB
	case StatusAccepted:
		status = "✅ Одобрено"
		color = 0x2ECC71
	case StatusRejected:
		status = "❌ Не принято"
		color = 0xE74C3C
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "Никнейм | статик", Value: item.NicknameStatic},
		{Name: "Доказательство", Value: item.ProofURL},
		{Name: "Тайм-коды", Value: item.Timecodes},
		{Name: "Фракция / Семья", Value: item.FactionFamily},
		{Name: "Статус проверки", Value: status},
	}

	if item.ReviewerDiscordID != nil {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Улика рассматривалась by",
			Value: "<@" + *item.ReviewerDiscordID + ">",
		})
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Улика №%d", item.ID),
		Description: fmt.Sprintf("Отправитель: <@%s>", item.AuthorDiscordID),
		Color:       color,
		Fields:      fields,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}
