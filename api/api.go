package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"

	"github.com/abates/monoprice"
	"github.com/gorilla/mux"
)

func ParseBool(str string) (interface{}, error) {
	b, err := strconv.ParseBool(str)
	if err == nil {
		if b {
			return "01", nil
		}
		return "00", nil
	}
	return "", err
}

func ParseInt(str string) (interface{}, error) {
	return strconv.Atoi(str)
}

type api struct {
	amp   *monoprice.Amplifier
	zones sync.Map
}

func New(amp *monoprice.Amplifier) *mux.Router {
	a := &api{amp: amp}

	for _, zone := range a.amp.Zones() {
		a.zones.Store(zone.ID(), zone)
	}

	r := mux.NewRouter()
	r.HandleFunc("/zones", http.HandlerFunc(a.listZones)).Methods("GET")
	r.HandleFunc("/{zone}/status", a.zoneHandler(a.status)).Methods("GET")
	r.HandleFunc("/{zone}/power/{power}", a.sendCommand(monoprice.SetPower, "power", ParseBool)).Methods("PUT")
	r.HandleFunc("/{zone}/mute/{mute}", a.sendCommand(monoprice.SetMute, "power", ParseBool)).Methods("PUT")
	r.HandleFunc("/{zone}/volume/{level}", a.sendCommand(monoprice.SetVolume, "level", ParseInt)).Methods("PUT")
	r.HandleFunc("/{zone}/treble/{level}", a.sendCommand(monoprice.SetTreble, "level", ParseInt)).Methods("PUT")
	r.HandleFunc("/{zone}/bass/{level}", a.sendCommand(monoprice.SetBass, "level", ParseInt)).Methods("PUT")
	r.HandleFunc("/{zone}/balance/{level}", a.sendCommand(monoprice.SetBalance, "level", ParseInt)).Methods("PUT")
	r.HandleFunc("/{zone}/source/{source}", a.sendCommand(monoprice.SetSource, "source", ParseInt)).Methods("PUT")
	r.HandleFunc("/{zone}/restore", a.zoneHandler(a.restore)).Methods("PUT")

	return r
}

func (a *api) listZones(w http.ResponseWriter, r *http.Request) {
	ids := []int{}
	a.zones.Range(func(key, value interface{}) bool {
		ids = append(ids, int(key.(monoprice.ZoneID)))
		return true
	})
	sort.Ints(ids)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ids)
}

func (a *api) zoneHandler(handler func(monoprice.Zone, http.ResponseWriter, *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["zone"])
		if err == nil {
			if zone, found := a.zones.Load(monoprice.ZoneID(id)); found {
				handler(zone.(monoprice.Zone), w, r)
			} else {
				log.Printf("Zone %d not found", id)
				http.Error(w, "Zone not found", http.StatusNotFound)
			}
		} else {
			log.Printf("Failed to convert zone %q to integer: %v", vars["zone"], err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
}

func (a *api) sendCommand(cmd monoprice.Command, v string, decoder func(string) (interface{}, error)) func(w http.ResponseWriter, r *http.Request) {
	return a.zoneHandler(func(zone monoprice.Zone, w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		arg, err := decoder(vars[v])
		if err == nil {
			err = zone.SendCommand(cmd, arg)

			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
			} else {
				log.Printf("Failed sending command to amp: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			log.Printf("Failed decoding command variable %q: %v", vars[v], err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})
}

func (a *api) status(zone monoprice.Zone, w http.ResponseWriter, r *http.Request) {
	state, err := zone.State()
	if err == nil || errors.Is(monoprice.ErrUnknownState, err) {
		w.Header().Set("Content-Type", "application/json")
		status := http.StatusOK
		if err != nil {
			status = http.StatusServiceUnavailable
		}
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(state)
	} else {
		log.Printf("Failed to determine zone status: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *api) restore(zone monoprice.Zone, w http.ResponseWriter, r *http.Request) {
}
