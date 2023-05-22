package middleware

import (
    obfuscate "github.com/blackhillsinfosec/skyhook-obfuscation"
    "github.com/gin-gonic/gin"
    "io"
)

type ByteReadCloser struct {
    Src   io.ReadCloser
    Chain *[]obfuscate.Obfuscator
}

func (b ByteReadCloser) Deobfuscated() ([]byte, error) {
    data, err := io.ReadAll(b.Src)
    if err == nil {
        data, err = obfuscate.Deobfuscate(data, *b.Chain)
    }
    return data, err
}

func (b ByteReadCloser) Read(d []byte) (int, error) {
    return b.Src.Read(d)
}

func (b ByteReadCloser) Close() error {
    return b.Src.Close()
}

func DeobfReqBody(chain *[]obfuscate.Obfuscator) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Request.Body = ByteReadCloser{
            Src:   c.Request.Body,
            Chain: chain,
        }
    }
}
