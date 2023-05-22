package server

import (
    "bytes"
    "embed"
    "encoding/json"
    "fmt"
    jwt "github.com/appleboy/gin-jwt/v2"
    obfuscate "github.com/blackhillsinfosec/skyhook-obfuscation"
    structs "github.com/blackhillsinfosec/skyhook/api_structs"
    "github.com/blackhillsinfosec/skyhook/config"
    "github.com/blackhillsinfosec/skyhook/log"
    "github.com/blackhillsinfosec/skyhook/server/chunk-fs"
    "github.com/blackhillsinfosec/skyhook/server/inspector"
    mw "github.com/blackhillsinfosec/skyhook/server/middleware"
    "github.com/blackhillsinfosec/skyhook/server/upload"
    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
    "github.com/impostorkeanu/go-commoners/rando"
    "golang.org/x/exp/maps"
    "golang.org/x/exp/slices"
    "io"
    "net/http"
    "path"
    "strings"
    "time"
)

var (
    //go:embed web_apps/file/templates web_apps/file/build
    FileSpa embed.FS
)

type SkyhookServer struct {
    Config          *config.FileServerOptions
    Tls             *config.ManualTlsOptions
    Users           *[]config.Credential
    ObfuscatorChain *[]obfuscate.Obfuscator
    Webroot         *string
    UploadManager   *upload.Manager
    WebrootFS       http.FileSystem
    FileServer      http.Handler
    Global          *config.SkyhookConfig

    LandingFiles          landingFiles
    LandingFileEncryption *config.LandingFileEncryptionOptions
    LandingFileObf        *obfuscate.XOR

    indexContent         []byte
    manifestContent      []byte
    assetManifestContent []byte
}

func (ss *SkyhookServer) Run(detach bool) (err error) {
    ss.Webroot = &ss.Config.RootDir
    ss.LandingFileEncryption = &ss.Config.EncryptedLoader
    ss.LandingFileObf = &obfuscate.XOR{ss.LandingFileEncryption.Key}

    for realPath, fakePath := range ss.Config.Routes.LandingPage {
        jsLoaderTempUrls.Insert(realPath, fakePath)
    }

    // Use default Gin settings (default error and logging functionality)
    r := gin.Default()
    r.SetTrustedProxies(nil)

    //==========================
    // MIDDLEWARE CONFIGURATIONS
    //==========================

    // JWT MIDDLEWARE
    // Reference: https://github.com/appleboy/gin-jwt
    var authMiddleWare *jwt.GinJWTMiddleware
    if authMiddleWare, err = jwt.New(&jwt.GinJWTMiddleware{
        Timeout:       7 * (24 * time.Hour),
        MaxRefresh:    7 * (24 * time.Hour),
        IdentityKey:   ss.Global.Auth.Jwt.FieldKeys.Username,
        Realm:         ss.Global.Auth.Jwt.Realm,
        Key:           []byte(ss.Global.Auth.Jwt.SigningKey),
        TokenLookup:   fmt.Sprintf("header: %s", ss.Global.Auth.Header.Name),
        TokenHeadName: ss.Global.Auth.Header.Scheme,
        Authorizator:  func(cred any, c *gin.Context) bool { return true },
        Unauthorized:  mw.JwtIsUnauthorized,
        Authenticator: mw.JwtLoginHandler(&ss.Global.Users, false),
        IdentityHandler: mw.JwtIdentityHandler(
            &ss.Global.Auth.Jwt.FieldKeys.Username,
            &ss.Global.Auth.Jwt.FieldKeys.Admin),
        PayloadFunc: mw.JwtPayloadFunc(ss.Global, ss.ObfuscatorChain)}); err != nil {

        log.ERR.Printf("Failed to initialize JWT auth: %v", err)
        return err
    }

    if err := authMiddleWare.MiddlewareInit(); err != nil {
        log.ERR.Printf("Failed to initialize Gin JWT middleware: %v", err)
        return err
    }

    // CORS MIDDLEWARE
    var corsFqdns []string
    corsFqdns = append(corsFqdns, "https://"+ss.Config.Socket())
    corsFqdns = append(corsFqdns, ss.Config.AddtlCorsUrls...)
    r.Use(cors.New(cors.Config{
        //AllowWildcard:    true,
        AllowOrigins:     corsFqdns,
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
        AllowHeaders:     []string{"Content-Type", "Authorization"},
        ExposeHeaders:    []string{"*", "Authorization", "Content-Length"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))

    //=====================
    // ROUTE CONFIGURATIONS
    //=====================

    // Default route
    r.NoRoute(authMiddleWare.MiddlewareFunc(), func(c *gin.Context) {
        c.JSON(http.StatusNotFound, gin.H{})
    })

    ss.initLandingFiles()
    for _, fakePath := range ss.Config.Routes.LandingPage {
        // Set route for the current randomized path
        r.GET(fakePath, ss.ServeLandingFile)
    }

    r.GET(ss.Config.Routes.EncryptedLoader.Js, ss.ServeLoaderFile)
    r.GET(ss.Config.Routes.EncryptedLoader.Html, ss.ServeLoaderFile)
    r.GET(ss.Config.Routes.EncryptedLoader.AutoHtml, ss.ServeLoaderFile)

    //=======================================================
    // REDIRECT TO index.html -ONLY- WHEN SPECIFIED IN ROUTES
    //=======================================================

    r.GET("/", func(c *gin.Context) {
        if slices.Contains(maps.Values(ss.Config.Routes.LandingPage), "/index.html") {
            c.Redirect(302, "/index.html")
        } else {
            c.Status(http.StatusNotFound)
        }
    })

    //======================
    // AUTHENTICATION ROUTES
    //======================

    // NOTE the "/login" route has to remain static because
    //  authMiddleWare.LoginHandler uses the JWT token as a method
    //  of communicating API routes to the JS web interface.
    //
    //  Note that the routes are XOR encrypted with the user's
    //  token value.
    r.GET("/login", authMiddleWare.RefreshHandler)
    r.POST("/login", authMiddleWare.LoginHandler)
    r.POST(ss.Config.Routes.Api.Logout, authMiddleWare.LogoutHandler)

    //=============================
    // OBFUSCATED FILE CHUNK ROUTES
    //=============================

    // SERVE OBFUSCATED FILE CHUNKS
    // Root directory containing files to be served.
    // - This object is responsible for deobfuscating requested file
    //   paths.
    ss.WebrootFS = chunk_fs.New(*ss.Webroot, ss.ObfuscatorChain)

    filesRoute := ss.Config.Routes.Api.Download
    l := len(ss.Config.Routes.Api.Download)
    if ss.Config.Routes.Api.Download[l-2:l-1] != "/" {
        filesRoute += "/"
    }

    ss.FileServer = http.StripPrefix(filesRoute, http.FileServer(ss.WebrootFS))
    inspectServer := http.StripPrefix(filesRoute, inspector.New(*ss.Webroot, ss.ObfuscatorChain))

    baseGroup := r.Group(filesRoute)
    baseGroup.Use(
        authMiddleWare.MiddlewareFunc(),
        mw.UpdateRangeHeader(&ss.Config.RangeHeaderOptions.Name, &ss.Config.RangeHeaderOptions.RangePrefix),
        mw.ObfResponse(ss.ObfuscatorChain, true))
    {
        // GET indicates that we're looking to retrieve a chunk of a file
        baseGroup.GET("*filepath", func(c *gin.Context) {
            // TODO derive method of stopping requests for file chunks
            //  on files that are registered as currently being uploaded
            ss.FileServer.ServeHTTP(c.Writer, c.Request)
        })
        // PATCH indicates that we're looking to inspect files
        baseGroup.PATCH("*filepath", func(c *gin.Context) {
            inspectServer.ServeHTTP(c.Writer, c.Request)
        })
    }

    //======================
    // CHUNKED UPLOAD ROUTES
    //======================

    // Header that will contain range offsets for uploads
    // - We have to handle this manually here since we're
    //   no longer reusing logic from http.fs.FileSystem.
    // - Expected format > Range: bytes=0-8686
    // - Multiple or inverted ranges are not supported!
    //rangeHeaderName := "Range"

    upGroup := r.Group(ss.Config.Routes.Api.Upload)
    upGroup.Use(
        authMiddleWare.MiddlewareFunc(),
        mw.ObfResponse(ss.ObfuscatorChain, false),
        mw.DeobfUploadFilePath(ss.Webroot, "filePath", ss.ObfuscatorChain))
    {
        // GET indicates that we're listing all uploads
        upGroup.GET("", ss.ListUploads)
        // POST indicates that we're creating an upload
        upGroup.PUT("/*filePath", ss.RegisterUpload)
        // PATCH indicates that an upload is finished
        upGroup.PATCH("/*filePath", ss.UploadFinished)
        // PUT indicates that the request contains an upload chunk
        upGroup.POST("/*filePath",
            mw.RangeHeader(&ss.Config.RangeHeaderOptions.Name, &ss.Config.RangeHeaderOptions.RangePrefix, true),
            mw.DeobfReqBody(ss.ObfuscatorChain),
            ss.ReceiveChunk)
        // DELETE indicates that the request aims to cancel an ongoing upload
        //  NOTE: this deletes any partial upload from disk
        upGroup.DELETE("/*filePath", ss.CancelUpload)
    }

    //============================
    // MONITOR FOR EXPIRED UPLOADS
    //============================

    go func() {
        log.INFO.Print("Starting upload expiration scanner")
        ss.UploadManager.ScanExpired()
    }()

    //===============
    // RUN THE SERVER
    //===============

    if detach {
        go ss.runFileServer(r)
    } else {
        var err error
        err = ss.runFileServer(r)
        if err != nil {
            log.ERR.Printf("Error running file server: %v", err)
        }
    }
    return nil
}

func (ss *SkyhookServer) initLandingFiles() {
    //===============================
    // LOAD LANDING FILES INTO MEMORY
    //===============================

    ss.initIndexTemplate()
    ss.initAssetManifest()
    landingContent := map[string][]byte{}

    for fileName, _ := range ss.Config.Routes.LandingPage {

        switch fileName {
        case "asset-manifest.json":
            landingContent[fileName] = ss.assetManifestContent
            continue
        case "index.html":
            landingContent[fileName] = ss.indexContent
            continue
        }

        //========================================
        // INSERT BYTE CONTENT INTO landingContent
        //========================================

        pth := lookupStatic(fileName)
        if content, err := FileSpa.ReadFile(pth); err == nil {

            if fileName == "main.js" {
                //======================================================
                // UPDATE ROOT ELEMENT IN main.js TO CONFIGURABLE VALUE
                //======================================================
                content = bytes.Replace(
                    content,
                    []byte("\"root\""),
                    []byte(fmt.Sprintf("\"%s\"", ss.Config.EncryptedLoader.RootElementId)), -1)
            }

            landingContent[fileName] = content

        } else {

            if strings.HasSuffix(fileName, "license.txt") {
                // TODO Viper creates a potential point of confusion in the future
                //  with lowercasing of configuration keys
                // ignore errors related to lower case licensing
                // this is an issue with viper, which tends to lowercase all
                // configuration keys
                continue
            }

            msg := fmt.Sprintf("failed to load landing file %s: %v", fileName, err)
            log.ERR.Println(msg)
            panic(msg)

        }
    }

    //===============================================
    // UPDATE LANDING FILE PATHS TO RANDOMIZED VALUES
    //===============================================

    for fileName, fakePath := range ss.Config.Routes.LandingPage {

        realPath := lookupStatic(fileName)
        if slices.Index(staticPaths, realPath) == -1 {
            continue
        }

        // make fileName relative
        //fileName = strings.TrimPrefix(realPath, "web_apps/file/build/")
        fakePath = fakePath[1:]
        _, fakeName := path.Split(fakePath)

        //================================================
        // REPLACE REAL PATH WITH FAKE PATH IN EACH FILE
        //================================================

        for _, iRealPath := range staticPaths {

            _, iFileName := path.Split(iRealPath)

            iRealPath = path.Join("/", lookupStatic(iFileName))

            landingContent[iFileName] = bytes.Replace(
                landingContent[iFileName], []byte(realPath), []byte(fakePath), -1)

            landingContent[iFileName] = bytes.Replace(
                landingContent[iFileName], []byte(iRealPath), []byte(fakePath), -1)

            landingContent[iFileName] = bytes.Replace(
                landingContent[iFileName], []byte(fileName), []byte(fakeName), -1)

            landingContent[iFileName] = bytes.Replace(
                landingContent[iFileName], []byte(".map.map"), []byte(".map"), -1)
        }

    }

    //================================
    // INITIALIZE LANDING FILE OBJECTS
    //================================

    ss.LandingFiles = landingFiles{}

    for fileName, fakePath := range ss.Config.Routes.LandingPage {
        ss.LandingFiles.Append(path.Join("/", landingPrefix, fileName), fakePath, landingContent[fileName], ss.LandingFileObf)
    }
}

func (ss *SkyhookServer) runFileServer(e *gin.Engine) (err error) {
    log.INFO.Printf("Listening and serving HTTPS on %s\n", ss.Config.Socket())

    defer func() { log.ERR.Println(err) }()

    server := &http.Server{
        Addr:              ss.Config.Socket(),
        Handler:           e.Handler(),
        TLSConfig:         nil,
        ReadTimeout:       0,
        ReadHeaderTimeout: 0,
        WriteTimeout:      0,
        IdleTimeout:       0,
        MaxHeaderBytes:    0,
        TLSNextProto:      nil,
        ConnState:         nil,
        ErrorLog:          log.FSERVER,
        BaseContext:       nil,
        ConnContext:       nil,
    }

    err = server.ListenAndServeTLS(ss.Tls.CertPath, ss.Tls.KeyPath)
    return err
}

func (ss *SkyhookServer) UploadFinished(c *gin.Context) {
    rfp := c.MustGet("relFilePath").(string)
    if err := ss.UploadManager.Deregister(rfp); err != nil {
        c.AbortWithStatus(http.StatusNotFound)
        return
    } else {
        c.JSON(http.StatusOK, structs.BaseResponse{
            Success: true,
            Message: "Upload deregistered.",
        })
    }
}

func (ss *SkyhookServer) ListUploads(c *gin.Context) {
    c.JSON(http.StatusOK, structs.ListUploadsResponse{
        BaseResponse: structs.BaseResponse{
            Success: true,
            Message: "Listing uploads.",
        },
        Uploads: ss.UploadManager.ListAll(),
    })
}

// CancelUpload cancels an ongoing upload. If any chunks of the file
// have been written to disk, they will be removed.
func (ss *SkyhookServer) CancelUpload(c *gin.Context) {
    rfp := c.MustGet("relFilePath").(string)
    if err := ss.UploadManager.CancelUpload(rfp); err != nil {
        c.AbortWithStatus(http.StatusNotFound)
        return
    } else {
        c.JSON(http.StatusOK, structs.BaseResponse{
            Success: true,
            Message: "Upload canceled.",
        })
    }
}

// RegisterUpload inspects a file path and registers a given file
// as currently being uploaded by passing it to UploadManager.
func (ss *SkyhookServer) RegisterUpload(c *gin.Context) {

    //====================================
    // CHECK FOR CURRENTLY EXISTING UPLOAD
    //====================================

    rfp := c.MustGet("relFilePath").(string)
    afp := c.MustGet("absFilePath").(string)
    if ss.UploadManager.RegistrantExists(rfp) {
        c.JSON(http.StatusConflict, structs.BaseResponse{
            Success: false,
            Message: "Upload for path is already registered.",
        })
        return
    }

    //============================
    // ATTEMPT UPLOAD REGISTRATION
    //============================

    if up, err := ss.UploadManager.Register(afp, rfp); err != nil {
        c.JSON(http.StatusNotAcceptable, structs.RegisterUploadResponse{
            BaseResponse: structs.BaseResponse{
                Success: false,
                Message: err.Error(),
            },
            RegisterUploadRequest: structs.RegisterUploadRequest{
                Path: up.RelPath,
            },
        })
    } else {
        c.JSON(http.StatusOK, structs.RegisterUploadResponse{
            BaseResponse: structs.BaseResponse{
                Success: true,
                Message: "Upload registered",
            },
            RegisterUploadRequest: structs.RegisterUploadRequest{
                Path: rfp,
            },
            //Id: up.Id,
        })
    }
}

// ReceiveChunk accepts an uploaded file chunk and passes it to
// UploadManager so it can be saved to disk.
func (ss *SkyhookServer) ReceiveChunk(c *gin.Context) {
    // Parse Range header
    rp := c.MustGet("relFilePath").(string)
    rStart := c.MustGet("rangeStart").(uint64)
    if data, err := c.Request.Body.(mw.ByteReadCloser).Deobfuscated(); err != nil {
        c.AbortWithStatus(http.StatusNotFound)
    } else {
        if err := ss.UploadManager.SaveChunk(rp, data, rStart); err != nil {
            c.AbortWithStatus(http.StatusNotFound)
            return
        }
        //go func() {
        //    ss.UploadManager.SaveChunk(rp, data, rStart)
        //}()
    }
}

// initAssetManifest reads the asset-manifest.json file from the NPM
// build directory and updates relevant paths with values configured
// in ss.Config.Routes.LandingPage.
func (ss *SkyhookServer) initAssetManifest() {

    //==========================
    // UPDATE THE ASSET MANIFEST
    //==========================

    manifest := assetManifest{}
    if b, err := FileSpa.ReadFile("web_apps/file/build/asset-manifest.json"); err != nil {
        log.ERR.Printf("Failed to open asset-manifest.json for reading: %v", err)
        panic(err)
    } else {

        // Load the manifest
        if err := json.Unmarshal(b, &manifest); err != nil {
            log.ERR.Printf("Failed to parse asset-manifest.json")
            panic(err)
        }

        // Update each file entry
        for _, fileKey := range maps.Keys(manifest.Files) {
            _, fName := path.Split(fileKey)
            manifest.Files[fileKey] = ss.Config.Routes.LandingPage[fName]
            //maniPath := strings.TrimPrefix("/landing/", manifest.Files[fileKey])
            //for realPath, fakePath := range ss.Config.Routes.LandingPage {
            //	if strings.HasSuffix(realPath, maniPath) {
            //		manifest.Files[fileKey] = fakePath
            //		continue
            //	}
            //}
        }

        // Update each entrypoint
        for fileInd := 0; fileInd < len(manifest.Entrypoints); fileInd++ {
            _, fName := path.Split(manifest.Entrypoints[fileInd])
            manifest.Entrypoints[fileInd] = ss.Config.Routes.LandingPage[fName]
        }
    }

    //=====================
    // MARSHAL THE MANIFEST
    //=====================

    if b, err := json.Marshal(manifest); err != nil {
        log.ERR.Printf("Failed to marshal asset-manifest.json: %v", err)
        panic(err)
    } else {
        ss.assetManifestContent = b
    }

}

// initIndexTemplate parses index.html from the NPM build directory and
// updates the relative URIs to point to the fake ones in
// ss.Config.Routes.LandingPage.
func (ss *SkyhookServer) initIndexTemplate() {
    var indexTemplate string
    if b, err := FileSpa.ReadFile("web_apps/file/build/index.html"); err != nil {
        panic("failed to read index.html file; has it been moved?")
    } else {
        indexTemplate = string(b)
        for fName, fakePath := range ss.Config.Routes.LandingPage {
            origPath := path.Join("/landing/", fName)
            indexTemplate = strings.Replace(indexTemplate, origPath, fakePath, -1)
            origPath = path.Join("/landing/static/js/", fName)
            indexTemplate = strings.Replace(indexTemplate, origPath, fakePath, -1)
            origPath = path.Join("/landing/static/css/", fName)
            indexTemplate = strings.Replace(indexTemplate, origPath, fakePath, -1)
        }
        indexTemplate = strings.Replace(
            indexTemplate,
            "\"root\"",
            fmt.Sprintf("\"%s\"", ss.Config.EncryptedLoader.RootElementId),
            -1)
    }
    ss.indexContent = []byte(indexTemplate)
}

// ServeLoaderFile serves encrypted loader files from disk.
func (ss *SkyhookServer) ServeLoaderFile(c *gin.Context) {

    fakePath := c.FullPath()
    if fakePath == "" {
        panic("unknown path requested")
    }

    var content []byte
    var mime string

    switch fakePath {
    case ss.Config.Routes.EncryptedLoader.Js:
        //=====================================
        // RETURN A RANDOMIZED ENCRYPTED LOADER
        //=====================================
        mime = "text/javascript"
        content = ss.GenEncryptedLoader()
    case ss.Config.Routes.EncryptedLoader.Html, ss.Config.Routes.EncryptedLoader.AutoHtml:
        //==========================
        // HANDLE HTML LANDING PAGES
        //==========================
        mime = "text/html"
        vals := loaderHtmlValues{}
        if fakePath == ss.Config.Routes.EncryptedLoader.AutoHtml {
            //================
            // EMBED JS LOADER
            //================
            vals.Script = string(ss.GenEncryptedLoader())
        }

        // Render the template
        buff := make([]byte, 0)
        w := bytes.NewBuffer(buff)
        if err := htmlLoaderTemp.Execute(w, vals); err != nil {
            panic(err)
        }

        // Extract the content
        content = make([]byte, w.Len())
        if _, err := w.Read(content); err != nil {
            panic(err)
        }
    }

    c.Data(http.StatusOK, mime, content)
}

func (ss *SkyhookServer) GenEncryptedLoader() []byte {
    randVars := randOverOneMany(5, 20)
    content := make([]byte, 0)
    lTemp := loaderTemplateValues{
        RootId: ss.Config.EncryptedLoader.RootElementId,
        //RootId:            rando.AnyAsciiString(20, false, ""),
        Stage0KeyVar: rando.AnyAsciiString(uint32(randVars[0]), false, ""),
        PayVar:       rando.AnyAsciiString(uint32(randVars[1]), false, ""),
        BuffVar:      rando.AnyAsciiString(uint32(randVars[2]), false, ""),
        Stage1KeyVar: rando.AnyAsciiString(uint32(randVars[3]), false, ""),
        QueryString:  fmt.Sprintf("%s=%s", ss.Config.EncryptedLoader.UriParam, rando.AnyString(uint32(randVars[4]), "")),
        Urls:         jsLoaderTempUrls,
    }

    buff := make([]byte, 0)
    w := bytes.NewBuffer(buff)
    if err := jsLoaderTemplS1.Execute(w, lTemp); err != nil {
        panic(fmt.Sprintf("Failed to render stage1 landing template: %v", err))
    }
    content, _ = io.ReadAll(w)
    content = doMinify(content)

    //===============================
    // SUBSTITUTION CIPHER THE LOADER
    //===============================

    alt := newAsciiSubCipher(true)
    key := make(map[string]string)
    ascii.SubCrypt(content, &alt, &key)
    lTemp.Pay = string(obfuscate.Base64Encode(content))

    //==========================
    // APPLY THE LOADER TEMPLATE
    //==========================

    // JSON key that will be embedded to decrypt the loader
    jKey, _ := json.Marshal(key)
    lTemp.Stage0Key = string(jKey)
    buff = make([]byte, 0)
    w = bytes.NewBuffer(buff)
    if err := jsLoaderTemplS0.Execute(w, lTemp); err != nil {
        panic(fmt.Sprintf("Failed to render stage0 landing template: %v", err))
    }

    content, _ = io.ReadAll(w)
    content = doMinify(content)

    return content
}

// ServeLandingFile serves file content that was updated
// with paths supplied via configuration file during initialization.
func (ss *SkyhookServer) ServeLandingFile(c *gin.Context) {

    if lF, err := ss.LandingFiles.Get(c.FullPath()); err != nil {

        //================================
        // FAILED TO RETRIEVE LANDING FILE
        //================================

        c.AbortWithStatus(http.StatusNotFound)

    } else {

        //============================================
        // NEGOTIATE PLAIN OR CIPHER TEXT FILE CONTENT
        //============================================

        var mime string
        var content []byte
        if c.Query(ss.LandingFileEncryption.UriParam) != "" {
            mime = "text"
            content = lF.Alt.Content
        } else {
            content = lF.Real.Content
            mime = parseFileMimetype(lF.Real.Name)
        }

        //==============
        // WRITE CONTENT
        //==============

        c.Data(http.StatusOK, mime, content)
    }

}
