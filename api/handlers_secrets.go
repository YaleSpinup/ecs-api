package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/ecs-api/apierror"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// SecretListHandler lists the secrets tagged with the org
func (s *server) SecretListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	smService, ok := s.smServices[account]
	if !ok {
		msg := fmt.Sprintf("secretsmanager service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	secrets, err := smService.ListSecretsWithFilter(r.Context(), func(sec *secretsmanager.SecretListEntry) bool {
		log.Debugf("checking tags for %s to be sure it's part of the org %s", aws.StringValue(sec.Name), s.org)
		for _, tag := range sec.Tags {
			if aws.StringValue(tag.Key) == "yale:org" && aws.StringValue(tag.Value) == s.org {
				log.Debugf("%s has matching org tag and is part of the %s org, adding to the list", aws.StringValue(sec.Name), s.org)
				return true
			}
		}
		return false
	})
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to list secrets from the secretsmanager service"))
		return
	}

	j, err := json.Marshal(secrets)
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to marshal response from the secretsmanager service"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// SecretCreateHandler decodes a body into CreateSecretInput and creates a secret
func (s *server) SecretCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	smService, ok := s.smServices[account]
	if !ok {
		msg := fmt.Sprintf("secretsmanager service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	input := &secretsmanager.CreateSecretInput{}
	err := json.NewDecoder(r.Body).Decode(input)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into create secret create input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}
	input.Tags = append(input.Tags, &secretsmanager.Tag{Key: aws.String("yale:org"), Value: aws.String(s.org)})

	out, err := smService.CreateSecret(r.Context(), input)
	if err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "failed to create secret", err))
		return
	}

	j, err := json.Marshal(out)
	if err != nil {
		handleError(w, apierror.New(apierror.ErrInternalError, "cannot marshal json response", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

func (s *server) SecretShowHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}

func (s *server) SecretDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}

func (s *server) SecretUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}
