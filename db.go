package main

import (
	"crypto/aes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"log"
	"sync"

	"libcomb"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const DB_PATH = "commits"

const DB_LEGACY_VERSION = 1
const DB_CURRENT_VERSION = 2

const DB_VERSION_KEY_LENGTH = 2
const DB_BLOCK_KEY_LENGTH = 8
const DB_COMMIT_KEY_LENGTH = 16

var db *leveldb.DB
var db_is_new bool
var db_mutex sync.Mutex

var DBInfo struct {
	Version          uint16
	Fingerprint      [32]byte
	FingerprintIndex map[uint64][32]byte
	StoredHeight     uint64
	CorruptedBlocks  map[uint64]struct{}
}

type BlockMetadata struct {
	Height      uint64
	Fingerprint [32]byte
	Root        [32]byte
}

func init() {
	DBInfo.FingerprintIndex = make(map[uint64][32]byte)
}

func fingerprint_write(key []byte, fingerprint [32]byte) [32]byte {
	var tmp [32]byte

	var aes, err = aes.NewCipher(key)
	if err != nil {
		log.Println("(db) error fingerprint key error (" + err.Error() + ")")
		return fingerprint
	}

	aes.Encrypt(tmp[0:16], fingerprint[0:16])
	aes.Decrypt(tmp[16:32], fingerprint[16:32])

	//swap tmp[8:16] with tmp[16:24]
	for i := 8; i < 16; i++ {
		tmp[i], tmp[8+i] = tmp[8+i], tmp[i]
	}

	aes.Encrypt(fingerprint[0:16], tmp[0:16])
	aes.Decrypt(fingerprint[16:32], tmp[16:32])

	return fingerprint
}

func fingerprint_unwrite(key []byte, fingerprint [32]byte) [32]byte {
	var tmp [32]byte

	var aes, err = aes.NewCipher(key)
	if err != nil {
		log.Println("(db) error fingerprint key error (" + err.Error() + ")")
		return fingerprint
	}

	aes.Decrypt(tmp[0:16], fingerprint[0:16])
	aes.Encrypt(tmp[16:32], fingerprint[16:32])

	//swap tmp[8:16] with tmp[16:24]
	for i := 8; i < 16; i++ {
		tmp[i], tmp[8+i] = tmp[8+i], tmp[i]
	}

	aes.Decrypt(fingerprint[0:16], tmp[0:16])
	aes.Encrypt(fingerprint[16:32], tmp[16:32])

	return fingerprint
}

func db_compute_fingerprint(height uint64) [32]byte {
	var db_fingerprint [32]byte
	var current_block BlockMetadata
	iter := db.NewIterator(nil, nil)
	for iok := iter.First(); iok; iok = iter.Next() {
		if len(iter.Key()) == DB_BLOCK_KEY_LENGTH {
			if binary.BigEndian.Uint64(iter.Key()) == height+1 {
				break
			}
			current_block = decode_block_metadata(iter.Key(), iter.Value())
			if db_fingerprint == empty {
				db_fingerprint = current_block.Fingerprint
			} else {
				db_fingerprint = fingerprint_write(current_block.Fingerprint[:], db_fingerprint)
			}
		}
	}
	iter.Release()
	return db_fingerprint
}

func db_get_fingerprint(height uint64) (fingerprint [32]byte) {
	if !*comb_fingerprint_index {
		return db_compute_fingerprint(height)
	}

	if f, ok := DBInfo.FingerprintIndex[height]; ok {
		return f
	}

	for {
		height--
		if height < p2wsh_height {
			return fingerprint
		}
		if f, ok := DBInfo.FingerprintIndex[height]; ok {
			return f
		}
	}
}

func db_compute_legacy_fingerprint(height uint64) [32]byte {
	var db_fingerprint [32]byte
	iter := db.NewIterator(nil, nil)
	for iok := iter.First(); iok; iok = iter.Next() {
		if len(iter.Key()) == DB_BLOCK_KEY_LENGTH {
			if binary.BigEndian.Uint64(iter.Key()) == height+1 {
				break
			}
		}
		if len(iter.Key()) == DB_COMMIT_KEY_LENGTH {
			db_fingerprint = fingerprint_write(iter.Value(), db_fingerprint)
		}
	}
	iter.Release()
	return db_fingerprint
}

func db_reconstruct_fingerprint_index() {
	DBInfo.FingerprintIndex = make(map[uint64][32]byte)
	DBInfo.Fingerprint = empty
	log.Printf("(db) indexing fingerprints...")
	var current_block BlockMetadata
	iter := db.NewIterator(nil, nil)
	for iok := iter.First(); iok; iok = iter.Next() {
		if len(iter.Key()) == DB_BLOCK_KEY_LENGTH {
			current_block = decode_block_metadata(iter.Key(), iter.Value())
			db_update_fingerprint(current_block.Height, current_block.Fingerprint)
		}
	}
	iter.Release()
}

func db_update_fingerprint(height uint64, fingerprint [32]byte) {
	if DBInfo.Fingerprint == empty {
		DBInfo.Fingerprint = fingerprint
	} else {
		DBInfo.Fingerprint = fingerprint_write(fingerprint[:], DBInfo.Fingerprint)
	}
	if *comb_fingerprint_index {
		DBInfo.FingerprintIndex[height] = DBInfo.Fingerprint
	}
}

func db_open() (err error) {
	var lvldb *leveldb.DB
	var options opt.Options
	options.Compression = opt.NoCompression

	path := DB_PATH

	//see if a db exists
	options.ErrorIfMissing = true
	lvldb, err = leveldb.OpenFile(path, &options)

	if err != nil {
		options.ErrorIfMissing = false
		//may not exist, try create one
		lvldb, err = leveldb.OpenFile(path, &options)
		if err == nil {
			db_is_new = true
		} else {
			//actually was some other error
			return err
		}
	}

	db_mutex.Lock()
	db = lvldb
	return nil
}

func db_close() {
	db.Close()
	db = nil
	db_mutex.Unlock()
}

func db_write_batch(batch *leveldb.Batch) (err error) {
	critical.Lock()
	err = db.Write(batch, &opt.WriteOptions{
		Sync: true,
	})
	batch.Reset()
	critical.Unlock()
	return err
}

func comb_process_block(height uint64, commits []libcomb.Commit) (err error) {
	if err = libcomb.LoadBlock(height, commits); err != nil {
		return err
	}

	COMBInfo.Height = libcomb.GetHeight()

	if DBInfo.StoredHeight != 0 {
		combcore_block_ingest()
	}
	return nil
}

func decode_commit(data []byte) (commit [32]byte) {
	copy(commit[:], data[0:32])
	return commit
}

func decode_tag(data []byte) (tag libcomb.UTXOtag) {
	tag.Height = binary.BigEndian.Uint64(data[0:8])
	tag.Commitnum = binary.BigEndian.Uint32(data[8:12])

	txnum := binary.BigEndian.Uint16(data[12:14])
	outnum := binary.BigEndian.Uint16(data[14:16])

	if txnum != 0 || outnum != 0 { //legacy tags
		tag.Commitnum = (uint32(txnum) << 16) + uint32(outnum)
	}
	return tag
}

func encode_tag(tag libcomb.UTXOtag) (data [16]byte) {
	binary.BigEndian.PutUint64(data[0:8], tag.Height)
	binary.BigEndian.PutUint32(data[8:12], tag.Commitnum)
	return data
}

func decode_block_metadata(key []byte, value []byte) (block BlockMetadata) {
	block.Height = binary.BigEndian.Uint64(key[0:8])
	copy(block.Fingerprint[:], value[0:32])
	copy(block.Root[:], value[32:64])
	return block
}

func encode_block_metadata(data BlockMetadata) (key [8]byte, value [64]byte) {
	binary.BigEndian.PutUint64(key[0:8], data.Height)
	copy(value[0:32], data.Fingerprint[:])
	copy(value[32:64], data.Root[:])
	return key, value
}

func db_inspect() {
	DBInfo.Version = db_get_version()

	iter := db.NewIterator(nil, nil)

	var sizes map[uint16]uint64 = make(map[uint16]uint64)

	var total uint64
	var total_keys uint64
	for iok := iter.First(); iok; iok = iter.Next() {
		total += uint64(len(iter.Key())) + uint64(len(iter.Value()))
		total_keys++
		sizes[uint16(len(iter.Key()))] += 1
	}
	log.Println("Database:")
	log.Printf("\tVersion: %d\n", DBInfo.Version)
	log.Printf("\tFingerprint: %X\n", DBInfo.Fingerprint)
	log.Printf("\tSize: %0.2f mb\n", float64(total)/(1024*1024))
	log.Printf("\tKeys: %d\n", total_keys)
	for key, value := range sizes {
		log.Printf("\t\t%d: %d\n", key, value)
	}
	iter.Release()
}

func db_get_version() uint16 {
	var key [2]byte
	var version uint16 = 1
	if data, err := db.Get(key[:], nil); err == nil {
		version = binary.BigEndian.Uint16(data)
	}
	return version
}

func db_remove_block(batch *leveldb.Batch, height uint64) (err error) {
	var prefix [8]byte
	binary.BigEndian.PutUint64(prefix[:], height)
	iter := db.NewIterator(util.BytesPrefix(prefix[:]), nil)
	for iter.Next() {
		batch.Delete(iter.Key())
	}
	iter.Release()
	if err = iter.Error(); err != nil {
		return err
	}
	return err
}

func db_store_block(batch *leveldb.Batch, block *BlockInfo) (err error) {
	var block_fingerprint hash.Hash = sha256.New()
	var block_metadata BlockMetadata
	block_metadata.Height = block.Height

	for _, commit := range block.Commits {
		tag_data := encode_tag(commit.Tag)
		batch.Put(tag_data[:], commit.Commit[:])
		block_fingerprint.Write(commit.Commit[:])
	}
	if len(block.Commits) > 0 {
		block_fingerprint.Sum(block_metadata.Fingerprint[0:0:32])
		db_update_fingerprint(block.Height, block_metadata.Fingerprint)
	}

	key, data := encode_block_metadata(block_metadata)
	batch.Put(key[:], data[:])
	return err
}

func db_process_block(batch *leveldb.Batch, block BlockInfo) (err error) {
	if err = comb_process_block(block.Height, block.Commits); err != nil {
		return err
	}

	block.Commits = libcomb.GetCommitDifference()

	if len(block.Commits) == 0 {
		return nil
	}

	if err = db_store_block(batch, &block); err != nil {
		return err
	}
	return nil
}

func db_load_blocks(end_height uint64) (err error) {
	var start_height uint64 = libcomb.GetHeight()
	if start_height == end_height {
		return nil
	}
	var block_fingerprint hash.Hash = sha256.New()
	block_fingerprint.Reset()
	var current_block BlockMetadata
	current_block.Height = start_height
	var current_block_corrupt bool
	var first_block bool = true
	var current_commit libcomb.Commit

	var block_commits []libcomb.Commit = nil

	var corrupt_blocks map[uint64]struct{} = make(map[uint64]struct{})

	//probably exists...
	var start_key [8]byte = uint64_to_bytes(start_height)

	iteration := 0
	iter := db.NewIterator(nil, nil)
	for iter.Seek(start_key[:]); ; iter.Next() {
		iteration++
		at_end := !iter.Valid()
		//process the last block, if one exists
		if (at_end || len(iter.Key()) == DB_BLOCK_KEY_LENGTH) && !first_block {
			var our_fingerprint [32]byte
			if len(block_commits) > 0 {
				block_fingerprint.Sum(our_fingerprint[0:0:32])
				block_fingerprint.Reset()
			}

			if our_fingerprint != current_block.Fingerprint {
				log.Printf("(db) fingerprint mismatch at height %d\n", current_block.Height)
				//log.Printf("%X\t%X\n", our_fingerprint, current_block.Fingerprint)
				current_block_corrupt = true
			}
			if current_block_corrupt {
				corrupt_blocks[current_block.Height] = struct{}{}
			}

			//dont load past any corruption
			if len(corrupt_blocks) == 0 {
				db_update_fingerprint(current_block.Height, our_fingerprint)
				if err = comb_process_block(current_block.Height, block_commits); err != nil {
					return err
				}
			}
		}
		if at_end {
			break
		}

		key := iter.Key()
		val := iter.Value()
		switch len(key) {
		case DB_BLOCK_KEY_LENGTH:
			current_block = BlockMetadata{}
			current_block_corrupt = false
			block_commits = nil
			current_block.Height = uint64(binary.BigEndian.Uint64(key[0:8]))
			first_block = false
			if len(val) != 64 {
				current_block_corrupt = true
				log.Printf("(db) malformed block header at height %d\n", current_block.Height)
				continue
			}
			current_block = decode_block_metadata(key, val)
		case DB_COMMIT_KEY_LENGTH:
			if len(val) != 32 {
				current_block_corrupt = true
				log.Printf("(db) malformed commit at height %d\n", current_block.Height)
				continue
			}

			current_commit.Tag = decode_tag(key)
			current_commit.Commit = decode_commit(val)
			block_commits = append(block_commits, current_commit)
			block_fingerprint.Write(current_commit.Commit[:])
		}
	}
	iter.Release()
	if err = iter.Error(); err != nil {
		return err
	}

	if DBInfo.CorruptedBlocks == nil {
		DBInfo.CorruptedBlocks = corrupt_blocks
	} else {
		return errors.New("old corrupted blocks are yet to be processed")
	}
	DBInfo.StoredHeight = current_block.Height

	log.Printf("(db) %d of %d blocks loaded. %d blocks corrupted\n", libcomb.GetHeight()-start_height, DBInfo.StoredHeight-start_height, len(DBInfo.CorruptedBlocks))
	return nil
}

func db_load_all() (err error) {
	libcomb.ResetCOMB()
	COMBInfo.Height = libcomb.GetHeight()
	err = db_load_blocks(^uint64(0))
	return err
}

func db_init() {
	batch := new(leveldb.Batch)
	var key [2]byte
	var value [2]byte
	binary.BigEndian.PutUint16(value[:], DB_CURRENT_VERSION)
	batch.Put(key[:], value[:])
	db_write_batch(batch)

	DBInfo.Version = DB_CURRENT_VERSION
}

func db_start() {
	var err error

	if db_is_new {
		log.Printf("(db) new database created (version %d)\n", DB_CURRENT_VERSION)
		db_init()
		return
	}

	log.Printf("(db) started. loading...")

	DBInfo.Version = db_get_version()
	DBInfo.StoredHeight = libcomb.GetHeight()

	switch DBInfo.Version {
	case DB_LEGACY_VERSION:
		err = db_legacy_migrate()
		if err != nil {
			log.Fatal("(db) migration error (" + err.Error() + ")")
		}
	case DB_CURRENT_VERSION:
	default:
		log.Fatal("(db) error unknown db version (" + fmt.Sprint(DBInfo.Version) + ")")
	}

	err = db_load_all()
	if err != nil {
		log.Printf("(db) load error (" + err.Error() + ")")
	}

	DBInfo.Fingerprint = db_get_fingerprint(COMBInfo.Height)
}
