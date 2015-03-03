package proto

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const MaxNickLength = 36

// An Identity maps to a global persona. It may exist only in the context
// of a single Room. An Identity may be anonymous.
type Identity interface {
	ID() string
	Name() string
	ServerID() string
	View() *IdentityView
}

type IdentityView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ServerID  string `json:"server_id"`
	ServerEra string `json:"server_era"`
}

// NormalizeNick validates and normalizes a proposed name from a user.
// If the proposed name is not valid, returns an error. Otherwise, returns
// the normalized form of the name. Normalization for a nick consists of:
//
// 1. Remove leading and trailing whitespace
// 2. Collapse all internal whitespace to single spaces
func NormalizeNick(name string) (string, error) {
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		return "", fmt.Errorf("invalid nick")
	}
	normalized := strings.Join(strings.Fields(name), " ")
	if utf8.RuneCountInString(normalized) > MaxNickLength {
		return "", fmt.Errorf("invalid nick")
	}
	return normalized, nil
}
