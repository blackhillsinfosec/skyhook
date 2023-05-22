package middleware

import (
    "bufio"
    "bytes"
    obfuscate "github.com/blackhillsinfosec/skyhook-obfuscation"
    "github.com/blackhillsinfosec/skyhook/log"
    "github.com/gin-gonic/gin"
    "golang.org/x/exp/slices"
    "io"
    "net"
    "net/http"
    "strconv"
)

// readerOnly implements io.Reader and no additional methods.
//
// This is useful because bytes.Reader implements io.WriterTo,
// which hijacks the flow of execution in io.copyBuffer.
//
// This is most relevant to ObfResponseWriter.ReadFrom.
type readerOnly struct {
    b *bytes.Reader
}

// Read satisfies io.Reader.
func (r readerOnly) Read(b []byte) (int, error) {
    return r.b.Read(b)
}

// ObfResponse returns a middleware that obfuscates
// all response data using chain.
//
// streamer determines if multiple writes to the writer
// will occur, such as when using http.FileServer to
// serve files directly from the filesystem.
func ObfResponse(chain *[]obfuscate.Obfuscator, streamer bool) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer = NewObfResponseWriter(c.Writer, chain, streamer)
    }
}

// NewObfResponseWriter initializes a response writer
// that will obfuscate the response body.
func NewObfResponseWriter(w gin.ResponseWriter, chain *[]obfuscate.Obfuscator, streamer bool) ObfResponseWriter {
    return ObfResponseWriter{
        w:        w,
        chain:    chain,
        streamer: streamer,
    }
}

// ObfResponseWriter ensures that all content written to a response
// is obfuscated. It behaves as a proxy to a gin.ResponseWriter
// object, which is responsible for crafting the final response.
type ObfResponseWriter struct {
    w        gin.ResponseWriter
    chain    *[]obfuscate.Obfuscator
    streamer bool
}

// Write proxies the Write call to gin.ResponseWriter. When an
// expected status code has been applied to the writer, b will
// be seamlessly obfuscated using the configured obfuscation chain
// prior to being written to the response.
func (tw ObfResponseWriter) Write(b []byte) (int, error) {
    if !tw.streamer && slices.Contains([]int{200, 206, 406, 409}, tw.Status()) {
        enc, _ := obfuscate.Obfuscate(b, *tw.chain)
        tw.Header().Set("Content-Length", strconv.FormatInt(int64(len(enc)), 10))
        tw.Header().Set("Content-Type", "text/plain")
        return tw.w.Write(enc)
    }
    return tw.w.Write(b)
}

// ReadFrom is implemented to intercept calls from io.CopyN, a
// functioned used by the net.http module when serving files.
// Intercepting at this point allows us to read in the chunk from
// the file and obfuscate it prior to writing it to the response.
//
// Additional Notes:
//
// io.CopyN is called by http.serveContent when serving contents of
// a file, which converts the source file to an io.LimitedReader that
// will read up to only N bytes (per the Range header).
func (tw ObfResponseWriter) ReadFrom(r io.Reader) (n int64, err error) {

    if l, ok := r.(*io.LimitedReader); ok {

        // Read in the chunk content
        buff := make([]byte, l.N)
        _, err = l.Read(buff)

        // Obfuscate the content
        buff, err = obfuscate.Obfuscate(buff, *tw.chain)
        if err != nil {
            log.ERR.Printf("Download Chunk Error: Failed to obfuscate data > %v", err)
            return n, err
        }

        // Set proper content length header
        tw.Header().Set("Content-Length", strconv.FormatInt(int64(len(buff)), 10))
        tw.Header().Set("Content-Type", "text/plain")

        // We use a readerOnly to ensure that no additional methods
        // interfere with io.copyBuffer (called by io.CopyN), which
        // will pass the response writer to any reader that implements
        // io.WriterTo.
        reader := readerOnly{bytes.NewReader(buff)}
        _, err = io.CopyBuffer(tw.w, reader, nil)

        if err != nil {
            log.ERR.Printf("Download Chunk Error: Failed to copy obfuscated data to the response > %v", err)
        }

        return l.N, err

    } else {

        panic("l must be an *io.LimitedReader")

    }
}

// Hijack proxies the method call to a gin.ResponseWriter.
func (tw ObfResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
    return tw.w.Hijack()
}

// Flush proxies the method call to a gin.ResponseWriter.
func (tw ObfResponseWriter) Flush() {
    println("flushed")
    tw.w.Flush()
}

// CloseNotify proxies the method call to a gin.ResponseWriter.
func (tw ObfResponseWriter) CloseNotify() <-chan bool {
    return tw.w.CloseNotify()
}

// Status proxies the method call to a gin.ResponseWriter.
func (tw ObfResponseWriter) Status() int {
    return tw.w.Status()
}

// Size proxies the method call to a gin.ResponseWriter.
func (tw ObfResponseWriter) Size() int {
    return tw.w.Size()
}

// WriteString proxies the method call to a gin.ResponseWriter.
func (tw ObfResponseWriter) WriteString(s string) (int, error) {
    return tw.w.WriteString(s)
}

// Written proxies the method call to a gin.ResponseWriter.
func (tw ObfResponseWriter) Written() bool {
    return tw.w.Written()
}

// WriteHeaderNow proxies the method call to a gin.ResponseWriter.
func (tw ObfResponseWriter) WriteHeaderNow() {
    tw.w.WriteHeaderNow()
}

// Pusher proxies the method call to a gin.ResponseWriter.
func (tw ObfResponseWriter) Pusher() http.Pusher {
    return tw.w.Pusher()
}

// Header proxies the method call to a gin.ResponseWriter.
func (tw ObfResponseWriter) Header() http.Header {
    return tw.w.Header()
}

// WriteHeader proxies the method call to a gin.ResponseWriter.
func (tw ObfResponseWriter) WriteHeader(s int) {
    tw.w.WriteHeader(s)
}
