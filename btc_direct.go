package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type RawData map[[32]byte]*BlockData

func direct_parse_block_file(data []byte, blocks *RawData, path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("(direct) cant open file %s (%s)\n", path, err.Error())
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
		block := new(BlockData)
		btc_parse_block(data[p:p+size], block)
		(*blocks)[block.Hash] = block
		p += size
	}
}

func direct_trace_chain(blocks *RawData, start_hash [32]byte, end_hash [32]byte, length uint64) (block_chain [][32]byte) {
	//start_hash exclusive, end_hash inclusive
	var hash [32]byte = end_hash
	for {
		if block, ok := (*blocks)[hash]; ok {
			block_chain = append(block_chain, hash)
			hash = block.Previous
			if block.Previous == start_hash {
				break
			}
		} else {
			break
		}
	}

	var progress float64 = (float64(len(block_chain)) / float64(length)) * 100.0

	log.Printf("(direct) %.2f%%\n", progress)

	if hash != start_hash {
		return nil
	}

	for i, j := 0, len(block_chain)-1; i < j; i, j = i+1, j-1 {
		block_chain[i], block_chain[j] = block_chain[j], block_chain[i]
	}

	return block_chain
}

func direct_load_trace(blocks *RawData, path string, start_hash, end_hash [32]byte, length uint64) (chain [][32]byte, err error) {
	//log.Printf("(direct) trace between %X -> %X\n", start_hash, end_hash)

	var block_data []byte = make([]byte, 128*1024*1024)
	var block_files []string
	if block_files, err = filepath.Glob(path + "/blocks/blk*.dat"); err != nil {
		return nil, err
	}
	for i, j := 0, len(block_files)-1; i < j; i, j = i+1, j-1 {
		block_files[i], block_files[j] = block_files[j], block_files[i]
	}

	for b := range block_files {
		direct_parse_block_file(block_data, blocks, block_files[b])
		chain = direct_trace_chain(blocks, start_hash, end_hash, length)

		if len(chain) != 0 {
			break
		}
	}
	return chain, nil
}

func direct_check_path(path string) (err error) {
	if path == "" {
		return fmt.Errorf("no path configured")
	}
	path = path + "/blocks"
	if _, err = os.Stat(path); err != nil {
		return err
	}
	var block_files []string
	if block_files, err = filepath.Glob(path + "/blk*.dat"); err != nil {
		return err
	}
	if len(block_files) == 0 {
		return fmt.Errorf("no block files found")
	}

	log.Printf("(direct) found %d block files\n", len(block_files))
	return nil
}
func direct_get_block_range(path string, start_hash [32]byte, end_hash [32]byte, length uint64, out chan<- BlockData) (err error) {
	defer close(out)
	var blocks RawData = make(RawData)
	var chain [][32]byte
	if chain, err = direct_load_trace(&blocks, path, start_hash, end_hash, length); err != nil {
		return err
	}
	for _, hash := range chain {
		out <- *blocks[hash]
	}
	return nil
}

func direct_get_block(path string, hash [32]byte) (block BlockData, err error) {
	channel := make(chan BlockData, 0)
	if err = direct_get_block_range(path, hash, hash, 1, channel); err != nil {
		block = <-channel
	}
	return block, err
}
