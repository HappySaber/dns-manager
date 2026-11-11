package model

// State is state of dns server and it's version
type State struct {
	Revision uint64
	Origin   string
	Servers  []string
}

func (s State) NewerThan(o State) bool {
	if s.Revision != o.Revision {
		return s.Revision > o.Revision
	}
	return s.Origin > o.Origin
}
