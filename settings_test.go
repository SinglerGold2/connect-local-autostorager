package main

import "testing"

func TestNormalizedUsesTheConfiguredLimit(t *testing.T) {
	folders := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}

	if got := normalized(folders, 4); len(got) != 4 {
		t.Fatalf("normalized(..., 4) kept %d folders, want 4", len(got))
	}

	if got := normalized(folders, 10); len(got) != 10 {
		t.Fatalf("normalized(..., 10) kept %d folders, want 10", len(got))
	}
}

func TestWithDefaultsClampsMaxFolders(t *testing.T) {
	if got := withDefaults(Settings{MaxFolders: 0}).MaxFolders; got != defaultMaxFolders {
		t.Fatalf("an unset limit became %d, want %d", got, defaultMaxFolders)
	}

	if got := withDefaults(Settings{MaxFolders: 10}).MaxFolders; got != 10 {
		t.Fatalf("a limit of 10 became %d", got)
	}

	if got := withDefaults(Settings{MaxFolders: maxFoldersLimit + 1}).MaxFolders; got != defaultMaxFolders {
		t.Fatalf("an oversized limit became %d, want the default", got)
	}
}
