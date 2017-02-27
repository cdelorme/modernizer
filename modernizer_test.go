package modernizer

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
)

type MockFile struct {
	bytes.Buffer
}

func (self *MockFile) Close() error { return nil }

var (
	fileMock = &MockFile{}
	errMock  = errors.New("mock error")

	errOpenFile error
	errRestart  error
	errRename   []error
)

func init() {
	restart = func(_ string) error { return errRestart }
	openfile = func(_ string, _ int, _ os.FileMode) (io.WriteCloser, error) { return fileMock, errOpenFile }
	remove = func(_ string) error { return nil }
	rename = func(_, _ string) error {
		if len(errRename) == 0 {
			return nil
		}
		err := errRename[0]
		errRename = errRename[1:]
		return err
	}
}

func TestPlacebo(_ *testing.T) {}

// @note: skilling os.OpenFile test (uncertain how to mock)

func TestCheckEmptyCdn(t *testing.T) {
	t.Parallel()
	if Check("", "") == nil {
		t.Logf("failed to return error on empty cdn...")
		t.FailNow()
	}
}

// @note: skipping os.Executable and Symlink error handlers

func TestCheckBadCdn(t *testing.T) {
	if Check("", "not-a-valid-website") == nil {
		t.Logf("failed to identify http failure...\n")
		t.FailNow()
	}
}

// @note: skipping body read error handler

func TestCheckNoVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	if err := Check("1", ts.URL); err == nil {
		t.Logf("failed to notice matching versions...")
		t.FailNow()
	}
}

func TestCheckMatchingVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/version" {
			w.Write([]byte("1"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	if err := Check("1", ts.URL); err == nil {
		t.Logf("failed to notice matching versions...")
		t.FailNow()
	}
}

func TestCheckNoSignatures(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/version" {
			w.Write([]byte("2"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	if err := Check("1", ts.URL); err == nil {
		t.Logf("failed to notice empty sha256sums...")
		t.FailNow()
	}
}

func TestCheckInvalidSignatures(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/version" {
			w.Write([]byte("2"))
		} else if r.RequestURI == "/sha256sums" {
			w.Write([]byte("hash fake"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	if err := Check("1", ts.URL); err == nil {
		t.Logf("failed to identify no matching signature...")
		t.FailNow()
	}
}

func TestCheckNoBinary(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/version" {
			w.Write([]byte("2"))
		} else if r.RequestURI == "/sha256sums" {
			w.Write([]byte("hash " + runtime.GOOS + "-" + runtime.GOARCH + "-binary"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	if err := Check("1", ts.URL); err == nil {
		t.Logf("failed to notice no binary exists...")
		t.FailNow()
	}
}

func TestCheckOpenFileError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/version" {
			w.Write([]byte("2"))
		} else if r.RequestURI == "/sha256sums" {
			w.Write([]byte("e49295702f7da8670778e9b95a281b72b41b31cb16afa376034b45f59a18ea3f " + runtime.GOOS + "-" + runtime.GOARCH + "-binary"))
		} else {
			w.Write([]byte("bananas"))
		}
	}))
	defer ts.Close()

	errOpenFile = errMock
	if err := Check("1", ts.URL); err == nil {
		t.Logf("failed to notice invalid binary shasum...")
		t.FailNow()
	}
	errOpenFile = nil
}

// @note: skipping io.Copy error handler

func TestCheckInvalidBinary(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/version" {
			w.Write([]byte("2"))
		} else if r.RequestURI == "/sha256sums" {
			w.Write([]byte("e49295702f7da8670778e9b95a281b72b41b31cb16afa376034b45f59a18ea3f " + runtime.GOOS + "-" + runtime.GOARCH + "-binary"))
		} else {
			w.Write([]byte("baddata"))
		}
	}))
	defer ts.Close()

	if err := Check("1", ts.URL); err == nil {
		t.Logf("failed to notice invalid binary shasum...")
		t.FailNow()
	}
}

func TestCheckRename(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/version" {
			w.Write([]byte("2"))
		} else if r.RequestURI == "/sha256sums" {
			w.Write([]byte("e49295702f7da8670778e9b95a281b72b41b31cb16afa376034b45f59a18ea3f " + runtime.GOOS + "-" + runtime.GOARCH + "-binary"))
		} else {
			w.Write([]byte("bananas\n"))
		}
	}))
	defer ts.Close()

	errRename = []error{errMock}
	if err := Check("1", ts.URL); err == nil {
		t.Logf("failed to catch first rename failure...")
		t.FailNow()
	}

	errRename = []error{nil, errMock}
	if err := Check("1", ts.URL); err == nil {
		t.Logf("failed to catch second rename failure...")
		t.FailNow()
	}

}

func TestCheckRestart(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/version" {
			w.Write([]byte("2"))
		} else if r.RequestURI == "/sha256sums" {
			w.Write([]byte("e49295702f7da8670778e9b95a281b72b41b31cb16afa376034b45f59a18ea3f " + runtime.GOOS + "-" + runtime.GOARCH + "-binary"))
		} else {
			w.Write([]byte("bananas\n"))
		}
	}))
	defer ts.Close()

	errRestart = errMock
	if err := Check("1", ts.URL); err == nil {
		t.Logf("failed to capture restart failure...")
		t.FailNow()
	}

	errRestart = nil
	if err := Check("1", ts.URL); err != nil {
		t.Logf("failed to succeed...")
		t.FailNow()
	}
}
