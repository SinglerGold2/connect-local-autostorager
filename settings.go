package main

// The settings a person edits on the page. The manifest's options are the
// operator's defaults; what is saved here wins, so a value typed on the page
// survives a restart.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// defaultMaxFolders is the minimum and the default; maxFoldersLimit keeps an
// operator from turning the page into an unbounded scan grid.
const (
	defaultMaxFolders = 4
	maxFoldersLimit   = 64
)

// merged reads the saved settings and lays them over what the host sent.
func merged(ctx context.Context, base Settings) Settings {
	if db == nil {
		return base
	}

	rows, err := db.QueryContext(ctx, `SELECT key, value FROM autostorager_settings`)
	if err != nil {
		return base
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return base
		}

		base = apply(base, key, value)
	}

	base.Folders = normalized(base.Folders, base.MaxFolders)

	return base
}

// apply writes one stored value onto a settings struct.
func apply(s Settings, key, value string) Settings {
	switch key {
	case "folders":
		var folders []string
		if err := json.Unmarshal([]byte(value), &folders); err == nil {
			s.Folders = folders
		}
	case "storage":
		if value != "" {
			s.Storage = value
		}
	case "interval_seconds":
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			s.IntervalSeconds = parsed
		}
	case "max_mb":
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			s.MaxMB = parsed
		}
	case "move_on_success":
		s.MoveOnSuccess = value == "true"
	}

	return s
}

// save writes every setting the page owns, in one transaction.
func save(ctx context.Context, settings Settings) error {
	if db == nil {
		return errors.New("this instance has no database")
	}

	folders, err := json.Marshal(settings.Folders)
	if err != nil {
		return err
	}

	values := map[string]string{
		"folders":          string(folders),
		"interval_seconds": strconv.Itoa(settings.IntervalSeconds),
		"storage":          settings.Storage,
		"max_mb":           strconv.Itoa(settings.MaxMB),
		"move_on_success":  strconv.FormatBool(settings.MoveOnSuccess),
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for key, value := range values {
		if err := upsert(ctx, tx, key, value); err != nil {
			_ = tx.Rollback()

			return err
		}
	}

	return tx.Commit()
}

// upsert writes one row.
func upsert(ctx context.Context, tx *sql.Tx, key, value string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO autostorager_settings (key, value) VALUES (?, ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, key, value)

	return err
}

// normalized drops empty folders and keeps the list within the limit.
func normalized(folders []string, limit int) []string {
	out := make([]string, 0, len(folders))

	for _, folder := range folders {
		trimmed := strings.TrimSpace(folder)
		if trimmed == "" {
			continue
		}

		out = append(out, trimmed)
	}

	if limit < defaultMaxFolders {
		limit = defaultMaxFolders
	}

	if limit > maxFoldersLimit {
		limit = maxFoldersLimit
	}

	if len(out) > limit {
		out = out[:limit]
	}

	return out
}

// validate refuses a value that cannot work, before it is stored.
func validate(s Settings) error {
	if s.IntervalSeconds < 1 || s.IntervalSeconds > 86400 {
		return fmt.Errorf("the interval must be between 1 and 86400 seconds")
	}

	if s.MaxMB < 1 || s.MaxMB > 4096 {
		return fmt.Errorf("the size limit must be between 1 and 4096 MB")
	}

	if s.Storage == "" {
		return fmt.Errorf("a storage name is required")
	}

	if s.MaxFolders < defaultMaxFolders || s.MaxFolders > maxFoldersLimit {
		return fmt.Errorf("max_folders must be between %d and %d", defaultMaxFolders, maxFoldersLimit)
	}

	return nil
}
