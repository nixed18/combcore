package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/vharitonsky/iniflags"

	"libcomb"
)

var critical sync.Mutex

var COMBInfo struct {
	Height uint64
}

func comb_sync_info() {
	COMBInfo.Height = libcomb.GetHeight()
}

func setup_graceful_shutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Printf("(combcore) terminate signal detected. shutting down...")
		critical.Lock()
		db.Close()
		os.Exit(-3)
	}()
}

func combcore_inspect() {
	db_inspect()
	neominer_inspect()
	directminer_inspect()
}

func main() {
	var err error
	iniflags.SetAllowMissingConfigFile(false)
	iniflags.SetAllowUnknownFlags(false)
	iniflags.SetConfigFile("config.ini")
	iniflags.Parse()

	setup_graceful_shutdown()

	if err = db_open(); err != nil {
		log.Fatal(err)
	}

	db_start()
	neominer_connect()
	directminer_start()

	if len(DBInfo.CorruptedBlocks) > 0 {
		if NodeInfo.alive {
			neominer_repair()
		} else {
			log.Printf("(combcore) cannot repair corruption. exitting...\n")
			combcore_inspect()
			os.Exit(1)
		}
	}

	if COMBInfo.Height != DBInfo.StoredHeight {
		log.Printf("(combcore) height mismatch (%d != %d), trying reload \n", COMBInfo.Height, DBInfo.StoredHeight)
		db_load_all()
		if COMBInfo.Height != DBInfo.StoredHeight {
			log.Printf("(combcore) height mismatch (%d != %d), unknown cause\n", COMBInfo.Height, DBInfo.StoredHeight)
			combcore_inspect()
			os.Exit(1)
		}
	}

	if err = rpc_serve(); err != nil {
		log.Printf("(rpc) failed to start (%v)\n", err)
	}

	/*go func() {
		for {
			guess_combbase()
			time.Sleep(500 * time.Millisecond)
		}
	}()*/

	for {
		log.Printf("(neominer) started. auto syncing...\n")
		neominer_blocking_connect()
		if err = neominer_sync(); err != nil {
			log.Printf("(combcore) sync failed (%v)\n", err)
		}
	}

}
