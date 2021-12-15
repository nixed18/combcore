package main

import (
	"log"

	"libcomb"

	"github.com/syndtr/goleveldb/leveldb"
)

var NeoInfo struct {
	BatchMode     bool
	BatchCapacity uint64
	BatchCached   uint64
	Batch         *leveldb.Batch
}

func neominer_inspect() {
	log.Println("BTC Node:")
	log.Printf("\tHeight: %d of %d\n", BTC.Chain.Height, BTC.Chain.KnownHeight)
	log.Println("COMB Node:")
	log.Printf("\tVersion: %s\n", libcomb.Version)
	log.Printf("\tHeight: %d\n", libcomb.GetHeight())
}

func neominer_enable_batch_mode() {
	NeoInfo.BatchCapacity = 10000
	NeoInfo.BatchCached = 0
	NeoInfo.Batch = new(leveldb.Batch)
	NeoInfo.BatchMode = true
}

func neominer_disable_batch_mode() {
	NeoInfo.BatchMode = false
	//flush cache
	neominer_write_batch(NeoInfo.Batch)
}

func neominer_write_batch(batch *leveldb.Batch) {
	log.Printf("(neominer) proccessed %d\n", COMBInfo.Height)
	if err := db_write(batch); err != nil {
		log.Panicf("(neominer) write batch failed (%s)\n", err.Error())
		return
	}
	if NeoInfo.BatchMode {
		NeoInfo.BatchCached = 0
	}
}

func neominer_process_block(block_data BlockData) {
	var err error
	var block Block
	var batch *leveldb.Batch

	if NeoInfo.BatchMode {
		batch = NeoInfo.Batch
		NeoInfo.BatchCached++
	} else {
		batch = new(leveldb.Batch)
	}

	block.Metadata.Hash = block_data.Hash
	block.Commits = block_data.Commits

	if block_data.Previous == COMBInfo.Hash {
		block.Metadata.Height = COMBInfo.Height + 1
	} else {
		log.Panicf("reorgs not supported %X %X", block_data.Hash, COMBInfo.Hash)
	}

	if err = db_process_block(batch, block); err != nil {
		log.Panicf("(neominer) ingest store block failed (%s)\n", err.Error())
		return
	}
	if err = combcore_process_block(block); err != nil {
		log.Panicf("(neominer) ingest process block failed (%s)\n", err.Error())
	}

	if (NeoInfo.BatchMode && NeoInfo.BatchCached >= NeoInfo.BatchCapacity) || !NeoInfo.BatchMode {
		neominer_write_batch(batch)
	}

}
