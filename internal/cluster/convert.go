package cluster

import (
	"slices"

	dnsv1 "dnsmanager/gen/dns-manager"
	"dnsmanager/internal/model"
)

func toProto(st model.State) *dnsv1.NodeState {
	return &dnsv1.NodeState{Revision: st.Revision, Origin: st.Origin, Servers: slices.Clone(st.Servers)}
}

func fromProto(p *dnsv1.NodeState) model.State {
	return model.State{Revision: p.GetRevision(), Origin: p.GetOrigin(), Servers: slices.Clone(p.GetServers())}
}
