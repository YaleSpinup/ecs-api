package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/pkg/errors"
)

var verifierTestCases = []struct {
	input      string
	err        error
	verifier   *Verifier
	exists     bool
	statusCode int
}{
	{
		input: "nginx",
		verifier: &Verifier{
			Domain: "docker.io",
			Name:   "docker.io/library/nginx",
			Host:   "registry-1.docker.io",
			Tag:    "latest",
			Scheme: "https",
			Path:   "library/nginx",
			Client: http.DefaultClient,
		},
		exists:     true,
		statusCode: http.StatusOK,
	},
	{
		input: "nginx:latest",
		verifier: &Verifier{
			Domain: "docker.io",
			Name:   "docker.io/library/nginx",
			Host:   "registry-1.docker.io",
			Tag:    "latest",
			Scheme: "https",
			Path:   "library/nginx",
			Client: http.DefaultClient,
		},
		exists:     true,
		statusCode: http.StatusOK,
	},
	{
		input: "library/nginx",
		verifier: &Verifier{
			Domain: "docker.io",
			Name:   "docker.io/library/nginx",
			Host:   "registry-1.docker.io",
			Tag:    "latest",
			Scheme: "https",
			Path:   "library/nginx",
			Client: http.DefaultClient,
		},
		exists:     true,
		statusCode: http.StatusOK,
	},
	{
		input: "library/nginx:latest",
		verifier: &Verifier{
			Domain: "docker.io",
			Name:   "docker.io/library/nginx",
			Host:   "registry-1.docker.io",
			Tag:    "latest",
			Scheme: "https",
			Path:   "library/nginx",
			Client: http.DefaultClient,
		},
		exists:     true,
		statusCode: http.StatusOK,
	},
	{
		input: "registry-1.docker.io/library/nginx",
		verifier: &Verifier{
			Domain: "registry-1.docker.io",
			Name:   "registry-1.docker.io/library/nginx",
			Host:   "registry-1.docker.io",
			Tag:    "latest",
			Scheme: "https",
			Path:   "library/nginx",
			Client: http.DefaultClient,
		},
		exists:     true,
		statusCode: http.StatusOK,
	},
	{
		input: "quay.io/coreos/etcd",
		verifier: &Verifier{
			Domain: "quay.io",
			Name:   "quay.io/coreos/etcd",
			Host:   "quay.io",
			Tag:    "latest",
			Scheme: "https",
			Path:   "coreos/etcd",
			Client: http.DefaultClient,
		},
		exists:     true,
		statusCode: http.StatusOK,
	},
	{
		input: "quay.io/coreos/etcd:latest",
		verifier: &Verifier{
			Domain: "quay.io",
			Name:   "quay.io/coreos/etcd",
			Host:   "quay.io",
			Tag:    "latest",
			Scheme: "https",
			Path:   "coreos/etcd",
			Client: http.DefaultClient,
		},
		exists:     true,
		statusCode: http.StatusOK,
	},
	{
		input: "quay.io/coreos/etcd:v3.3.17",
		verifier: &Verifier{
			Domain: "quay.io",
			Name:   "quay.io/coreos/etcd",
			Host:   "quay.io",
			Tag:    "v3.3.17",
			Scheme: "https",
			Path:   "coreos/etcd",
			Client: http.DefaultClient,
		},
		exists:     true,
		statusCode: http.StatusOK,
	},
	{
		input: "quay.io/mojanalytics/rstudio",
		verifier: &Verifier{
			Domain: "quay.io",
			Name:   "quay.io/mojanalytics/rstudio",
			Host:   "quay.io",
			Tag:    "latest",
			Scheme: "https",
			Path:   "mojanalytics/rstudio",
			Client: http.DefaultClient,
		},
		exists:     true,
		statusCode: http.StatusOK,
	},
	{
		input: "quay.io/mojanalytics/rstudio:latest",
		verifier: &Verifier{
			Domain: "quay.io",
			Name:   "quay.io/mojanalytics/rstudio",
			Host:   "quay.io",
			Tag:    "latest",
			Scheme: "https",
			Path:   "mojanalytics/rstudio",
			Client: http.DefaultClient,
		},
		exists:     true,
		statusCode: http.StatusOK,
	},
	{
		input: "quay.io/mojanalytics/rstudio:1.2.1335-r3.5.1-python3.7.1-conda-3",
		verifier: &Verifier{
			Domain: "quay.io",
			Name:   "quay.io/mojanalytics/rstudio",
			Host:   "quay.io",
			Tag:    "1.2.1335-r3.5.1-python3.7.1-conda-3",
			Scheme: "https",
			Path:   "mojanalytics/rstudio",
			Client: http.DefaultClient,
		},
		exists:     true,
		statusCode: http.StatusOK,
	},
	{
		input: "cloudmatin.io/sochill/datalibrary",
		verifier: &Verifier{
			Domain: "cloudmatin.io",
			Name:   "cloudmatin.io/sochill/datalibrary",
			Host:   "cloudmatin.io",
			Tag:    "latest",
			Scheme: "https",
			Path:   "sochill/datalibrary",
			Client: http.DefaultClient,
		},
		exists:     false,
		statusCode: http.StatusTeapot,
	},
	{
		input: "cloudmatin.io/sochill/datalibrary:latest",
		verifier: &Verifier{
			Domain: "cloudmatin.io",
			Name:   "cloudmatin.io/sochill/datalibrary",
			Host:   "cloudmatin.io",
			Tag:    "latest",
			Scheme: "https",
			Path:   "sochill/datalibrary",
			Client: http.DefaultClient,
		},
		exists:     false,
		statusCode: http.StatusBadRequest,
	},
	{
		input: "cloudmatin.io/sochill/datalibrary:12.13.14",
		verifier: &Verifier{
			Domain: "cloudmatin.io",
			Name:   "cloudmatin.io/sochill/datalibrary",
			Host:   "cloudmatin.io",
			Tag:    "12.13.14",
			Scheme: "https",
			Path:   "sochill/datalibrary",
			Client: http.DefaultClient,
		},
		exists:     false,
		statusCode: http.StatusNotFound,
	},
}

func TestNewVerifier(t *testing.T) {
	for _, v := range verifierTestCases {
		t.Logf("testing with testverifier %+v", v)

		verifier, err := NewVerifier(v.input, false)
		t.Logf("got %+v, %s", v, reflect.TypeOf(verifier).String())

		if err != v.err {
			t.Errorf("expected error to be %s, got %s", v.err, err)
		}

		if !reflect.DeepEqual(verifier, v.verifier) {
			t.Errorf("expected %+v, got %v", v.verifier, verifier)
		}
	}
}

func TestVerify(t *testing.T) {
	for _, v := range verifierTestCases {
		t.Logf("testing with testverifier %+v", v)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(v.statusCode)
		}))
		defer ts.Close()

		v.verifier.Host = ts.Listener.Addr().String()
		v.verifier.Scheme = "http"

		exists, err := v.verifier.Verify(context.TODO())
		if err != v.err {
			t.Errorf("expected error to be %s, got %s", v.err, err)
		}
		t.Logf("got exists '%t' for %+v", exists, v.verifier)

		if exists != v.exists {
			t.Errorf("expected %+v exists to be %t, got %v", v.verifier, v.exists, exists)
		}
	}
}

func TestBearerTokenAuth(t *testing.T) {
	testHeaders := []struct {
		scope      string
		service    string
		token      string
		statusCode int
		err        error
	}{
		{
			scope:      "library/nginx",
			service:    "registry-1.docker.io",
			token:      "abc123",
			statusCode: http.StatusOK,
		},
		{
			scope:      "sochill/datalibrary",
			service:    "cloudmatin.io",
			token:      "$031337",
			statusCode: http.StatusOK,
		},
		{
			scope:      "derpy/derp",
			service:    "brokenservice.ly",
			token:      "zzzzzzzzz",
			statusCode: http.StatusBadRequest,
			err:        errors.New("something"),
		},
		{
			scope:      "library/nginx",
			service:    "registry-1.docker.io",
			token:      "abc123",
			statusCode: http.StatusUnauthorized,
			err:        errors.New("something"),
		},
	}

	testVerifier := Verifier{}
	for _, h := range testHeaders {
		t.Logf("testing with testheader %+v", h)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(h.statusCode)
			j, _ := json.Marshal(struct {
				Token string `json:"token"`
			}{
				Token: h.token,
			})
			w.Write(j)
		}))
		defer ts.Close()

		header := fmt.Sprintf("Bearer \"realm=%s\",\"scope=%s\",\"service=%s\",\"foo=bar\"", ts.URL, h.scope, h.service)
		testVerifier.Client = ts.Client()
		token, err := testVerifier.bearerTokenAuth(context.TODO(), header)
		if h.err == nil && err != nil {
			t.Errorf("expected nil error, got %s", err)
		} else if h.err != nil && err == nil {
			t.Errorf("expected an error, but got nil")
		}

		if err == nil && token != h.token {
			t.Errorf("expected token %s, got %s", h.token, token)
		}
	}

	// test missing header
	_, err := testVerifier.bearerTokenAuth(context.TODO(), "")
	if err == nil {
		t.Error("expected error, got nil")
	}
	t.Logf("got err response for missing header '%s'", err)

	// test missing Bearer
	_, err = testVerifier.bearerTokenAuth(context.TODO(), "\"realm=foobar\",\"scope=foo\",\"service=bar\"")
	if err == nil {
		t.Error("expected error, got nil")
	}
	t.Logf("got err response for missing Bearer '%s'", err)

	// bad json response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{fuuuuuuuu"))
	}))
	defer ts.Close()

	testVerifier.Client = ts.Client()
	_, err = testVerifier.bearerTokenAuth(context.TODO(), fmt.Sprintf("Bearer \"realm=%s\",\"scope=foo\",\"service=bar\"", ts.URL))
	if err == nil {
		t.Error("expected error, got nil")
	}
	t.Logf("got err response for bad JSON '%s'", err)
}
