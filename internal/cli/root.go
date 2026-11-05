package cli

import (
	"context"
	dnsv1 "dnsmanager/gen/dns-manager"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type options struct {
	addr    string
	timeout time.Duration
}

func NewRootCmd() *cobra.Command {
	opts := &options{}

	root := &cobra.Command{
		Use:   "dns-cli",
		Short: "Manage DNS servers on the storage system",
	}
	root.PersistentFlags().StringVar(&opts.addr, "addr", "localhost:50051", "dnsd server address")
	root.PersistentFlags().DurationVar(&opts.timeout, "timeout", 5*time.Second, "request timeout")

	root.AddCommand(newListCmd(opts), newAddCmd(opts), newRemoveCmd(opts))
	return root
}

func (o *options) call(cmd *cobra.Command, fn func(ctx context.Context, c dnsv1.DNSServiceClient) error) error {
	conn, err := grpc.NewClient(o.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("connect to %s: %w", o.addr, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(cmd.Context(), o.timeout)
	defer cancel()

	return fn(ctx, dnsv1.NewDNSServiceClient(conn))
}

func Message(err error) string {
	if s, ok := status.FromError(err); ok {
		return s.Message()
	}
	return err.Error()
}
