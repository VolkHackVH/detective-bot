package evidence

import (
	"github.com/bwmarrin/discordgo"
)

const (
	createPanelCommand = "!улика"

	createEvidenceButtonID = "evidence:new"
	submitEvidenceModalID  = "evidence:submit"
)

func evidencePanelMessage() *discordgo.MessageSend {
	return &discordgo.MessageSend{
		Content: "Для подачи улики нажмите кнопку **«Подать улику»**.",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Подать улику",
						Style:    discordgo.PrimaryButton,
						CustomID: createEvidenceButtonID,
					},
				},
			},
		},
	}
}

func evidenceModal() *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: submitEvidenceModalID,
			Title:    "Подача улики",
			Components: []discordgo.MessageComponent{
				textInputRow(
					"nickname_static",
					"Никнейм и статик",
					"Ivan Ivanov | 12345",
					discordgo.TextInputShort,
					2,
					100,
				),
				textInputRow(
					"proof_url",
					"Доказательство (ссылка)",
					"https://...",
					discordgo.TextInputShort,
					5,
					500,
				),
				textInputRow(
					"timecodes",
					"Тайм-коды",
					"00:15 — нарушение",
					discordgo.TextInputParagraph,
					1,
					1000,
				),
				textInputRow(
					"faction_family",
					"Фракция / Семья",
					"Название фракции или семьи",
					discordgo.TextInputShort,
					1,
					100,
				),
			},
		},
	}
}

func textInputRow(
	customID string,
	label string,
	placeholder string,
	style discordgo.TextInputStyle,
	minLength int,
	maxLength int,
) discordgo.ActionsRow {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.TextInput{
				CustomID:    customID,
				Label:       label,
				Placeholder: placeholder,
				Style:       style,
				Required:    true,
				MinLength:   minLength,
				MaxLength:   maxLength,
			},
		},
	}
}
