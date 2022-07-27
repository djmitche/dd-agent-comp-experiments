// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package flare

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/djmitche/dd-agent-comp-experiments/comp/ipcapi"
	"github.com/mholt/archiver"
)

type flare struct {
	// Mutex covers all fields
	sync.Mutex

	// callbacks contains all registered callbacks (as would be given to Register)
	callbacks []func(string) error
}

func newFlare(ipcapi ipcapi.Component) Component {
	f := &flare{}
	ipcapi.Register("/agent/flare", f.ipcHandler)

	// XXX: there's a possible dependency cycle here if comp/util/log were to
	// Register its own callback to record agent logs, because this component
	// depends on comp/ipcapi which depends on comp/util/log.  The easiest way
	// to break that cycle is likely for this component to register its own
	// handler for agent logs.

	return f
}

func newMock(ipcapi ipcapi.Component) Component {
	// mock is just like the real thing, but doesn't use ipcapi.
	return &flare{}
}

// RegisterFile implements Component#RegisterFile.
func (f *flare) RegisterFile(filename string, cb func() (string, error)) {
	if filepath.IsAbs(filename) {
		panic(fmt.Sprintf("path %s is not relative", filename))
	}

	// register this as a closure capturing filename and cb
	f.Register(func(flareDir string) error {
		content, err := cb()
		if err != nil {
			return err
		}

		fullpath := filepath.Join(flareDir, filename)
		parentDir := filepath.Dir(fullpath)
		err = os.MkdirAll(parentDir, 0o700)
		if err != nil {
			return err
		}

		return ioutil.WriteFile(fullpath, []byte(content), 0o600)
	})
}

// Register implements Component#Register.
func (f *flare) Register(cb func(flareDir string) error) {
	f.Lock()
	defer f.Unlock()
	f.callbacks = append(f.callbacks, cb)
}

// CreateFlare implements Component#CreateFlare.
func (f *flare) CreateFlare() (string, error) {
	f.Lock()
	defer f.Unlock()

	// make a root temporary directory
	rootDir, err := ioutil.TempDir("", "experiment-flare-*")
	if err != nil {
		return "", err
	}

	flareDir := filepath.Join(rootDir, "hostname") // TODO: figure out hostname

	err = os.MkdirAll(flareDir, 0o700)
	if err != nil {
		return "", err
	}

	// on completion, remove the flareDir, but leave the archiveFile.
	defer os.RemoveAll(flareDir)

	err = f.writeFlareFiles(flareDir, false)
	if err != nil {
		return "", err
	}

	archiveFile := filepath.Join(rootDir, "hostname-timestamp.zip")
	err = f.createArchive(flareDir, archiveFile)
	if err != nil {
		return "", err
	}

	return archiveFile, nil
}

// CreateFlareRemote implements Component#CreateFlareRemote.
func (f *flare) CreateFlareRemote() (string, error) {
	panic("TODO")
}

// GetFlareFile implements Mock#GetFlareFile.
func (f *flare) GetFlareFile(t *testing.T, filename string) (string, error) {
	f.Lock()
	defer f.Unlock()

	flareDir := t.TempDir()
	err := f.writeFlareFiles(flareDir, true)
	if err != nil {
		return "", err
	}

	content, err := ioutil.ReadFile(filepath.Join(flareDir, filename))
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// ipcHandler serves the /agent/flare endpoint.  On success, this returns a 200
// with {"filename": <filename>} giving the local filename of the flare file.
func (f *flare) ipcHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header()["Content-Type"] = []string{"application/json; charset=UTF-8"}

	archiveFile, err := f.CreateFlare()
	if err != nil {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"filename": archiveFile,
	})
}

// writeFlareFiles calls all of the callbacks to write all flare files to disk.
// If returnErrors is true then errors from callbacks are returned immediately
// (for testing).
//
// XXX note that this does no scrubbing
//
// It assumes f is locked.
func (f *flare) writeFlareFiles(flareDir string, returnErrors bool) error {
	errors := []string{}
	for _, cb := range f.callbacks {
		err := cb(flareDir)
		if err != nil {
			if returnErrors {
				return err
			}
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		// attempt to write FLARE-ERRORS.txt; an error here is actually fatal
		err := ioutil.WriteFile(
			filepath.Join(flareDir, "FLARE-ERRORS.txt"),
			[]byte(strings.Join(errors, "\n")),
			0o600)
		if err != nil {
			return err
		}
	}

	return nil
}

// createArchive creates an archive of the flareDir.
//
// It assumes f is locked.
func (f *flare) createArchive(flareDir, archiveFile string) error {
	return archiver.Archive([]string{flareDir}, archiveFile)
}
