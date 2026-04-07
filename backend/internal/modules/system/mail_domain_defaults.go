package system

import (
	"net"
	"net/url"
	"strings"
)

const (
	legacySMTPHostname = "mail.shiro.local"
	legacyDKIMTarget   = "shiro._domainkey.shiro.local"
)

var commonSecondLevelSuffixes = map[string]struct{}{
	"ac":  {},
	"co":  {},
	"com": {},
	"edu": {},
	"gov": {},
	"net": {},
	"org": {},
}

func deriveMailTargetsFromAppBaseURL(appBaseURL string) (string, string, bool) {
	rootDomain := deriveRootDomainFromAppBaseURL(appBaseURL)
	if rootDomain == "" {
		return "", "", false
	}
	return "smtp." + rootDomain, "shiro._domainkey." + rootDomain, true
}

func deriveRootDomainFromAppBaseURL(appBaseURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(appBaseURL))
	if err != nil {
		return ""
	}

	host := strings.TrimSpace(strings.ToLower(parsed.Hostname()))
	if host == "" || host == "localhost" || net.ParseIP(host) != nil {
		return ""
	}

	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return ""
	}
	if len(labels) == 2 {
		return host
	}

	last := labels[len(labels)-1]
	secondLast := labels[len(labels)-2]
	if len(last) == 2 {
		if _, ok := commonSecondLevelSuffixes[secondLast]; ok && len(labels) >= 3 {
			return strings.Join(labels[len(labels)-3:], ".")
		}
	}

	return strings.Join(labels[len(labels)-2:], ".")
}

func shouldUseDerivedMailTarget(value string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	return trimmed == "" || trimmed == legacySMTPHostname || trimmed == legacyDKIMTarget
}
