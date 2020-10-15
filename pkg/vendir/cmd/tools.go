package cmd

import (
	"github.com/spf13/cobra"
)

func NewToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tools",
		Aliases: []string{"t"},
		Short:   "Tools",
	}
	return cmd
}
