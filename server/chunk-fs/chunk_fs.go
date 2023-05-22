package chunk_fs

import (
    obfuscate "github.com/blackhillsinfosec/skyhook-obfuscation"
    "github.com/blackhillsinfosec/skyhook/log"
    "net/http"
)

func New(root string, obfsChain *[]obfuscate.Obfuscator) http.FileSystem {
    fs := http.Dir(root)
    return &ObfChunkFilesystem{httpFs: fs, chain: obfsChain}
}

// ObfChunkFilesystem implements Open such that the path
// to the requested file decoded prior to opening it.
//
// - Requested file path (name) is expected to have been obfuscated
//   by the requesting client using the same chain as configured
//   in the server.
//   - Value is deobfuscated prior to attempting to access the
//     target file.
// - Automatically pulls files from http.FileServer
// - Chunks are specified via the Content-Range header sent by
//   clients
//   - ObfResponseWriter obfuscates the data prior to writing to
//     http.ResponseWriter
type ObfChunkFilesystem struct {
    httpFs http.FileSystem
    chain  *[]obfuscate.Obfuscator
}

// Open deobfuscates name and proxies the file request to an
// upstream http.Filesystem object, which returns the desired
// http.File object.
//
// Each individual chunk is obfuscated using ObfChunkFilesystem.chain.
func (fs ObfChunkFilesystem) Open(name string) (http.File, error) {
    if len(name) > 0 && name[0:1] == "/" {
        name = name[1:]
    }

    if dec, err := obfuscate.Deobfuscate([]byte(name), *fs.chain); err != nil {
        log.ERR.Printf("Failed to decode: %v", err)
        return nil, err
    } else {
        return fs.httpFs.Open(string(dec))
    }
}
