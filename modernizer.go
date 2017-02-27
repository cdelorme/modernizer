package modernizer

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	errNoCdn     = errors.New("no cdn provided...")
	errVersion   = errors.New("unable to get version file...")
	errNewest    = errors.New("no new versions available...")
	errSignature = errors.New("unable to get sha256sums file...")
	errNoMatch   = errors.New("no matching signature found...")
	errBinary    = errors.New("unable to download new binary...")
	errBadSum    = errors.New("new binary did not match sha256sum...")

	rename   = os.Rename
	remove   = os.Remove
	openfile = func(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
		f, err := os.OpenFile(name, flag, perm)
		return f, err
	}
)

func Check(version, cdn string) error {
	if cdn = strings.TrimSuffix(cdn, "/"); cdn == "" {
		return errNoCdn
	}

	executableOrigin, err := os.Executable()
	if err != nil {
		return err
	}

	executable, err := filepath.EvalSymlinks(executableOrigin)
	if err != nil {
		return err
	}

	executableName := filepath.Base(executable)
	executableDir := filepath.Dir(executable)
	executableNew := filepath.Join(executableDir, "."+executableName+".new")
	executableOld := filepath.Join(executableDir, "."+executableName+".old")

	remove(executableOld)

	versionResponse, err := http.Get(cdn + "/version")
	if err != nil {
		return err
	}
	defer versionResponse.Body.Close()
	if versionResponse.StatusCode != http.StatusOK {
		return errVersion
	}

	cdnVersion := bytes.NewBuffer([]byte{})
	if _, err := cdnVersion.ReadFrom(versionResponse.Body); err != nil {
		return err
	} else if cdnVersion.String() == version {
		return errNewest
	}

	signatureResponse, err := http.Get(cdn + "/sha256sums")
	if err != nil {
		return err
	}
	defer signatureResponse.Body.Close()
	if signatureResponse.StatusCode != http.StatusOK {
		return errSignature
	}

	cdnSignatures := bytes.NewBuffer([]byte{})
	if _, err := cdnSignatures.ReadFrom(signatureResponse.Body); err != nil {
		return err
	}

	cdnSignatureScanner := bufio.NewScanner(cdnSignatures)
	var binaryRemoteFile, binaryRemoteSum string
	for cdnSignatureScanner.Scan() {
		if strings.Contains(strings.ToLower(cdnSignatureScanner.Text()), runtime.GOOS) && strings.Contains(cdnSignatureScanner.Text(), runtime.GOARCH) {
			if combo := strings.SplitN(cdnSignatureScanner.Text(), " ", 2); len(combo) == 2 {
				binaryRemoteSum, binaryRemoteFile = combo[0], combo[1]
				break
			}
		}
	}
	if binaryRemoteFile == "" || binaryRemoteSum == "" {
		return errNoMatch
	}

	binaryResponse, err := http.Get(cdn + "/" + binaryRemoteFile)
	if err != nil {
		return err
	}
	defer binaryResponse.Body.Close()
	if binaryResponse.StatusCode != http.StatusOK {
		return errBinary
	}

	binaryNew, err := openfile(executableNew, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	hasher := sha256.New()
	if _, err := io.Copy(binaryNew, io.TeeReader(binaryResponse.Body, hasher)); err != nil {
		binaryNew.Close()
		remove(executableNew)
		return err
	} else if hex.EncodeToString(hasher.Sum(nil)) != binaryRemoteSum {
		binaryNew.Close()
		remove(executableNew)
		return errBadSum
	}
	binaryNew.Close()

	if err := rename(executable, executableOld); err != nil {
		remove(executableNew)
		return err
	}
	if err := rename(executableNew, executable); err != nil {
		remove(executableNew)
		remove(executable)
		rename(executableOld, executable)
		return err
	}

	if err := restart(executable); err != nil {
		remove(executable)
		rename(executableOld, executable)
		return err
	}

	return nil
}
