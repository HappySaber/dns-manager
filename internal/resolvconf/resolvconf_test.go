package resolvconf

import (
	"slices"
	"testing"
)

func TestParse_Nameservers(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"single", "nameserver 8.8.8.8\n", []string{"8.8.8.8"}},
		{"ipv6", "nameserver 2001:4860:4860::8888\n", []string{"2001:4860:4860::8888"}},
		{"comments and blanks", "# c\n\nnameserver 1.1.1.1\n; x\n", []string{"1.1.1.1"}},
		{"extra spaces and tabs", "nameserver\t  9.9.9.9  \n", []string{"9.9.9.9"}},
		{"trailing comment", "nameserver 8.8.4.4 # google\n", []string{"8.8.4.4"}},
		{"invalid ignored", "nameserver abc\nnameserver 1.1.1.1\n", []string{"1.1.1.1"}},
		{"no trailing newline", "nameserver 8.8.8.8", []string{"8.8.8.8"}},
		{"crlf", "nameserver 8.8.8.8\r\n", []string{"8.8.8.8"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, a := range Parse([]byte(tt.in)).Nameservers() {
				got = append(got, a.String())
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
