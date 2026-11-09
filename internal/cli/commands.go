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

func newAddCmd(o *options) *cobra.Command {
	return &cobra.Command{
		Use:   "add <ip>",
		Short: "Add a DNS server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.call(cmd, func(ctx context.Context, c dnsv1.DNSServiceClient) error {
				_, err := c.AddServer(ctx, &dnsv1.AddServerRequest{Adresses: args[0]})
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "ok")
				return nil
			})
		},
	}
}

func newRemoveCmd(o *options) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <ip>",
		Short: "Remove a DNS server by ip",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.call(cmd, func(ctx context.Context, c dnsv1.DNSServiceClient) error {
				_, err := c.RemoveServer(ctx, &dnsv1.RemoveServerRequest{Adresses: args[0]})
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "ok")
				return nil
			})
		},
	}
}
