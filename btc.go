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
		BTC.Chain.KnownHeight = 0 //signals we are disconnected
		return
	}

	if COMBInfo.Hash == BTC.Chain.TopHash {
		//log.Printf("nothing to do %X == %X\n", COMBInfo.Hash, BTC.Chain.TopHash)
		return
	}

	delta = int64(BTC.Chain.Height) - int64(COMBInfo.Height)
	//delta does not include reorgs, dont rely on this, only use for status info etc
	log.Printf("(btc) %d blocks behind...\n", delta)

	var blocks chan BlockData = make(chan BlockData)
	var wait sync.Mutex
	wait.Lock()
	go func() {
		for block := range blocks {
			neominer_process_block(block)
		}
		neominer_write()
		wait.Unlock()
	}()

	var target [32]byte = BTC.Chain.TopHash
	if err = btc_get_block_range(target, &COMBInfo.Chain, uint64(delta), blocks); err != nil {
		log.Printf("(btc) failed to get blocks (%s)\n", err.Error())
	}
	wait.Lock() //dont leave before neominer is finished (only a problem if we use a buffered channel)
}

func btc_get_block_range(target [32]byte, chain *map[[32]byte][32]byte, delta uint64, blocks chan<- BlockData) (err error) {
	if BTC.DirectPath != "" && delta > 10 {
		if err = direct_get_block_range(BTC.DirectPath, target, chain, delta, blocks); err != nil {
			return err
		}
	} else {
		if err = rest_get_block_range(BTC.RestClient, BTC.RestURL, target, chain, delta, blocks); err != nil {
			return err
		}
	}
	return nil
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
