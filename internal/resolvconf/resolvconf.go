package resolvconf

import (
	"net/netip"
	"strings"
)

type line struct {
	raw  string
	addr netip.Addr
	isNS bool
}

type File struct {
	lines []line
}

// Parse raws if cant parse, than save raw as it was
func Parse(data []byte) *File {
	f := &File{}
	text := strings.TrimRight(string(data), "\r\n")
	if text == "" {
		return f
	}
	for _, raw := range strings.Split(text, "\n") {
		f.lines = append(f.lines, parseLine(strings.TrimRight(raw, "\r")))
	}
	return f
}

func parseLine(raw string) line {
	fields := strings.Fields(raw)
	if len(fields) >= 2 && fields[0] == "nameserver" {
		addr, err := netip.ParseAddr(fields[1])
		if err == nil {

			return line{
				raw:  raw,
				addr: addr,
				isNS: true,
			}
		}
	}
	return line{
		raw: raw,
	}
}

func (f *File) Nameservers() []netip.Addr {
	var res []netip.Addr
	for _, line := range f.lines {
		if line.isNS {
			res = append(res, line.addr)
		}
	}
	return res
}

// SetNameservers replace all nameservers raws
func (f *File) SetNameservers(adrrs []netip.Addr) {
	out := make([]line, 0, len(f.lines)+len(adrrs))
	placed := false

	for _, l := range f.lines {
		if !l.isNS {
			out = append(out, l)
			continue
		}
		if !placed {
			out = appendNS(out, adrrs)
			placed = true
		}
	}
	if !placed {
		out = appendNS(out, adrrs)
	}
	f.lines = out
}

func appendNS(dst []line, addrs []netip.Addr) []line {
	for _, a := range addrs {
		dst = append(dst, line{raw: "nameserver " + a.String(), addr: a, isNS: true})
	}
	return dst
}

func (f *File) Render() []byte {
	if len(f.lines) == 0 {
		return nil
	}
	var b strings.Builder
	for _, l := range f.lines {
		b.WriteString(l.raw)
		b.WriteByte('\n')
	}
	return []byte(b.String())
}
