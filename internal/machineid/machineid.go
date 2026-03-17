package machineid

import (
	"fmt"
	"hash/fnv"
	"net"
	"os"
)

// Resolve determines a node ID (0–1023) from the environment.
//
// Resolution order:
//  1. SNOWFLAKE_NODE_ID env var (handled externally via config, not here)
//  2. Kubernetes: hash HOSTNAME (pod name) mod 1024
//  3. Fallback: lower 10 bits of the first private IPv4 address
func Resolve() (int64, error) {
	// Strategy 1: Kubernetes pod name
	if hostname := os.Getenv("HOSTNAME"); hostname != "" {
		if _, inK8s := os.LookupEnv("KUBERNETES_SERVICE_HOST"); inK8s {
			return hashMod(hostname, 1024), nil
		}
	}

	// Strategy 2: Lower 10 bits of private IP
	if id, err := fromPrivateIP(); err == nil {
		return id, nil
	}

	return 0, fmt.Errorf("machineid: unable to auto-detect node ID; set SNOWFLAKE_NODE_ID explicitly")
}

func hashMod(s string, mod int64) int64 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int64(h.Sum32()) % mod
}

func fromPrivateIP() (int64, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return 0, err
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipNet.IP.To4()
		if ip == nil || ip.IsLoopback() {
			continue
		}
		// Use lower 10 bits from the last two octets
		return (int64(ip[2])<<8 | int64(ip[3])) & 0x3FF, nil
	}

	return 0, fmt.Errorf("machineid: no suitable private IP found")
}
