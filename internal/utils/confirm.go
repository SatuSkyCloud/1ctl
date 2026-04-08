package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm asks the user for y/N confirmation. Returns true if confirmed.
// Returns true immediately when yes is true (--yes flag).
func Confirm(prompt string, yes bool) bool {
	if yes {
		return true
	}
	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
