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

	COMBInfo.Status = "Initializing..."
	combcore_init()
	rpc_start()

	if err = db_open(); err != nil {
		log.Fatal(err)
	}
	COMBInfo.Status = "Loading..."
	db_start()

	btc_init()
	for {
		btc_sync()
		COMBInfo.Status = "Idle"
		time.Sleep(time.Second * 10)
	}
}
