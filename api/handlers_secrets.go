package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/YaleSpinup/apierror"

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

	tagsFilter := map[string]string{"spinup:org": s.org}
	q := r.URL.Query()
	if len(q) > 0 {
		log.Debugf("parsing query parameters %+v", q)
		for k, v := range q {
			log.Debugf("key: %s, value: %+v", k, v)
			// append tag filters, silently ignore attempts to override the org
			if k != "spinup:org" {
				tagsFilter[k] = v[0]
			}
		}
	}

	secrets, err := smService.ListSecretsWithFilter(r.Context(), func(sec *secretsmanager.SecretListEntry) bool {
		for k, v := range tagsFilter {
			log.Debugf("checking %s tags for %s = %s", aws.StringValue(sec.Name), k, v)

			found := false
			for _, tag := range sec.Tags {
				if aws.StringValue(tag.Key) == k && aws.StringValue(tag.Value) == v {
					found = true
					break
				}
			}

			if !found {
				log.Debugf("didn't find tag (%s = %s) for %s", k, v, aws.StringValue(sec.Name))
				return false
			}
		}

		log.Debugf("%s matched all tags", aws.StringValue(sec.Name))
		return true
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
	input.Tags = append(input.Tags, &secretsmanager.Tag{Key: aws.String("spinup:org"), Value: aws.String(s.org)})

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
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	smService, ok := s.smServices[account]
	if !ok {
		msg := fmt.Sprintf("secretsmanager service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}
	id := vars["secret"]
	secret, err := smService.GetSecretMetaDataWithFilter(r.Context(), id, func(out *secretsmanager.DescribeSecretOutput) bool {
		log.Debugf("checking tags for %s to be sure it's part of the org %s", aws.StringValue(out.Name), s.org)
		for _, tag := range out.Tags {
			if aws.StringValue(tag.Key) == "spinup:org" && aws.StringValue(tag.Value) == s.org {
				log.Debugf("%s has matching org tag and is part of the %s org, adding to the list", aws.StringValue(out.Name), s.org)
				return true
			}
		}
		return false
	})
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to get secret from the secretsmanager service"))
		return
	}

	j, err := json.Marshal(secret)
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to marshal response from the secretsmanager service"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

func (s *server) SecretDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	smService, ok := s.smServices[account]
	if !ok {
		msg := fmt.Sprintf("secretsmanager service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	id := vars["secret"]
	window := int64(7)
	q := r.URL.Query()
	if wq := q.Get("window"); wq != "" {
		i, err := strconv.ParseInt(wq, 10, 64)
		if err != nil {
			msg := fmt.Sprintf("failed to parse window size as integer: %s", wq)
			handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
			return
		}
		window = i
	}

	// first check the secret matches our filters (ie. it's part of the org)
	_, err := smService.GetSecretMetaDataWithFilter(r.Context(), id, func(out *secretsmanager.DescribeSecretOutput) bool {
		log.Debugf("checking tags for %s to be sure it's part of the org %s", aws.StringValue(out.Name), s.org)
		for _, tag := range out.Tags {
			if aws.StringValue(tag.Key) == "spinup:org" && aws.StringValue(tag.Value) == s.org {
				log.Debugf("%s has matching org tag and is part of the %s org", aws.StringValue(out.Name), s.org)
				return true
			}
		}
		return false
	})
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to get secret from the secretsmanager service"))
		return
	}

	// then delete it
	out, err := smService.DeleteSecret(r.Context(), id, window)
	if err != nil {
		handleError(w, err)
	}

	j, err := json.Marshal(out)
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to marshal response from the secretsmanager service"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

func (s *server) SecretUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	smService, ok := s.smServices[account]
	if !ok {
		msg := fmt.Sprintf("secretsmanager service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}
	id := vars["secret"]

	var input = struct {
		Secret string
		Tags   []*secretsmanager.Tag
	}{}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into create update secret input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}

	if len(input.Tags) > 0 {
		for _, t := range input.Tags {
			if aws.StringValue(t.Key) == "spinup:org" {
				handleError(w, apierror.New(apierror.ErrBadRequest, "illegal update of org tag", err))
				return
			}
		}

		if err := smService.UpdateSecretTags(r.Context(), id, input.Tags); err != nil {
			handleError(w, apierror.New(apierror.ErrBadRequest, "failed to update tags for secret", err))
			return
		}
	}

	if input.Secret == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte{})
		return
	}

	out, err := smService.UpdateSecret(r.Context(), &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(id),
		SecretString: aws.String(input.Secret),
		VersionStages: []*string{
			aws.String("AWSCURRENT"),
		},
	})
	if err != nil {
		handleError(w, apierror.New(apierror.ErrBadRequest, "failed to update secret value", err))
		return
	}

	j, err := json.Marshal(out)
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to marshal response from the secretsmanager service"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}
