package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Boards whose BIOS was never given a real UUID report one of these, so every
// unit of the same model would claim to be the same machine. The AMI default is
// listed in both byte orders because SMBIOS readers disagree on the first three
// fields.
var placeholderMachineIDs = map[string]bool{
	"00000000-0000-0000-0000-000000000000": true,
	"ffffffff-ffff-ffff-ffff-ffffffffffff": true,
	"03000200-0400-0500-0006-000700080009": true,
	"00020003-0004-0005-0006-000700080009": true,
}

func isUsableMachineID(id string) bool {
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return false
	}
	return !placeholderMachineIDs[parsed.String()]
}

// loadOrCreateMachineID is the last resort for a PC that exposes no usable ID.
// An ID that could not be saved is never returned: it would change on every
// restart and break the PC's terminal binding each time.
func loadOrCreateMachineID(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err == nil && isUsableMachineID(string(data)) {
		return strings.TrimSpace(string(data)), nil
	}
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("read generated machine ID: %w", err)
	}

	id := uuid.NewString()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("create machine ID dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(id+"\n"), 0644); err != nil {
		return "", fmt.Errorf("write generated machine ID: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", fmt.Errorf("save generated machine ID: %w", err)
	}
	return id, nil
}
