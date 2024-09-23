package choiceflag

import (
	"strings"
	"testing"
)

func TestChoiceFlag_Set(t *testing.T) {
	tests := []struct {
		name      string
		choices   []string
		value     string
		expectErr bool
	}{
		{
			name:    "Valid choice",
			choices: []string{"option1", "option2", "option3"},
			value:   "option1",
		},
		{
			name:      "Invalid choice",
			choices:   []string{"option1", "option2", "option3"},
			value:     "invalid",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf := NewChoiceFlag(tt.choices)
			err := cf.Set(tt.value)
			if (err != nil) != tt.expectErr {
				t.Errorf("Set() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr && cf.String() != tt.value {
				t.Errorf("Set() = %v, want %v", cf.String(), tt.value)
			}
		})
	}
}

func TestChoiceFlag_String(t *testing.T) {
	cf := NewChoiceFlag([]string{"option1", "option2"})
	cf.Set("option1")
	if cf.String() != "option1" {
		t.Errorf("String() = %v, want %v", cf.String(), "option1")
	}
}

func TestChoiceFlag_Type(t *testing.T) {
	choices := []string{"option1", "option2"}
	cf := NewChoiceFlag(choices)
	if cf.Type() != strings.Join(choices, "|") {
		t.Errorf("Type() = %v, want %v", cf.Type(), "string")
	}
}

func TestNewChoiceFlag(t *testing.T) {
	choices := []string{"option1", "option2"}
	cf := NewChoiceFlag(choices)
	if len(cf.choices) != len(choices) {
		t.Errorf("NewChoiceFlag() choices = %v, want %v", cf.choices, choices)
	}
}
