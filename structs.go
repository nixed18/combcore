package main

import (
	"errors"
	"fmt"
	"strings"

	"libcomb"
)

type WalletKey struct {
	Public  string
	Private [21]string
}

type Stack struct {
	Destination string
	Sum         uint64
	Change      string
}

type Transaction struct {
	Source      string
	Destination string
}

type SignedTransaction struct {
	Tx        Transaction
	Signature [21]string
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

func (w *WalletKey) Parse() (lw libcomb.WalletKey, err error) {
	if lw.Public, err = parse_hex(w.Public); err != nil {
		return libcomb.WalletKey{}, err
	}
	for i := range w.Private {
		if lw.Private[i], err = parse_hex(w.Private[i]); err != nil {
			return libcomb.WalletKey{}, err
		}
	}
	return lw, err
}

func (s *Stack) Parse() (ls libcomb.Stack, err error) {
	if ls.Destination, err = parse_hex(s.Destination); err != nil {
		return libcomb.Stack{}, err
	}
	ls.Sum = s.Sum
	if ls.Change, err = parse_hex(s.Change); err != nil {
		return libcomb.Stack{}, err
	}
	return ls, err
}

func (tx *Transaction) Parse() (ltx libcomb.Transaction, err error) {
	if ltx.Destination, err = parse_hex(tx.Destination); err != nil {
		return libcomb.Transaction{}, err
	}
	if ltx.Source, err = parse_hex(tx.Source); err != nil {
		return libcomb.Transaction{}, err
	}
	return ltx, err
}

func (d *Decider) Parse() (ld libcomb.Decider, err error) {
	if ld.Private[0], err = parse_hex(d.Private[0]); err != nil {
		return libcomb.Decider{}, err
	}
	if ld.Private[1], err = parse_hex(d.Private[1]); err != nil {
		return libcomb.Decider{}, err
	}
	return ld, err
}

func (m *MerkleSegment) Parse() (lm libcomb.MerkleSegment, err error) {
	if lm.Short[0], err = parse_hex(m.Short[0]); err != nil {
		return libcomb.MerkleSegment{}, err
	}
	if lm.Short[1], err = parse_hex(m.Short[1]); err != nil {
		return libcomb.MerkleSegment{}, err
	}
	if lm.Long[0], err = parse_hex(m.Long[0]); err != nil {
		return libcomb.MerkleSegment{}, err
	}
	if lm.Long[1], err = parse_hex(m.Long[1]); err != nil {
		return libcomb.MerkleSegment{}, err
	}
	if lm.Leaf, err = parse_hex(m.Leaf); err != nil {
		return libcomb.MerkleSegment{}, err
	}
	if lm.Next, err = parse_hex(m.Next); err != nil {
		return libcomb.MerkleSegment{}, err
	}
	for i := range m.Branches {
		if lm.Branches[i], err = parse_hex(m.Branches[i]); err != nil {
			return libcomb.MerkleSegment{}, err
		}
	}

	return lm, err
}

func (c *Contract) Parse() (lc libcomb.Contract, err error) {
	if lc.Short[0], err = parse_hex(c.Short[0]); err != nil {
		return libcomb.Contract{}, err
	}
	if lc.Short[1], err = parse_hex(c.Short[1]); err != nil {
		return libcomb.Contract{}, err
	}
	if lc.Next, err = parse_hex(c.Next); err != nil {
		return libcomb.Contract{}, err
	}
	if lc.Root, err = parse_hex(c.Root); err != nil {
		return libcomb.Contract{}, err
	}
	return lc, err
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

func wallet_key_stringify(w libcomb.WalletKey) (sw WalletKey) {
	sw.Public = fmt.Sprintf("%X", w.Public)
	for i := range w.Private {
		sw.Private[i] = fmt.Sprintf("%X", w.Private[i])
	}
	return sw
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
