package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUploadHandler(t *testing.T) {
	config := conf{ListenPort: "9666", ShieldURL: "blah.com"}
	uploadHandle := uploadHandler(config)
	// GET
	req, _ := http.NewRequest("GET", "", nil)
	w := httptest.NewRecorder()
	uploadHandle.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("uploadHandler GET returned %v, should be %v", w.Code, http.StatusMethodNotAllowed)
	}
}
