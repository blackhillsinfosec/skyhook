package middleware

import (
    "fmt"
    obfuscate "github.com/blackhillsinfosec/skyhook-obfuscation"
    structs "github.com/blackhillsinfosec/skyhook/api_structs"
    "github.com/blackhillsinfosec/skyhook/server/inspector"
    "github.com/gin-gonic/gin"
    "net/http"
)

// DeobfUploadFilePath extracts and deobfuscates the filePath
// route variable and assigns two variables to gin.Context:
//
// 1. relFilePath - Relative file path to the target upload file.
// 2. absFilePath - Absolute file path to the target upload file.
func DeobfUploadFilePath(webroot *string, paramName string, chain *[]obfuscate.Obfuscator) gin.HandlerFunc {
    return func(c *gin.Context) {

        //=================
        // HANDLE FILE PATH
        //=================

        pathString := c.Param(paramName)
        if len(pathString) > 0 && pathString[0:1] == "/" {
            pathString = pathString[1:]
        }

        //=========================
        // HANDLE NO-PARAM SCENARIO
        //=========================
        // - Simply set these to empty string values
        // - This is needed to allow upload listing such that all
        //   routes can belong to the same group.

        if len(pathString) == 0 {
            c.Set("relFilePath", "")
            c.Set("absFilePath", "")
            return
        }

        //======================
        // DEOBFUSCATE FILE PATH
        //======================

        if pathBytes, err := obfuscate.Deobfuscate([]byte(pathString), *chain); err == nil {

            //================
            // SET relFilePath
            //================

            pathString = string(pathBytes)

            if pathString[0:1] != "/" {
                // Require a leading slash in the registration path, forming
                // an absolute "web path" to the resource.
                c.AbortWithStatusJSON(http.StatusNotAcceptable, structs.BaseResponse{
                    Success: false,
                    Message: fmt.Sprintf("Upload registration paths must begin with a slash, i.e., \"/%s\"", pathString),
                })
                return
            }

            c.Set("relFilePath", pathString)

            //================
            // SET absFilePath
            //================

            if abs, err := inspector.ToAbs(*webroot, pathString); err != nil {
                c.AbortWithStatus(http.StatusNotFound)
                return
            } else {
                c.Set("absFilePath", abs)
            }

        } else {

            c.AbortWithStatus(http.StatusNotFound)
            return

        }
    }
}
