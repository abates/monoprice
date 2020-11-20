package main

import (
	"crypto/rand"
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

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "server":
		server()
	case "keygen":
		keygen()
	default:
		log.Fatalf("Usage: %s [server|keygen]", filepath.Base(os.Args[0]))
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
	if len(apiKey) == 0 {
		log.Fatal("ampserver requires an API_KEY environment variable.")
	}

	port := getEnv("AMP_PORT", "/dev/ttyUSB0")
	speed := getIntEnv("AMP_SPEED", 9600)
	listenPort := getIntEnv("LISTEN_PORT", 8000)

	c := &serial.Config{Name: port, Baud: speed}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatalf("Failed to open serial port: %v", err)
	}

	amp := monoprice.New(s)
	zones := []string{}
	z, err := amp.Zones()
	if err != nil {
		log.Fatalf("Failed to determine zones from amplifier: %v", err)
	}

	for _, zone := range z {
		zones = append(zones, fmt.Sprintf("%d", zone.ID()))
	}
	log.Printf("Connected to amplifier, found zones %s", strings.Join(zones, ","))

	router, err := api.New(amp)
	if err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}
	log.Printf("API Server started, listening on port %d", listenPort)
	//router.Use(authMiddleware(apiKey))

	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", listenPort),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
