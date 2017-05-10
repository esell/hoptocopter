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

func TestToInt(t *testing.T) {
	tempInt := toInt("20")
	if tempInt != 20 {
		t.Errorf("toInt returned %v, should be %v", tempInt, 20)
	}

}

func TestParseProfile(t *testing.T) {
	tempProfs, err := ParseProfiles("samples/coverage.out")
	if err != nil {
		t.Errorf("ParseProfile returned error: %v", err)
	}
	if len(tempProfs) != 1 {
		t.Errorf("ParseProfile length returned %v, should be %v", len(tempProfs), 1)
	}
	if tempProfs[0].FileName != "github.com/esell/hoptocopter/main.go" {
		t.Errorf("ParseProfile FileName returned %v, should be %v", tempProfs[0].FileName, "github.com/esell/hoptocopter/main.go")
	}

}

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
	config := conf{ListenPort: "9666", ShieldURL: "https://img.shields.io/badge"}
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

	// real GET - should fail (500)
	req, _ = http.NewRequest("GET", "?repo=blah", nil)
	w = httptest.NewRecorder()
	displayHandle.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("displayHandler GET returned %v, should be %v", w.Code, http.StatusInternalServerError)
	}

	// real GET - should work
	// stage test files

	origCvr, err := os.Open("samples/coverage.out")
	if err != nil {
		t.Errorf("error opening up sample coverage.out: %s", err)
	}
	defer origCvr.Close()
	if err := os.MkdirAll("blah", 0755); err != nil {
		t.Errorf("error creating directory for %s: %s\n", "blah", err)
	}
	copyCvr, err := os.Create("blah/coverage.out")
	if err != nil {
		t.Errorf("error creating copy of coverage.out: %s", err)
	}
	_, err = io.Copy(copyCvr, origCvr)
	if err != nil {
		t.Errorf("error writing copy of coverage.out: %s", err)
	}
	if err := copyCvr.Close(); err != nil {
		t.Errorf("error saving copy of coverage.out: %s", err)
	}
	req, _ = http.NewRequest("GET", "?repo=blah", nil)
	w = httptest.NewRecorder()
	displayHandle.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("displayHandler GET returned %v, should be %v", w.Code, http.StatusOK)
	}

}
