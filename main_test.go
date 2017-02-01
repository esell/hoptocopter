package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
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

	// POST
	sampleCover, err := os.Open("samples/coverage.out")
	if err != nil {
		t.Errorf("error opening sample coverage file: %s", err)
	}
	defer sampleCover.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "coverage.out")
	if err != nil {
		t.Errorf("error FormFile: %s", err)
	}
	_, err = io.Copy(part, sampleCover)
	if err != nil {
		t.Errorf("error copying sampleCover to FormFile: %s", err)
	}
	if err := writer.Close(); err != nil {
		t.Errorf("error closing form writer: %s", err)
	}

	req, _ = http.NewRequest("POST", "?repo=blah", body)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	uploadHandle.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Errorf("uploadHandler POST returned %v, should be %v", w.Code, http.StatusSeeOther)
	}

	// cleanup
	if err := os.RemoveAll("blah"); err != nil {
		t.Errorf("error cleaning up after uploadHandler(): %s", err)
	}
}

func TestDisplayHandler(t *testing.T) {
	config := conf{ListenPort: "9666", ShieldURL: "blah.com"}
	displayHandle := displayHandler(config)

	// GET
	req, _ := http.NewRequest("GET", "", nil)
	w := httptest.NewRecorder()
	displayHandle.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Errorf("displayHandler GET returned %v, should be %v", w.Code, http.StatusSeeOther)
	}

	// POST
	req, _ = http.NewRequest("POST", "?repo=blah", nil)
	w = httptest.NewRecorder()
	displayHandle.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("displayHandler POST returned %v, should be %v", w.Code, http.StatusMethodNotAllowed)
	}

}
