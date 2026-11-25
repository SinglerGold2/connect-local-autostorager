package main

// The imports that survive a restart. The card reads the newest ten and the
// page the newest twenty-five; both come from this table, not from a slice
// that a restart would empty.

import (
	"context"
	"errors"
	"fmt"
)

// loadImports returns the newest imports, oldest first, so the page can append
// a new one at the end.
func loadImports(ctx context.Context) ([]Import, error) {
	if db == nil {
		return nil, errors.New("this instance has no database")
	}

	rows, err := db.QueryContext(ctx, `SELECT at, folder, source, target
		FROM autostorager_imports ORDER BY id DESC LIMIT ?`, recentLimit)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	imports := make([]Import, 0, recentLimit)
	for rows.Next() {
		var entry Import
		if err := rows.Scan(&entry.At, &entry.Folder, &entry.Source, &entry.Target); err != nil {
			return nil, err
		}

		entry.Result = "imported"
		imports = append(imports, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	for left, right := 0, len(imports)-1; left < right; left, right = left+1, right-1 {
		imports[left], imports[right] = imports[right], imports[left]
	}

	return imports, nil
}

// recordImport writes one committed import. A failure here does not undo the
// import; it only means the page will not remember it.
func recordImport(ctx context.Context, entry Import) error {
	if db == nil {
		return errors.New("this instance has no database")
	}

	_, err := db.ExecContext(ctx, `INSERT INTO autostorager_imports (at, folder, source, target)
		VALUES (?, ?, ?, ?)`, entry.At, entry.Folder, entry.Source, entry.Target)
	if err != nil {
		return fmt.Errorf("record import: %w", err)
	}

	return nil
}
