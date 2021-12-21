package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"

	"libcomb"
)

type Control struct{}

func (c *Control) LoadTransaction(args *Transaction, reply *string) (err error) {
	var tx libcomb.Transaction

	if _, tx, err = args.Parse(); err != nil {
		return err
	}

	var id [32]byte
	if id, err = libcomb.LoadTransaction(tx); err != nil {
		return err
	}

	Wallet.TXs[id] = tx
	*reply = fmt.Sprintf("%X", id)
	return nil
}
func (c *Control) LoadKey(args *Key, reply *string) (err error) {
	var w libcomb.Key
	if _, w, err = args.Parse(); err != nil {
		return err
	}
	var address [32]byte = libcomb.LoadKey(w)
	Wallet.Keys[address] = w
	*reply = fmt.Sprintf("%X", address)
	return nil
}
func (c *Control) LoadStack(args *Stack, reply *string) (err error) {
	var s libcomb.Stack
	if _, s, err = args.Parse(); err != nil {
		return err
	}
	var address [32]byte = libcomb.LoadStack(s)
	Wallet.Stacks[address] = s
	*reply = fmt.Sprintf("%X", address)
	return nil
}
func (c *Control) LoadDecider(args *Decider, reply *string) (err error) {
	var d libcomb.Decider
	if _, d, err = args.Parse(); err != nil {
		return err
	}
	*reply = fmt.Sprintf("%X", libcomb.LoadDecider(d))
	return nil
}
func (c *Control) LoadMerkleSegment(args *MerkleSegment, reply *string) (err error) {
	var m libcomb.MerkleSegment
	if _, m, err = args.Parse(); err != nil {
		return err
	}
	*reply = fmt.Sprintf("%X", libcomb.LoadMerkleSegment(m))
	return nil
}

func (c *Control) GenerateKey(args *interface{}, reply *Key) error {
	key := libcomb.GenerateKey()
	*reply = key_stringify(key)
	return nil
}

func (c *Control) ConstructStack(args *Stack, reply *Stack) (err error) {
	*reply = *args

	var address [32]byte
	if address, _, err = args.Parse(); err != nil {
		return err
	}
	(*reply).Address = fmt.Sprintf("%X", address)
	return err
}

func (c *Control) GenerateDecider(args *interface{}, reply *Decider) error {
	decider := libcomb.GenerateDecider()
	*reply = decider_stringify(decider)
	return nil
}

func (c *Control) ConstructTransaction(args *RawTransaction, result *Transaction) (err error) {
	var rtx libcomb.RawTransaction
	if _, rtx, err = args.Parse(); err != nil {
		return err
	}

	var tx libcomb.Transaction
	if tx, err = libcomb.SignTransaction(rtx); err != nil {
		return err
	}

	*result = tx_stringify(tx)
	return nil
}

type SignDeciderArgs struct {
	Decider Decider
	Number  uint16
}

func (c *Control) SignDecider(args *SignDeciderArgs, result *string) (err error) {
	var d libcomb.Decider
	var l libcomb.LongDecider
	if _, d, err = args.Decider.Parse(); err != nil {
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
	if _, contract, err = args.Contract.Parse(); err != nil {
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

	return nil
}

type SweepArgs struct {
	Target string
	Range  uint64
}

func (c *Control) DoSweep(args *SweepArgs, reply *struct{}) (err error) {
	log.Printf("sweep %s %d\n", args.Target, args.Range)
	var stack libcomb.Stack
	var address [32]byte
	if stack.Change, err = parse_hex(args.Target); err != nil {
		return err
	}
	for i := uint64(0); i < args.Range; i++ {
		binary.BigEndian.PutUint64(stack.Destination[:], i)
		address = libcomb.GetStackAddress(stack)
		if libcomb.GetAddressBalance(libcomb.CommitAddress(address)) > 0 {
			log.Println(i, stack_export(stack))
		}
	}
	log.Printf("done\n")
	return nil
}

func (c *Control) CommitAddress(args *string, reply *string) (err error) {
	var address [32]byte
	if address, err = parse_hex(*args); err != nil {
		return err
	}
	address = libcomb.CommitAddress(address)
	*reply = fmt.Sprintf("%X", address)
	return nil
}

func (c *Control) GetMissingCommits(args *[]string, reply *[]string) (err error) {
	//takes list of addresses and checks if they have been commited, returns commits that are missing
	var address [32]byte

	for _, a := range *args {
		if address, err = parse_hex(a); err != nil {
			return err
		}
		address = libcomb.CommitAddress(address)
		if !libcomb.HaveCommit(address) {
			*reply = append(*reply, a)
		}
	}
	return nil
}

func (c *Control) FindCommitOccurances(args *string, reply *[]uint64) (err error) {
	var address [32]byte
	if address, err = parse_hex(*args); err != nil {
		return err
	}
	*reply = db_find_commits(address)
	for _, i := range *reply {
		log.Println(i)
	}
	return nil
}

func (c *Control) LoadWallet(args *string, reply *struct{}) (err error) {
	err = wallet_load(*args)
	return err
}

func (c *Control) SaveWallet(args *struct{}, reply *string) (err error) {
	*reply = wallet_export()
	return err
}

func (c *Control) GetWallet(args *struct{}, reply *StringWallet) (err error) {
	*reply = wallet_stringify()
	return nil
}

type StatusInfo struct {
	COMBHeight     uint64
	BTCHeight      uint64
	BTCKnownHeight uint64
	Commits        uint64
	Status         string
	Network        string
}

func (c *Control) GetStatus(args *struct{}, reply *StatusInfo) (err error) {
	reply.COMBHeight = COMBInfo.Height
	reply.BTCHeight = BTC.Chain.Height
	reply.BTCKnownHeight = BTC.Chain.KnownHeight
	reply.Commits = libcomb.GetCommitCount()
	reply.Status = COMBInfo.Status
	reply.Network = COMBInfo.Network
	return nil
}

func (c *Control) DoDump(args *struct{}, reply *struct{}) (err error) {
	combcore_dump()
	return nil
}
