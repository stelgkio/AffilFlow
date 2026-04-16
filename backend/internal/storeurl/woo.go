package storeurl

import "strings"

// NormalizeWooSiteBase trims space and trailing slashes (no scheme changes).
func NormalizeWooSiteBase(u string) string {
	u = strings.TrimSpace(u)
	u = strings.TrimSuffix(u, "/")
	return u
}

// WooSiteCandidates returns URL forms to match WooCommerce X-WC-Webhook-Source against stored site_base_url.
func WooSiteCandidates(source string) []string {
	u := NormalizeWooSiteBase(source)
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(u)
	add(u + "/")
	if strings.HasPrefix(u, "https://") {
		add("http://" + strings.TrimPrefix(u, "https://"))
		add("http://" + strings.TrimPrefix(u, "https://") + "/")
	} else if strings.HasPrefix(u, "http://") {
		add("https://" + strings.TrimPrefix(u, "http://"))
		add("https://" + strings.TrimPrefix(u, "http://") + "/")
	}
	return out
}
