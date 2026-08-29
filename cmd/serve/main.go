// Command serve is a tiny static file server for trying the wasm build locally.
//
//	make wasm
//	go run ./cmd/serve            # http://localhost:8000/
//	go run ./cmd/serve -addr :9000 -dir ./static
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := flag.String("addr", ":8000", "listen address")
	dir := flag.String("dir", "./static", "directory to serve")
	flag.Parse()

	if _, err := os.Stat(*dir); err != nil {
		log.Fatal(err)
	}

	host := *addr
	if host[0] == ':' {
		host = "localhost" + host
	}
	fmt.Printf("serving %s at http://%s/\n", *dir, host)
	log.Fatal(http.ListenAndServe(*addr, http.FileServer(http.Dir(*dir))))
}
