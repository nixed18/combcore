package main

import (
	"fmt"

	"libcomb"
)

type P2P struct{}

type InfoResponse struct {
	Version string `json:"comb"`
	Height  uint64 `json:"height"`
}

func (p *P2P) GetInfo(args *interface{}, reply *InfoResponse) error {
	reply.Version = libcomb.Version
	reply.Height = libcomb.GetHeight()
	return nil
}

func (p *P2P) GetFingerprint(args *uint64, reply *string) error {
	*reply = fmt.Sprintf("%X", db_compute_fingerprint(*args))
	return nil
}
