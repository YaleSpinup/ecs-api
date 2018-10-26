package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// PingHandler responds to ping requests
func PingHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	log.Debug("Ping/Pong")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

// VersionHandler responds to version requests
func VersionHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte{})
		return
	}
	w.WriteHeader(http.StatusOK)

	data, err := json.Marshal(struct {
		Version    string `json:"version"`
		GitHash    string `json:"githash"`
		BuildStamp string `json:"buildstamp"`
	}{
		Version:    fmt.Sprintf("%s%s", Version, VersionPrerelease),
		GitHash:    githash,
		BuildStamp: buildstamp,
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
