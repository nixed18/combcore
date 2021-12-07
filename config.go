package main

import (
	"flag"
)

var (
	btc_host     = flag.String("btc_host", "127.0.0.1", "IP/Hostname of COMBs BTC peer")
	btc_port     = flag.Uint("btc_port", 8332, "Port of BTC peer RPC interface")
	btc_username = flag.String("btc_user", "user", "Username for BTC peer RPC interface")
	btc_password = flag.String("btc_pass", "pass", "Password for BTC peer RPC interface")
	btc_data     = flag.String("btc_data", "", "Local BTC data directory")

	comb_host      = flag.String("comb_host", "127.0.0.1", "Bind addresss for COMBCores RPC interface")
	comb_port      = flag.Uint("comb_port", 2211, "Port for COMBCores RPC interface")
	comb_username  = flag.String("comb_user", "user", "Username for COMBCores Control RPC interface")
	comb_password  = flag.String("comb_pass", "pass", "Password for COMBCores Control RPC interface")
	comb_whitelist = flag.String("comb_whitelist", "127.0.0.1", "IP Whitelist for COMBCores Control RPC interface")

	comb_fingerprint_index = flag.Bool("comb_fingerprint_index", false, "Keep block fingerprint index in memory")
)
