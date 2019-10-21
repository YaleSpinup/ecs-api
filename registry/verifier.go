package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Verifier is an image verification client
type Verifier struct {
	Client *http.Client
	Name   string
	Scheme string
	Domain string
	Host   string
	Path   string
	Tag    string
}

// NewVerifier creates an image verification struct from an image input string
func NewVerifier(input string, insecure bool) (*Verifier, error) {
	log.Infof("creating new verifier from '%s'", input)

	verifier := Verifier{}

	if insecure {
		log.Debugf("setting scheme to http")
		verifier.Scheme = "http"
	} else {
		log.Debugf("setting scheme to https")
		verifier.Scheme = "https"
	}

	// parse the reference input
	ref, err := reference.ParseAnyReference(input)
	if err != nil {
		return nil, err
	}

	// set the domain
	n, ok := ref.(reference.Named)
	if ok {
		verifier.Name = n.Name()
		verifier.Path = reference.Path(n)
		log.Debugf("checking for domain in reference: %s (%s)", n.Name(), reference.Domain(n))
		if d := reference.Domain(n); d != "" {
			log.Debugf("setting domain to %s", d)
			verifier.Domain = d
		}
	}

	// set the host to the domain unless we are using the docker hub
	if verifier.Domain == "docker.io" {
		verifier.Host = "registry-1.docker.io"
	} else {
		verifier.Host = verifier.Domain
	}

	verifier.Tag = "latest"
	nt, ok := ref.(reference.NamedTagged)
	if ok {
		log.Debugf("checking for tag in reference: %s (%s", nt.Name(), nt.Tag())
		if t := nt.Tag(); t != "" {
			log.Debugf("setting tag to %s", t)
			verifier.Tag = t
		}
	}

	verifier.Client = http.DefaultClient

	return &verifier, nil
}

// Verify executes image verification for a verifier
func (v *Verifier) Verify(ctx context.Context) (bool, error) {
	url := v.Scheme + "://" + v.Host + "/v2/" + v.Path + "/manifests/" + v.Tag
	log.Infof("verifying with URL %s", url)

	// setup HTTP request to registry URL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, errors.Wrap(err, "unable to create new request for "+url)
	}
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	// make the HTTP request to the registry
	res, err := v.Client.Do(req)
	if err != nil {
		return false, errors.Wrap(err, "unable to make http request")
	}
	defer res.Body.Close()

	code := res.StatusCode
	log.Debugf("got response status %s(%d) and headers when requesting manifest: %+v", res.Status, code, res.Header)

	// if we're not authenticated, let's try to auth
	if code == 401 {
		token, err := v.bearerTokenAuth(ctx, res.Header.Get("Www-Authenticate"))
		if err != nil {
			return false, errors.Wrap(err, "failed to get bearer token")
		}

		log.Debugf("got bearer token for authentication %s", token)
		req.Header.Set("Authorization", "Bearer "+token)

		authres, err := v.Client.Do(req)
		if err != nil {
			return false, errors.Wrap(err, "unable to make authenticated request")
		}
		defer authres.Body.Close()

		log.Debugf("got response status %s(%d) and headers when requesting authenticated manifest: %+v", authres.Status, authres.StatusCode, authres.Header)
		code = authres.StatusCode
	}

	if code > 299 && code <= 399 {
		log.Warnf("got unexpexted redirection code for %s during verification %d", v.Name, code)
	}

	if code > 499 {
		msg := fmt.Sprintf("verify failed, status: %d", code)
		return false, errors.New(msg)
	}

	if code > 299 && code <= 499 {
		return false, nil
	}

	log.Debugf("image %s seems to exist, got status %d", v.Name, code)
	return true, nil
}

// bearerTokenAuth executes the request for the bearer token regerenced in a Www-Authenticate header
func (v *Verifier) bearerTokenAuth(ctx context.Context, header string) (string, error) {
	if header == "" {
		return "", errors.New("empty authenticate header")
	}
	log.Debugf("parsing bearer token header: %s", header)

	realm := ""
	scope := ""
	service := ""

	value := strings.TrimPrefix(header, "Bearer ")
	parts := strings.Split(value, ",")
	for _, p := range parts {
		p = strings.ReplaceAll(p, "\"", "")
		pv := strings.SplitN(p, "=", 2)
		switch pv[0] {
		case "realm":
			realm = pv[1]
		case "scope":
			scope = pv[1]
		case "service":
			service = pv[1]
		default:
			log.Warnf("unexpected part of Www-Authenticate header: %s", p)
		}
	}

	url := fmt.Sprintf("%s?scope=%s&service=%s", realm, scope, service)
	log.Debugf("requesting auth token from url %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", errors.Wrap(err, "unable to create new request for "+url)
	}

	res, err := v.Client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "unable to request token")
	}
	defer res.Body.Close()

	if res.StatusCode > 299 {
		return "", errors.New("bad response when requesting token: " + res.Status)
	}

	response := struct {
		Token string `json:"token"`
	}{}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		log.Errorf("failed to decode token: %s", err)
		return "", err
	}

	return response.Token, nil
}
