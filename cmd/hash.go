package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// RunHashPassword reads a password from stdin and outputs its bcrypt hash.
func RunHashPassword() error {
	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read password from stdin: %w", err)
	}

	// Trim newline
	password = strings.TrimSuffix(password, "\n")
	password = strings.TrimSuffix(password, "\r")

	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	// Generate bcrypt hash
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %w", err)
	}

	// Output just the hash
	fmt.Println(string(hash))
	return nil
}
