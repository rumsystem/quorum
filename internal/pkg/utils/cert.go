package utils

import "net"

func GetPublicIPs(ips []net.IP) []net.IP {
	pubIps := []net.IP{}
	for _, v := range ips {
		if !v.IsPrivate() && !v.IsLoopback() {
			pubIps = append(pubIps, v)
		}
	}

	return pubIps
}
