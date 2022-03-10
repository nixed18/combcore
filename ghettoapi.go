package main

import (
	"encoding/json"
	"fmt"
	"libcomb"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

var gapi_db_mutex sync.Mutex
var gcontrol *Control

// Exists temporarily to serve scanner needs
func ghetto_rpc() {

	// Public
	if *public_api_bind != "" {
		publicln, err6 := net.Listen("tcp", *public_api_bind)
		if err6 != nil {
			log.Fatal(err6)
		}

		publicr := mux.NewRouter()
		s0 := publicr.PathPrefix("/public").Subrouter()

		s0.HandleFunc("/lib/get_commit_count", api_lib_get_commit_count)
		s0.HandleFunc("/lib/get_height", api_lib_get_height)
		s0.HandleFunc("/lib/get_block_commits/{block}", api_lib_get_block_commits)
		s0.HandleFunc("/lib/get_block_by_height/{height}", api_lib_get_block_by_height)
		s0.HandleFunc("/lib/get_block_by_hash/{hash}", api_lib_get_block_by_hash)
		s0.HandleFunc("/lib/get_block_coinbase_commit/{block}", api_lib_get_block_coinbase_commit)

		s0.HandleFunc("/db/get_block_metadata_by_height/{height}", api_db_get_block_metadata_by_height)
		s0.HandleFunc("/db/get_full_block_by_height/{height}", api_db_get_full_block_by_height)



		
		srv := &http.Server{
			Handler: publicr,
			WriteTimeout: 24 * time.Hour,
			ReadTimeout:  24 * time.Hour,
		}

		go func(s *http.Server) {
			err := s.Serve(publicln)
			if err!= nil {
				fmt.Println(err)
				log.Fatal()
			}

		}(srv)

	}
	
	
	// Private
	// !!! This is not secure for normal use currently !!!
	if *private_api_bind != "" {
		privateln, err6 := net.Listen("tcp", *private_api_bind)
		if err6 != nil {
			log.Fatal(err6)
		}
	
		privater := mux.NewRouter()
		s0 := privater.PathPrefix("/private").Subrouter()
		s0.HandleFunc("/db/remove_blocks_after_height/{height}", api_db_remove_blocks_after_height)
		s0.HandleFunc("/db/push_block/{block_data}", api_db_push_block) // This should be converted to "push_blocks" with a json array as the argument, but it'll do for now.
	
		srv := &http.Server{
			Handler: privater,
			WriteTimeout: 24 * time.Hour,
			ReadTimeout:  24 * time.Hour,
		}
	
		go func(s *http.Server) {
			err := s.Serve(privateln)
			if err!= nil {
				fmt.Println(err)
				log.Fatal()
			}
	
		}(srv)
	}

}


type commitTagPair struct {
	commit string
	tag libcomb.Tag
}

func api_lib_get_block_commits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	h, err := strconv.Atoi(vars["block"])
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}
	raw_commits := libcomb.GetBlockCommits(uint64(h))
	out := []commitTagPair{}

	// Non binary for now because I'm lazy, change to binary in the future.
	for key, val := range raw_commits {
		insert_at := 0
		for i, pair := range out {
			if val.Order < pair.tag.Order {
				break
			}
			// Insert before
			insert_at = i+1
		}
		if len(out) == insert_at {
			out = append(out, commitTagPair{
				commit: strings.ToUpper(fmt.Sprintf("%x", key)),
				tag: val,
			})
		} else {
			out = append(out[:insert_at+1], out[insert_at:]...)
			out[insert_at] = commitTagPair{
				commit: strings.ToUpper(fmt.Sprintf("%x", key)),
				tag: val,
			}
		}
	}

	fmt.Fprintf(w, fmt.Sprint(out))
}

func api_lib_get_height(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, fmt.Sprint(libcomb.GetHeight()))
}

func api_lib_get_commit_count(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, fmt.Sprint(libcomb.GetCommitCount()))

}

func api_lib_get_block_by_height(w http.ResponseWriter, r *http.Request) {
}

func api_lib_get_block_by_hash(w http.ResponseWriter, r *http.Request) {

}

func api_lib_get_block_coinbase_commit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	h, err := strconv.Atoi(vars["block"])
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}
	commits := libcomb.GetBlockCommits(uint64(h))
	combbase := ""
	for commit, tag := range commits {
		if tag.Order == 0 {
			combbase = strings.ToUpper(fmt.Sprintf("%x", commit))
			
		}
	}
	fmt.Fprintf(w, fmt.Sprint(combbase))
}

func api_db_get_block_metadata_by_height(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	h, err:= strconv.Atoi(vars["height"])
	if err != nil {
		fmt.Println("conv error:", err, vars["height"])
		log.Println("conv error:", err, vars["height"])
		return
	}
	gapi_db_mutex.Lock()
	raw_data := db_get_block_by_height(uint64(h))
	gapi_db_mutex.Unlock()

	
	out, err := json.Marshal(raw_data)
	if err != nil {
		fmt.Println("ERROR marshalling block data", err, raw_data)
		log.Fatal("ERROR marshalling block data", err, raw_data)
	}

	fmt.Fprint(w, string(out))
}

func api_db_get_full_block_by_height(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	h, err:= strconv.Atoi(vars["height"])
	if err != nil {
		fmt.Println("conv error:", err, vars["height"])
		log.Println("conv error:", err, vars["height"])
		return
	}
	gapi_db_mutex.Lock()
	raw_data := db_get_full_block_by_height(uint64(h))
	gapi_db_mutex.Unlock()
	
	out, err := json.Marshal(raw_data)
	if err != nil {
		fmt.Println("ERROR marshalling block data", err, raw_data)
		log.Fatal("ERROR marshalling block data", err, raw_data)
	}

	fmt.Fprint(w, string(out))
}


// --- Private ---
func api_db_remove_blocks_after_height(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if *node_mode != MID_NODE_REMOTE {
		return
	}
	h, err:= strconv.Atoi(vars["height"])
	if err != nil {
		fmt.Println("conv error:", err, vars["height"])
		log.Println("conv error:", err, vars["height"])
		return
	}
	gapi_db_mutex.Lock()
	defer gapi_db_mutex.Unlock()
	db_remove_blocks_after(uint64(h))
}

func api_db_push_block(w http.ResponseWriter, r *http.Request) {
	// Ingests a processed comb block.

	// Only function if this node is a mid-level remote node; it has a local DB but no BTC node to pull from, and receives blocks without putting out a request to peers
	// This conditional needs to be made way better, for now it does the job though
	if *node_mode != MID_NODE_REMOTE {
		return
	}

	vars := mux.Vars(r)

	// Unmarshal block data json into block data
	inc_block := BlockData{}
	err := json.Unmarshal([]byte(vars["block_data"]), &inc_block)

	if err != nil {
		fmt.Fprint(w, "ERROR: problem unmarshalling block_json", err)
		fmt.Println("ERROR: problem unmarshalling block_json", err)
		log.Println("ERROR: problem unmarshalling block_json", err, inc_block)
		return
	}

	// Dunno if this'll work by itself lol, we'll see I guess. Feels sketchy.
	gapi_db_mutex.Lock()
	defer gapi_db_mutex.Unlock()
	neominer_process_block(inc_block)
	neominer_write()
}

