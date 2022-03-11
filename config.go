package main

import (
	"flag"
)

const (
	FULL_NODE = 0 // Has its own BTC chain to reference
	MID_NODE = 1	// Has a list of truested COMB peers to pull commits from and build a local DB
	MID_NODE_REMOTE = 2 // Relies on external data pushing to build own DB
	LIGHT_NODE = 3 // Has a list of trusted COMB peers top query individual comit statuses from
)

var (
	btc_peer = flag.String("btc_peer", "127.0.0.1", "")
	btc_port = flag.Uint("btc_port", 8332, "")
	btc_data = flag.String("btc_data", "", "")

	comb_host    = flag.String("comb_host", "127.0.0.1", "")
	comb_port    = flag.Uint("comb_port", 2211, "")
	comb_network = flag.String("comb_network", "mainnet", "")

	comb_fingerprint_index = flag.Bool("comb_fingerprint_index", false, "")

	public_api_bind = flag.String("public_api_bind", "", "")
	private_api_bind = flag.String("private_api_bind", "", "")
	node_mode = flag.Uint("node_mode", 0, "")
)
