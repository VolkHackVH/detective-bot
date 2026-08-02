package evidence

type Status string

const (
	StatusSubmitted Status = "submitted"
	StatusInReview  Status = "in_review"
	StatusAccepted  Status = "accepted"
	StatusRejected  Status = "rejected"
)

type Evidence struct {
	ID                int64
	AuthorDiscordID   string
	GuildID           string
	NicknameStatic    string
	ProofURL          string
	Timecodes         string
	FactionFamily     string
	Status            Status
	ReviewerDiscordID *string
	RejectionReason   *string
	ReviewChannelID   *string
	ReviewMessageID   *string
}

type CreateEvidence struct {
	AuthorDiscordID string
	GuildID         string
	NicknameStatic  string
	ProofURL        string
	Timecodes       string
	FactionFamily   string
}
