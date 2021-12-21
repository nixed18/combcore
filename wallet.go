package main

import (
	"encoding/binary"
	"errors"
	"log"
	"strings"

	"libcomb"
)

var Wallet struct {
	Keys     map[[32]byte]libcomb.Key
	Stacks   map[[32]byte]libcomb.Stack
	TXs      map[[32]byte]libcomb.Transaction
	Deciders map[[32]byte]libcomb.Decider
	Merkles  map[[32]byte]libcomb.MerkleSegment
}

func wallet_clear() {
	Wallet.Keys = make(map[[32]byte]libcomb.Key)
	Wallet.Stacks = make(map[[32]byte]libcomb.Stack)
	Wallet.TXs = make(map[[32]byte]libcomb.Transaction)
	Wallet.Deciders = make(map[[32]byte]libcomb.Decider)
	Wallet.Merkles = make(map[[32]byte]libcomb.MerkleSegment)
}

func init() {
	wallet_clear()
}

func wallet_load_stack(data []byte) (address [32]byte, err error) {
	var stack libcomb.Stack
	if len(data) != 32+32+8 {
		return address, errors.New("stack data malformed")
	}
	copy(stack.Change[:], data[0:32])
	copy(stack.Destination[:], data[32:64])
	stack.Sum = binary.BigEndian.Uint64(data[64:])
	libcomb.LoadStack(stack)
	address = libcomb.Hash(data[:])
	Wallet.Stacks[address] = stack
	return address, nil
}

func wallet_load_key(data []byte) (address [32]byte, err error) {
	var key libcomb.Key
	if len(data) != 21*32 {
		return address, errors.New("key data malformed")
	}
	for i := range key.Private {
		copy(key.Private[i][:], data[i*32:(i+1)*32])
	}

	key.Public = libcomb.LoadKey(key)
	Wallet.Keys[key.Public] = key
	return key.Public, nil
}

func wallet_load_transaction(data []byte) (address [32]byte, err error) {
	var tx libcomb.Transaction
	if len(data) != 23*32 {
		return address, errors.New("tx data malformed")
	}

	copy(tx.Source[:], data[0:32])
	copy(tx.Destination[:], data[32:64])

	for i := range tx.Signature {
		copy(tx.Signature[i][:], data[i*32+64:(i+1)*32+64])
	}
	address, err = libcomb.LoadTransaction(tx)
	Wallet.TXs[address] = tx
	return address, err
}

func wallet_load_merkle_segment(data []byte) (address [32]byte, err error) {
	var m libcomb.MerkleSegment
	if len(data) != 22*32 {
		return address, errors.New("merkle segment data malformed")
	}
	copy(m.Short[0][:], data[0:32])
	copy(m.Short[1][:], data[32:64])
	copy(m.Long[0][:], data[64:96])
	copy(m.Long[1][:], data[96:128])

	data = data[128:]
	for i := range m.Branches {
		copy(m.Branches[i][:], data[i*32:(i+1)*32])
	}
	data = data[16*32:]

	copy(m.Leaf[:], data[0:32])
	copy(m.Next[:], data[32:64])
	address = libcomb.LoadMerkleSegment(m)
	Wallet.Merkles[address] = m
	return address, nil
}

func wallet_load_decider(data []byte) (short [32]byte, err error) {
	var d libcomb.Decider
	if len(data) != 2*32 {
		return short, errors.New("decider data malformed")
	}

	copy(d.Private[0][:], data[0:32])
	copy(d.Private[1][:], data[32:64])

	short = libcomb.LoadDecider(d)
	Wallet.Deciders[short] = d
	return short, nil
}

func wallet_load_construct(construct string) (address [32]byte, err error) {
	var data []byte
	if strings.HasPrefix(construct, COMBInfo.Prefix["stack"]) {
		construct = strings.TrimPrefix(construct, COMBInfo.Prefix["stack"])
		data = libcomb.ParseHexSlice([]byte(construct))
		address, err = wallet_load_stack(data)
	}
	if strings.HasPrefix(construct, COMBInfo.Prefix["tx"]) {
		construct = strings.TrimPrefix(construct, COMBInfo.Prefix["tx"])
		data = libcomb.ParseHexSlice([]byte(construct))
		address, err = wallet_load_transaction(data)
	}

	if strings.HasPrefix(construct, COMBInfo.Prefix["key"]) {
		construct = strings.TrimPrefix(construct, COMBInfo.Prefix["key"])
		data = libcomb.ParseHexSlice([]byte(construct))
		address, err = wallet_load_key(data)
	}

	if strings.HasPrefix(construct, COMBInfo.Prefix["merkle"]) {
		construct = strings.TrimPrefix(construct, COMBInfo.Prefix["merkle"])
		data = libcomb.ParseHexSlice([]byte(construct))
		address, err = wallet_load_merkle_segment(data)
	}

	if strings.HasPrefix(construct, COMBInfo.Prefix["decider"]) {
		construct = strings.TrimPrefix(construct, COMBInfo.Prefix["decider"])
		data = libcomb.ParseHexSlice([]byte(construct))
		address, err = wallet_load_decider(data)
	}
	return address, err
}

func wallet_load(data string) (err error) {
	var lines []string = strings.Split(data, "\n")
	var address [32]byte
	for _, line := range lines {
		if line == "" {
			continue
		}
		if address, err = wallet_load_construct(line); err != nil {
			log.Printf("(import) load construct error (%s)", err.Error())
		} else {
			log.Printf("(import) loaded construct (%X)", address)
		}
	}
	return nil
}
