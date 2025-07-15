package choiceflag

import (
	"fmt"
	"slices"
	"strings"
)

// ChoiceFlag is a custom flag type for limited choice options
type ChoiceFlag struct {
	choices []string
	value   string
}

// String returns the string representation of the flag value
func (c *ChoiceFlag) String() string {
	return c.value
}

// Set validates and sets the flag value
func (c *ChoiceFlag) Set(value string) error {
	if slices.Contains(c.choices, value) {
		c.value = value
		return nil
	}
	return fmt.Errorf("valid choices are: [%s]", strings.Join(c.choices, ", "))
}

// Type returns the type of the flag
func (e *ChoiceFlag) Type() string {
	return strings.Join(e.choices, "|")
}

// NewChoiceFlag creates a new ChoiceFlag with the given choices
func NewChoiceFlag(choices []string) *ChoiceFlag {
	return &ChoiceFlag{
		choices: choices,
	}
}
