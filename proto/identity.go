package proto

import (
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strings"
	"unicode/utf8"
)

var validEmoji = map[string]string{}

var possibleEmoji = regexp.MustCompile(":[\\S]+?:")

const MaxNickLength = 36

const (
	ltrEmbed    = '\u202A'
	rtlEmbed    = '\u202B'
	ltrOverride = '\u202D'
	rtlOverride = '\u202E'
	ltrIsolate  = '\u2066'
	rtlIsolate  = '\u2067'
	fsIsolate   = '\u2068'

	bidiExplicitPop = '\u202C'
	bidiIsolatePop  = '\u2069'
)

type UserID string

func (uid UserID) String() string { return string(uid) }

func (uid UserID) Parse() (kind, id string) {
	parts := strings.SplitN(string(uid), ":", 2)
	if len(parts) < 2 {
		return "", string(uid)
	}
	return parts[0], parts[1]
}

// An Identity maps to a global persona. It may exist only in the context
// of a single Room. An Identity may be anonymous.
type Identity interface {
	ID() UserID
	Name() string
	ServerID() string
	View() IdentityView
}

type IdentityView struct {
	ID        UserID `json:"id"`         // the id of an agent or account
	Name      string `json:"name"`       // the name-in-use at the time this view was captured
	ServerID  string `json:"server_id"`  // the id of the server that captured this view
	ServerEra string `json:"server_era"` // the era of the server that captured this view
}

// LoadEmoji takes a json key-value object stored in the file at path and
// unmarshals it into the global validEmoji map[string]string.
func LoadEmoji(path string) error {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &validEmoji); err != nil {
		return err
	}
	return nil
}

func nickLen(nick string) int {
	indices := possibleEmoji.FindAllStringIndex(nick, 36)
	for _, item := range indices {
		s := nick[item[0]+1 : item[1]-1]
		_, ok := validEmoji[s]
		if ok {
			ret := 1 + nickLen(nick[0:item[0]])
			if item[1] < len(nick) {
				ret += nickLen(nick[item[1]:])
			}
			return ret
		}
	}
	return utf8.RuneCountInString(nick)
}

// NormalizeNick validates and normalizes a proposed name from a user.
// If the proposed name is not valid, returns an error. Otherwise, returns
// the normalized form of the name. Normalization for a nick consists of:
//
// 1. Remove leading and trailing whitespace
// 2. Collapse all internal whitespace to single spaces
// 3. Replace all
func NormalizeNick(name string) (string, error) {
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		return "", ErrInvalidNick
	}
	normalized := strings.Join(strings.Fields(name), " ")
	if nickLen(normalized) > MaxNickLength {
		return "", ErrInvalidNick
	}
	return normalizeBidi(normalized), nil
}

// normalizeBidi attempts to prevent names from using bidi control codes to
// screw up our layout
func normalizeBidi(name string) string {
	bidiExplicitDepth := 0
	bidiIsolateDepth := 0

	for _, c := range name {
		switch c {
		case ltrEmbed, rtlEmbed, ltrOverride, rtlOverride:
			bidiExplicitDepth++
		case bidiExplicitPop:
			bidiExplicitDepth--
		case ltrIsolate, rtlIsolate, fsIsolate:
			bidiIsolateDepth++
		case bidiIsolatePop:
			bidiIsolateDepth--
		}
	}
	if bidiExplicitDepth+bidiIsolateDepth > 0 {
		pops := make([]byte,
			bidiExplicitDepth*utf8.RuneLen(bidiExplicitPop)+bidiIsolateDepth+utf8.RuneLen(bidiIsolatePop))
		i := 0
		for ; bidiExplicitDepth > 0; bidiExplicitDepth-- {
			i += utf8.EncodeRune(pops[i:], bidiExplicitPop)
		}
		for ; bidiIsolateDepth > 0; bidiIsolateDepth-- {
			i += utf8.EncodeRune(pops[i:], bidiIsolatePop)
		}
		return name + string(pops[:i])
	}
	return name
}
