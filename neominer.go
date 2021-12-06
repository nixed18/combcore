package main

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"libcomb"

	"github.com/syndtr/goleveldb/leveldb"
)

type BlockInfo struct {
	Height  uint64
	Commits []libcomb.Commit
}

var NodeInfo struct {
	alive                bool
	version              string
	height, known_height uint64
	hash                 [32]byte
	guess                [32]byte
}

func neominer_get_info(client *http.Client) {
	NodeInfo.hash, NodeInfo.height, NodeInfo.known_height, _ = btc_get_sync_state(client)
}

func neominer_inspect() {
	log.Println("BTC Node:")
	log.Printf("\tVersion: %s\n", NodeInfo.version)
	log.Printf("\tHeight: %d of %d\n", NodeInfo.height, NodeInfo.known_height)
	if !NodeInfo.alive {
		var client *http.Client = &http.Client{}
		_, _, err := btc_is_alive(client)
		log.Printf("\tError: %s\n", err.Error())
	}
	log.Println("COMB Node:")
	log.Printf("\tVersion: %s\n", libcomb.Version)
	log.Printf("\tHeight: %d\n", libcomb.GetHeight())
}

func neominer_repair() {
	var client *http.Client = &http.Client{}
	var block BlockInfo
	var err error
	if len(DBInfo.CorruptedBlocks) > 0 {
		batch := new(leveldb.Batch)
		ordered := make([]uint64, 0, len(DBInfo.CorruptedBlocks))
		for height := range DBInfo.CorruptedBlocks {
			ordered = append(ordered, height)
		}
		sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
		log.Printf("(neominer) repairing %d blocks...\n", len(ordered))

		for _, height := range ordered {
			if err = db_load_blocks(height - 1); err != nil {
				log.Fatalln("(neominer) load error (" + err.Error() + ")")
			}
			if block, err = btc_get_block(client, height); err == nil {
				log.Printf("(neominer) repaired block %d\n", height)
				err = db_process_block(batch, block)
				db_write_batch(batch)
			}
			if err != nil {
				log.Fatalln("(neominer) repair error (" + err.Error() + ")")
			}
		}

		DBInfo.CorruptedBlocks = nil
	}
}

func neominer_download_batch(start_height, end_height uint64) (blocks []BlockInfo, err error) {
	//add multiprocessing!
	client := &http.Client{}
	var delta uint64 = 0
	for height := start_height; height <= end_height; height++ {
		if block, err := btc_get_block(client, height); err == nil {
			blocks = append(blocks, block)
			delta++
		}
		if delta == 10 {
			break
		}
	}
	return blocks, err
}

func neominer_load_batch(blocks []BlockInfo) (err error) {
	batch := new(leveldb.Batch)
	for _, b := range blocks {
		if b.Height == 0 {
			continue
		}
		if err = db_process_block(batch, b); err != nil {
			return err
		}
	}
	db_write_batch(batch)
	return nil
}

func neominer_sync() (err error) {
	var client *http.Client = &http.Client{}
	for {
		if err = neominer_start(); err != nil {
			return err
		}
		if err = btc_wait_for_block(client); err != nil {
			return err
		}
	}
}

func neominer_start() (err error) {
	var client *http.Client = &http.Client{}
	var blocks []BlockInfo
	var download_err, load_err error
	var diff int = int(NodeInfo.height) - int(COMBInfo.Height)
	if diff > 0 && (diff < 10000 || DataInfo.Path == "") {
		for {
			blocks, download_err = neominer_download_batch(COMBInfo.Height+1, NodeInfo.height)
			load_err = neominer_load_batch(blocks)
			if download_err != nil {
				log.Fatalln("(neominer) sync download error (" + download_err.Error() + ")")
			}
			if load_err != nil {
				log.Fatalln("(neominer) sync load error (" + load_err.Error() + ")")
			}
			if COMBInfo.Height == NodeInfo.height {
				break
			}
		}
	} else if diff >= 10000 && DataInfo.Path != "" {
		log.Printf("(neominer) delegating to directminer (%d blocks)\n", diff)
		neominer_get_info(client)
		var hash string
		if hash, err = btc_get_block_hash(client, COMBInfo.Height); err != nil {
			log.Fatalln("(neominer) get hash error (" + err.Error() + ")")
		}
		hash = strings.ToUpper(hash[1:65])
		start_hash := hex2byte32([]byte(hash))

		directminer_mine(COMBInfo.Height, start_hash, NodeInfo.hash)
		log.Printf("(neominer) height %d\n", COMBInfo.Height)
	}
	return nil
}

func neominer_blocking_connect() (err error) {
	var first = true
	var client *http.Client = &http.Client{}
	for !NodeInfo.alive {
		NodeInfo.alive, NodeInfo.version, err = btc_is_alive(client)
		if NodeInfo.alive {
			if !first {
				log.Printf("(neominer) connected\n")
			}
			break
		}
		if first {
			log.Printf("(neominer) %s\n", err.Error())
			log.Printf("(neominer) retrying...\n")
			first = false
		}
		time.Sleep(time.Second)
	}
	return
}

func neominer_connect() {
	var client *http.Client = &http.Client{}
	NodeInfo.alive, NodeInfo.version, _ = btc_is_alive(client)
	if !NodeInfo.alive {
		return
	}
	neominer_get_info(client)
}
