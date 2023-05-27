package cmd

import (
    "fmt"
    obfs "github.com/blackhillsinfosec/skyhook-obfuscation"
    "github.com/blackhillsinfosec/skyhook/config"
    "github.com/blackhillsinfosec/skyhook/log"
    "github.com/blackhillsinfosec/skyhook/server"
    "github.com/blackhillsinfosec/skyhook/server/upload"
    "github.com/fsnotify/fsnotify"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/impostorkeanu/go-commoners/rando"
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "golang.org/x/exp/maps"
    "golang.org/x/exp/slices"
    "golang.org/x/sync/semaphore"
    "gopkg.in/yaml.v3"
    "io/fs"
    "os"
    "path"
    "strings"
    "sync"
    "time"
)

var (
    conSem = semaphore.NewWeighted(1)

    //===============
    // COBRA COMMANDS
    //===============

    serverCmd = &cobra.Command{
        Use:     "server",
        Aliases: []string{"s", "srv"},
        Short:   "Configure and run Skyhook servers.",
    }
    runServersCmd = &cobra.Command{
        Use:     "run",
        Aliases: []string{"r", "start"},
        Short:   "Run the Skyhook servers.",
        RunE:    runSkyhook,
    }
    genServerConfigCmd = &cobra.Command{
        Use:     "generate-config",
        Aliases: genAliases,
        Short:   "Generate a Skyhook server configuration file.",
        Run:     genSkyhookConfig,
    }

    randApiPathsLen = uint8(0)

    //================
    // OTHER VARIABLES
    //================

    // gConfig is the global SkyhookConfig.
    gConfig = &config.SkyhookConfig{}

    // fsConfig is a reference to the standard file server
    // configuration.
    fsConfig *config.FileServerOptions

    // asConfig is a reference to the admin server configuration.
    asConfig *config.AdminServerOptions

    // fServer is the file server config.
    fServer server.SkyhookServer

    // aServer is the admin server config.
    aServer server.AdminServer

    _viper = viper.NewWithOptions(viper.KeyDelimiter("|"))
)

func init() {
    gin.SetMode(gin.ReleaseMode)
    RootCmd.AddCommand(serverCmd)
    serverCmd.AddCommand(genServerConfigCmd, runServersCmd)
    runServersCmd.Flags().StringVarP(&configFile, "config-file", "c",
        "", "Configuration file.")
    runServersCmd.MarkFlagRequired("config-file")
    runServersCmd.Flags().Bool("no-admin-server", false,
        "Run only the file server. Make any updates by updating the config file.")

    genServerConfigCmd.Flags().Uint8VarP(&randApiPathsLen, "rand-api-path-min-len", "r",
        randApiPathsLen, "Randomize API paths up to the supplied length. Supplying a non-zero value enables this functionality.")
}

func runSkyhook(cmd *cobra.Command, args []string) (err error) {

    //===========================
    // APPLY VIPER CONFIGURATIONS
    //===========================

    log.INFO.Printf("Using config file at %s", configFile)
    if _, err = os.Stat(configFile); err != nil {
        return err
    }

    _viper.SetConfigType("yaml")
    _viper.SetConfigFile(configFile)

    if err = _viper.ReadInConfig(); err != nil {
        log.ERR.Printf("Failed to read config file: %v", err)
        return err
    }

    if err = _viper.UnmarshalExact(gConfig); err != nil {
        log.ERR.Printf("Failed to unmarshal config file (poorly formatted YAML?): %v", err)
        return err
    }

    //================================
    // VALIDATE CONFIG AND RUN SERVERS
    //================================

    if err = gConfig.Validate(); err != nil {
        log.ERR.Printf("Configuration failed validation")
        return err
    }

    noRunAdmin, _ := cmd.Flags().GetBool("no-admin-server")
    if noRunAdmin {
        err = runWithoutAdmin()
        fServer.Run(false)
    } else {
        err = runWithAdmin()
    }

    return err
}

func runWithoutAdmin() (err error) {
    _viper.OnConfigChange(func(e fsnotify.Event) {
        log.INFO.Printf("Config file changed: %s", e.Name)
        log.INFO.Printf("Reloading server config")
        if err := loadConfig(); err != nil {
            log.WARN.Printf("Failed to reload configuration file: %v", err)
            log.INFO.Printf("Preserving previously loaded config file")
        }
    })
    _viper.WatchConfig()
    return loadConfig()
}

func loadConfig() (err error) {

    //=================================
    // LOAD AND VALIDATE THE NEW CONFIG
    //=================================

    if err = conSem.Acquire(ctx{}, 1); err != nil {
        log.ERR.Println("Failed to acquire semaphore to reload config file")
        return err
    }
    defer conSem.Release(1)

    if err = _viper.ReadInConfig(); err != nil {
        log.ERR.Printf("Failed to read config file: %v", err)
        return err
    }

    buff := config.SkyhookConfig{}

    if err = _viper.UnmarshalExact(&buff); err != nil {
        log.ERR.Printf("Failed to unmarshal config file (poorly formatted YAML?): %v", err)
        return err
    }

    if err = buff.Validate(); err != nil {
        log.ERR.Printf("New configuration failed validation: %v", err)
        return err
    }

    //=======================
    // NOTIFY OF PORT CHANGES
    //=======================

    restartMsg := "Stop and restart the server to implement this change"

    if fsConfig != nil && buff.FileServer.Port != fsConfig.Port {
        log.WARN.Printf("File server port changed to: %d", buff.FileServer.Port)
        log.WARN.Println(restartMsg)
    }

    if fsConfig != nil && buff.FileServer.Interface != fsConfig.Interface {
        log.WARN.Printf("File server interface changed to: %v", buff.FileServer.Interface)
        log.WARN.Println(restartMsg)
    }

    if len(buff.Users) == 0 {
        log.WARN.Print("Zero (0) users have been configured.")
        log.ERR.Print("Configure a user to access the file server")
    }

    //========================
    // UPDATE TO LATEST CONFIG
    //========================

    gConfig = &buff
    fsConfig = &buff.FileServer
    fServer.Config = fsConfig
    fServer.Tls = &gConfig.Tls
    fServer.Users = &gConfig.Users

    obfsChain, failures := obfs.ParseObfuscators(&fsConfig.Obfuscators)
    if len(failures) > 0 {
        log.ERR.Printf("Failed to parse obfuscator(s): %s", strings.Join(failures, ", "))
    }

    if len(fsConfig.Obfuscators) == 0 {
        log.WARN.Println("Zero (0) obfuscators have been configured")
        log.WARN.Println("File obfuscation will be disabled")
    } else {
        var algos []string
        for _, o := range fsConfig.Obfuscators {
            algos = append(algos, o.Algo)
        }
        log.INFO.Printf("Current obfuscation pipeline: %s", strings.Join(algos, "|"))
    }
    fServer.ObfuscatorChain = obfsChain

    return err
}

// runWithAdmin runs both the file and admin server, allowing for a
// secondary admin application to manage the file server.
func runWithAdmin() (err error) {

    //======================
    // START THE FILE SERVER
    //======================

    // Get a reference to the file server config.
    fsConfig = &gConfig.FileServer
    asConfig = &gConfig.AdminServer

    // Parse the slice of obfuscators.
    obfsChain, failures := obfs.ParseObfuscators(&fsConfig.Obfuscators)
    if len(failures) > 0 {
        log.ERR.Printf("Failed to parse obfuscator(s): %s", strings.Join(failures, ", "))
    }

    log.INFO.Printf("File server started on: %s", fsConfig.Socket())
    log.INFO.Printf("Admin server started on: %s", asConfig.Socket())
    log.INFO.Printf("Blocking until shutdown request")

    // Initialize the server.
    upMgr, err := upload.NewManager(&fsConfig.UploadOptions.RegistrantsFile, &fsConfig.UploadOptions.MaxUploadDuration)
    if err != nil {
        log.ERR.Printf("Failed to initialize upload manager: %v", err)
        panic(err)
    }
    fServer = server.SkyhookServer{
        Config:          fsConfig,
        Tls:             &gConfig.Tls,
        Users:           &gConfig.Users,
        ObfuscatorChain: obfsChain,
        UploadManager:   &upMgr,
        Global:          gConfig,
    }

    fServer.Run(true)

    //=======================
    // START THE ADMIN SERVER
    //=======================
    //
    // This intentionally blocks.

    aServer = server.AdminServer{
        Config:               asConfig,
        Users:                &gConfig.Users,
        Tls:                  &gConfig.Tls,
        ObfuscatorChain:      obfsChain,
        Kill:                 make(chan uint8, 1),
        ConfigFileMu:         sync.Mutex{},
        ConfigFile:           &configFile,
        Global:               gConfig,
        EncryptedJsGenerator: fServer.GenEncryptedLoader,
    }

    err = aServer.Run()

    gConfig.FileServer.Obfuscators = *obfs.UnparseObfuscators(aServer.ObfuscatorChain)

    // This error is expected.
    if strings.Contains(strings.ToLower(err.Error()), "http: server closed") {
        err = nil
    }

    //======================================
    // WRITE THE CURRENT CONFIG FILE TO DISK
    //======================================

    log.INFO.Print("Attempting to save current config file")
    if buff, err := yaml.Marshal(&gConfig); err != nil {
        log.ERR.Printf("Failed to marshal config file for writing: %v", err)
    } else {
        if cBuff, err := os.ReadFile(configFile); err == nil {

            //========================
            // SAVE BACKUP CONFIG FILE
            //========================

            backFile := fmt.Sprintf("%s.%s.%v.%s",
                strings.Trim(configFile, ".yml"),
                "backup",
                time.Now().Unix(),
                "yml")
            log.INFO.Printf("Writing backup config file: %s", backFile)
            if err := os.WriteFile(backFile, cBuff, 0600); err != nil {
                log.ERR.Printf("Failed to write backup config: %v", err)
            } else {

                //=========================
                // SAVE CURRENT CONFIG FILE
                //=========================

                log.INFO.Printf("Writing config file: %v", configFile)
                if err := os.WriteFile(configFile, buff, 0600); err != nil {
                    log.ERR.Printf("Failed to write config file: %v", err)
                }
            }

        } else {

            //===================================
            // FAILED TO READ CURRENT CONFIG FILE
            //===================================

            log.ERR.Printf("Failed to read config file: %v", err)

        }
    }

    return err

}

func handleDir(dEnt fs.DirEntry, root string, routes *map[string]string) {
    infos, _ := server.FileSpa.ReadDir(path.Join(root, dEnt.Name()))
    for _, idEnt := range infos {
        if idEnt.IsDir() {
            handleDir(idEnt, path.Join(root, dEnt.Name()), routes)
        } else {
            handleFile(idEnt, path.Join(root, dEnt.Name()), routes)
        }
    }
}

func handleFile(fEnt fs.DirEntry, root string, routes *map[string]string) {

    realPath := path.Join("/", root, fEnt.Name())
    (*routes)[parseFile(realPath)] = realPath

    var fakepath string
    if randApiPathsLen > 0 {

        //====================
        // RANDOMIZE THE PATHS
        //====================

        s := strings.Split(realPath, ".")
        var ext string
        if len(s) > 1 {
            ext = "." + s[len(s)-1]
        }

        for fakepath == "" || fakepath == realPath || slices.Contains(maps.Keys(*routes), fakepath) {
            fakepath = "/" + rando.AnyString(uint32(randApiPathsLen), "/") + ext
        }

    } else {

        //===================
        // STANDARD FILE PATH
        //===================

        _, fakepath = path.Split(realPath)
        if fakepath == "index.html" {
            fakepath = path.Join("/", fakepath)
        } else {
            fakepath = path.Join("/landing", fakepath)
        }

    }

    (*routes)[parseFile(realPath)] = fakepath

}

func parseFile(s string) string {
    _, s = path.Split(s)
    return s
}

func genSkyhookConfig(cmd *cobra.Command, args []string) {

    //=========================
    // CONFIGURE LANDING ROUTES
    //=========================
    // Landing apiRoutes are apiRoutes pointing to files that support
    // the landing page, e.g., index.html and favicon.ico.
    //
    // This will provide an index mapping randomized paths back
    // to the embedded filesystem's true path.

    landingRoutes := make(map[string]string)

    infos, _ := server.FileSpa.ReadDir("web_apps/file/build")
    for _, dEnt := range infos {
        if strings.HasSuffix(dEnt.Name(), ".bak") {
            continue
        } else if dEnt.IsDir() {
            handleDir(dEnt, "web_apps/file/build", &landingRoutes)
        } else {
            handleFile(dEnt, "web_apps/file/build", &landingRoutes)
        }
    }

    // Reconcile map files to have same names (non-map-file is authoritative)
    for _, realPath := range maps.Keys(landingRoutes) {
        if !strings.HasSuffix(realPath, ".map") {
            continue
        }
        landingRoutes[parseFile(realPath)] = landingRoutes[strings.TrimSuffix(realPath, ".map")] + ".map"
    }

    //====================
    // CONFIGURE API PATHS
    //====================

    apiRoutes := map[string]string{
        //"login":    "/login",
        "logout":   "/logout",
        "download": "/files",
        "upload":   "/upload",
        "config":   "/config",
    }

    if randApiPathsLen > 0 {

        log.WARN.Printf("Randomizing API apiRoutes (%v minimum length)", randApiPathsLen)
        for k, v := range apiRoutes {
            orig := v[:]
            for v == orig || slices.Contains(maps.Values(apiRoutes), v) {
                //for v == orig || slices.Contains(maps.Values(apiRoutes), v) || slices.Contains(maps.Keys(landingRoutes), v) {
                v = "/" + rando.AnyString(uint32(randApiPathsLen), "/")
            }
            apiRoutes[k] = v
        }

    } else {

        log.WARN.Printf("Using default API apiRoutes")

    }

    //=======================
    // CONFIGURE LOADER PATHS
    //=======================

    var loaderRoutes config.EncryptedInterfaceLoaderRoutes
    if randApiPathsLen > 0 {
        loaderRoutes = config.EncryptedInterfaceLoaderRoutes{
            AutoHtml: "/" + rando.AnyString(uint32(randApiPathsLen), "/") + ".html",
            Html:     "/" + rando.AnyString(uint32(randApiPathsLen), "/") + ".html",
            Js:       "/" + rando.AnyString(uint32(randApiPathsLen), "/") + ".js",
        }
    } else {
        loaderRoutes = config.EncryptedInterfaceLoaderRoutes{
            AutoHtml: "/enc/loader_auto.html",
            //AutoJs:   "/enc/loader_auto.js",
            Html: "/enc/loader.html",
            Js:   "/enc/loader.js",
        }
    }

    //===============================
    // CONFIGURE & DUMP A CONFIG FILE
    //===============================

    var configBytes []byte
    configBytes, _ = yaml.Marshal(&config.SkyhookConfig{
        Tls: config.ManualTlsOptions{
            CertPath: "",
            KeyPath:  "",
        },
        FileServer: config.FileServerOptions{
            LinkFqdns: []string{"your.fqdn.here"},
            EncryptedLoader: config.LandingFileEncryptionOptions{
                Key:           rando.AnyString(uint32(10), "-"),
                UriParam:      rando.AnyString(uint32(10), ""),
                RootElementId: rando.AnyString(uint32(10), "-"),
            },
            ServerOptions: config.ServerOptions{
                AddtlCorsUrls: []string{"*"},
                Interface:     "eth0",
                Port:          443,
            },
            RootDir: "webroot",
            UploadOptions: config.FileServerUploadOptions{
                RegistrantsFile:   "skyhook_upload_registrants.json",
                MaxUploadDuration: 24,
            },
            RangeHeaderOptions: config.FileServerRangeHeaderOptions{
                Name:        rando.AnyString(uint32(20), ""),
                RangePrefix: "bytes",
            },
            Routes: config.FileServerRouteOptions{
                Api: config.FileServerApiRoutes{
                    Logout:          apiRoutes["logout"],
                    Download:        apiRoutes["download"],
                    Upload:          apiRoutes["upload"],
                    OperatingConfig: apiRoutes["config"],
                },
                LandingPage:     landingRoutes,
                EncryptedLoader: loaderRoutes,
            },
        },
        AdminServer: config.AdminServerOptions{
            ServerOptions: config.ServerOptions{
                AddtlCorsUrls: []string{"*"},
                Interface:     "lo",
                Port:          65535,
            },
        },
        Auth: config.AuthOptions{
            Header: config.AdminAuthHeaderOptions{
                Name:   "Authorization",
                Scheme: "Bearer",
            },
            Jwt: config.JwtOptions{
                SafeJwtOptions: config.SafeJwtOptions{
                    Realm: "sh",
                    FieldKeys: config.JwtFieldKeys{
                        Username: "id",
                        Admin:    "ad",
                        Config:   "c",
                    },
                },
                SigningKey: uuid.New().String(),
            },
        },
        Users: []config.Credential{
            {
                Username: rando.AnyString(uint32(7), "-"),
                Password: rando.AnyString(uint32(20), " "),
                IsAdmin:  true,
                Token:    rando.AnyAsciiString(uint32(10), true, ""),
            },
            {
                Username: rando.AnyString(uint32(7), "-"),
                Password: rando.AnyString(uint32(20), " "),
                IsAdmin:  false,
                Token:    rando.AnyAsciiString(uint32(10), true, ""),
            }}})
    fmt.Println(string(configBytes))
}
