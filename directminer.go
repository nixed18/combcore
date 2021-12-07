package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"log"
	"os"
	"path/filepath"
	"sync"

	"libcomb"

	"github.com/syndtr/goleveldb/leveldb"
)

type RawBlockData struct {
	Root     [32]byte
	Previous [32]byte
	Commits  []libcomb.Commit
	Active   bool
}

type RawData map[[32]byte]*RawBlockData

var DataInfo struct {
	Path string
}

func directminer_inspect() {
	log.Println("Block Data:")
	var file_count int = 0
	if DataInfo.Path != "" {
		var block_files []string
		block_files, _ = filepath.Glob(DataInfo.Path + "/blocks/blk*.dat")
		file_count = len(block_files)
	}
	log.Printf("\tPath: %s\n", DataInfo.Path)
	log.Printf("\tFiles: %d\n", file_count)
}

func directminer_parse_varint(data []byte) (value uint64, advance uint8) {
	prefix := data[0]

	switch prefix {
	case 0xfd:
		value = uint64(binary.LittleEndian.Uint16(data[1:]))
		advance = 3
	case 0xfe:
		value = uint64(binary.LittleEndian.Uint32(data[1:]))
		advance = 5
	case 0xff:
		value = uint64(binary.LittleEndian.Uint64(data[1:]))
		advance = 9
	default:
		value = uint64(prefix)
		advance = 1
	}

	return value, advance
}

func directminer_swap_endian(block *RawBlockData) {
	block.Root = swap_endian(block.Root)
	block.Previous = swap_endian(block.Previous)
}

func directminer_parse_block(data []byte) (hash [32]byte, block *RawBlockData) {
	block = new(RawBlockData)
	var current_commit libcomb.Commit
	hash = sha256.Sum256(data[0:80])
	hash = sha256.Sum256(hash[:])
	data = data[4:] //version(4)
	copy(block.Previous[:], data[0:32])
	data = data[32:] //previous(32)
	copy(block.Root[:], data[0:32])
	data = data[44:] //merkle root(32),time(4),bits(4),nonce(4)

	tx_count, adv := directminer_parse_varint(data[:])
	data = data[adv:] //tx count(var)

	var segwit bool
	for t := 0; t < int(tx_count); t++ {
		segwit = false
		data = data[4:] //version(4)
		in_count, adv := directminer_parse_varint(data[:])

		if in_count == 0 { //segwit marker is 0x00
			segwit = true
			data = data[2:] //marker(1),flag(1)
			in_count, adv = directminer_parse_varint(data[:])
		}

		data = data[adv:] //vin count(var)
		for i := 0; i < int(in_count); i++ {
			data = data[36:] //txid(32), vout(4)
			sig_size, adv := directminer_parse_varint(data[:])
			data = data[uint64(adv)+sig_size+4:] //sig size(var), sig(var),sequence(4)
		}
		out_count, adv := directminer_parse_varint(data[:])
		data = data[adv:] //vout count(var)
		for i := 0; i < int(out_count); i++ {
			data = data[8:] //value(8)
			pub_size, adv := directminer_parse_varint(data[:])
			data = data[uint64(adv):] //pub size(var)
			if pub_size == 34 && data[0] == 0 && data[1] == 32 {
				copy(current_commit.Commit[:], data[2:34])
				block.Commits = append(block.Commits, current_commit)
			}
			data = data[pub_size:] //pub (var)
		}
		if segwit {
			for i := 0; i < int(in_count); i++ {
				witness_count, adv := directminer_parse_varint(data[:])
				data = data[adv:] //witness count(var)
				for w := 0; w < int(witness_count); w++ {
					witness_size, adv := directminer_parse_varint(data[:])
					data = data[uint64(adv)+witness_size:] //witness size(var), witness(var)
				}
			}
		}
		data = data[4:] //locktime(4)
	}

	directminer_swap_endian(block)
	hash = swap_endian(hash)

	return hash, block
}

func directminer_parse_block_file(data []byte, m *sync.Mutex, blocks *RawData, path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("(directminer) cant open file %s (%s)\n", path, err.Error())
		return
	}
	stats, _ := f.Stat()
	data = data[:stats.Size()]
	buf := bufio.NewReader(f)
	buf.Read(data)
	f.Close()

	var p int = 0
	var size int
	var magic uint32 = binary.LittleEndian.Uint32([]byte{0xf9, 0xbe, 0xb4, 0xd9})
	for {
		if p >= len(data) || binary.LittleEndian.Uint32(data[p:p+4]) != magic {
			break
		}
		p += 4
		size = int(binary.LittleEndian.Uint32(data[p : p+4]))
		p += 4

		hash, raw_data := directminer_parse_block(data[p : p+size])
		m.Lock()
		(*blocks)[hash] = raw_data
		m.Unlock()

		p += size
	}
}

func directminer_load_all_blocks(blocks *RawData, block_data_path string, max_open uint) (err error) {
	var blocks_mutex sync.Mutex

	const file_size uint = 128 * 1024 * 1024
	var block_data []byte = make([]byte, max_open*file_size)
	var available chan uint = make(chan uint, max_open)
	for i := uint(0); i < max_open; i++ {
		available <- i
	}
	log.Printf("(directminer) started. %d concurrent files\n", max_open)

	var block_files []string
	block_files, err = filepath.Glob(block_data_path + "/blocks/blk*.dat")
	if err != nil {
		return err
	}

	var block_file_count int = len(block_files)
	log.Printf("(directminer) found %d block data files\n", block_file_count)
	log.Printf("(directminer) mining...\n")

	proccessed := 0
	for b := range block_files {
		a := <-available
		go func(a uint, b int) {
			ram := block_data[a*file_size : (a+1)*file_size]
			directminer_parse_block_file(ram, &blocks_mutex, blocks, block_files[b])
			proccessed++
			if proccessed%200 == 0 {
				log.Printf("(directminer) loaded %d/%d\n", proccessed, block_file_count)
			}
			available <- a
		}(a, b)
	}

	for i := uint(0); i < max_open; i++ {
		<-available
	}

	return nil
}

func directminer_trace_chain(blocks *RawData, start_hash [32]byte, end_hash [32]byte) (block_chain [][32]byte) {
	var hash [32]byte = end_hash
	for {
		if block, ok := (*blocks)[hash]; ok {
			block_chain = append(block_chain, hash)
			block.Active = true
			hash = block.Previous
			if hash == start_hash {
				block_chain = append(block_chain, hash)
				break
			}
		} else {
			break
		}
	}

	if hash != start_hash {
		return nil
	}

	//reverse the chain since we traced from the top
	for i, j := 0, len(block_chain)-1; i < j; i, j = i+1, j-1 {
		block_chain[i], block_chain[j] = block_chain[j], block_chain[i]
	}

	//remove inactive blocks
	for hash, block := range *blocks {
		if !block.Active {
			delete(*blocks, hash)
		}
	}

	return block_chain
}

func directminer_store_chain(blocks *RawData, start_height uint64, chain [][32]byte) (count uint64, err error) {
	var current_block BlockInfo
	current_block.Height = libcomb.GetHeight()
	var height uint64

	batch := new(leveldb.Batch)

	for relative_height, hash := range chain {
		height = start_height + uint64(relative_height)
		if uint64(height) > current_block.Height {
			current_block.Commits = (*blocks)[hash].Commits
			current_block.Height = uint64(height)
			for i := range current_block.Commits {
				current_block.Commits[i].Tag.Height = current_block.Height
				current_block.Commits[i].Tag.Commitnum = uint32(i)
			}

			if err = db_process_block(batch, current_block); err != nil {
				return count, err
			}
			if height%1000 == 0 {
				db_write_batch(batch)
			}
			count++
		}
	}
	db_write_batch(batch)
	db_compute_fingerprint(libcomb.GetHeight())
	return count, err
}

func directminer_load_trace(blocks *RawData, block_data_path string, max_open uint, start_hash, end_hash [32]byte) (err error) {
	var blocks_mutex sync.Mutex
	var chain [][32]byte

	const file_size uint = 128 * 1024 * 1024
	var block_data []byte = make([]byte, max_open*file_size)
	var available chan uint = make(chan uint, max_open)
	for i := uint(0); i < max_open; i++ {
		available <- i
	}
	log.Printf("(directminer) started. %d concurrent files\n", max_open)

	var block_files []string
	block_files, err = filepath.Glob(block_data_path + "/blocks/blk*.dat")
	if err != nil {
		return err
	}
	for i, j := 0, len(block_files)-1; i < j; i, j = i+1, j-1 {
		block_files[i], block_files[j] = block_files[j], block_files[i]
	}

	var block_file_count int = len(block_files)
	log.Printf("(directminer) found %d block data files\n", block_file_count)
	log.Printf("(directminer) mining...\n")

	proccessed := 0
	finished := false
	for b := range block_files {
		if finished {
			break
		}
		a := <-available
		go func(a uint, b int) {
			ram := block_data[a*file_size : (a+1)*file_size]
			directminer_parse_block_file(ram, &blocks_mutex, blocks, block_files[b])
			proccessed++
			if proccessed%100 == 0 {
				log.Printf("(directminer) loaded %d files\n", proccessed)
			}
			blocks_mutex.Lock()
			chain = directminer_trace_chain(blocks, start_hash, end_hash)
			blocks_mutex.Unlock()
			if len(chain) > 0 {
				finished = true
			}
			available <- a
		}(a, b)
	}

	for i := uint(0); i < max_open; i++ {
		<-available
	}
	log.Printf("(directminer) loaded %d files\n", proccessed)
	return nil
}

func directminer_start(start_height uint64, start_hash, end_hash [32]byte) (err error) {
	var blocks RawData = make(RawData)
	var chain [][32]byte
	var new_blocks uint64
	if err = directminer_load_trace(&blocks, DataInfo.Path, 4, start_hash, end_hash); err != nil {
		return err
	}
	chain = directminer_trace_chain(&blocks, start_hash, end_hash)
	if len(chain) != 0 {
		log.Printf("(directminer) found chain of length %d\n", len(chain)-1)
		log.Printf("(directminer) storing chain...\n")
		if new_blocks, err = directminer_store_chain(&blocks, start_height, chain); err != nil {
			return err
		}
		log.Printf("(directminer) finished. loaded %d new blocks.\n", new_blocks)
	} else {
		log.Printf("(directminer) failed. no active chain found\n")
	}
	return nil
}

func directminer_init() {
	DataInfo.Path = *btc_data
}
