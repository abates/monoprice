package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/abates/monoprice"
	"github.com/gorilla/mux"
)

type api struct {
	amp   *monoprice.Amplifier
	zones sync.Map
}

func New(amp *monoprice.Amplifier) (http.Handler, error) {
	a := &api{amp: amp}

	zones, err := a.amp.Zones()
	if err != nil {
		return nil, err
	}

	for _, zone := range zones {
		a.zones.Store(zone.ID(), zone)
	}

	r := mux.NewRouter()
	r.HandleFunc("/zones", http.HandlerFunc(a.listZones)).Methods("GET")
	r.HandleFunc("/{zone}/status", a.zoneHandler(a.status)).Methods("GET")
	r.HandleFunc("/{zone}/power/{power}", a.sendCommand(monoprice.SetPower, "power", monoprice.ParseBool)).Methods("PUT")
	r.HandleFunc("/{zone}/mute/{mute}", a.sendCommand(monoprice.SetMute, "power", monoprice.ParseBool)).Methods("PUT")
	r.HandleFunc("/{zone}/volume/{level}", a.sendCommand(monoprice.SetVolume, "level", monoprice.ParseInt)).Methods("PUT")
	r.HandleFunc("/{zone}/treble/{level}", a.sendCommand(monoprice.SetTreble, "level", monoprice.ParseInt)).Methods("PUT")
	r.HandleFunc("/{zone}/bass/{level}", a.sendCommand(monoprice.SetBass, "level", monoprice.ParseInt)).Methods("PUT")
	r.HandleFunc("/{zone}/balance/{level}", a.sendCommand(monoprice.SetBalance, "level", monoprice.ParseInt)).Methods("PUT")
	r.HandleFunc("/{zone}/source/{source}", a.sendCommand(monoprice.SetSource, "source", monoprice.ParseInt)).Methods("PUT")
	r.HandleFunc("/{zone}/restore", a.zoneHandler(a.restore)).Methods("PUT")

	return r, nil
}

func (a *api) listZones(w http.ResponseWriter, r *http.Request) {
	ids := []int{}
	a.zones.Range(func(key, value interface{}) bool {
		ids = append(ids, value.(int))
		return true
	})
	json.NewEncoder(w).Encode(ids)
}

func (a *api) zoneHandler(handler func(monoprice.Zone, http.ResponseWriter, *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["zone"])
		if err == nil {
			if zone, found := a.zones.Load(id); found {
				handler(zone.(monoprice.Zone), w, r)
			} else {
				http.Error(w, "Zone not found", http.StatusNotFound)
			}
		} else {
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
				w.WriteHeader(http.StatusOK)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})
}

func (a *api) status(zone monoprice.Zone, w http.ResponseWriter, r *http.Request) {
	state, err := zone.State()
	if err == nil {
		json.NewEncoder(w).Encode(state)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *api) restore(zone monoprice.Zone, w http.ResponseWriter, r *http.Request) {
}
