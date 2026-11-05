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
