package main

import (
	"encoding/json"
	"libcomb"
	"log"
	"net/http"
	"sort"
	"strings"
)

type mempool map[string]struct {
	WTXID string
	Fees  struct {
		Base float64
	}
	SpentBy []string
}

type rawtx struct {
	VOut []struct {
		ScriptPubKey struct {
			Type string
			Hex  string
		}
	}
}

var AutoInfo struct {
	Guess   [32]byte
	Commits [][32]byte
}

func guess_combbase() {
	var err error
	var j json.RawMessage
	var combbase [32]byte
	client := &http.Client{}
	if j, err = btc_rpc_call(client, "getrawmempool", "true"); err == nil {
		var pool mempool = make(mempool)
		err = json.Unmarshal(j, &pool)
		if err != nil {
			log.Println(err.Error())
		}
		ids := make([]string, 0, len(pool))
		for k := range pool {
			ids = append(ids, k)
		}
		sort.SliceStable(ids, func(i, j int) bool {
			return pool[ids[i]].Fees.Base > pool[ids[j]].Fees.Base
		})

		var tx rawtx
		for _, id := range ids {
			if j, err = btc_rpc_call(client, "getrawtransaction", "\""+id+"\",true"); err != nil {
				log.Println(err.Error())
			}

			if err = json.Unmarshal(j, &tx); err != nil {
				log.Println(err.Error())
			}

			for _, vout := range tx.VOut {
				data := vout.ScriptPubKey
				if data.Type == "witness_v0_scripthash" {
					//remove the 0020 (opcode for "push 32 bytes onto the stack")
					data.Hex = strings.ToUpper(data.Hex[4:])
					combbase = hex2byte32([]byte(data.Hex))

					if !libcomb.HaveCommit(combbase) {
						NodeInfo.guess = combbase
						return
					}

				}
			}
		}
		var empty [32]byte
		NodeInfo.guess = empty
		log.Printf("empty\n")
	} else {
		log.Printf("(autoclaim) %s\n", err.Error())
	}
}
