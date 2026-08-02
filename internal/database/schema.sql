CREATE TABLE IF NOT EXISTS evidence (
  id INTEGER PRIMARY KEY AUTOINCREMENT,

 author_discord_id TEXT NOT NULL,
 guild_id TEXT NOT NULL,

 nickname_static TEXT NOT NULL,
 proof_url TEXT NOT NULL,
 timecodes TEXT NOT NULL,
 faction_family TEXT NOT NULL,

status TEXT NOT NULL DEFAULT 'submitted'
    CHECK (status IN (
                      'submitted',
                      'in_review',
                      'accepted',
                      'rejected',
                      'publish_failed'
                     )),

reviewer_discord_id TEXT,
rejection_reason TEXT,

review_channel_id TEXT,
review_message_id TEXT,

created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
claimed_at TEXT,
resolved_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_evidence_status
    ON evidence(status);

CREATE INDEX IF NOT EXISTS idx_evidence_author
    ON evidence(author_discord_id);
