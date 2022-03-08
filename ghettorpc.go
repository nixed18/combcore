package main

import (
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
	ln, err6 := net.Listen("tcp", "10.0.0.75:3232")
	if err6 != nil {
		log.Fatal(err6)
	}

	r := mux.NewRouter()
	s0 := r.PathPrefix("/").Subrouter()
	s0.HandleFunc("/push_block/{blockjson}", push_block)
	s0.HandleFunc("/get_commit_count", get_commit_count)
	s0.HandleFunc("/get_block_commits/{block}", get_block_commits)
	s0.HandleFunc("/get_block_by_height/{height}", get_block_by_height)
	s0.HandleFunc("/get_block_by_hash/{hash}", get_block_by_hash)
	s0.HandleFunc("/get_block_coinbase_commit/{block}", get_block_coinbase_commit)


	srv := &http.Server{
		Handler: r,
		WriteTimeout: 24 * time.Hour,
		ReadTimeout:  24 * time.Hour,
	}

	err := srv.Serve(ln)
	if err!= nil {
		fmt.Println(err)
		log.Fatal()
	}

}

func push_block(w http.ResponseWriter, r *http.Request) {

}

func get_block_combspawn(w http.ResponseWriter, r *http.Request) {

}

type commitTagPair struct {
	commit string
	tag libcomb.Tag
}

func get_block_commits(w http.ResponseWriter, r *http.Request) {
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


func get_commit_count(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, fmt.Sprint(libcomb.GetCommitCount()))

}

func get_block_by_height(w http.ResponseWriter, r *http.Request) {

}

func get_block_by_hash(w http.ResponseWriter, r *http.Request) {

}

func get_block_coinbase_commit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	h, err := strconv.Atoi(vars["block"])
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}
	commits := libcomb.GetBlockCommits(uint64(h))
	combbase := ""
	for commit, tag := range commits {
		// For now it looks like libcomb is only storing the first seen, but the DB is storing them all. Alright, cool
		if tag.Order == 0 {
			combbase = strings.ToUpper(fmt.Sprintf("%x", commit))
			
		}
	}
	fmt.Fprintf(w, fmt.Sprint(combbase))
}