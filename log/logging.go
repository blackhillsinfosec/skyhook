package log

import (
	"log"
	"os"
)

var (
	INFO    *log.Logger
	WARN    *log.Logger
	ERR     *log.Logger
	FSERVER *log.Logger
	flags   = log.Ltime | log.Ldate
)

func init() {
	INFO = log.New(os.Stderr, "[Skyhook] [INF] ", flags)
	WARN = log.New(os.Stderr, "[Skyhook] [WRN] ", flags)
	ERR = log.New(os.Stderr, "[Skyhook] [ERR] ", flags)
	FSERVER = log.New(os.Stderr, "[Skyhook File Server] [ERR] ", flags)
}
