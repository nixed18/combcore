package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"libcomb"
	"net/http"
	"strings"
	"time"
)

type RPCResult struct {
	Result json.RawMessage
}

var btc_client = &http.Client{}

func init() {
	btc_client.Timeout = time.Second * 10
}

func btc_rpc_call(client *http.Client, method, params string) (json.RawMessage, error) {
	body := strings.NewReader("{\"jsonrpc\":\"1.0\",\"id\":\"" + method + "\",\"method\":\"" + method + "\",\"params\":[" + params + "]}")
	request, _ := http.NewRequest("POST", "http://"+*btc_username+":"+*btc_password+"@"+*btc_host+":"+fmt.Sprint(*btc_port), body)
	request.Header.Set("Content-Type", "text/plain")
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("btc rpc request failed (" + err.Error() + ")")
	}

	response_data, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil, errors.New("btc rpc response io error (" + err.Error() + ")")
	}

	var response_result RPCResult

	err = json.Unmarshal(response_data, &response_result)
	if err != nil {
		return nil, errors.New("btc rpc response is gibberish (" + err.Error() + ")")
	}

	return response_result.Result, nil
}

func btc_is_alive(client *http.Client) (alive bool, version string, err error) {
	var info_json json.RawMessage
	var info struct{ Subversion string }

	//try to get the node version as an alive test
	if info_json, err = btc_rpc_call(client, "getnetworkinfo", ""); err != nil {
		return false, "", err
	}

	if err = json.Unmarshal(info_json, &info); err != nil {
		return false, "", err
	}
	//if BTC doesnt know its own version, its not alive
	return info.Subversion != "", info.Subversion, nil
}

func btc_get_sync_state(client *http.Client) (active_hash [32]byte, active_height, known_height uint64, err error) {
	var chains_json json.RawMessage
	var chains []struct {
		Hash   string
		Height uint64
		Status string
	}

	//get a list of all the chains BTC is aware of
	if chains_json, err = btc_rpc_call(client, "getchaintips", ""); err != nil {
		return active_hash, 0, 0, err
	}

	if err = json.Unmarshal(chains_json, &chains); err != nil {
		return active_hash, 0, 0, err
	}

	if len(chains) == 0 {
		return active_hash, 0, 0, err
	}

	//BTC is not synced if it knows about a valid chain thats longer than the active one
	for _, tip := range chains {
		switch tip.Status {
		//the longest fully validated chain (headers + blocks)
		case "active":
			hash := strings.ToUpper(tip.Hash)
			active_hash = hex2byte32([]byte(hash))
			active_height = tip.Height
			if known_height < tip.Height {
				known_height = tip.Height
			}
		//headers-only:		chain with valid headers
		//valid-headers: 	above + blocks are downloaded
		//valid-fork: 		above + blocks are validated
		case "headers-only", "valid-headers", "valid-fork":
			if known_height < tip.Height {
				known_height = tip.Height
			}
		}
	}

	return active_hash, active_height, known_height, nil
}

func btc_get_block_hash(client *http.Client, height uint64) (hash string, err error) {
	var block_hash json.RawMessage
	if block_hash, err = btc_rpc_call(client, "getblockhash", fmt.Sprint(height)); err != nil {
		return "", err
	}
	hash = string(block_hash)
	return hash, nil
}

func btc_get_block(client *http.Client, height uint64) (block BlockInfo, err error) {
	var block_json json.RawMessage
	var block_hash string
	var rawblock struct {
		Height uint64
		TX     []struct {
			VOut []struct {
				ScriptPubKey struct {
					Type string
					Hex  string
				}
			}
		}
	}

	if block_hash, err = btc_get_block_hash(client, height); err != nil {
		return block, err
	}

	if block_json, err = btc_rpc_call(client, "getblock", block_hash+",2"); err != nil {
		return block, err
	}

	if err = json.Unmarshal(block_json, &rawblock); err != nil {
		return block, err
	}

	var commit libcomb.Commit
	block.Height = rawblock.Height
	commit.Tag.Height = rawblock.Height
	//pull out the p2wsh vouts
	for _, tx := range rawblock.TX {
		for _, vout := range tx.VOut {
			data := vout.ScriptPubKey
			if data.Type == "witness_v0_scripthash" {
				//remove the 0020 (opcode for "push 32 bytes onto the stack")
				data.Hex = strings.ToUpper(data.Hex[4:])
				commit.Commit = hex2byte32([]byte(data.Hex))
				commit.Tag.Commitnum++
				block.Commits = append(block.Commits, commit)
			}
		}
	}

	return block, nil
}

func btc_wait_for_block(client *http.Client) (err error) {
	var block_json json.RawMessage
	var block struct {
		Hash   string
		Height uint64
	}

	if block_json, err = btc_rpc_call(client, "waitfornewblock", ""); err != nil {
		return err
	}

	if err = json.Unmarshal(block_json, &block); err != nil {
		return err
	}
	NodeInfo.height = block.Height
	block.Hash = strings.ToUpper(block.Hash)
	NodeInfo.hash = hex2byte32([]byte(block.Hash))
	return nil
}
