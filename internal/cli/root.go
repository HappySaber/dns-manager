package cli

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {

	root := &cobra.Command{
		Use:   "dns-cli",
		Short: "Manage DNS servers on the storage system",
	}

	root.AddCommand(newListCmd(), newAddCmd(), newRemoveCmd())
	return root
}
