package cmd

import (
    "context"
    "crypto/tls"
    "fmt"
    "github.com/blackhillsinfosec/skyhook/config"
    "github.com/blackhillsinfosec/skyhook/log"
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "golang.org/x/crypto/acme/autocert"
    "gopkg.in/yaml.v3"
    "net/http"
    "os"
    "time"
)

var (
    acmeCmd = &cobra.Command{
        Use:     "acme",
        Aliases: []string{"a"},
        Short:   "Get certificates via Lets Encrypt.",
    }
    runAcmeCmd = &cobra.Command{
        Use:     "run",
        Aliases: []string{"run"},
        Short:   "Get a certificate via Lets Encrypt",
        RunE:    runAcmeServer,
    }
    genAcmeConfigCmd = &cobra.Command{
        Use:     "generate-config",
        Aliases: genAliases,
        Short: "Generate a config file to pull a certificate via " +
            "Lets Encrypt.",
        Run: genAcmeConfig,
    }
)

type configChunk struct {
    TlsConfig config.ManualTlsOptions `yaml:"tls_config"`
}

func init() {
    RootCmd.AddCommand(acmeCmd)
    acmeCmd.AddCommand(genAcmeConfigCmd, runAcmeCmd)
    runAcmeCmd.Flags().StringVarP(&configFile, "config-file", "c",
        "", "Configuration file.")
    runAcmeCmd.MarkFlagRequired("config-file")
}

func genAcmeConfig(cmd *cobra.Command, args []string) {
    var configBytes []byte
    configBytes, _ = yaml.Marshal(config.AcmeOptions{
        CertDir: "skyhook-acme",
        Fqdn:    "your.domain.com",
        Email:   "optional@email.com",
    })
    fmt.Println(string(configBytes))
}

func runAcmeServer(cmd *cobra.Command, args []string) (err error) {

    log.INFO.Println("Preparing to acquire Lets Encrypt certificate")

    //==========================
    // CONFIGURE THE CONFIG FILE
    //==========================

    viper.SetConfigType("yaml")
    viper.SetConfigFile(configFile)
    if err = viper.ReadInConfig(); err != nil {
        log.ERR.Print("Failed to read configuration file")
        return err
    }

    //==========================
    // UNMARSHAL THE CONFIG FILE
    //==========================

    conf := config.AcmeOptions{}
    if err = viper.UnmarshalExact(&conf); err != nil {
        log.ERR.Printf("Failed to unmarshal config file: %v", err)
        return err
    }

    if err = conf.Validate(); err != nil {
        log.ERR.Printf("Failed to validate config file: %v", err)
        return err
    }

    //==============
    // INITIATE ACME
    //==============

    log.INFO.Print("Starting the ACME server")
    die := make(chan uint8, 1)
    go startAcme(&conf, die)

    // Allow time for the server to start
    time.Sleep(3 * time.Second)

    // Send HTTPS request to the listening port, inducing certificate
    // generation
    tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
    hClient := &http.Client{Transport: tr}

    //=======================
    // FORCE CERTIFICATE PULL
    //=======================

    log.INFO.Println("Initiating certificate pull")
    if _, err := hClient.Get(fmt.Sprintf("https://%s/ping", conf.Fqdn)); err != nil {

        log.ERR.Printf("%v", err)

    } else {

        //=========================
        // WAIT FOR THE CERTIFICATE
        //=========================

        certPath := fmt.Sprintf("%s/%s", conf.CertDir, conf.Fqdn)
        for i := 0; i <= 5; i++ {

            if _, err := os.Stat(certPath); err == nil {
                log.INFO.Printf("Certificate obtained: %s", certPath)
                log.WARN.Print("Set the below value in the server config:")
                chunk, _ := yaml.Marshal(
                    &configChunk{
                        TlsConfig: config.ManualTlsOptions{
                            CertPath: certPath,
                            KeyPath:  certPath}})
                fmt.Printf("\n\n%s\n\n", chunk)
                break
            }

            if i != 5 {
                log.INFO.Println("Still waiting for certificate")
                time.Sleep(3 * time.Second)
            } else {
                log.ERR.Printf("Failed to obtain certificate: %s", certPath)
            }

        }

    }

    //================
    // KILL THE SERVER
    //================

    log.INFO.Println("Killing the ACME server")
    die <- 1

    log.INFO.Println("Exiting")
    return err
}

func startAcme(conf *config.AcmeOptions, die chan uint8) {

    //=============================
    // CONFIGURE AND RUN THE SERVER
    //=============================

    certMan := autocert.Manager{
        Prompt:     autocert.AcceptTOS,
        HostPolicy: autocert.HostWhitelist(conf.Fqdn),
        Cache:      autocert.DirCache(conf.CertDir),
    }
    srv := &http.Server{
        Addr:      ":https",
        Handler:   nil,
        TLSConfig: certMan.TLSConfig(),
    }
    http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "pong")
    })
    go srv.ListenAndServeTLS("", "")

    //================================
    // WAIT FOR KILL SIGNAL & SHUTDOWN
    //================================

    <-die
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer func() {
        cancel()
    }()
    if err := srv.Shutdown(ctx); err != nil {
        log.ERR.Printf("Failed to gracefully shut down ACME server: %v", err)
    }
}
