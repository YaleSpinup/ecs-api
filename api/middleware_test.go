package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func TestTokenMiddleware(t *testing.T) {
	psk := []byte("sometesttoken")
	tokenHeader, _ := bcrypt.GenerateFromPassword(psk, bcrypt.DefaultCost)

	// Test handler that just returns 200 OK
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Got request on %+v", r.URL)
		w.WriteHeader(http.StatusOK)
	})

	pubUrls := map[string]string{
		"/foo": "public",
		"/bar": "public",
		"/baz": "public",
	}

	// Start a new server with our token middleware and test handler
	server := httptest.NewServer(TokenMiddleware(psk, pubUrls, okHandler))
	defer server.Close()

	// Test some public urls
	for u := range pubUrls {
		url := fmt.Sprintf("%s%s", server.URL, u)
		t.Logf("Getting %s", url)
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Received %d for public url '%s', expected %d", resp.StatusCode, u, http.StatusOK)
		}
	}

	// Test a bad URI
	_, err := http.Get(fmt.Sprintf("%s/\n", server.URL))
	if err == nil {
		t.Fatal("expected error for bad URL")
	}

	// Test a private URL without an auth token
	resp, err := http.Get(fmt.Sprintf("%s/private", server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Received status: %d for '%s/private', expected %d", resp.StatusCode, server.URL, http.StatusForbidden)
	}

	// Test a private URL _with_ an auth-token
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/private", server.URL), nil)
	req.Header.Add("X-Auth-Token", string(tokenHeader))
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Received %d for '%s/private', expected %d", resp.StatusCode, server.URL, http.StatusOK)
	}

	// Test a private URL with options
	req, _ = http.NewRequest(http.MethodOptions, fmt.Sprintf("%s/optionstuff", server.URL), nil)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Received %d for '%s/optionstuff', expected %d", resp.StatusCode, server.URL, http.StatusOK)
	}

	testHeaders := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Headers": "X-Auth-Token",
	}

	for k, v := range testHeaders {
		if h, ok := resp.Header[k]; !ok || h[0] != v {
			t.Errorf("Expected response header %s from OPTIONS request to be %s, got %s", k, v, h[0])
		}
	}
}

func BenchmarkTokenMiddlewarePub(b *testing.B) {
	log.SetLevel(log.ErrorLevel)

	psk := []byte("0232ecdb-8ce2-4125-808b-8056b24d3a49")

	// Test handler that just returns 200 OK
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	pubUrls := map[string]string{
		"/public": "public",
		"/bar":    "public",
		"/baz":    "public",
	}

	// Start a new server with our token middleware and test handler
	server := httptest.NewServer(TokenMiddleware(psk, pubUrls, okHandler))
	defer server.Close()

	for n := 0; n < b.N; n++ {
		// Test a private URL _with_ an auth-token
		client := &http.Client{}
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/public", server.URL), nil)
		_, err := client.Do(req)
		if err != nil {
			log.Errorf("unexpected error: %s", err)
		}
	}
}

func BenchmarkTokenMiddlewarePrivMinCost(b *testing.B) {
	log.SetLevel(log.ErrorLevel)

	psk := []byte("0232ecdb-8ce2-4125-808b-8056b24d3a49")
	tokenHeader, _ := bcrypt.GenerateFromPassword(psk, bcrypt.MinCost)

	// Test handler that just returns 200 OK
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	pubUrls := map[string]string{
		"/public": "public",
		"/bar":    "public",
		"/baz":    "public",
	}

	// Start a new server with our token middleware and test handler
	server := httptest.NewServer(TokenMiddleware(psk, pubUrls, okHandler))
	defer server.Close()

	for n := 0; n < b.N; n++ {
		// Test a private URL _with_ an auth-token
		client := &http.Client{}
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/private", server.URL), nil)
		req.Header.Add("X-Auth-Token", string(tokenHeader))
		_, err := client.Do(req)
		if err != nil {
			log.Errorf("unexpected error: %s", err)
		}
	}
}

func BenchmarkTokenMiddlewarePrivDefCost(b *testing.B) {
	log.SetLevel(log.ErrorLevel)

	psk := []byte("0232ecdb-8ce2-4125-808b-8056b24d3a49")
	tokenHeader, _ := bcrypt.GenerateFromPassword(psk, bcrypt.DefaultCost)

	// Test handler that just returns 200 OK
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	pubUrls := map[string]string{
		"/public": "public",
		"/bar":    "public",
		"/baz":    "public",
	}

	// Start a new server with our token middleware and test handler
	server := httptest.NewServer(TokenMiddleware(psk, pubUrls, okHandler))
	defer server.Close()

	for n := 0; n < b.N; n++ {
		// Test a private URL _with_ an auth-token
		client := &http.Client{}
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/private", server.URL), nil)
		req.Header.Add("X-Auth-Token", string(tokenHeader))
		_, err := client.Do(req)
		if err != nil {
			log.Errorf("unexpected error: %s", err)
		}
	}
}

func BenchmarkTokenMiddlewarePrivCostSix(b *testing.B) {
	log.SetLevel(log.ErrorLevel)

	psk := []byte("0232ecdb-8ce2-4125-808b-8056b24d3a49")
	tokenHeader, _ := bcrypt.GenerateFromPassword(psk, 6)

	// Test handler that just returns 200 OK
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	pubUrls := map[string]string{
		"/public": "public",
		"/bar":    "public",
		"/baz":    "public",
	}

	// Start a new server with our token middleware and test handler
	server := httptest.NewServer(TokenMiddleware(psk, pubUrls, okHandler))
	defer server.Close()

	for n := 0; n < b.N; n++ {
		// Test a private URL _with_ an auth-token
		client := &http.Client{}
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/private", server.URL), nil)
		req.Header.Add("X-Auth-Token", string(tokenHeader))
		_, err := client.Do(req)
		if err != nil {
			log.Errorf("unexpected error: %s", err)
		}
	}
}
