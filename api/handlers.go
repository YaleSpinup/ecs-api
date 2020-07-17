package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/apierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// PingHandler responds to ping requests
func (s *server) PingHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	log.Debug("Ping/Pong")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

// VersionHandler responds to version requests
func (s *server) VersionHandler(w http.ResponseWriter, r *http.Request) {
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
		Version:    fmt.Sprintf("%s%s", s.version.Version, s.version.VersionPrerelease),
		GitHash:    s.version.GitHash,
		BuildStamp: s.version.BuildStamp,
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleError handles standard apierror return codes
func handleError(w http.ResponseWriter, err error) {
	log.Error(err.Error())
	if aerr, ok := errors.Cause(err).(apierror.Error); ok {
		switch aerr.Code {
		case apierror.ErrForbidden:
			w.WriteHeader(http.StatusForbidden)
		case apierror.ErrNotFound:
			w.WriteHeader(http.StatusNotFound)
		case apierror.ErrConflict:
			w.WriteHeader(http.StatusConflict)
		case apierror.ErrBadRequest:
			w.WriteHeader(http.StatusBadRequest)
		case apierror.ErrLimitExceeded:
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(aerr.Message))
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}
