package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// ParamListHandler lists the params tagged with the org
func (s *server) ParamListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	ssmService, ok := s.ssmServices[account]
	if !ok {
		msg := fmt.Sprintf("ssm service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	prefix := vars["prefix"]
	if prefix == "" {
		handleError(w, apierror.New(apierror.ErrBadRequest, "prefix is required", nil))
		return
	}

	path := fmt.Sprintf("/%s/%s", s.org, prefix)
	params, err := ssmService.ListParametersByPath(r.Context(), path)
	if err != nil {
		msg := fmt.Sprintf("unable to list params from the ssm service path %s", path)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	j, err := json.Marshal(params)
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to marshal response from the ssm service"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// ParamCreateHandler creates a parameter store parameter
func (s *server) ParamCreateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	ssmService, ok := s.ssmServices[account]
	if !ok {
		msg := fmt.Sprintf("ssm service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	prefix := vars["prefix"]
	if prefix == "" {
		handleError(w, apierror.New(apierror.ErrBadRequest, "prefix is required", nil))
		return
	}

	input := ssm.PutParameterInput{}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into put parameter input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}
	defer r.Body.Close()

	// ensure the name doesn't contain any /'s
	if strings.Contains(aws.StringValue(input.Name), "/") {
		msg := fmt.Sprintf("name (%s) cannot contain '/'", aws.StringValue(input.Name))
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, nil))
		return
	}

	// prepend the param name with org and prefix
	path := fmt.Sprintf("/%s/%s/%s", s.org, prefix, aws.StringValue(input.Name))
	input.Name = aws.String(path)

	newTags := []*ssm.Tag{
		{
			Key:   aws.String("spinup:org"),
			Value: aws.String(s.org),
		},
	}

	for _, t := range input.Tags {
		if aws.StringValue(t.Key) != "spinup:org" && aws.StringValue(t.Key) != "yale:org" {
			newTags = append(newTags, t)
		}
	}
	input.Tags = newTags

	// default to SecureString type if none is passed
	if aws.StringValue(input.Type) == "" {
		input.Type = aws.String("SecureString")
	}

	// default to Default KMS Key if none is passed
	if aws.StringValue(input.Type) == "SecureString" && aws.StringValue(input.KeyId) == "" {
		input.KeyId = aws.String(ssmService.DefaultKmsKeyId)
	}

	err = ssmService.CreateParameter(r.Context(), &input)
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to create params for the ssm service"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// ParamShowHandler gets the details of a param
func (s *server) ParamShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	ssmService, ok := s.ssmServices[account]
	if !ok {
		msg := fmt.Sprintf("ssm service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	prefix := vars["prefix"]
	if prefix == "" {
		handleError(w, apierror.New(apierror.ErrBadRequest, "prefix is required", nil))
		return
	}

	param := vars["param"]
	if param == "" {
		handleError(w, apierror.New(apierror.ErrNotFound, "param is required", nil))
		return
	}

	path := fmt.Sprintf("/%s/%s", s.org, prefix)

	meta, err := ssmService.GetParameterMetadata(r.Context(), path, param)
	if err != nil {
		msg := fmt.Sprintf("unable to get parameter metadata from the ssm service path %s/%s", path, param)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	parameter, err := ssmService.GetParameter(r.Context(), path, param)
	if err != nil {
		msg := fmt.Sprintf("unable to get parameter from the ssm service path %s/%s", path, param)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	tags, err := ssmService.ListParameterTags(r.Context(), path, param)
	if err != nil {
		msg := fmt.Sprintf("unable to get parameter tags from the ssm parameter id %s", aws.StringValue(meta.Name))
		handleError(w, errors.Wrap(err, msg))
		return
	}

	out := struct {
		ARN              *string
		Name             *string
		Description      *string
		KeyId            *string
		Type             *string
		Tags             []*ssm.Tag
		LastModifiedDate string
		Version          *int64
	}{
		parameter.ARN,
		meta.Name,
		meta.Description,
		meta.KeyId,
		meta.Type,
		tags,
		meta.LastModifiedDate.String(),
		meta.Version,
	}

	j, err := json.Marshal(out)
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to marshal response from the ssm service"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// ParamDeleteHandler deletes a parameter store parameter
func (s *server) ParamDeleteHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	ssmService, ok := s.ssmServices[account]
	if !ok {
		msg := fmt.Sprintf("ssm service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	prefix := vars["prefix"]
	if prefix == "" {
		handleError(w, apierror.New(apierror.ErrBadRequest, "prefix is required", nil))
		return
	}

	param := vars["param"]
	if param == "" {
		handleError(w, apierror.New(apierror.ErrNotFound, "param is required", nil))
		return
	}

	path := fmt.Sprintf("/%s/%s/%s", s.org, prefix, param)
	err := ssmService.DeleteParameter(r.Context(), path)
	if err != nil {
		msg := fmt.Sprintf("unable to delete parameter from the ssm service path %s", path)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// ParamDeleteAllHandler deletes all parameter store parameters in a path
func (s *server) ParamDeleteAllHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	ssmService, ok := s.ssmServices[account]
	if !ok {
		msg := fmt.Sprintf("ssm service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	prefix := vars["prefix"]
	if prefix == "" {
		handleError(w, apierror.New(apierror.ErrBadRequest, "prefix is required", nil))
		return
	}

	path := fmt.Sprintf("/%s/%s", s.org, prefix)
	params, err := ssmService.ListParametersByPath(r.Context(), path)
	if err != nil {
		msg := fmt.Sprintf("unable to list params from the ssm service prefix %s", prefix)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	for _, param := range params {
		p := fmt.Sprintf("/%s/%s/%s", s.org, prefix, param)
		err := ssmService.DeleteParameter(r.Context(), p)
		if err != nil {
			msg := fmt.Sprintf("unable to delete all parameters from the ssm service path %s, failed to delete %s", path, p)
			handleError(w, errors.Wrap(err, msg))
			return
		}
	}

	j, err := json.Marshal(
		struct {
			Message string
			Deleted int
		}{
			Message: "OK",
			Deleted: len(params),
		})
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to marshal response from the ssm service"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// ParamUpdateHandler updates a parameter store parameter
func (s *server) ParamUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	ssmService, ok := s.ssmServices[account]
	if !ok {
		msg := fmt.Sprintf("ssm service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	prefix := vars["prefix"]
	if prefix == "" {
		handleError(w, apierror.New(apierror.ErrBadRequest, "prefix is required", nil))
		return
	}

	paramName := vars["param"]
	if paramName == "" {
		handleError(w, apierror.New(apierror.ErrBadRequest, "param name is required", nil))
		return
	}

	input := &ssm.PutParameterInput{}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		msg := fmt.Sprintf("cannot decode body into put parameter input: %s", err)
		handleError(w, apierror.New(apierror.ErrBadRequest, msg, err))
		return
	}
	defer r.Body.Close()

	path := fmt.Sprintf("/%s/%s", s.org, prefix)
	parameter, err := ssmService.GetParameterMetadata(r.Context(), path, paramName)
	if err != nil {
		msg := fmt.Sprintf("unable to get parameter from the ssm service path %s/%s", path, paramName)
		handleError(w, errors.Wrap(err, msg))
		return
	}

	// if new tags are passed, update the tags
	if input.Tags != nil {
		newTags := []*ssm.Tag{}
		for _, t := range input.Tags {
			if aws.StringValue(t.Key) != "spinup:org" && aws.StringValue(t.Key) != "yale:org" {
				newTags = append(newTags, t)
			}
		}
		input.Tags = nil

		err := ssmService.UpdateParameterTags(r.Context(), aws.StringValue(parameter.Name), newTags)
		if err != nil {
			handleError(w, errors.Wrap(err, "failed to add tag to resource"))
			return
		}
	}

	// if a new value is passed, update the parameter
	if aws.StringValue(input.Value) != "" {
		input.Overwrite = aws.Bool(true)
		input.Name = aws.String(path + "/" + paramName)

		// default to SecureString if none is provided
		if aws.StringValue(input.Type) == "" {
			input.Type = aws.String("SecureString")
		}

		// default to default KMS key if none is provided and type is SecureString
		if aws.StringValue(input.Type) == "SecureString" && aws.StringValue(input.KeyId) == "" {
			input.KeyId = aws.String(ssmService.DefaultKmsKeyId)
		}

		err = ssmService.UpdateParameter(r.Context(), input)
		if err != nil {
			msg := fmt.Sprintf("unable to create params from the ssm service prefix %s", prefix)
			handleError(w, errors.Wrap(err, msg))
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
