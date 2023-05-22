package fs

import (
	"net/http"
)

// HttpFsProxy proxies Open requests to files on a Dst http.FileSystem.
// Upon Open, NameFunc is executed to generate the target file name on
// Dst.
//
// This is primarily useful when Dst has been derived from an embed.FS,
// object granting control over the path of the target file.
type HttpFsProxy struct {
	// Dst http.FileSystem.
	Dst http.FileSystem
	// NameFunc is used to generate the value prefixed to the
	// target file targeted via name. Output, i.e., target, should
	// resolve to a file on Dst.
	NameFunc func(name string) (target string)
}

// Open generates the target file name via NameFunc and proxies the
// open request to Dst.
func (prx HttpFsProxy) Open(name string) (http.File, error) {
	return prx.Dst.Open(prx.NameFunc(name))
}
