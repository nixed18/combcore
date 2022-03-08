package main

import (
	"fmt"
	"libcomb"
	"log"
	"net"
	"net/http"
	"strconv"
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
	s0.HandleFunc("get_block_commits/{block}", get_block_commits)
	s0.HandleFunc("get_block_by_height/{height}", get_block_by_height)
	s0.HandleFunc("get_block_by_hash/{hash}", get_block_by_hash)
	s0.HandleFunc("/get_coinbase", get_coinbase)

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

func get_block_commits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	h, err := strconv.Atoi(vars["block"])
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}
	fmt.Fprintf(w, fmt.Sprint(libcomb.GetBlockCommits(uint64(h))))
}

func get_commit_count(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, fmt.Sprint(libcomb.GetCommitCount()))

}

func get_block_by_height(w http.ResponseWriter, r *http.Request) {

}

func get_block_by_hash(w http.ResponseWriter, r *http.Request) {

}

func get_coinbase(w http.ResponseWriter, r *http.Request) {
}