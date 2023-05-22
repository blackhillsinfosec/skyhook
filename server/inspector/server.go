package inspector

import (
    "encoding/json"
    "errors"
    obfuscate "github.com/blackhillsinfosec/skyhook-obfuscation"
    "github.com/blackhillsinfosec/skyhook/log"
    "io/fs"
    "net/http"
    "os"
    "path"
    "path/filepath"
    "sort"
    "strings"
    "time"
)

type InspectResponse struct {
    Target  string     `json:"target" yaml:"target" binding:"required"`
    Entries []FileInfo `yaml:"entries" json:"entries"`
}

// fileInfoItems provides a name method for the fs.FileInfo
// elements.
type fileInfoItems []fs.FileInfo

// name accepts an element id and returns its name.
func (d fileInfoItems) name(i int) string {
    return d[i].Name()
}

// InspectFileServer provides a ServeHTTP method that functions
// similarly to the standard net.http.FileServer:
//
// - It accepts a file name, which is expected to be obfuscated.
// - The file name is base64 decoded and deobfuscated
type InspectFileServer struct {
    Webroot string
    chain   *[]obfuscate.Obfuscator
}

func (is InspectFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

    //====================================
    // PARSE FILE PATH & GENERATE RESPONSE
    //====================================

    dec, _ := obfuscate.Deobfuscate([]byte(r.URL.Path), *is.chain)

    if fName, err := ToAbs(is.Webroot, string(dec)); err != nil {

        //===============================
        // FAILED TO DERIVE ABSOLUTE PATH
        //===============================

        log.ERR.Printf("Failed to open requested target for inspection: %v", err)
        http.Error(w, "Not found.", http.StatusNotFound)

    } else {

        //====================================
        // OPEN FILE/DIRECTORY FOR ENUMERATION
        //====================================

        var f http.File
        if f, err = os.Open(fName); err != nil {

            //=========================
            // DIRECTORY/FILE NOT FOUND
            //=========================

            log.ERR.Printf("Failed to open requested target for inspection: %v", err)
            http.Error(w, "Not found.", http.StatusNotFound)

        } else {

            //=====================
            // DIRECTORY/FILE FOUND
            //=====================

            var stat fs.FileInfo
            if stat, err = f.Stat(); err != nil {

                //====================
                // FAILED TO STAT FILE
                //====================

                log.ERR.Printf("Failed to stat requested target for inspection: %v", err)
                http.Error(w, "Not found.", http.StatusNotFound)
            }

            var entries []FileInfo
            if stat.IsDir() {

                //===================
                // DIRECTORY RESPONSE
                //===================
                // Enumerate, extract, and sort items.

                // Read directory items into a fileInfoItems slice
                var infos fileInfoItems
                infos, err = f.Readdir(-1)

                if err != nil {
                    // Failed to read directory items
                    http.Error(w, "Error reading directory", http.StatusInternalServerError)
                    return
                }

                // Sort the items by name
                sort.Slice(infos, func(i, j int) bool {
                    return infos.name(i) < infos.name(j)
                })

                // Convert the sorted slice of info objects to NewFileInfo objects
                for _, i := range infos {
                    entries = append(entries, NewFileInfo(i))
                }

            } else {

                //==============
                // FILE RESPONSE
                //==============
                // A single file was requested

                entries = []FileInfo{NewFileInfo(stat)}

            }

            //==================================
            // MARSHAL THE RESPONSE DATA AS JSON
            //==================================

            if b, err := json.Marshal(InspectResponse{
                Target:  string(dec),
                Entries: entries,
            }); err != nil {
                log.ERR.Printf("Failed to marshal response data for inspection: %v", err)
                http.Error(w, "Not found.", http.StatusNotFound)
            } else {
                b, _ = obfuscate.Obfuscate(b, *is.chain)
                w.Write(b)
            }

        }
    }
}

// New returns an InspectFileServer capable of
// inspecting and returning JSON formatted fs.FileInfo
// data structures.
func New(webroot string, obfChain *[]obfuscate.Obfuscator) InspectFileServer {
    return InspectFileServer{
        Webroot: webroot,
        chain:   obfChain,
    }
}

// FileInfo provides a response structure for the
// inspect endpoint, allowing callers to retrieve
// information about files and directories on the
// file server.
type FileInfo struct {
    Name    string      `json:"name"`
    Size    int64       `json:"size"`
    Mode    fs.FileMode `json:"mode"`
    ModTime time.Time   `json:"mod_time"`
    IsDir   bool        `json:"is_dir"`
}

// NewFileInfo returns a FileInfo struct initialized from a
// fs.FileInfo object.
func NewFileInfo(fi fs.FileInfo) FileInfo {
    return FileInfo{
        Name:    fi.Name(),
        Size:    fi.Size(),
        Mode:    fi.Mode(),
        ModTime: fi.ModTime(),
        IsDir:   fi.IsDir(),
    }
}

// ToAbs accepts an absolute path to a root directory and a
// file/directory name, sanitizes the name, and then returns
// an absolute path to the targeted file.
//
// An error is returned only when the name contains invalid
// characters.
func ToAbs(root, name string) (abs string, err error) {
    if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
        return abs, errors.New("http: invalid character in file path")
    }

    if root == "" {
        root = "."
    }

    abs = filepath.Join(root, filepath.FromSlash(path.Clean("/"+name)))
    return abs, err
}
