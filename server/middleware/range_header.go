package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
)

// UpdateRangeHeader is used to adapt
func UpdateRangeHeader(headerName, rangePrefix *string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if *headerName != "Range" || *rangePrefix != "bytes" {
			value := c.Request.Header.Get(*headerName)
			if value != "" {
				value = strings.Replace(value, fmt.Sprintf("%s", *rangePrefix), "bytes", 1)
				c.Request.Header.Set("Range", value)
			}
		}
	}
}

func RangeHeader(headerName, rangePrefix *string, required bool) gin.HandlerFunc {

	return func(c *gin.Context) {

		h := c.Request.Header.Get(*headerName)

		if required && h == "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		} else if h == "" {
			c.Set("hasRange", false)
		} else {
			c.Set("hasRange", true)

			//===================
			// PARSE RANGE HEADER
			//===================
			// Logic was roughly derived from https.fs.parseRange
			// Expected header format is "Range: bytes=0-123456"
			// Multiple ranges is not supported

			pre := fmt.Sprintf("%s=", *rangePrefix)
			if !strings.HasPrefix(h, pre) {
				c.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
				return
			}

			ra := strings.TrimPrefix(h, pre)
			ra = textproto.TrimString(ra)

			if ra == "" {
				c.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
				return
			}

			// Split range on "-" and capture values
			start, end, ok := strings.Cut(ra, "-")
			if !ok || start == "" || end == "" {
				c.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
				return
			}

			// Extract range start
			sI, err := strconv.ParseUint(start, 10, 64)
			if err != nil {
				c.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
				return
			}

			// Extract range end
			eI, err := strconv.ParseUint(end, 10, 64)
			if err != nil {
				c.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
				return
			}

			// Ensure there's an actual range
			if sI >= eI {
				c.AbortWithStatus(http.StatusRequestedRangeNotSatisfiable)
				return
			}

			c.Set("rangeStart", sI)
			c.Set("rangeEnd", eI)
		}
	}
}
