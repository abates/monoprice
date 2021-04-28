package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/abates/monoprice"
	"github.com/abates/monoprice/api"
	"github.com/gorilla/mux"
	"github.com/tarm/serial"
)

var verbose bool
var disableAuth bool
var apiKey string

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func getIntEnv(key string, fallback int) int {
	value := getEnv(key, fmt.Sprintf("%d", fallback))
	v, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Failed to parse %s env variable %s, falling back to %d", key, value, fallback)
		v = fallback
	}
	return v
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <flags> [server|keygen]\n", filepath.Base(os.Args[0]))
	flag.PrintDefaults()
}

func main() {
	flag.BoolVar(&verbose, "verbose", false, "verbose logging")
	flag.BoolVar(&disableAuth, "noauth", false, "disable authentication middleware (useful for testing)")
	flag.Usage = usage
	flag.Parse()

	cmd := ""
	if len(flag.Args()) > 0 {
		cmd = flag.Args()[0]
	}

	switch cmd {
	case "server":
		server()
	case "keygen":
		keygen()
	default:
		usage()
		os.Exit(-1)
	}
}

func keygen() {
	b := make([]byte, 64)
	rand.Read(b)
	fmt.Printf("%x\n", b)
}

func authMiddleware(apiKey string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqToken := r.Header.Get("X-Auth-Key")
			if reqToken == apiKey {
				next.ServeHTTP(w, r)
			} else {
				http.Error(w, "Not Authorized", http.StatusUnauthorized)
			}
		})
	}
}

func server() {
	apiKey = getEnv("API_KEY", "")
	if len(apiKey) == 0 && !disableAuth {
		log.Fatal("ampserver requires an API_KEY environment variable.")
	}

	port := getEnv("AMP_PORT", "/dev/ttyUSB0")
	speed := getIntEnv("AMP_SPEED", 9600)
	listenPort := getIntEnv("LISTEN_PORT", 8000)

	c := &serial.Config{Name: port, Baud: speed, ReadTimeout: time.Second}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatalf("Failed to open serial port: %v", err)
	}

	options := []monoprice.Option{}
	if verbose {
		options = append(options, monoprice.VerboseOption())
	}

	amp, err := monoprice.New(s, options...)
	if err != nil {
		log.Fatalf("Failed to initialize amplifier: %v", err)
	}

	zones := []string{}
	z := amp.Zones()

	for _, zone := range z {
		zones = append(zones, fmt.Sprintf("%d", zone.ID()))
	}
	log.Printf("Connected to amplifier, found zones %s", strings.Join(zones, ","))

	router := api.New(amp)
	log.Printf("API Server started, listening on port %d", listenPort)
	if !disableAuth {
		router.Use(authMiddleware(apiKey))
	}

	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", listenPort),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
