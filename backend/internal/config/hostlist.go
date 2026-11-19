package config

import (
	"fmt"
	"regexp"
	"strings"
)

// HostList adalah daftar nama host (domain) yang boleh dipakai mengakses aplikasi.
// Daftar kosong = semua host diizinkan. Mendukung wildcard satu tingkat awalan: "*.contoh.id".
type HostList struct {
	exact    map[string]bool
	suffixes []string // ".contoh.id" untuk entri "*.contoh.id"
}

var hostRe = regexp.MustCompile(`^(\*\.)?[a-z0-9]([a-z0-9._-]*[a-z0-9])?$`)

// ParseHostList mengurai daftar host dipisah koma. Port dan skema tidak boleh disertakan.
func ParseHostList(entries []string) (*HostList, error) {
	l := &HostList{exact: map[string]bool{}}
	for _, e := range entries {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if strings.ContainsAny(e, "/:@ ") && !strings.HasPrefix(e, "[") {
			return nil, fmt.Errorf("host tidak valid %q (tanpa skema, port, atau path; contoh: link.contoh.id)", e)
		}
		e = strings.Trim(e, "[]") // IPv6 literal, mis. [::1]
		if strings.Contains(e, ":") {
			l.exact[e] = true
			continue
		}
		if !hostRe.MatchString(e) {
			return nil, fmt.Errorf("host tidak valid %q", e)
		}
		if strings.HasPrefix(e, "*.") {
			l.suffixes = append(l.suffixes, e[1:])
			continue
		}
		l.exact[e] = true
	}
	return l, nil
}

func (l *HostList) Enabled() bool { return l != nil && (len(l.exact) > 0 || len(l.suffixes) > 0) }

// Allows memeriksa nama host (tanpa port). Wildcard "*.x.id" cocok untuk subdomain, bukan "x.id" sendiri.
func (l *HostList) Allows(host string) bool {
	if !l.Enabled() {
		return true
	}
	host = strings.Trim(strings.ToLower(strings.TrimSpace(host)), "[]")
	host = strings.TrimSuffix(host, ".") // "contoh.id." = "contoh.id"
	if host == "" {
		return false
	}
	if l.exact[host] {
		return true
	}
	for _, s := range l.suffixes {
		if strings.HasSuffix(host, s) && len(host) > len(s) {
			return true
		}
	}
	return false
}

func (l *HostList) Strings() []string {
	if l == nil {
		return []string{}
	}
	out := make([]string, 0, len(l.exact)+len(l.suffixes))
	for h := range l.exact {
		out = append(out, h)
	}
	for _, s := range l.suffixes {
		out = append(out, "*"+s)
	}
	return out
}
