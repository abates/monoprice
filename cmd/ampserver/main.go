package main

import (
	"log"
	"net/http"
	"time"

	"github.com/abates/monoprice"
	"github.com/abates/monoprice/api"
	"github.com/tarm/serial"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

func main() {
	c := &serial.Config{Name: "/dev/ttyUSB0", Baud: 9600}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}

	amp := monoprice.New(s)
	log.Printf("Amp is setup, creating router")
	router, err := api.New(amp)
	if err != nil {
		log.Fatal(err)
	}

	router.Use(loggingMiddleware)

	log.Printf("Router created, error: %v", err)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Handler: router,
		Addr:    ":8000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
