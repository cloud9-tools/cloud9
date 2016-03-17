package main

import (
	"log"
	"net"

	"github.com/cloud9-tools/cloud9/server"
)

func main() {
	srv, err := server.New("/srv/c9")
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	defer srv.Close()
	err = srv.ListenAndServe("tcp", ":8002")
	if err != nil {
		operr, ok := err.(*net.OpError)
		if !ok || operr.Op != "accept" || operr.Err.Error() != "use of closed network connection" {
			log.Fatalf("error: %#v\n", err)
		}
	}
}
