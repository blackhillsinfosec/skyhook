package upload

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blackhillsinfosec/skyhook/log"
	"golang.org/x/exp/maps"
	"io"
	"os"
	"path"
	"sync"
	"time"
)

type Manager struct {
	registrants       map[string]*Upload
	registrantsFile   *string
	maxUploadDuration *uint
	writeMu           sync.Mutex
}

// Register manages creation of upload registrants.
func (m Manager) Register(afp, rfp string) (up Upload, err error) {
	if !m.RegistrantExists(rfp) {
		if up, err = NewUpload(afp, rfp, *m.maxUploadDuration); err == nil {
			m.registrants[rfp] = &up
			if err = m.SaveRegistrants(); err != nil {
				err = errors.New(fmt.Sprintf("failed to write upload registrant file: %v", err))
			} else {
				log.INFO.Printf("Created new upload for: %s", up.RelPath)
			}
		}
	} else {
		log.WARN.Printf("Upload already exists for: %s", up.RelPath)
		log.WARN.Printf("Failed to create new upload for: %s", up.RelPath)
		err = errors.New("upload already exists")
	}
	return up, err
}

// Deregister removes a registered upload from the registrants list
// and updates the manifest file.
func (m Manager) Deregister(relPath string) (err error) {
	if !m.RegistrantExists(relPath) {
		err = errors.New("upload does not exist")
	} else {
		log.INFO.Printf("Upload finished: %v", relPath)
		delete(m.registrants, relPath)
		m.SaveRegistrants()
	}
	return err
}

// SaveRegistrants is responsible for saving current registrants
// to disk, allowing the server to recover from fatal events.
func (m Manager) SaveRegistrants() error {
	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	if f, err := os.OpenFile(*m.registrantsFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600); err == nil {
		var data []byte
		if data, err = json.Marshal(&m.registrants); err == nil {
			_, err = f.Write(data)
		}
		return f.Close()
	} else {
		return err
	}
}

// CancelUpload is responsible for removing any partially uploaded
// files and removing the registered upload.
func (m Manager) CancelUpload(relPath string) error {
	up := m.registrants[relPath]
	if up == nil {
		return errors.New("unknown upload requested")
	}

	up.mu.Lock()
	defer func() {
		up.mu.Unlock()
		delete(m.registrants, up.RelPath)
		m.SaveRegistrants()
	}()

	if _, err := os.Stat(up.AbsPath); err == nil {
		if err := os.Remove(up.AbsPath); err != nil {
			return errors.New("failed to cancel upload")
		}
	}

	log.INFO.Printf("Canceling upload: %v", relPath)

	return nil
}

// ScanExpired iterates over each registered upload and removes
// those with an expired timestamp.
//
// This is effectively housekeeping to clean up after failures.
func (m Manager) ScanExpired() {
	for {
		for _, up := range m.registrants {
			if up.mu.TryLock() {
				if time.Now().After(up.Expiration) {
					log.INFO.Printf("Upload expired: %v", up.RelPath)
					up.mu.Unlock()
					m.CancelUpload(up.RelPath)
				} else {
					up.mu.Unlock()
				}
			}
		}
		time.Sleep(time.Minute)
	}
}

// SaveChunk saves a chunk of data to the Upload.AbsPath
// of the Upload identified by relPath. The chunk is written
// to the file at the byte offset identified by off.
//
// An error is returned when opening or writing to the file
// fails.
func (m Manager) SaveChunk(relPath string, chunk []byte, off uint64) error {
	up := m.registrants[relPath]
	if up == nil {
		return errors.New("unknown upload specified")
	}

	up.mu.Lock()
	defer up.mu.Unlock()

	// Open/create the file for writing
	if f, err := os.OpenFile(up.AbsPath, os.O_CREATE|os.O_RDWR, 0600); err != nil {
		return errors.New("failed to open upload file for writing")
	} else {
		// Write at the specified offset
		if _, err := f.WriteAt(chunk, int64(off)); err != nil {
			return errors.New("failed to write to upload file")
		}
		return f.Close()
	}
}

// Get attempts to retrieve the upload tracked by target.
func (m Manager) Get(relPath string) (Upload, error) {
	if m.registrants[relPath] == nil {
		return Upload{}, errors.New("upload not found")
	} else {
		return *m.registrants[relPath], nil
	}
}

func (m Manager) ListAll() (u []Upload) {
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	for _, v := range maps.Values(m.registrants) {
		u = append(u, *v)
	}
	return u
}

// RegistrantExists searches registrants to determine if one
// exists.
//
// See Get for more information on target.
func (m Manager) RegistrantExists(relPath string) bool {
	if _, err := m.Get(relPath); err != nil {
		return false
	}
	return true
}

// NewManager initializes an Manager. Should regFile
// be non-nil, logic will attempt to read the file from disk and
// parse it accordingly.
func NewManager(regFile *string, maxUploadDuration *uint) (um Manager, err error) {

	r := make(map[string]*Upload)
	if regFile != nil {

		//===========================
		// LOAD REGISTRANTS FROM FILE
		//===========================

		var f *os.File
		if f, err = os.OpenFile(*regFile, os.O_CREATE|os.O_RDONLY, 0600); err == nil {
			var b []byte
			if b, err = io.ReadAll(f); err == nil && len(b) > 0 {
				if err = json.Unmarshal(b, &r); err != nil {
					return um, err
				}
			}
			if err = f.Close(); err != nil {
				return um, err
			}
		}

	}

	return Manager{
		registrants:       r,
		registrantsFile:   regFile,
		maxUploadDuration: maxUploadDuration,
		writeMu:           sync.Mutex{},
	}, err
}

type Upload struct {
	AbsPath    string    `json:"abs_path" yaml:"abs_path"`
	RelPath    string    `json:"rel_path" yaml:"rel_path"`
	Expiration time.Time `json:"expiration" yaml:"expiration"`
	mu         sync.Mutex
}

func NewUpload(abs, rel string, maxDuration uint) (u Upload, err error) {
	// Assumptions
	//  - Caller has already ensured that relPath is in the webroot
	//  - relPath has been cleansed of malicious characters

	if rel == "" || abs == "" {
		return u, errors.New("destination file must have a value")
	}

	// Ensure target file doesn't exist, rejecting if so
	if _, err := os.Stat(abs); err == nil {
		return u, errors.New("file already exists")
	}

	// Ensure directory leading to relPath exists
	dir, _ := path.Split(abs)
	if dir != "" {
		if _, err := os.Stat(dir); err != nil {
			return u, errors.New("upload directory doesn't exist")
		}
	}

	// TODO should probably check disk space, too
	u = Upload{
		//Id:         uuid.New().String(),
		AbsPath:    abs,
		RelPath:    rel,
		Expiration: time.Now().Add(time.Duration(maxDuration) * time.Hour),
		mu:         sync.Mutex{},
	}
	return u, err
}
