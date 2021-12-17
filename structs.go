package main

import (
	"errors"
	"fmt"
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

func parse_hex(hex string) (raw [32]byte, err error) {
	if len(hex) < 64 {
		err = errors.New("hex too short")
		return raw, err
	}
	if len(hex) > 64 {
		err = errors.New("hex too long")
		return raw, err
	}

	hex = strings.ToUpper(hex)

	if err = checkHEX32(hex); err != nil {
		return raw, err
	}

	raw = hex2byte32([]byte(hex))
	return raw, nil
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

func wallet_export() (out string) {
	var empty [32]byte
	for _, k := range Wallet.Keys {
		out += k.Export() + "\n"
	}
	for _, s := range Wallet.Stacks {
		out += s.Export() + "\n"
	}
	for _, tx := range Wallet.TXs {
		out += tx.Export() + "\n"
	}
	for _, d := range Wallet.Deciders {
		out += d.Export(empty) + "\n"
	}
	for _, m := range Wallet.Merkles {
		out += m.Export() + "\n"
	}
	return out
}
