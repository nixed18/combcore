package main

import (
	"fmt"
	"libcomb"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/vharitonsky/iniflags"
)

const p2wsh_height uint64 = 481824

var critical sync.Mutex
var empty [32]byte
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

func combcore_init() {
	iniflags.SetAllowMissingConfigFile(false)
	iniflags.SetAllowUnknownFlags(false)
	iniflags.SetConfigFile("config.ini")
	iniflags.Parse()

	setup_graceful_shutdown()
}

func combcore_dump() {
	db_inspect()
	neominer_inspect()
	directminer_inspect()
}

func combcore_check() (err error) {
	//check for fatal post-load problems
	if len(DBInfo.CorruptedBlocks) > 0 {
		if NodeInfo.alive {
			neominer_repair()
		} else {
			return fmt.Errorf("(combcore) cannot repair corruption")
		}
	}

	if COMBInfo.Height != DBInfo.StoredHeight {
		log.Printf("(combcore) height mismatch (%d != %d), trying reload\n", COMBInfo.Height, DBInfo.StoredHeight)
		db_load_all()
		if COMBInfo.Height != DBInfo.StoredHeight {
			return fmt.Errorf("(combcore) height mismatch (%d != %d), unknown cause", COMBInfo.Height, DBInfo.StoredHeight)
		}
	}

	return nil
}

func combcore_block_ingest() {
	//log.Printf("(combcore) height %d\n", COMBInfo.Height)
	if combbase, err := libcomb.GetCOMBBase(COMBInfo.Height); err == nil {
		autoclaim_process_combbase(combbase)
		NodeInfo.last_block = time.Now().Unix()
	}
}
