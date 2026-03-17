package main

import (
	"log"
	"time"

	"github.com/hra4h03/snowflake-uuid/internal/config"
	"github.com/hra4h03/snowflake-uuid/internal/machineid"
	"github.com/hra4h03/snowflake-uuid/internal/server"
	"github.com/hra4h03/snowflake-uuid/snowflake"
)

func main() {
	cfg := config.Load()

	// Resolve node ID
	nodeID := cfg.NodeID
	if nodeID < 0 {
		if cfg.AutoNodeID {
			resolved, err := machineid.Resolve()
			if err != nil {
				log.Fatalf("auto node ID resolution failed: %v", err)
			}
			nodeID = resolved
			log.Printf("auto-detected node ID: %d", nodeID)
		} else {
			log.Fatal("node ID is required: set -node-id flag, SNOWFLAKE_NODE_ID env var, or enable -auto-node-id")
		}
	}

	epoch := time.UnixMilli(cfg.EpochMs)

	gen, err := snowflake.New(nodeID, snowflake.WithEpoch(epoch))
	if err != nil {
		log.Fatalf("failed to create generator: %v", err)
	}

	if err := server.Run(cfg.ListenAddr, gen, epoch); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
