package main

import (
	"encoding/binary"
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
	Height  uint64
	Hash    [32]byte
	Chain   map[[32]byte][32]byte
	Status  string
	Network string
	Magic   uint32
	Prefix  map[string]string
	Path    string
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

	COMBInfo.Network = *comb_network
	combcore_set_network()
	setup_graceful_shutdown()

	COMBInfo.Chain = make(map[[32]byte][32]byte)
	//load our checkpoint (chain start)
	COMBInfo.Chain[COMBInfo.Hash] = empty
}

func combcore_set_network() {
	COMBInfo.Prefix = make(map[string]string)
	log.Printf("(combcore) loading in %s mode\n", COMBInfo.Network)
	//every difference between the networks is here (minus libcomb.SwitchToTestnet)
	switch COMBInfo.Network {
	case "mainnet":
		COMBInfo.Height = 481823
		COMBInfo.Hash, _ = parse_hex("000000000000000000cbeff0b533f8e1189cf09dfbebf57a8ebe349362811b80")
		COMBInfo.Magic = binary.LittleEndian.Uint32([]byte{0xf9, 0xbe, 0xb4, 0xd9})
		COMBInfo.Path = "commits"
		COMBInfo.Prefix["stack"] = "/stack/data/"
		COMBInfo.Prefix["tx"] = "/tx/recv/"
		COMBInfo.Prefix["key"] = "/wallet/data/"
		COMBInfo.Prefix["merkle"] = "/merkle/data/"
		COMBInfo.Prefix["decider"] = "/purse/data/"
	case "testnet":
		COMBInfo.Height = 0
		COMBInfo.Hash, _ = parse_hex("000000000933ea01ad0ee984209779baaec3ced90fa3f408719526f8d77f4943")
		COMBInfo.Magic = binary.LittleEndian.Uint32([]byte{0x0B, 0x11, 0x09, 0x07})
		COMBInfo.Path = "commits_testnet"
		COMBInfo.Prefix["stack"] = "\\stack\\data\\"
		COMBInfo.Prefix["tx"] = "\\tx\\recv\\"
		COMBInfo.Prefix["key"] = "\\wallet\\data\\"
		COMBInfo.Prefix["merkle"] = "\\merkle\\data\\"
		COMBInfo.Prefix["decider"] = "\\purse\\data\\"
		libcomb.SwitchToTestnet()
	default:
		log.Panicf("unknown network %s\n", COMBInfo.Network)
	}

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
	COMBInfo.Chain[block.Metadata.Hash] = COMBInfo.Hash
	COMBInfo.Hash = block.Metadata.Hash

	return nil
}

func combcore_reorg(target [32]byte) {
	//target is the highest common block between our chain and the new reorged chain
	//this function should remove all block data after target, and rollback libcomb to target
	var ok bool
	var metadata = db_get_block_by_hash(target)

	log.Printf("(combcore) reorg encountered, rolling back to block %d\n", metadata.Height)

	//trace back our in memory chain
	for COMBInfo.Hash != target {
		if COMBInfo.Hash, ok = COMBInfo.Chain[COMBInfo.Hash]; !ok {
			log.Panicf("reorg past checkpoint is not possible\n")
		}
	}

	//remove reorg'd blocks from the db
	db_remove_blocks_after(metadata.Height + 1)

	//unload libcomb to the target height
	for COMBInfo.Height != metadata.Height {
		libcomb.UnloadBlock()
		COMBInfo.Height = libcomb.GetHeight()
	}
	COMBInfo.Hash = target
}
