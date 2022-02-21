package main

import (
	"flag"
)

var (
	btc_peer = flag.String("btc_peer", "127.0.0.1", "")
	btc_port = flag.Uint("btc_port", 8332, "")
	btc_data = flag.String("btc_data", "", "")

	comb_host    = flag.String("comb_host", "127.0.0.1", "")
	comb_port    = flag.Uint("comb_port", 2211, "")
	comb_network = flag.String("comb_network", "mainnet", "")

	comb_fingerprint_index = flag.Bool("comb_fingerprint_index", false, "")
)
