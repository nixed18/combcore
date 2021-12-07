package main

import (
	"encoding/json"
	"libcomb"
	"log"
	"strings"
	"sync"
	"time"
)

type block_template struct {
	Transactions []struct {
		Data string
		TXID string
		Fee  uint64
	}
}

type rawtx struct {
	VSize uint64
	VOut  []struct {
		ScriptPubKey struct {
			Type string
			Hex  string
		}
	}
}

var AutoInfo struct {
	Active  bool
	Buy     float64
	Guess   [32]byte
	Commits [][32]byte
	Fees    map[[32]byte]float64
	Mutex   sync.Mutex
}

func init() {
	AutoInfo.Fees = make(map[[32]byte]float64)
}

func autoclaim_start() {
	AutoInfo.Active = true
	go func() {
		for {
			autoclaim_guess_combbase()
			time.Sleep(10 * time.Second)
		}
	}()
}

func autoclaim_process_combbase(combbase [32]byte) {
	if !AutoInfo.Active {
		return
	}
	//var idx int = -1
	var fee float64
	var ok bool

	AutoInfo.Mutex.Lock()
	/*for i := range AutoInfo.Commits {
		if AutoInfo.Commits[i] == combbase {
			idx = i
		}
	}*/

	if fee, ok = AutoInfo.Fees[combbase]; !ok {
		fee = 0
	}

	//log.Printf("(autoclaim) combbase %f (%d) vs %f\n", fee, idx, AutoInfo.Fees[AutoInfo.Guess])
	if AutoInfo.Buy > 0 {
		if AutoInfo.Buy > fee {
			log.Printf("(autoclaim) buy successful")
		} else {
			log.Printf("(autoclaim) buy failed")
		}
	}
	AutoInfo.Buy = 0
	AutoInfo.Fees = make(map[[32]byte]float64)
	AutoInfo.Commits = nil
	AutoInfo.Mutex.Unlock()
}

func autoclaim_evaluate_signal(fee float64, delta int64, size float64) {
	cost := uint64(153*fee + 301)
	if AutoInfo.Buy == 0 && cost < 9000 && delta >= 8*60 {
		AutoInfo.Buy = fee * 1.1
		log.Printf("(autoclaim) buy %d@%f\n", cost, AutoInfo.Fees[AutoInfo.Commits[0]])
	}
}

func autoclaim_guess_combbase() {
	var err error
	var j json.RawMessage
	var commit [32]byte

	if !NodeInfo.alive {
		return
	}

	if j, err = btc_rpc_call(btc_client, "getblocktemplate", "{\"rules\": [\"segwit\"]}"); err == nil {
		var block block_template
		err = json.Unmarshal(j, &block)
		if err != nil {
			log.Println(err.Error())
		}

		var tx rawtx
		AutoInfo.Mutex.Lock()
		var size int = 80

		for k := range block.Transactions {
			id := block.Transactions[k].TXID
			if j, err = btc_rpc_call(btc_client, "getrawtransaction", "\""+id+"\",true"); err != nil {
				log.Println(err.Error())
			}

			if err = json.Unmarshal(j, &tx); err != nil {
				log.Println(err.Error())
			}
			fee := float64(block.Transactions[k].Fee) / float64(tx.VSize)

			for _, vout := range tx.VOut {
				data := vout.ScriptPubKey
				if data.Type == "witness_v0_scripthash" {
					data.Hex = strings.ToUpper(data.Hex[4:])
					commit = hex2byte32([]byte(data.Hex))

					if !libcomb.HaveCommit(commit) {
						AutoInfo.Commits = append(AutoInfo.Commits, commit)
						AutoInfo.Fees[commit] = fee
					}
				}
			}
			size += len(block.Transactions[k].Data) / 2
		}

		if len(AutoInfo.Commits) != 0 {
			fee := AutoInfo.Fees[AutoInfo.Commits[0]]
			if AutoInfo.Guess != AutoInfo.Commits[0] {
				//log.Printf("(autoclaim) guess %f\n", fee)
			}
			AutoInfo.Guess = AutoInfo.Commits[0]

			delta := time.Now().Unix() - NodeInfo.last_block
			autoclaim_evaluate_signal(fee, delta, float64(size)/1048576.0)
		} else {
			AutoInfo.Guess = empty
		}
		AutoInfo.Mutex.Unlock()
	} else {
		log.Printf("(autoclaim) %s\n", err.Error())
	}
}
