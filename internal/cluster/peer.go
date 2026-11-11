package cluster

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	dnsv1 "dnsmanager/gen/dns-manager"
	"dnsmanager/internal/model"
)

type Peer struct {
	conn   *grpc.ClientConn
	client dnsv1.ClusterServiceClient
}

func NewPeer(addr string, extra ...grpc.DialOption) (*Peer, error) {
	opts := append([]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, extra...)
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("create peer client for %s: %w", addr, err)
	}
	return &Peer{conn: conn, client: dnsv1.NewClusterServiceClient(conn)}, nil
}

func (p *Peer) Close() error { return p.conn.Close() }

func (p *Peer) GetState(ctx context.Context) (model.State, error) {
	resp, err := p.client.GetState(ctx, &dnsv1.GetStateRequest{})
	if err != nil {
		return model.State{}, fmt.Errorf("peer GetState: %w", err)
	}
	return fromProto(resp.GetState()), nil
}

func (p *Peer) Replicate(ctx context.Context, st model.State) error {
	if _, err := p.client.Replicate(ctx, &dnsv1.ReplicateRequest{State: toProto(st)}); err != nil {
		return fmt.Errorf("peer Replicate: %w", err)
	}
	return nil
}
