//go:build !windows && !darwin && !linux

package config

func GetMachineID() (string, error) {
	return "unknown-platform", nil
}
