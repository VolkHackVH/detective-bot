package evidence

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(
	ctx context.Context,
	input CreateEvidence,
) (*Evidence, error) {
	result, err := r.db.ExecContext(ctx, `
        INSERT INTO evidence (
            author_discord_id,
            guild_id,
            nickname_static,
            proof_url,
            timecodes,
            faction_family
        )
        VALUES (?, ?, ?, ?, ?, ?)
    `,
		input.AuthorDiscordID,
		input.GuildID,
		input.NicknameStatic,
		input.ProofURL,
		input.Timecodes,
		input.FactionFamily,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, id)
}

func (r *Repository) Claim(
	ctx context.Context,
	evidenceID int64,
	reviewerID string,
) (*Evidence, bool, error) {
	result, err := r.db.ExecContext(ctx, `
        UPDATE evidence
        SET
            status = 'in_review',
            reviewer_discord_id = ?,
            claimed_at = CURRENT_TIMESTAMP
        WHERE id = ?
          AND status = 'submitted'
    `, reviewerID, evidenceID)
	if err != nil {
		return nil, false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, false, err
	}

	evidence, err := r.GetByID(ctx, evidenceID)
	return evidence, affected == 1, err
}

func (r *Repository) Resolve(
	ctx context.Context,
	evidenceID int64,
	reviewerID string,
	status Status,
) (*Evidence, bool, error) {
	if status != StatusAccepted && status != StatusRejected {
		return nil, false, fmt.Errorf(
			"invalid final status: %s",
			status,
		)
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE evidence
		SET
			status = ?,
			reviewer_discord_id = ?,
			resolved_at = CURRENT_TIMESTAMP
		WHERE id = ?
		  AND status = 'in_review'
	`,
		status,
		reviewerID,
		evidenceID,
	)
	if err != nil {
		return nil, false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, false, err
	}

	item, err := r.GetByID(ctx, evidenceID)

	return item, affected == 1, err
}

func (r *Repository) GetByID(
	ctx context.Context,
	id int64,
) (*Evidence, error) {
	var item Evidence
	var reviewerID sql.NullString
	var channelID sql.NullString
	var messageID sql.NullString

	err := r.db.QueryRowContext(ctx, `
        SELECT
            id,
            author_discord_id,
            guild_id,
            nickname_static,
            proof_url,
            timecodes,
            faction_family,
            status,
            reviewer_discord_id,
            review_channel_id,
            review_message_id
        FROM evidence
        WHERE id = ?
    `, id).Scan(
		&item.ID,
		&item.AuthorDiscordID,
		&item.GuildID,
		&item.NicknameStatic,
		&item.ProofURL,
		&item.Timecodes,
		&item.FactionFamily,
		&item.Status,
		&reviewerID,
		&channelID,
		&messageID,
	)
	if err != nil {
		return nil, err
	}

	if reviewerID.Valid {
		item.ReviewerDiscordID = &reviewerID.String
	}
	if channelID.Valid {
		item.ReviewChannelID = &channelID.String
	}
	if messageID.Valid {
		item.ReviewMessageID = &messageID.String
	}

	return &item, nil
}

func (r *Repository) SetReviewMessage(
	ctx context.Context,
	evidenceID int64,
	channelID string,
	messageID string,
) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE evidence
        SET review_channel_id = ?,
            review_message_id = ?
        WHERE id = ?
    `, channelID, messageID, evidenceID)

	return err
}
