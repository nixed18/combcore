package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"log"

	"github.com/syndtr/goleveldb/leveldb"
)

type LegacyBlockMetadata struct {
	Height      uint64
	Hash        string
	Fingerprint string
	Corrupt     bool
}

func db_legacy_clean() (err error) {
	var db_fingerprint [32]byte
	var block_fingerprint hash.Hash = sha256.New()
	var current_block LegacyBlockMetadata

	var corrupt_blocks map[uint64]struct{} = make(map[uint64]struct{})
	log.Println("(legacy) finding corrupt blocks...")

	iter := db.NewIterator(nil, nil)
	for iter.First(); ; iter.Next() {
		at_end := !iter.Valid()
		//process the last block, if one exists
		if (at_end || len(iter.Key()) == DB_BLOCK_KEY_LENGTH) && current_block.Height != 0 {
			if !current_block.Corrupt {
				block_fingerprint.Write([]byte(current_block.Hash[:]))
				our_fingerprint := fmt.Sprintf("%x", block_fingerprint.Sum(nil))
				stored_fingerprint := current_block.Fingerprint
				block_fingerprint.Reset()
				if our_fingerprint != stored_fingerprint {
					current_block.Corrupt = true
					log.Printf("(legacy) fingerprint mismatch detected at height %d\n", current_block.Height)
				}
			}
			if current_block.Corrupt {
				corrupt_blocks[current_block.Height] = struct{}{}
			}
		}
		if at_end {
			break
		}

		key := iter.Key()
		val := iter.Value()
		switch len(key) {
		case DB_BLOCK_KEY_LENGTH:
			current_block = LegacyBlockMetadata{}
			current_block.Height = uint64(binary.BigEndian.Uint64(key[0:8]))
			if len(val) != 128 {
				current_block.Corrupt = true
				log.Printf("(legacy) malformed block header detected at height %d\n", current_block.Height)
				continue
			}
			current_block.Hash = string(val[0:64])
			current_block.Fingerprint = string(val[64:128])
		case DB_COMMIT_KEY_LENGTH:
			if len(val) != 32 {
				current_block.Corrupt = true
				log.Printf("(legacy) malformed commit detected at height %d\n", current_block.Height)
				continue
			}
			block_fingerprint.Write(key)
			block_fingerprint.Write(val)
			db_fingerprint = fingerprint_write(val, db_fingerprint)
		}
	}
	iter.Release()
	if err = iter.Error(); err != nil {
		return err
	}

	if len(corrupt_blocks) != 0 {
		log.Printf("(legacy) removing %d corrupt blocks...", len(corrupt_blocks))
		batch := new(leveldb.Batch)
		iter = db.NewIterator(nil, nil)
		for iok := iter.First(); iok; iok = iter.Next() {
			key := iter.Key()
			switch len(key) {
			case DB_BLOCK_KEY_LENGTH:
				current_block = LegacyBlockMetadata{}
				current_block.Height = uint64(binary.BigEndian.Uint64(key[0:8]))
			}
			if _, ok := corrupt_blocks[current_block.Height]; ok {
				batch.Delete(key)
			}

		}
		iter.Release()
		if err = iter.Error(); err != nil {
			return err
		}
		if err = db_write_batch(batch); err != nil {
			return err
		}
	} else {
		log.Println("(legacy) no corrupt blocks found")
	}

	return nil
}

func db_legacy_migrate() (err error) {
	var block_fingerprint hash.Hash = sha256.New()
	var current_block BlockMetadata
	var current_block_has_commits bool = false

	log.Println("(legacy) migration to v2 started")

	db_legacy_clean()

	log.Println("(legacy) upgrading...")
	batch := new(leveldb.Batch)
	iter := db.NewIterator(nil, nil)
	for iter.First(); ; iter.Next() {
		at_end := !iter.Valid()
		//process the last block, if one exists
		if (at_end || len(iter.Key()) == DB_BLOCK_KEY_LENGTH) && current_block.Height != 0 {
			if current_block_has_commits {
				block_fingerprint.Sum(current_block.Fingerprint[0:0:32])
			}
			block_fingerprint.Reset()
			key, new_header := encode_block_metadata(current_block)
			batch.Put(key[:], new_header[:])
		}
		if at_end {
			break
		}

		key := iter.Key()
		switch len(key) {
		case DB_BLOCK_KEY_LENGTH:
			current_block = BlockMetadata{}
			current_block.Height = uint64(binary.BigEndian.Uint64(key[0:8]))
			current_block_has_commits = false
		case DB_COMMIT_KEY_LENGTH:
			current_block_has_commits = true
			block_fingerprint.Write(iter.Value())
		}
	}
	iter.Release()
	if err = iter.Error(); err != nil {
		return err
	}
	var key [2]byte
	var value [2]byte = [2]byte{0, 2}
	batch.Put(key[:], value[:])

	if err = db_write_batch(batch); err != nil {
		return err
	}

	log.Println("(legacy) migration finished")

	return iter.Error()
}
