package server

import (
	"log"
	"net/http"
)

func Run(handler *http.ServeMux, port string) (err error) {
	log.Printf("service is running on port %s...", port)
	if err = http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("can't run server. Err: %v", err)
	}
	return
}
