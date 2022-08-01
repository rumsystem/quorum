package utils

import (
	"net"
	"regexp"
)

func IsPublicIP(v string) bool {
	ip := net.ParseIP(v)
	if ip == nil {
		return false
	}

	if !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsUnspecified() {
		return true
	}
	return false
}

func IsDomainName(domain string) bool {
	RegExp := regexp.MustCompile(`^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z]{2,3})$`)

	return RegExp.MatchString(domain)
}
