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
	"time"

	"github.com/gorilla/mux"
)

// Exists temporarily to serve scanner needs
func ghetto_rpc() {

	// Public
	if *public_api_bind != "" {
		publicln, err6 := net.Listen("tcp", *public_api_bind)
		if err6 != nil {
			log.Fatal(err6)
		}

		publicr := mux.NewRouter()
		s0 := publicr.PathPrefix("/lib/").Subrouter()
		s0.HandleFunc("/get_commit_count", api_lib_get_commit_count)
		s0.HandleFunc("/get_height", api_lib_get_height)
		s0.HandleFunc("/get_block_commits/{block}", api_lib_get_block_commits)
		s0.HandleFunc("/get_block_by_height/{height}", api_lib_get_block_by_height)
		s0.HandleFunc("/get_block_by_hash/{hash}", api_lib_get_block_by_hash)
		s0.HandleFunc("/get_block_coinbase_commit/{block}", api_lib_get_block_coinbase_commit)


		
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
	if *private_api_bind != "" {
		privateln, err6 := net.Listen("tcp", "127.0.0.1:4343")
		if err6 != nil {
			log.Fatal(err6)
		}
	
		privater := mux.NewRouter()
		s1 := privater.PathPrefix("/private/").Subrouter()
		s1.HandleFunc("/{ciphertext}", api_handle_private_command)
	
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



// --- Private ---
type PrivateComms struct {
	Command string `json:"cmd"`
	Args map[string]string `json:"args"`
}

func api_handle_private_command(w http.ResponseWriter, r *http.Request) {
	// Check for conditions
	if *comms_key == "" {
		fmt.Fprintf(w, "ERROR missing comms_key")
		return
	}

	vars := mux.Vars(r)

	// Decrypt ciphertext
	decryptext := aes_decrypt(vars["ciphertext"], *comms_key)

	// Check out validity of decryption
	if !good_cryption(decryptext) {
		fmt.Fprint(w, "ERROR cryption error: ", decryptext)
	}

	// Unmarshal
	comm := PrivateComms{}
	err := json.Unmarshal([]byte(decryptext), &comm)
	if err != nil {
		fmt.Fprint(w, "ERROR invalid comms json: ", decryptext)
		log.Println(err)
		return
	}

	// Engage
	switch comm.Command {
	case "push_comb_block":
		// Ingests a processed comb block.

		// Only function if this node is a mid-level remote node; it has a local DB but no BTC node to pull from, and receives blocks without putting out a request to peers
		// This conditional needs to be made way better, for now it does the job though
		if *node_mode != MID_NODE_REMOTE {
			return
		}

		// Unmarshal block data json into block data
		inc_block := BlockData{}
		err := json.Unmarshal([]byte(comm.Args["block_data"]), &inc_block)

		if err != nil {
			fmt.Fprint(w, "ERROR: problem unmarshalling block_json")
		}

		// Dunno if this'll work by itself lol, we'll see I guess. Feels sketchy.
		neominer_process_block(inc_block)
	}

}