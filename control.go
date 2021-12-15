package main

import (
	"errors"
	"fmt"

	"libcomb"
)

type Control struct{}

func (c *Control) LoadTransaction(args *SignedTransaction, reply *struct{}) (err error) {
	var tx libcomb.Transaction
	var signature [21][32]byte

	if tx, err = args.Tx.Parse(); err != nil {
		return err
	}
	for i := range args.Signature {
		if signature[i], err = parse_hex(args.Signature[i]); err != nil {
			return err
		}
	}

	libcomb.LoadTransaction(tx, signature)
	return nil
}
func (c *Control) LoadWalletKey(args *WalletKey, reply *string) (err error) {
	var w libcomb.WalletKey
	if w, err = args.Parse(); err != nil {
		return err
	}
	*reply = fmt.Sprintf("%X", libcomb.LoadWalletKey(w))
	return nil
}
func (c *Control) LoadStack(args *Stack, reply *string) (err error) {
	var s libcomb.Stack
	if s, err = args.Parse(); err != nil {
		return err
	}
	*reply = fmt.Sprintf("%X", libcomb.LoadStack(s))
	return nil
}
func (c *Control) LoadDecider(args *Decider, reply *string) (err error) {
	var d libcomb.Decider
	if d, err = args.Parse(); err != nil {
		return err
	}
	*reply = fmt.Sprintf("%X", libcomb.LoadDecider(d))
	return nil
}
func (c *Control) LoadMerkleSegment(args *MerkleSegment, reply *string) (err error) {
	var m libcomb.MerkleSegment
	if m, err = args.Parse(); err != nil {
		return err
	}
	*reply = fmt.Sprintf("%X", libcomb.LoadMerkleSegment(m))
	return nil
}

func (c *Control) GenerateWalletKey(args *interface{}, reply *WalletKey) error {
	key := libcomb.GenerateWalletKey()
	*reply = wallet_key_stringify(key)
	return nil
}

func (c *Control) GenerateDecider(args *interface{}, reply *Decider) error {
	decider := libcomb.GenerateDecider()
	*reply = decider_stringify(decider)
	return nil
}

func (c *Control) SignTransaction(args *Transaction, result *[21]string) (err error) {
	var tx libcomb.Transaction
	if tx, err = args.Parse(); err != nil {
		return err
	}
	var signature [21][32]byte = libcomb.SignTransaction(tx)
	for i := range signature {
		result[i] = fmt.Sprintf("%X", signature[i])
	}
	return nil
}

type SignDeciderArgs struct {
	Decider Decider
	Number  uint16
}

func (c *Control) SignDecider(args *SignDeciderArgs, result *string) (err error) {
	var d libcomb.Decider
	var l libcomb.LongDecider
	if d, err = args.Decider.Parse(); err != nil {
		return err
	}
	l = libcomb.SignDecider(d, args.Number)
	*result = fmt.Sprintf("%X", l.Signature[0]) + fmt.Sprintf("%X", l.Signature[1])
	return nil
}

type ConstructContractArgs struct {
	Short [2]string
	Tree  [65536]string
}

func (c *Control) ConstructContract(args *ConstructContractArgs, result *Contract) (err error) {
	var short libcomb.ShortDecider
	var tree [65536][32]byte
	if short.Public[0], err = parse_hex(args.Short[0]); err != nil {
		return err
	}
	if short.Public[1], err = parse_hex(args.Short[1]); err != nil {
		return err
	}
	for i := range args.Tree {
		if tree[i], err = parse_hex(args.Tree[i]); err != nil {
			return err
		}
	}

	var contract libcomb.Contract = libcomb.ConstructContract(tree, short)
	*result = contract_stringify(contract)
	return nil
}

type DecideContractArgs struct {
	Contract Contract
	Long     string
	Tree     [65536]string
}

func (c *Control) DecideContract(args *DecideContractArgs, result *MerkleSegment) (err error) {
	var contract libcomb.Contract
	var long libcomb.LongDecider
	var tree [65536][32]byte
	if contract, err = args.Contract.Parse(); err != nil {
		return err
	}

	if len(args.Long) != 128 {
		return errors.New("long decider not 64 bytes")
	}

	if long.Signature[0], err = parse_hex(args.Long[0:64]); err != nil {
		return err
	}
	if long.Signature[1], err = parse_hex(args.Long[64:128]); err != nil {
		return err
	}
	for i := range args.Tree {
		if tree[i], err = parse_hex(args.Tree[i]); err != nil {
			return err
		}
	}
	var m libcomb.MerkleSegment = libcomb.DecideContract(contract, long, tree)
	*result = merkle_segment_stringify(m)
	return nil
}

func (c *Control) GetAddressBalance(args *string, reply *uint64) (err error) {
	var address [32]byte
	if address, err = parse_hex(*args); err != nil {
		return err
	}
	*reply = libcomb.GetAddressBalance(address)

	address = libcomb.CommitAddress(address)
	fmt.Printf("alt %d\n", libcomb.GetAddressBalance(address))

	return nil
}

func (c *Control) LoadSave(args *string, reply *struct{}) (err error) {
	err = import_load_save_file(*args)
	return err
}

func (c *Control) DoDump(args *struct{}, reply *struct{}) (err error) {
	combcore_dump()
	return nil
}
