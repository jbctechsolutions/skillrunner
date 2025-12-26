// Package session provides domain entities for session management.
package session

import (
	"fmt"
	"math/rand"
	"time"
)

var (
	// adjectives for session names
	adjectives = []string{
		"brave", "swift", "bold", "keen", "calm",
		"wise", "quick", "bright", "steady", "sharp",
		"clever", "nimble", "proud", "noble", "fierce",
		"gentle", "mighty", "agile", "astute", "daring",
	}

	// pioneers for session names
	pioneers = []string{
		"turing", "lovelace", "hopper", "dijkstra", "knuth",
		"ritchie", "thompson", "backus", "codd", "shannon",
		"neumann", "babbage", "boole", "church", "curry",
		"edsger", "grace", "ada", "alan", "dennis",
	}

	rng *rand.Rand
)

func init() {
	// Initialize random number generator with current time
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// GenerateSessionName generates a session name in the format "adjective-pioneer"
// Examples: "brave-turing", "swift-lovelace", "bold-hopper"
func GenerateSessionName() string {
	adjective := adjectives[rng.Intn(len(adjectives))]
	pioneer := pioneers[rng.Intn(len(pioneers))]
	return fmt.Sprintf("%s-%s", adjective, pioneer)
}

// GenerateUniqueSessionName generates a unique session name with an optional numeric suffix
// to ensure uniqueness. It takes a function that checks if a name already exists.
func GenerateUniqueSessionName(exists func(string) bool) string {
	// Try without suffix first
	name := GenerateSessionName()
	if !exists(name) {
		return name
	}

	// Try up to 100 times to find a unique name
	for i := 0; i < 100; i++ {
		name = GenerateSessionName()
		if !exists(name) {
			return name
		}
	}

	// If still not unique, add a timestamp suffix
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%d", GenerateSessionName(), timestamp)
}

// IsValidSessionName checks if a session name follows the expected format
func IsValidSessionName(name string) bool {
	if name == "" {
		return false
	}

	// Basic validation: should contain at least one hyphen
	// More detailed validation could be added here
	for i := 0; i < len(name); i++ {
		if name[i] == '-' {
			return true
		}
	}

	return false
}
