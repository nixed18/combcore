package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func rest_trace_chain(client *http.Client, url string, start_hash [32]byte, end_hash [32]byte, length uint64) (chain [][32]byte, err error) {
	//start_hash exclusive, end_hash inclusive
	var raw_json json.RawMessage
	var headers []struct {
		Height            uint64
		Previousblockhash string
	}

	if start_hash == end_hash {
		chain = append(chain, end_hash)
		return chain, nil
	}

	var hash [32]byte = end_hash
	for {
		if hash == start_hash {
			break
		}
		chain = append(chain, hash)
		if raw_json, err = btc_rest_call(client, fmt.Sprintf("%s/headers/1/%x.json", url, hash)); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw_json, &headers); err != nil {
			return nil, fmt.Errorf("header is gibberish (%s) got (%s)", err.Error(), string(raw_json))
		}
		if len(headers) == 0 {
			return nil, fmt.Errorf("cannot find header for %X", hash)
		}
		if hash, err = parse_hex(headers[0].Previousblockhash); err != nil {
			return nil, err
		}

		var progress float64 = (float64(len(chain)) / float64(length)) * 100.0
		COMBInfo.Status = fmt.Sprintf("Tracing (%.2f%%)...", progress)
	}

	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	return chain, nil
}

func rest_get_block_range(client *http.Client, url string, start_hash [32]byte, end_hash [32]byte, length uint64, out chan<- BlockData) (err error) {
	defer close(out)
	var chain [][32]byte
	var block BlockData
	if chain, err = rest_trace_chain(client, url, start_hash, end_hash, length); err != nil {
		return err
	}

	for i, h := range chain {
		if block, err = rest_get_block(client, url, h); err != nil {
			return err
		}
		var progress float64 = (float64(i) / float64(length)) * 100.0
		COMBInfo.Status = fmt.Sprintf("Mining (%.2f%%)...", progress)
		out <- block
	}

	return nil
}

func rest_get_block(client *http.Client, url string, hash [32]byte) (block BlockData, err error) {
	var raw_data []byte
	var raw_block *BlockData = new(BlockData)

	if raw_data, err = btc_rest_call(client, fmt.Sprintf("%s/block/%x.bin", url, hash)); err != nil {
		return block, err
	}

	btc_parse_block(raw_data, raw_block)

	if raw_block.Hash != hash {
		log.Panicf("recieved wrong block %X != %X", raw_block.Hash, hash)
	}

	block.Hash = hash
	block.Commits = raw_block.Commits

	return *raw_block, nil
}

func rest_get_hash(client *http.Client, url string, height uint64) (hash [32]byte, err error) {
	var raw_json json.RawMessage
	var raw_blockhash struct {
		Blockhash string
	}

	if raw_json, err = btc_rest_call(client, fmt.Sprintf("%s/blockhashbyheight/%d.json", url, height)); err != nil {
		return hash, err
	}
	if err = json.Unmarshal(raw_json, &raw_blockhash); err != nil {
		return hash, fmt.Errorf("blockhashbyheight is gibberish (%s) got (%s)", err.Error(), string(raw_json))
	}

	raw_blockhash.Blockhash = strings.ToUpper(raw_blockhash.Blockhash)
	if err = checkHEX32(raw_blockhash.Blockhash); err != nil {
		return hash, err
	}

	hash = hex2byte32([]byte(raw_blockhash.Blockhash))
	return hash, nil
}

func rest_get_height(client *http.Client, url string, hash [32]byte) (height uint64, err error) {
	var raw_json json.RawMessage
	var raw_headers []struct {
		Height uint64
	}

	if raw_json, err = btc_rest_call(client, fmt.Sprintf("%s/headers/1/%x.json", url, hash)); err != nil {
		return height, err
	}
	if err = json.Unmarshal(raw_json, &raw_headers); err != nil {
		return height, fmt.Errorf("header data is gibberish (%s) got (%s)", err.Error(), string(raw_json))
	}

	if len(raw_headers) == 0 {
		return height, errors.New("cannot find header")
	}

	height = raw_headers[0].Height
	return height, nil
}

func rest_get_chains(client *http.Client, url string) (chain ChainData, err error) {
	var raw_json json.RawMessage
	var raw_chain struct {
		Blocks        uint64
		Headers       uint64
		BestBlockHash string
	}

	if raw_json, err = btc_rest_call(client, fmt.Sprintf("%s/chaininfo.json", url)); err != nil {
		return chain, err
	}

	if err = json.Unmarshal(raw_json, &raw_chain); err != nil {
		return chain, fmt.Errorf("chain data is gibberish (%s) got (%s)", err.Error(), string(raw_json))
	}

	if chain.TopHash, err = parse_hex(raw_chain.BestBlockHash); err != nil {
		return chain, err
	}

	chain.Height = raw_chain.Blocks
	chain.KnownHeight = raw_chain.Headers

	return chain, nil
}

func rest_try_connect(client *http.Client, url string) (err error) {
	var raw_json json.RawMessage
	var raw_chain struct {
		Blocks        uint64
		Headers       uint64
		BestBlockHash string
	}

	if raw_json, err = btc_rest_call(client, fmt.Sprintf("%s/chaininfo.json", url)); err != nil {
		return err
	} else {
		if err = json.Unmarshal(raw_json, &raw_chain); err != nil {
			return fmt.Errorf("peer is loading")
		}
		if raw_chain.Blocks == 0 {
			return fmt.Errorf("peer is loading")
		}
	}

	return nil
}

func btc_rest_call(client *http.Client, url string) (json.RawMessage, error) {
	request, _ := http.NewRequest("GET", url, nil)
	request.Header.Set("Content-Type", "text/plain")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	response_data, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil, err
	}

	return response_data, nil
}
