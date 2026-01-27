//go:build !windows && !darwin

package server

import "fmt"

func playSystemSound() {
	// Fallback to terminal bell for other systems (Linux, etc)
	fmt.Print("\a")
}
