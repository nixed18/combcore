package main

import (
	"fmt"

	"libcomb"
)

type P2P struct{}

type InfoResponse struct {
	Version     string `json:"comb"`
	Height      uint64 `json:"height"`
	Fingerprint string `json:"fingerprint"`
}

func (p *P2P) GetInfo(args *interface{}, reply *InfoResponse) error {
	reply.Version = libcomb.Version
	reply.Height = libcomb.GetHeight()
	reply.Fingerprint = fmt.Sprintf("%X", DBInfo.Fingerprint)
	return nil
}

func (p *P2P) GetFingerprint(args *uint64, reply *string) error {
	f := db_get_fingerprint(*args)
	f2 := db_compute_fingerprint(*args)
	*reply = fmt.Sprintf("%X %X", f, f2)
	return nil
}

func (p *P2P) GetLegacyFingerprint(args *uint64, reply *string) error {
	*reply = fmt.Sprintf("%X", db_compute_legacy_fingerprint(*args))
	return nil
}
