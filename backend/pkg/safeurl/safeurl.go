package safeurl

import (
	"net/url"
	"strings"
)

// RedirectTarget returns next if its host is allowlisted; otherwise defaultURL.
func RedirectTarget(allowHosts []string, defaultURL, next string) string {
	if next == "" {
		return defaultURL
	}
	u, err := url.Parse(next)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return defaultURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return defaultURL
	}
	host := strings.ToLower(u.Hostname())
	for _, h := range allowHosts {
		h = strings.TrimSpace(strings.ToLower(h))
		if h == "" {
			continue
		}
		// allow "localhost:3001" style in allow list
		allowHost, _, _ := strings.Cut(h, ":")
		if host == strings.ToLower(allowHost) || host == h {
			return next
		}
		if strings.Contains(h, ":") && strings.ToLower(u.Host) == h {
			return next
		}
	}
	return defaultURL
}
