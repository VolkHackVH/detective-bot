package evidence

import "github.com/bwmarrin/discordgo"

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

	switch customID {
	case createEvidenceButtonID:
		if err := session.InteractionRespond(
			interaction.Interaction,
			evidenceModal(),
		); err != nil {
			slog.Error("cannot show evidence modal", "error", err)
		}
	}
}
