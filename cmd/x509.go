package cmd

import (
    "bytes"
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "crypto/x509/pkix"
    "encoding/pem"
    "errors"
    "fmt"
    "github.com/blackhillsinfosec/skyhook/config"
    "github.com/blackhillsinfosec/skyhook/log"
    "github.com/blackhillsinfosec/skyhook/util"
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"
    "os"
)

var (
    // x509Cmd is the subcommand used to generate
    // various files for operation of Skyhook.
    x509Cmd = &cobra.Command{
        Use:     "x509",
        Aliases: []string{"x"},
        Short:   "Generate self-signed X509 files.",
    }

    // genX509ConfigCmd generates and dumps configuration
    // files for Skyhook.
    genX509ConfigCmd = &cobra.Command{
        Use:     "generate-config",
        Aliases: genAliases,
        Short:   "Generate a config to create X509 files.",
        Run:     genX509Config,
    }

    // genX509FilesCmd generates x509 certificates.
    genX509FilesCmd = &cobra.Command{
        Use:     "generate-certs",
        Aliases: []string{"run"},
        Short:   "Generate and dump an x509 key pair",
        RunE:    generateX509,
    }
)

func init() {
    RootCmd.AddCommand(x509Cmd)
    x509Cmd.AddCommand(genX509ConfigCmd, genX509FilesCmd)

    //==============================
    // ARGUMENTS FOR X509 GENERATION
    //==============================

    genX509FilesCmd.Flags().StringVarP(&configFile, "config-file", "c",
        "", "Config file containing configuration parameters to generate "+
            "a x509 certificate keypair.")
    genX509FilesCmd.Flags().StringVarP(&manCertFile, "cert-file", "",
        "cert.pem", "File that will receive the X509 certificate.")
    genX509FilesCmd.Flags().StringVarP(&manKeyFile, "key-file", "",
        "key.pem", "File that will receive the X509 key.")
    genX509FilesCmd.Flags().Bool("overwrite", false, "Overwrite output files.")
}

// genX509Config is the function executed by the "generate" subcommand
// to produce config files.
func genX509Config(cmd *cobra.Command, args []string) {
    var configBytes []byte
    configBytes, _ = yaml.Marshal(config.DefaultX509Config())
    fmt.Println(string(configBytes))
}

// generateX509 generates a x509 certificate and key.
func generateX509(cmd *cobra.Command, args []string) (err error) {
    var c *config.GenerateX509Config
    if configFile != "" {
        //===============================================
        // GENERATE A X509 CERT FROM USER SUPPLIED CONFIG
        //===============================================

        if _, err := os.Stat(configFile); err != nil {
            log.ERR.Printf("Config file does not exist: %s", configFile)
            return err
        }
        c = &config.GenerateX509Config{}
        util.UnmarshalFileInto(&configFile, c)

    } else {
        //==================================================
        // GENERATE STANDARD X509 WHEN NO CONFIG IS SUPPLIED
        //==================================================

        c = config.DefaultX509Config()
    }

    //==================
    // GENERATE THE X509
    //==================

    o, _ := cmd.Flags().GetBool("overwrite")
    return GenerateX509(c, manCertFile, manKeyFile, o)
}

func GenerateX509(config *config.GenerateX509Config, certOutFile, keyOutFile string, overwrite bool) (err error) {

    //=====================
    // PREPARE OUTPUT FILES
    //=====================

    if certOutFile == "" {
        log.WARN.Println("No certificate file name provided. Using default: cert.pem")
        certOutFile = "cert.pem"
    }

    if keyOutFile == "" {
        log.WARN.Println("No key file name provided. Using default: cert.pem")
        keyOutFile = "key.pem"
    }

    if overwrite {
        log.WARN.Println("File overwriting enabled")
    }

    if _, e := os.Stat(certOutFile); e == nil {
        if !overwrite {
            return errors.New("certificate file exists")
        } else {
            log.INFO.Printf("Removing file: %s", certOutFile)
            os.Remove(certOutFile)
        }
    }

    if _, e := os.Stat(keyOutFile); e == nil {
        if !overwrite {
            return errors.New("key file exists")
        } else {
            log.INFO.Printf("Removing file: %s", keyOutFile)
            os.Remove(keyOutFile)
        }
    }

    //=========================
    // GENERATE THE CERTIFICATE
    //=========================

    log.INFO.Println("Generating X509 certificate")
    pKey, _ := rsa.GenerateKey(rand.Reader, 4028)

    subject := pkix.Name{
        Country:            []string{config.Subject.Country},
        Province:           []string{config.Subject.Province},
        Locality:           []string{config.Subject.Locality},
        Organization:       []string{config.Subject.Organization},
        OrganizationalUnit: []string{config.Subject.OrganizationalUnit},
        CommonName:         config.Subject.CommonName,
    }

    cert := &x509.Certificate{
        Subject:            subject,
        SerialNumber:       config.SerialNumber,
        NotBefore:          config.Timeline.NotBefore,
        NotAfter:           config.Timeline.NotAfter,
        Issuer:             subject,
        SignatureAlgorithm: x509.SHA256WithRSA,
    }

    certBytes, _ := x509.CreateCertificate(rand.Reader, cert, cert, &pKey.PublicKey, pKey)

    //===================
    // WRITE OUTPUT FILES
    //===================

    certPEM := &bytes.Buffer{}
    pem.Encode(certPEM, &pem.Block{
        Type:  "CERTIFICATE",
        Bytes: certBytes,
    })

    certPrivKeyPEM := &bytes.Buffer{}
    pem.Encode(certPrivKeyPEM, &pem.Block{
        Type:  "RSA PRIVATE KEY",
        Bytes: x509.MarshalPKCS1PrivateKey(pKey),
    })

    log.INFO.Printf("Writing certificate file: %s", certOutFile)
    pemFile, err := os.OpenFile(certOutFile, os.O_WRONLY|os.O_CREATE, 0600)

    if err != nil {
        return err
    }
    defer pemFile.Close()
    pemFile.Write(certPEM.Bytes())

    log.INFO.Printf("Writing certificate file: %s", keyOutFile)
    keyFile, err := os.OpenFile(keyOutFile, os.O_WRONLY|os.O_CREATE, 0600)

    if err != nil {
        return err
    }
    defer keyFile.Close()
    keyFile.Write(certPrivKeyPEM.Bytes())

    log.INFO.Println("Exiting")

    return err
}
