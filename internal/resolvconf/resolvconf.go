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

// Parse splits data into lines and recognizes valid "nameserver <ip>" entries.
// It never fails: unknown or malformed lines (including "nameserver abc") are kept
// as plain text, because resolv.conf is written by many tools and we must not
// crash or drop content because of a line we do not understand.
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

// SetNameservers replaces all nameserver lines with addrs. The new block takes the
// place of the first old nameserver line; if there was none, it is appended to the end.
// Other lines are not touched.
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

// Render serializes the file back to bytes. Every line ends with "\n",
// and an empty file produces empty output.
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
