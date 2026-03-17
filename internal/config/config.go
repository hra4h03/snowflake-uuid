package config

import (
	"flag"
	"os"
	"strconv"
)

// Config holds all service configuration.
type Config struct {
	NodeID     int64
	ListenAddr string
	AutoNodeID bool
	EpochMs    int64
}

// Load parses configuration from environment variables and command-line flags.
// Flags take precedence over environment variables over defaults.
func Load() *Config {
	c := &Config{}

	// Defaults, overridden by env vars if set
	defaultNodeID := int64(-1) // -1 signals "not set"
	if v, ok := os.LookupEnv("SNOWFLAKE_NODE_ID"); ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			defaultNodeID = n
		}
	}

	defaultAddr := ":8080"
	if v, ok := os.LookupEnv("SNOWFLAKE_LISTEN_ADDR"); ok {
		defaultAddr = v
	}

	defaultAutoNodeID := false
	if v, ok := os.LookupEnv("SNOWFLAKE_AUTO_NODE_ID"); ok {
		defaultAutoNodeID = v == "true" || v == "1"
	}

	defaultEpochMs := int64(1704067200000) // 2024-01-01 00:00:00 UTC
	if v, ok := os.LookupEnv("SNOWFLAKE_EPOCH_MS"); ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			defaultEpochMs = n
		}
	}

	flag.Int64Var(&c.NodeID, "node-id", defaultNodeID, "machine/node ID (0-1023)")
	flag.StringVar(&c.ListenAddr, "listen-addr", defaultAddr, "HTTP listen address")
	flag.BoolVar(&c.AutoNodeID, "auto-node-id", defaultAutoNodeID, "auto-detect node ID from environment")
	flag.Int64Var(&c.EpochMs, "epoch", defaultEpochMs, "custom epoch in Unix milliseconds")
	flag.Parse()

	return c
}
