package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_addDefaultCommandIfNeeded(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			"get args as is when blueprint command is included",
			[]string{"xl-bp", "blueprint"},
			[]string{"xl-bp", "blueprint"},
		},
		{
			"get default when no subcommand is included",
			[]string{"xl-bp"},
			[]string{"xl-bp", "blueprint"},
		},
		{
			"get args as is when a subcommand is included",
			[]string{"xl-bp", "version"},
			[]string{"xl-bp", "version"},
		},
		{
			"get args as is when a subcommand is included with flags",
			[]string{"xl-bp", "version", "-v"},
			[]string{"xl-bp", "version", "-v"},
		},
		{
			"get args as is when a subcommand is included with flags in different order",
			[]string{"xl-bp", "-v", "version", "-h"},
			[]string{"xl-bp", "-v", "version", "-h"},
		},
		{
			"get default when command with flags",
			[]string{"xl-bp", "-v"},
			[]string{"xl-bp", "blueprint", "-v"},
		},
		{
			"get default when command with multiple flags",
			[]string{"xl-bp", "-v", "-h"},
			[]string{"xl-bp", "blueprint", "-v", "-h"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addDefaultCommandIfNeeded(tt.args)
			assert.Equal(t, tt.want, got)
		})
	}
}
