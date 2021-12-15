package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"log"
	"os"
	"strings"

	"libcomb"
)

func import_load_stack(data []byte) (address [32]byte, err error) {
	var stack libcomb.Stack
	if len(data) != 32+32+8 {
		return address, errors.New("stack data malformed")
	}
	copy(stack.Change[:], data[0:32])
	copy(stack.Destination[:], data[32:64])
	stack.Sum = binary.BigEndian.Uint64(data[64:])
	libcomb.LoadStack(stack)
	return sha256.Sum256(data[:]), nil
}

func import_load_wallet_key(data []byte) (address [32]byte, err error) {
	var key libcomb.WalletKey
	if len(data) != 21*32 {
		return address, errors.New("wallet key data malformed")
	}
	for i := range key.Private {
		copy(key.Private[i][:], data[i*32:(i+1)*32])
	}
	address = libcomb.LoadWalletKey(key)
	return address, nil
}

func import_load_transaction(data []byte) (address [32]byte, err error) {
	var tx libcomb.Transaction
	var sig [21][32]byte
	if len(data) != 23*32 {
		return address, errors.New("tx data malformed")
	}

	copy(tx.Source[:], data[0:32])
	copy(tx.Destination[:], data[32:64])

	for i := range sig {
		copy(sig[i][:], data[i*32+64:(i+1)*32+64])
	}
	address, err = libcomb.LoadTransaction(tx, sig)
	return address, err
}

func import_load_merkle_segment(data []byte) (address [32]byte, err error) {
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
	return address, nil
}

func import_load_decider(data []byte) (short [32]byte, err error) {
	var d libcomb.Decider
	if len(data) != 2*32 {
		return short, errors.New("decider data malformed")
	}

	copy(d.Private[0][:], data[0:32])
	copy(d.Private[1][:], data[32:64])

	short = libcomb.LoadDecider(d)
	return short, nil
}

func import_load_construct(construct string) (address [32]byte, err error) {
	var data []byte
	if strings.HasPrefix(construct, "/stack/data/") {
		construct = strings.TrimPrefix(construct, "/stack/data/")
		data = hex2byte([]byte(construct))
		address, err = import_load_stack(data)
	}
	if strings.HasPrefix(construct, "/tx/recv/") {
		construct = strings.TrimPrefix(construct, "/tx/recv/")
		data = hex2byte([]byte(construct))
		address, err = import_load_transaction(data)
	}

	if strings.HasPrefix(construct, "/wallet/data/") {
		construct = strings.TrimPrefix(construct, "/wallet/data/")
		data = hex2byte([]byte(construct))
		address, err = import_load_wallet_key(data)
	}

	if strings.HasPrefix(construct, "/merkle/data/") {
		data = hex2byte([]byte(construct))
		address, err = import_load_merkle_segment(data)
	}

	if strings.HasPrefix(construct, "/purse/data/") {
		data = hex2byte([]byte(construct))
		address, err = import_load_decider(data)
	}
	return address, err
}

func import_load_save_file(file string) (err error) {
	var f *os.File
	if f, err = os.OpenFile(file, os.O_RDONLY, os.ModePerm); err != nil {
		log.Printf("(import) cannot load file (%s)", err.Error())
		return err
	}
	defer f.Close()

	var scanner *bufio.Scanner = bufio.NewScanner(f)
	var address [32]byte
	for scanner.Scan() {
		construct := scanner.Text()
		if address, err = import_load_construct(construct); err != nil {
			log.Printf("(import) load construct error (%s)", err.Error())
		} else {
			log.Printf("(import) loaded construct (%X)", address)
		}
	}
	return nil
}
