package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DiscordToken string `json:"-"`
	DatabasePath string `json:"-"`

	Discord  DiscordConfig  `json:"discord"`
	Evidence EvidenceConfig `json:"evidence"`
}

type DiscordConfig struct {
	GuildID string `json:"guild_id"`
}

type EvidenceConfig struct {
	IntakeChannelID string   `json:"intake_channel_id"`
	ReviewChannelID string   `json:"review_channel_id"`
	ReviewerRoleIDs []string `json:"reviewer_role_ids"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}

	var cfg Config

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config file %q: %w", path, err)
	}

	cfg.DiscordToken = strings.TrimSpace(
		os.Getenv("DISCORD_TOKEN"),
	)
	cfg.DatabasePath = strings.TrimSpace(
		os.Getenv("DATABASE_PATH"),
	)

	cfg.normalize()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) normalize() {
	c.Discord.GuildID = strings.TrimSpace(c.Discord.GuildID)

	c.Evidence.IntakeChannelID = strings.TrimSpace(
		c.Evidence.IntakeChannelID,
	)
	c.Evidence.ReviewChannelID = strings.TrimSpace(
		c.Evidence.ReviewChannelID,
	)

	roleIDs := make([]string, 0, len(c.Evidence.ReviewerRoleIDs))
	seen := make(map[string]struct{})

	for _, roleID := range c.Evidence.ReviewerRoleIDs {
		roleID = strings.TrimSpace(roleID)
		if roleID == "" {
			continue
		}

		if _, exists := seen[roleID]; exists {
			continue
		}

		roleIDs = append(roleIDs, roleID)
		seen[roleID] = struct{}{}
	}

	c.Evidence.ReviewerRoleIDs = roleIDs
}

func (c *Config) validate() error {
	if c.DiscordToken == "" {
		return fmt.Errorf("environment variable DISCORD_TOKEN is required")
	}

	if c.DatabasePath == "" {
		return fmt.Errorf("environment variable DATABASE_PATH is required")
	}

	if c.Discord.GuildID == "" {
		return fmt.Errorf("discord.guild_id is required in config.json")
	}

	if c.Evidence.IntakeChannelID == "" {
		return fmt.Errorf(
			"evidence.intake_channel_id is required in config.json",
		)
	}

	if c.Evidence.ReviewChannelID == "" {
		return fmt.Errorf(
			"evidence.review_channel_id is required in config.json",
		)
	}

	if len(c.Evidence.ReviewerRoleIDs) == 0 {
		return fmt.Errorf(
			"evidence.reviewer_role_ids must contain at least one role ID",
		)
	}

	return nil
}
