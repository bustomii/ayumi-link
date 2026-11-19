package config

import (
	"fmt"
	"net/netip"
	"strings"
)

// IPList adalah daftar alamat/CIDR yang diizinkan. Daftar kosong = semua diizinkan.
type IPList struct {
	prefixes []netip.Prefix
}

// ParseIPList mengurai daftar dipisah koma, mis. "203.0.113.7, 192.168.1.0/24, ::1".
func ParseIPList(entries []string) (*IPList, error) {
	l := &IPList{}
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if strings.Contains(e, "/") {
			p, err := netip.ParsePrefix(e)
			if err != nil {
				return nil, fmt.Errorf("entri CIDR tidak valid %q", e)
			}
			l.prefixes = append(l.prefixes, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(e)
		if err != nil {
			return nil, fmt.Errorf("alamat IP tidak valid %q", e)
		}
		a = a.Unmap()
		l.prefixes = append(l.prefixes, netip.PrefixFrom(a, a.BitLen()))
	}
	return l, nil
}

// Enabled: apakah ada pembatasan (daftar tidak kosong).
func (l *IPList) Enabled() bool { return l != nil && len(l.prefixes) > 0 }

// Allows memeriksa sebuah alamat. Alamat tidak valid selalu ditolak bila pembatasan aktif.
func (l *IPList) Allows(ip string) bool {
	if !l.Enabled() {
		return true
	}
	a, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return false
	}
	a = a.Unmap() // ::ffff:1.2.3.4 dianggap 1.2.3.4
	for _, p := range l.prefixes {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// Strings mengembalikan daftar dalam bentuk teks (untuk ditampilkan).
func (l *IPList) Strings() []string {
	if l == nil {
		return []string{}
	}
	out := make([]string, 0, len(l.prefixes))
	for _, p := range l.prefixes {
		out = append(out, p.String())
	}
	return out
}
