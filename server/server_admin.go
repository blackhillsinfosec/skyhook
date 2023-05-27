package server

import (
    "context"
    "embed"
    "errors"
    "fmt"
    jwt "github.com/appleboy/gin-jwt/v2"
    obfs "github.com/blackhillsinfosec/skyhook-obfuscation"
    structs "github.com/blackhillsinfosec/skyhook/api_structs"
    "github.com/blackhillsinfosec/skyhook/config"
    "github.com/blackhillsinfosec/skyhook/log"
    mw "github.com/blackhillsinfosec/skyhook/server/middleware"
    fsUtil "github.com/blackhillsinfosec/skyhook/util/fs"
    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
    "golang.org/x/exp/slices"
    "gopkg.in/yaml.v3"
    "net/http"
    "os"
    "path"
    "strings"
    "sync"
    "time"
)

var (
    //go:embed web_apps/admin/build
    adminSpa embed.FS
)

// AdminServer hosts the administrative web application, which
// exists as a method of managing the underlying configuration
// file for the file server.
//
// While this can be Internet-exposed, it's recommended
// that it listens only on 127.0.0.1.
//
// JWT authentication is used.
type AdminServer struct {
    Config          *config.AdminServerOptions
    Tls             *config.ManualTlsOptions
    Users           *[]config.Credential
    ObfuscatorChain *[]obfs.Obfuscator
    // Kill is a channel used to tell AdminServer that it
    // should die.
    Kill chan uint8
    // ConfigFileMu is used to control RW operations to
    // the config file.
    ConfigFileMu sync.Mutex
    // ConfigFile points to the config file on disk and is
    // used during RW operations.
    ConfigFile *string
    // Global points to the global configuration and is
    // used during RW operations to the config file.
    Global               *config.SkyhookConfig
    EncryptedJsGenerator func() []byte
}

// Run runs the admin server.
func (as *AdminServer) Run() (err error) {

    eng := gin.Default()
    //=========================
    // CONFIGURE JWT MIDDLEWARE
    //=========================

    // Reference: https://github.com/appleboy/gin-jwt
    var authMiddleWare *jwt.GinJWTMiddleware
    if authMiddleWare, err = jwt.New(&jwt.GinJWTMiddleware{
        Timeout:       7 * (24 * time.Hour),
        MaxRefresh:    7 * (24 * time.Hour),
        IdentityKey:   as.Global.Auth.Jwt.FieldKeys.Username,
        Realm:         as.Global.Auth.Jwt.Realm,
        Key:           []byte(as.Global.Auth.Jwt.SigningKey),
        TokenLookup:   fmt.Sprintf("header: %s", as.Global.Auth.Header.Name),
        TokenHeadName: as.Global.Auth.Header.Scheme,
        Authorizator:  mw.JwtIsCredAdmin,
        Unauthorized:  mw.JwtIsUnauthorized,
        Authenticator: mw.JwtLoginHandler(&as.Global.Users, true),
        IdentityHandler: mw.JwtIdentityHandler(
            &as.Global.Auth.Jwt.FieldKeys.Username,
            &as.Global.Auth.Jwt.FieldKeys.Admin),
        PayloadFunc: mw.JwtPayloadFunc(as.Global)}); err != nil {

        log.ERR.Printf("Failed to initialize JWT auth: %v", err)
        return err

    }

    if err = authMiddleWare.MiddlewareInit(); err != nil {
        log.ERR.Printf("Failed to initialize Gin JWT middleware: %v", err)
        return err
    }

    //===============
    // CONFIGURE CORS
    //===============

    var corsFqdns []string
    corsFqdns = append(corsFqdns, "https://"+as.Config.Socket())
    corsFqdns = append(corsFqdns, as.Config.AddtlCorsUrls...)
    cors := cors.New(cors.Config{
        //AllowAllOrigins: true,
        AllowWildcard:    true,
        AllowOrigins:     corsFqdns,
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
        AllowHeaders:     []string{"*"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    })
    eng.Use(cors)

    if err = eng.SetTrustedProxies(nil); err != nil {
        log.ERR.Printf("Failed set trusted proxies on admin server: %v", err)
        panic(err)
    }

    eng.NoRoute(authMiddleWare.MiddlewareFunc(), func(ctx *gin.Context) {
        //claims := jwt.ExtractClaims(ctx)
        ctx.JSON(http.StatusNotFound, gin.H{})
    })

    //=================
    // ANONYMOUS ROUTES
    //=================

    proxFs := fsUtil.HttpFsProxy{
        Dst: http.FS(adminSpa),
        NameFunc: func(name string) string {
            return path.Join("web_apps/admin/build", name)
        },
    }

    eng.StaticFS("/landing", proxFs)
    eng.GET("/", func(c *gin.Context) {
        c.FileFromFS("/", proxFs)
    })

    eng.GET("/ping", as.PingHandler)
    eng.POST("/login", authMiddleWare.LoginHandler)
    eng.GET("/login", authMiddleWare.RefreshHandler)
    eng.POST("/logout", authMiddleWare.LogoutHandler)

    //=====================
    // AUTHENTICATED ROUTES
    //=====================

    auth := eng.Group("/admin")
    auth.Use(authMiddleWare.MiddlewareFunc())
    {
        auth.GET("/ping", as.PingHandler)
        auth.GET("/links", as.GetLinks)

        auth.GET("/users", as.GetUsers)
        auth.PUT("/users", as.SaveUsers)

        auth.GET("/obfs", as.ListObfuscators)
        auth.GET("/obfs/config", as.GetObfuscators)
        auth.PUT("/obfs/config", as.SaveObfuscators)

        auth.GET("/advanced", as.GetAdvancedConfig)
        auth.GET("/landing", as.GetFileServerLandingUri)
        auth.GET("/js", as.GetEncryptedJs)
    }

    //=================
    // START THE SERVER
    //=================

    srv := http.Server{
        Addr:    as.Config.Socket(),
        Handler: eng,
    }

    go func() {
        err = srv.ListenAndServeTLS(as.Tls.CertPath, as.Tls.KeyPath)
        if err != nil {
            as.Kill <- 2
        }
    }()

    out := <-as.Kill
    if out != 2 {
        _, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer func() {
            cancel()
        }()
        log.WARN.Printf("Shutting down admin server")
    } else if err != nil {
        log.ERR.Printf("Failed to start admin server: %v", err)
    }

    return err
}

func (as *AdminServer) GetLinks(c *gin.Context) {

    allLinks := structs.LinksResponse{}

    fs := as.Global.FileServer

    for _, fqdn := range fs.LinkFqdns {

        links := structs.Links{}
        base := "https://" + fqdn

        for r, f := range fs.Routes.LandingPage {
            if r == "index.html" {
                links.Standard.Html = base + f
            }
        }

        links.Encrypted.Js = base + fs.Routes.EncryptedLoader.Js
        links.Encrypted.Html = fmt.Sprintf("%s%s#%s", base, fs.Routes.EncryptedLoader.Html, fs.EncryptedLoader.Key)
        links.Encrypted.AutoloadHtml = fmt.Sprintf("%s%s#%s", base, fs.Routes.EncryptedLoader.AutoHtml, fs.EncryptedLoader.Key)

        allLinks[fqdn] = links
    }

    c.JSON(http.StatusOK, allLinks)

}

func (as *AdminServer) GetEncryptedJs(c *gin.Context) {
    c.JSON(http.StatusOK, structs.EncryptedJsResponse{
        BaseResponse: structs.BaseResponse{
            Success: true,
            Message: "Randomized encrypted JS returned.",
        },
        EncryptedJs: string(as.EncryptedJsGenerator()),
    })
}

func (as *AdminServer) writeGlobalConfig(backup bool) (err error) {
    as.ConfigFileMu.Lock()
    defer as.ConfigFileMu.Unlock()

    //===========================
    // MARSHAL THE CURRENT CONFIG
    //===========================

    var buff []byte
    if buff, err = yaml.Marshal(as.Global); err != nil {
        return errors.New(
            fmt.Sprintf("failed to marshal config: %v", err))
    }

    if backup {

        //===========================
        // BACK UP CONFIGURATION FILE
        //===========================

        var cBuff []byte
        if cBuff, err = os.ReadFile(*as.ConfigFile); err == nil {

            backFile := fmt.Sprintf("%s.%s.%v.%s",
                strings.Trim(*as.ConfigFile, ".yml"),
                "backup",
                time.Now().Unix(),
                "yml")

            if err = os.WriteFile(backFile, cBuff, 0600); err != nil {

                //=======================
                // FAILED TO WRITE BACKUP
                //=======================

                return errors.New(fmt.Sprintf("failed to write backup config: %v", err))
            }

        } else {

            //===================================
            // FAILED TO READ CURRENT CONFIG FILE
            //===================================

            return errors.New(fmt.Sprintf("failed to read config file: %v", err))

        } // Config file read

    } // Backup

    //=========================
    // SAVE CURRENT CONFIG FILE
    //=========================

    if err = os.WriteFile(*as.ConfigFile, buff, 0600); err != nil {
        err = errors.New(fmt.Sprintf("failed to write config file: %v", err))
    }

    return err
}

//==================
// ENDPOINT HANDLERS
//==================

// WriteGlobalConfigFile writes the global config file to disk.
func (as *AdminServer) WriteGlobalConfigFile(c *gin.Context) {
    go func() {
        log.WARN.Print("Writing global config file")
        as.writeGlobalConfig(false)
    }()
    resp := structs.BaseSuccessResponse()
    resp.Message = "Config file is being written to disk."
    c.JSON(http.StatusOK, resp)
}

// GetUsers returns all users from the config file.
//
// Responses:
// - CredListResponse.
func (as *AdminServer) GetUsers(c *gin.Context) {
    c.JSON(http.StatusOK, structs.CredListResponse{
        BaseResponse: structs.BaseSuccessResponse(),
        CredList:     structs.CredList{Users: *as.Users},
        //ConfigUri:    as.Global.FileServer.Routes.Api.Config,
    })
}

// SaveUsers saves users from the payload to the config file.
//
// Responses:
//
// - When successful or standard errors occur, structs.BaseResponse.
// - When duplicate usernames are detected, DuplicateUsernameResponse
func (as *AdminServer) SaveUsers(c *gin.Context) {

    //=========================
    // PARSE SUPPLIED USER LIST
    //=========================

    payload := structs.CredList{}
    if err := c.BindJSON(&payload); err != nil {
        c.JSON(http.StatusBadRequest, structs.BaseResponse{Message: "Poorly formatted request payload."})
        return
    }

    //===============================
    // ENSURE CURRENT USER IS PRESENT
    //===============================

    var found bool
    creds := mw.JwtExtractCtxClaims(
        as.Global.Auth.Jwt.FieldKeys.Username,
        as.Global.Auth.Jwt.FieldKeys.Admin, c).(*config.Credential)
    for _, cred := range payload.Users {
        if cred.Username == creds.Username {
            found = true
            if !cred.IsAdmin {
                c.JSON(http.StatusBadRequest,
                    structs.BaseResponse{Message: "Admin cannot remove admin access from their own account"})
                return
            }
            break
        }
    }

    if !found {
        c.JSON(http.StatusBadRequest, structs.BaseResponse{Message: "Current admin user not found in user list"})
        return
    }

    //==========================
    // CHECK FOR DUPLICATE USERS
    //==========================

    var dupes, known []string
    for _, cred := range payload.Users {
        if slices.Index(known, cred.Username) > 0 {
            dupes = append(dupes, cred.Username)
        }
    }

    if len(dupes) > 0 {
        c.JSON(http.StatusBadRequest, structs.DuplicateUsernameResponse{
            BaseResponse: structs.BaseResponse{
                Message: "Duplicate usernames supplied",
            },
            Usernames: nil,
        })
        return
    }

    //==============================================
    // CHECKS PASSED -- UPDATE CURRENT LIST OF USERS
    //==============================================

    *as.Users = payload.Users

    go func() {
        as.writeGlobalConfig(false)
    }()
    c.JSON(http.StatusOK, structs.BaseSuccessResponse())
}

// ListObfuscators returns a list of all supported obfuscators.
//
// Responses:
//
// - ListObfuscatorsResponse
func (as *AdminServer) ListObfuscators(c *gin.Context) {

    b := structs.ListObfuscatorsResponse{
        BaseResponse: structs.BaseResponse{
            Success: true,
            Message: "Listing all supported obfuscators.",
        },
        Obfuscators: structs.Obfuscators,
    }
    c.JSON(http.StatusOK, b)
}

// GetObfuscators returns all currently configured obfuscators.
//
// Responses:
//
// - GetObfuscatorsResponse
func (as *AdminServer) GetObfuscators(c *gin.Context) {

    //===================================================
    // INTROSPECT THE OBFUSCATOR CHAIN INTO A JSON OBJECT
    //===================================================

    obfs := obfs.UnparseObfuscators(as.ObfuscatorChain)

    c.JSON(http.StatusOK, structs.GetObfuscatorsResponse{
        BaseResponse: structs.BaseResponse{
            Success: true,
            Message: "Listing currently configured obfuscators.",
        },
        Obfuscators: *obfs,
    })
}

// PingHandler is for API ping requests.
//
// Response Structure: PingResponse
func (as *AdminServer) PingHandler(c *gin.Context) {
    c.JSON(http.StatusOK, structs.PingResponse{
        BaseResponse: structs.BaseSuccessResponse(),
        Response:     "pong"})
}

// SaveObfuscators saves the supplied obfuscators configuration.
//
// Responses:
//
// - Upon success, SaveObfuscatorsResponse.
// - Upon error, structs.BaseResponse.
func (as *AdminServer) SaveObfuscators(c *gin.Context) {

    var msg string
    p := structs.ObfuscatorsPayload{}
    if err := c.BindJSON(&p); err != nil {
        msg = fmt.Sprintf("Failed to parse JSON payload while saving obfuscators: %v", err)
        log.INFO.Print(msg)
        c.JSON(http.StatusBadRequest, structs.BaseResponse{Message: msg})
        return
    }

    //====================================
    // PARSE THE OBFUSCATOR CONFIGURATIONS
    //====================================

    var unparsed structs.ObfuscatorsPayload
    if latest, failures := obfs.ParseObfuscators(&p.Obfuscators); len(failures) > 0 {
        msg = fmt.Sprintf("Failed to parse obfuscators: %s", strings.Join(failures, ", "))
        log.ERR.Print(msg)
    } else {

        if len(*latest) == 0 {

            msg = "Zero (0) obfuscators have been configured. No obfuscation will occur."
            log.WARN.Print(msg)

        } else {

            //=============================
            // LOG LATEST OBFUSCATION CHAIN
            //=============================

            unparsed = structs.ObfuscatorsPayload{
                Obfuscators: *obfs.UnparseObfuscators(latest),
            }

            var strChain string
            if bChain, err := yaml.Marshal(unparsed); err != nil {
                log.ERR.Printf("Failed to convert obfs chain into YML for logging: %v", err)
            } else {
                strChain = string(bChain)
            }

            msg = fmt.Sprintf("Setting the current obfuscation chain:\n\n%s\n\n", strChain)

        }

        //=========================
        // SET NEW OBFUSCATOR CHAIN
        //=========================

        as.Global.FileServer.Obfuscators = p.Obfuscators
        *as.ObfuscatorChain = *latest

    }

    log.WARN.Print(msg)

    go func() {
        as.writeGlobalConfig(false)
    }()

    c.JSON(http.StatusOK, structs.SaveObfuscatorsResponse{
        BaseResponse: structs.BaseResponse{
            Success: true,
            Message: msg,
        },
        ObfuscatorsPayload: unparsed,
    })

}

func (as *AdminServer) GetAdvancedConfig(c *gin.Context) {
    c.JSON(http.StatusOK, structs.AdvancedConfigResponse{
        BaseResponse: structs.BaseResponse{
            Success: true,
            Message: "Current advanced configurations returned.",
        },
        ApiRoutes:   as.Global.FileServer.Routes.Api,
        Obfuscators: *obfs.UnparseObfuscators(as.ObfuscatorChain),
        AuthConfig: config.SafeAuthOptions{
            Header: as.Global.Auth.Header,
            Jwt:    as.Global.Auth.Jwt.SafeJwtOptions,
        },
    })
}

func (as *AdminServer) GetFileServerLandingUri(c *gin.Context) {
    c.JSON(http.StatusOK, structs.BaseResponse{
        Success: true,
        Message: as.Global.FileServer.Routes.LandingPage["/web_apps/file/build/index.html"],
    })
}
