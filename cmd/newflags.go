package cmd

import (
	"flag"
	"strings"
)

type NewFlagsValue struct {
	FlagSet *flag.FlagSet
	Flags   []string
}

func (v *NewFlagsValue) String() string { return strings.Join(v.Flags, ",") }

func (v *NewFlagsValue) Set(flags string) error {
	// parse the list of potential new flags
	v.Flags = strings.Split(flags, ",")

	// define the given flags if they haven't been defined already
	for _, flagName := range v.Flags {
		if v.FlagSet.Lookup(flagName) == nil {
			v.FlagSet.String(flagName, "", "not yet implemented")
		}
	}

	return nil
}

// Install -newflags on the default flagset.
var newFlags = NewFlagsFlag(flag.CommandLine, "newflags")

// NewFlagsFlag returns a special flag value installed on the given flagset
// with the given name. When this flag is encountered while parsing, it
// ensures that a definition for all the given flag names exists.
func NewFlagsFlag(flagSet *flag.FlagSet, name string) *NewFlagsValue {
	v := &NewFlagsValue{FlagSet: flagSet}
	flagSet.Var(v, name, "comma-separated list of flag names to ignore errors on")
	return v
}
