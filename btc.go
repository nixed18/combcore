package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type BlockData struct {
	Hash     [32]byte
	Previous [32]byte
	Commits  [][32]byte
}

type ChainData struct {
	Height      uint64
	KnownHeight uint64
	TopHash     [32]byte
}

var BTC struct {
	RestClient *http.Client
	RestURL    string
	DirectPath string
	Chain      ChainData
}

func btc_init() {
	BTC.RestURL = fmt.Sprintf("http://%s:%d/rest", *btc_peer, *btc_port)
	BTC.RestClient = &http.Client{}

	if err := direct_check_path(*btc_data); err != nil {
		log.Printf("(btc) direct mining disabled (%s)\n", err.Error())
	} else {
		BTC.DirectPath = *btc_data
	}
}

func btc_sync() {
	var err error
	var delta int64
	if BTC.Chain, err = rest_get_chains(BTC.RestClient, BTC.RestURL); err != nil {
		log.Printf("(btc) failed to get chains (%s)\n", err.Error())
		return
	}

	delta = int64(BTC.Chain.Height) - int64(COMBInfo.Height)

	log.Printf("(btc) %d blocks behind...\n", delta)

	if delta == 1 {
		var block BlockData
		if block, err = rest_get_block(BTC.RestClient, BTC.RestURL, BTC.Chain.TopHash); err != nil {
			log.Printf("(btc) failed to get block (%s)\n", err.Error())
			return
		}
		neominer_process_block(block)
		return
	}

	if delta > 1 {
		neominer_enable_batch_mode()
		var blocks chan BlockData = make(chan BlockData)
		var wait sync.Mutex
		wait.Lock()
		go func() {
			for block := range blocks {
				neominer_process_block(block)
			}
			wait.Unlock()
			neominer_disable_batch_mode()
		}()

		var start [32]byte = COMBInfo.Hash
		var end [32]byte = BTC.Chain.TopHash

		if BTC.DirectPath != "" && delta > 10 {
			if err = direct_get_block_range(BTC.DirectPath, start, end, uint64(delta), blocks); err != nil {
				log.Printf("(btc) failed to get blocks (direct) (%s)\n", err.Error())
				return
			}
		} else {
			if err = rest_get_block_range(BTC.RestClient, BTC.RestURL, start, end, blocks); err != nil {
				log.Printf("(btc) failed to get blocks (rest) (%s)\n", err.Error())
				return
			}
		}
		wait.Lock() //fix the race condition if we used a buffered channel
		return
	}
}

func btc_parse_varint(data []byte) (value uint64, advance uint8) {
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

func btc_parse_block(data []byte, block *BlockData) {
	var current_commit [32]byte
	block.Hash = sha256.Sum256(data[0:80])
	block.Hash = sha256.Sum256(block.Hash[:])
	data = data[4:] //version(4)
	copy(block.Previous[:], data[0:32])
	data = data[32:] //previous(32)
	data = data[44:] //merkle root(32),time(4),bits(4),nonce(4)

	tx_count, adv := btc_parse_varint(data[:])
	data = data[adv:] //tx count(var)

	var segwit bool
	for t := 0; t < int(tx_count); t++ {
		segwit = false
		data = data[4:] //version(4)
		in_count, adv := btc_parse_varint(data[:])

		if in_count == 0 { //segwit marker is 0x00
			segwit = true
			data = data[2:] //marker(1),flag(1)
			in_count, adv = btc_parse_varint(data[:])
		}

		data = data[adv:] //vin count(var)
		for i := 0; i < int(in_count); i++ {
			data = data[36:] //txid(32), vout(4)
			sig_size, adv := btc_parse_varint(data[:])
			data = data[uint64(adv)+sig_size+4:] //sig size(var), sig(var),sequence(4)
		}
		out_count, adv := btc_parse_varint(data[:])
		data = data[adv:] //vout count(var)
		for i := 0; i < int(out_count); i++ {
			data = data[8:] //value(8)
			pub_size, adv := btc_parse_varint(data[:])
			data = data[uint64(adv):] //pub size(var)
			if pub_size == 34 && data[0] == 0 && data[1] == 32 {
				copy(current_commit[:], data[2:34])
				block.Commits = append(block.Commits, current_commit)
			}
			data = data[pub_size:] //pub (var)
		}
		if segwit {
			for i := 0; i < int(in_count); i++ {
				witness_count, adv := btc_parse_varint(data[:])
				data = data[adv:] //witness count(var)
				for w := 0; w < int(witness_count); w++ {
					witness_size, adv := btc_parse_varint(data[:])
					data = data[uint64(adv)+witness_size:] //witness size(var), witness(var)
				}
			}
		}
		data = data[4:] //locktime(4)
	}

	block.Hash = swap_endian(block.Hash)
	block.Previous = swap_endian(block.Previous)
}
