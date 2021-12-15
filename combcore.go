package main

import (
	"libcomb"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/vharitonsky/iniflags"
)

var critical sync.Mutex
var shutdown sync.Mutex
var empty [32]byte
var COMBInfo struct {
	Height uint64
	Hash   [32]byte
}

func setup_graceful_shutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Printf("(combcore) terminate signal detected. shutting down...")
		critical.Lock()
		db.Close()
		shutdown.Unlock()
		os.Exit(-3)
	}()
	shutdown.Lock()
}

func combcore_init() {
	iniflags.SetAllowMissingConfigFile(false)
	iniflags.SetAllowUnknownFlags(false)
	iniflags.SetConfigFile("config.ini")
	iniflags.Parse()

	COMBInfo.Height = 481823
	COMBInfo.Hash, _ = parse_hex("000000000000000000cbeff0b533f8e1189cf09dfbebf57a8ebe349362811b80")

	setup_graceful_shutdown()
}

func combcore_dump() {
	db_inspect()
	neominer_inspect()
}

func combcore_process_block(block Block) (err error) {
	var lib_block libcomb.Block
	var lib_commit libcomb.Commit
	lib_block.Height = block.Metadata.Height
	lib_commit.Tag.Height = block.Metadata.Height

	for i, c := range block.Commits {
		lib_commit.Commit = c
		lib_commit.Tag.Commitnum = uint32(i)
		lib_block.Commits = append(lib_block.Commits, lib_commit)
	}

	if err = libcomb.LoadBlock(lib_block); err != nil {
		return err
	}
	COMBInfo.Height = libcomb.GetHeight()
	if COMBInfo.Height == block.Metadata.Height {
		COMBInfo.Hash = block.Metadata.Hash
	}
	return nil
}
