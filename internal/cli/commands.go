package cli

import (
	"context"
	dnsv1 "dnsmanager/gen/dns-manager"
	"fmt"

	"github.com/spf13/cobra"
)

func newListCmd(o *options) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show configured DNS servers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return o.call(cmd, func(ctx context.Context, c dnsv1.DNSServiceClient) error {
				resp, err := c.ListServers(ctx, &dnsv1.ListServersRequest{})
				if err != nil {
					return err
				}
				for _, a := range resp.GetAdresses() {
					fmt.Fprintln(cmd.OutOrStdout(), a)
				}
				return nil
			})
		},
	}
}

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Show configured DNS servers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Println("newaddcom")
			return nil
		},
	}
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Show configured DNS servers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return nil
		},
	}
}
