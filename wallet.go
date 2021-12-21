package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"strings"

	"libcomb"
)

type Key struct {
	Public  string
	Private [21]string
	Balance uint64
}

type Stack struct {
	Destination string
	Sum         uint64
	Change      string
	Address     string
}

type RawTransaction struct {
	Source      string
	Destination string
	ID          string
}

type Transaction struct {
	Source      string
	Destination string
	Signature   [21]string
	Active      bool
	ID          string
}

type Decider struct {
	Private [2]string
}

type MerkleSegment struct {
	Short    [2]string
	Long     [2]string
	Branches [16]string
	Leaf     string
	Next     string
}

type Contract struct {
	Short [2]string
	Next  string
	Root  string
}

type StringWallet struct {
	Keys     []Key
	Stacks   []Stack
	TXs      []Transaction
	Deciders []Decider
	Merkles  []MerkleSegment
}

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
		data = hex2byte([]byte(construct))
		address, err = wallet_load_stack(data)
	}
	if strings.HasPrefix(construct, COMBInfo.Prefix["tx"]) {
		construct = strings.TrimPrefix(construct, COMBInfo.Prefix["tx"])
		data = hex2byte([]byte(construct))
		address, err = wallet_load_transaction(data)
	}

	if strings.HasPrefix(construct, COMBInfo.Prefix["key"]) {
		construct = strings.TrimPrefix(construct, COMBInfo.Prefix["key"])
		data = hex2byte([]byte(construct))
		address, err = wallet_load_key(data)
	}

	if strings.HasPrefix(construct, COMBInfo.Prefix["merkle"]) {
		construct = strings.TrimPrefix(construct, COMBInfo.Prefix["merkle"])
		data = hex2byte([]byte(construct))
		address, err = wallet_load_merkle_segment(data)
	}

	if strings.HasPrefix(construct, COMBInfo.Prefix["decider"]) {
		construct = strings.TrimPrefix(construct, COMBInfo.Prefix["decider"])
		data = hex2byte([]byte(construct))
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

func (w *Key) Parse() (id [32]byte, lw libcomb.Key, err error) {
	if lw.Public, err = parse_hex(w.Public); err != nil {
		return id, libcomb.Key{}, err
	}
	for i := range w.Private {
		if lw.Private[i], err = parse_hex(w.Private[i]); err != nil {
			return id, libcomb.Key{}, err
		}
	}
	return lw.Public, lw, err
}

func (s *Stack) Parse() (id [32]byte, ls libcomb.Stack, err error) {
	if ls.Destination, err = parse_hex(s.Destination); err != nil {
		return id, libcomb.Stack{}, err
	}
	ls.Sum = s.Sum
	if ls.Change, err = parse_hex(s.Change); err != nil {
		return id, libcomb.Stack{}, err
	}
	id = libcomb.GetStackAddress(ls)
	return id, ls, err
}

func (tx *RawTransaction) Parse() (id [32]byte, ltx libcomb.RawTransaction, err error) {
	if ltx.Destination, err = parse_hex(tx.Destination); err != nil {
		return id, libcomb.RawTransaction{}, err
	}
	if ltx.Source, err = parse_hex(tx.Source); err != nil {
		return id, libcomb.RawTransaction{}, err
	}
	var tmptx libcomb.Transaction
	tmptx.Source = ltx.Source
	tmptx.Destination = ltx.Destination
	id = libcomb.GetTXID(tmptx)
	return id, ltx, err
}

func (tx *Transaction) Parse() (id [32]byte, ltx libcomb.Transaction, err error) {
	if ltx.Destination, err = parse_hex(tx.Destination); err != nil {
		return id, libcomb.Transaction{}, err
	}
	if ltx.Source, err = parse_hex(tx.Source); err != nil {
		return id, libcomb.Transaction{}, err
	}
	for i := range tx.Signature {
		if ltx.Signature[i], err = parse_hex(tx.Signature[i]); err != nil {
			return id, libcomb.Transaction{}, err
		}
	}
	id = libcomb.GetTXID(ltx)
	return id, ltx, err
}

func (d *Decider) Parse() (id [32]byte, ld libcomb.Decider, err error) {
	if ld.Private[0], err = parse_hex(d.Private[0]); err != nil {
		return id, libcomb.Decider{}, err
	}
	if ld.Private[1], err = parse_hex(d.Private[1]); err != nil {
		return id, libcomb.Decider{}, err
	}

	id = libcomb.ComputeDeciderAddress(ld)
	return id, ld, err
}

func (m *MerkleSegment) Parse() (id [32]byte, lm libcomb.MerkleSegment, err error) {
	if lm.Short[0], err = parse_hex(m.Short[0]); err != nil {
		return id, libcomb.MerkleSegment{}, err
	}
	if lm.Short[1], err = parse_hex(m.Short[1]); err != nil {
		return id, libcomb.MerkleSegment{}, err
	}
	if lm.Long[0], err = parse_hex(m.Long[0]); err != nil {
		return id, libcomb.MerkleSegment{}, err
	}
	if lm.Long[1], err = parse_hex(m.Long[1]); err != nil {
		return id, libcomb.MerkleSegment{}, err
	}
	if lm.Leaf, err = parse_hex(m.Leaf); err != nil {
		return id, libcomb.MerkleSegment{}, err
	}
	if lm.Next, err = parse_hex(m.Next); err != nil {
		return id, libcomb.MerkleSegment{}, err
	}
	for i := range m.Branches {
		if lm.Branches[i], err = parse_hex(m.Branches[i]); err != nil {
			return id, libcomb.MerkleSegment{}, err
		}
	}

	id = libcomb.ComputeMerkleSegmentAddress(lm)
	return id, lm, err
}

func (c *Contract) Parse() (id [32]byte, lc libcomb.Contract, err error) {
	if lc.Short[0], err = parse_hex(c.Short[0]); err != nil {
		return id, libcomb.Contract{}, err
	}
	if lc.Short[1], err = parse_hex(c.Short[1]); err != nil {
		return id, libcomb.Contract{}, err
	}
	if lc.Next, err = parse_hex(c.Next); err != nil {
		return id, libcomb.Contract{}, err
	}
	if lc.Root, err = parse_hex(c.Root); err != nil {
		return id, libcomb.Contract{}, err
	}

	return id, lc, err
}

func (sw *StringWallet) Parse() {
	for _, k := range sw.Keys {
		if id, lk, err := k.Parse(); err == nil {
			Wallet.Keys[id] = lk
		}
	}
	for _, s := range sw.Stacks {
		if id, ls, err := s.Parse(); err == nil {
			Wallet.Stacks[id] = ls
		}
	}
	for _, tx := range sw.TXs {
		if id, ltx, err := tx.Parse(); err == nil {
			Wallet.TXs[id] = ltx
		}
	}
	for _, d := range sw.Deciders {
		if id, ld, err := d.Parse(); err == nil {
			Wallet.Deciders[id] = ld
		}
	}
	for _, m := range sw.Merkles {
		if id, lm, err := m.Parse(); err == nil {
			Wallet.Merkles[id] = lm
		}
	}
}

func key_stringify(w libcomb.Key) (sw Key) {
	sw.Public = fmt.Sprintf("%X", w.Public)
	for i := range w.Private {
		sw.Private[i] = fmt.Sprintf("%X", w.Private[i])
	}
	sw.Balance = libcomb.GetAddressBalance(w.Public)
	return sw
}

func stack_stringify(s libcomb.Stack) (ss Stack) {
	ss.Change = fmt.Sprintf("%X", s.Change)
	ss.Destination = fmt.Sprintf("%X", s.Destination)
	ss.Sum = s.Sum
	ss.Address = fmt.Sprintf("%X", libcomb.GetStackAddress(s))
	return ss
}

func tx_stringify(tx libcomb.Transaction) (stx Transaction) {
	stx.Source = fmt.Sprintf("%X", tx.Source)
	stx.Destination = fmt.Sprintf("%X", tx.Destination)
	for i := range tx.Signature {
		stx.Signature[i] = fmt.Sprintf("%X", tx.Signature[i])
	}
	stx.ID = fmt.Sprintf("%X", libcomb.GetTXID(tx))
	stx.Active = libcomb.IsTransactionActive(tx.Source, tx.Destination)
	return stx
}

func decider_stringify(d libcomb.Decider) (sd Decider) {
	sd.Private[0] = fmt.Sprintf("%X", d.Private[0])
	sd.Private[1] = fmt.Sprintf("%X", d.Private[1])
	return sd
}

func contract_stringify(c libcomb.Contract) (sc Contract) {
	sc.Next = fmt.Sprintf("%X", c.Next)
	sc.Root = fmt.Sprintf("%X", c.Root)
	sc.Short[0] = fmt.Sprintf("%X", c.Short[0])
	sc.Short[1] = fmt.Sprintf("%X", c.Short[1])
	return sc
}

func merkle_segment_stringify(m libcomb.MerkleSegment) (sm MerkleSegment) {
	sm.Short[0] = fmt.Sprintf("%X", m.Short[0])
	sm.Short[1] = fmt.Sprintf("%X", m.Short[1])

	sm.Long[0] = fmt.Sprintf("%X", m.Long[0])
	sm.Long[1] = fmt.Sprintf("%X", m.Long[1])

	sm.Leaf = fmt.Sprintf("%X", m.Leaf)
	sm.Next = fmt.Sprintf("%X", m.Next)

	for i := range m.Branches {
		sm.Branches[i] = fmt.Sprintf("%X", m.Branches[i])
	}

	return sm
}

func wallet_stringify() StringWallet {
	var w StringWallet
	for _, k := range Wallet.Keys {
		w.Keys = append(w.Keys, key_stringify(k))
	}
	for _, s := range Wallet.Stacks {
		w.Stacks = append(w.Stacks, stack_stringify(s))
	}
	for _, tx := range Wallet.TXs {
		w.TXs = append(w.TXs, tx_stringify(tx))
	}
	for _, d := range Wallet.Deciders {
		w.Deciders = append(w.Deciders, decider_stringify(d))
	}
	for _, m := range Wallet.Merkles {
		w.Merkles = append(w.Merkles, merkle_segment_stringify(m))
	}
	return w
}

func key_export(w libcomb.Key) (out string) {
	out = COMBInfo.Prefix["key"]
	for _, k := range w.Private {
		out += fmt.Sprintf("%X", k)
	}
	return out
}

func stack_export(s libcomb.Stack) (out string) {
	var sum [8]byte = uint64_to_bytes(s.Sum)
	out = fmt.Sprintf("%s%X%X%X", COMBInfo.Prefix["stack"], s.Change, s.Destination, sum)
	return out
}

func transaction_export(tx libcomb.Transaction) (out string) {
	out = fmt.Sprintf("%s%X%X", COMBInfo.Prefix["tx"], tx.Source, tx.Destination)
	for _, k := range tx.Signature {
		out += fmt.Sprintf("%X", k)
	}
	return out
}

func decider_export(d libcomb.Decider, next [32]byte) (out string) {
	out = fmt.Sprintf("%s%X%X%X", COMBInfo.Prefix["decider"], next, d.Private[0], d.Private[1])
	return out
}

func merkle_export(m libcomb.MerkleSegment) (out string) {
	out = fmt.Sprintf("%s%X%X%X%X", COMBInfo.Prefix["merkle"], m.Short[0], m.Short[1], m.Long[0], m.Long[1])
	for _, b := range m.Branches {
		out += fmt.Sprintf("%X", b)
	}
	out += fmt.Sprintf("%X%X", m.Leaf, m.Next)
	return out
}

func wallet_export() (out string) {
	var empty [32]byte
	for _, k := range Wallet.Keys {
		out += key_export(k) + "\n"
	}
	for _, s := range Wallet.Stacks {
		out += stack_export(s) + "\n"
	}
	for _, tx := range Wallet.TXs {
		out += transaction_export(tx) + "\n"
	}
	for _, d := range Wallet.Deciders {
		out += decider_export(d, empty) + "\n"
	}
	for _, m := range Wallet.Merkles {
		out += merkle_export(m) + "\n"
	}
	return out
}
