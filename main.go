package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

func main() {
	f, _ := os.OpenFile("combcore.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer f.Close()
	wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(wrt)

	var err error

	combcore_set_status("Initializing...")
	combcore_init()
	neominer_init()
	rpc_start()

	if err = db_open(); err != nil {
		log.Fatal(err)
	}
	combcore_set_status("Loading...")
	db_start()
	combcore_set_status("Idle")

	fmt.Println("Nodetype = ", fmt.Sprint(*node_mode))

	// Start here to prevent db load collisions
	go ghetto_rpc()

	// Limits db mining race conditions; this is shit code and there's a more elegant way to do this, but it works for now
	switch *node_mode {
	case FULL_NODE:
		btc_init()
		for {
			btc_sync()
			combcore_set_status("Idle")
			time.Sleep(time.Second * 10)
		}

	case MID_NODE:
		// Insert the appropriate peer check when it exists
		for {
			time.Sleep(time.Second * 10)
		}
		
	default:
		for {
			time.Sleep(time.Second * 10)
		}
	}
}
