package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsUsableMachineID(t *testing.T) {
	cases := map[string]bool{
		"6f1c2a3e-8b4d-4e2f-9a1b-0c3d5e7f9a2b":   true,
		"{6F1C2A3E-8B4D-4E2F-9A1B-0C3D5E7F9A2B}": true,
		"03000200-0400-0500-0006-000700080009":   false,
		"00020003-0004-0005-0006-000700080009":   false,
		"00000000-0000-0000-0000-000000000000":   false,
		"FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF":   false,
		"  03000200-0400-0500-0006-000700080009": false,
		"To be filled by O.E.M.":                 false,
		"":                                       false,
	}
	for id, want := range cases {
		if got := isUsableMachineID(id); got != want {
			t.Errorf("isUsableMachineID(%q) = %v, want %v", id, got, want)
		}
	}
}

func TestLoadOrCreateMachineIDIsStableAcrossCalls(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ts-escpos", "machine-id")
	first, err := loadOrCreateMachineID(path)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if !isUsableMachineID(first) {
		t.Fatalf("generated ID %q is not usable", first)
	}
	second, err := loadOrCreateMachineID(path)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if second != first {
		t.Fatalf("second call = %q, want %q", second, first)
	}
}

func TestLoadOrCreateMachineIDReplacesUnusableFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "machine-id")
	if err := os.WriteFile(path, []byte("03000200-0400-0500-0006-000700080009\n"), 0644); err != nil {
		t.Fatal(err)
	}
	id, err := loadOrCreateMachineID(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !isUsableMachineID(id) {
		t.Fatalf("got unusable ID %q", id)
	}
	saved, _ := os.ReadFile(path)
	if strings.TrimSpace(string(saved)) != id {
		t.Fatalf("saved %q, want %q", saved, id)
	}
}

// An ID that cannot be persisted would differ on the next start, so the call
// must fail instead of returning one.
func TestLoadOrCreateMachineIDFailsWhenItCannotSave(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, nil, 0644); err != nil {
		t.Fatal(err)
	}
	id, err := loadOrCreateMachineID(filepath.Join(blocker, "machine-id"))
	if err == nil {
		t.Fatalf("got ID %q, want an error", id)
	}
}
