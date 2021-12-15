package main

import (
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
	combcore_init()

	if err = db_open(); err != nil {
		log.Fatal(err)
	}

	db_start()
	rpc_start()

	btc_init()
	for {
		btc_sync()
		time.Sleep(time.Second * 10)
	}
}
