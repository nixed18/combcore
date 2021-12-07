package main

import (
	"fmt"
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
	last_block           int64
}

func init() {
	NodeInfo.last_block = time.Now().Unix()
}

func neominer_get_info(client *http.Client) {
	NodeInfo.hash, NodeInfo.height, NodeInfo.known_height, _ = btc_get_sync_state(client)
}

func neominer_inspect() {
	log.Println("BTC Node:")
	log.Printf("\tVersion: %s\n", NodeInfo.version)
	log.Printf("\tHeight: %d of %d\n", NodeInfo.height, NodeInfo.known_height)
	if !NodeInfo.alive {
		_, _, err := btc_is_alive(btc_client)
		log.Printf("\tError: %s\n", err.Error())
	}
	log.Println("COMB Node:")
	log.Printf("\tVersion: %s\n", libcomb.Version)
	log.Printf("\tHeight: %d\n", libcomb.GetHeight())
}

func neominer_repair() {
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
			if block, err = btc_get_block(btc_client, height); err == nil {
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
	var delta uint64 = 0
	for height := start_height; height <= end_height; height++ {
		if block, err := btc_get_block(btc_client, height); err == nil {
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

func neominer_mine() (err error) {
	var blocks []BlockInfo
	var download_err, load_err error
	var diff int = int(NodeInfo.height) - int(COMBInfo.Height)
	if diff > 0 && (diff < 10000 || DataInfo.Path == "") {
		for {
			start := COMBInfo.Height + 1
			if start < 481824 {
				start = 481824
			}
			blocks, download_err = neominer_download_batch(start, NodeInfo.height)
			load_err = neominer_load_batch(blocks)
			if download_err != nil {
				return fmt.Errorf("download error (%s)", download_err.Error())
			}
			if load_err != nil {
				return fmt.Errorf("load error (%s)", load_err.Error())
			}
			if COMBInfo.Height == NodeInfo.height {
				break
			}
		}
	} else if diff >= 10000 && DataInfo.Path != "" {
		log.Printf("(neominer) delegating to directminer (%d blocks)\n", diff)
		neominer_get_info(btc_client)
		start := COMBInfo.Height
		if start < 481824 {
			start = 481824
		}
		var hash string
		if hash, err = btc_get_block_hash(btc_client, COMBInfo.Height); err != nil {
			return fmt.Errorf("(neominer) get hash error (%s)", err.Error())
		}
		hash = strings.ToUpper(hash[1:65])
		start_hash := hex2byte32([]byte(hash))

		if err = directminer_start(COMBInfo.Height, start_hash, NodeInfo.hash); err != nil {
			return fmt.Errorf("(neominer) directminer error (%s)", err.Error())
		}
		log.Printf("(neominer) height %d\n", COMBInfo.Height)
	}
	return nil
}

func neominer_sync() (err error) {
	client := &http.Client{}
	for {
		if err = neominer_mine(); err != nil {
			return err
		}
		if err = btc_wait_for_block(client); err != nil {
			return err
		}
	}
}

func neominer_blocking_connect() {
	var err error
	var first = true
	for first || !NodeInfo.alive {
		NodeInfo.alive, NodeInfo.version, err = btc_is_alive(btc_client)
		if NodeInfo.alive {
			if !first {
				log.Printf("(neominer) connected\n")
			}
			break
		}
		if first {
			log.Printf("(neominer) failed to connect (%s)\n", err.Error())
			log.Printf("(neominer) retrying...\n")
			first = false
		}
		time.Sleep(time.Second)
	}
}

func neominer_try_connect() {
	var err error
	NodeInfo.alive, NodeInfo.version, err = btc_is_alive(btc_client)
	if !NodeInfo.alive {
		log.Printf("(neominer) failed to connect (%s)", err.Error())
		return
	}
	neominer_get_info(btc_client)
}

func neominer_start() {
	log.Printf("(neominer) started. auto syncing...\n")
	for {
		neominer_blocking_connect()
		log.Printf("(neominer) connected\n")
		if err := neominer_sync(); err != nil {
			log.Printf("(neominer) sync failed (%v)\n", err)
		}
	}
}
