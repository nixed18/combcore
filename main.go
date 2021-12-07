package main

import (
	"log"
)

func main() {
	var err error
	combcore_init()

	if err = db_open(); err != nil {
		log.Fatal(err)
	}

	db_start()
	rpc_start()
	autoclaim_start()
	//neominer_try_connect()
	directminer_init()

	if err = combcore_check(); err != nil {
		combcore_dump()
		log.Fatalln(err.Error())
	}

	neominer_start()
}
