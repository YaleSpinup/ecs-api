package api

import (
	"net/http"

	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/YaleSpinup/ecs-api/registry"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// ImageVerificationHandler checks if an image exists
func (s *server) ImageVerificationHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	image := vars["image"]

	log.Debugf("verifying image '%s' from query", image)

	verifier, err := registry.NewVerifier(image, false)
	if err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "unable to create new image verifier", err))
		return
	}

	exists, err := verifier.Verify(r.Context())
	if err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "unable to verify image", err))
		return
	}

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
