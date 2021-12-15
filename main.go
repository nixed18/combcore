package main

import (
	"log"
)

func main() {
	var err error
	combcore_init()

	if err = db_open(); err != nil {
		log.Fatal(err)
	}

	db_start()
	rpc_start()

	btc_init()
	for {
		btc_sync()
	}
}
