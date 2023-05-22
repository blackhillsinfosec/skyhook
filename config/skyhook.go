package config

import (
    "fmt"
    obfs "github.com/blackhillsinfosec/skyhook-obfuscation"
    "github.com/blackhillsinfosec/skyhook/log"
    "os"
)

// ManualTlsOptions are the values used to configure
// self-managed SSL certificates.
type ManualTlsOptions struct {
    // CertPath is the path to the certificate file.
    CertPath string `nonzero:"" mapstructure:"cert_path" yaml:"cert_path"`
    // KeyPath is the path to the key file.
    KeyPath string `nonzero:"" mapstructure:"key_path" yaml:"key_path"`
}

// AcmeOptions has values to configured automated
// certificate management via LetsEncrypt.
type AcmeOptions struct {
    // CertDir is a path to the directory where
    // AutoTLS will store certificate files.
    CertDir string `nonzero:"skyhook-acme" mapstructure:"cert_directory" yaml:"cert_directory"`
    // Fqdn determines the FQDN to pull a certificate for.
    Fqdn string `nonzero:""`
    // Email address to use while interacting with LetsEncrypt.
    //
    // This value is optional.
    Email string
}

// Validate AcmeOptions.
func (l *AcmeOptions) Validate() (err error) {
    if err = CheckNonZeroFormat(l); err == nil {
        if _, err := os.Stat(l.CertDir); err != nil {
            log.WARN.Printf("Creating cache directory: %s", l.CertDir)
            if err := os.Mkdir(l.CertDir, 0700); err != nil {
                panic(fmt.Sprintf("Failed to create auto TLS cache directory: %s", err))
            }
        }
    }
    return err
}

// FileServerUploadOptions defines options related to file upload.
type FileServerUploadOptions struct {
    RegistrantsFile   string `nonzero:"" yaml:"registrants_file" json:"registrants_file" mapstructure:"registrants_file"`
    MaxUploadDuration uint   `nonzero:"24" yaml:"max_upload_duration" json:"max_upload_duration" mapstructure:"max_upload_duration"`
}

// FileServerRangeHeaderOptions enables configuration of the HTTP Range
// header, allowing us to bypass cloud services that often strip the
// Range header, such as CDNs.
type FileServerRangeHeaderOptions struct {
    // Name of the header.
    Name string `nonzero:"Range" yaml:"name" json:"name" mapstructure:"name"`
    // RangePrefix is the prefix set before the range, e.g., "bytes=0-10".
    RangePrefix string `nonzero:"bytes" yaml:"range_prefix" json:"range_prefix" mapstructure:"range_prefix"`
}

// EncryptedInterfaceLoaderRoutes is used to configure routes
// for the encrypted loader.
type EncryptedInterfaceLoaderRoutes struct {
    // AutoHtml the web path used to reference the HTML
    // document that will automatically load the Js.
    AutoHtml string `yaml:"auto_html" json:"auto_html" mapstructure:"auto_html"`
    // Html is the web path referencing a blank HTML document
    // that Js can be pasted into via the developer tools, enabling
    // proper configuration of CORS.
    //
    // See Js for a description on how to decrypt and initiate the loader.
    Html string `yaml:"html" json:"html" mapstructure:"html"`
    // Js is the web path referencing the JS encrypted loader.
    //
    // This loader is dynamically generated at each reference. It's
    // encrypted with a secret shared among all users defined in
    // LandingFileEncryptionOptions.Key.
    //
    // # Loader Decryption Methods
    //
    // Operators can decrypt the loader using one of two methods:
    //
    // ## Pass the Key to Developer Tools Variable
    //
    // Opening the browser's developer tools and observing
    // a variable name logged to the script console. Set this
    // variable to LandingFileEncryptionOptions.Key to initiate
    // decryption.
    //
    // ## Pass the Key to the Browser's URL Bar
    //
    // If access to developer tools has been restricted, the
    // encryption key can be passed to JS via browser URL bar. This
    // technique makes use of the value set to
    // LandingFileEncryptionOptions.UriParam.
    //
    // Example where curly brackets indicate parameter values:
    //
    // https://files.somedomain.com/path/to/AutoHTML#{UriParam}={Key}
    //
    // NOTE: Although the value appears in the URL bar of the browser,
    // the key _is not_ sent to the server. (tested in Chrome)
    Js string `yaml:"js" json:"js" mapstructure:"js"`
}

// FileServerApiRoutes defines web routes to the file server's
// API endpoints.
type FileServerApiRoutes struct {
    Logout   string `nonzero:"/logout" yaml:"logout" mapstructure:"logout" json:"logout"`
    Download string `nonzero:"/files" yaml:"download" mapstructure:"download" json:"download"`
    Upload   string `nonzero:"/upload" yaml:"upload" mapstructure:"upload" json:"upload"`
}

// FileServerRouteOptions aggregates various sets of options
// for web routes hosted by the file server.
type FileServerRouteOptions struct {
    Api             FileServerApiRoutes            `yaml:"api" mapstructure:"api"`
    EncryptedLoader EncryptedInterfaceLoaderRoutes `json:"encrypted_loader" yaml:"encrypted_loader" mapstructure:"encrypted_loader"`
    // LandingPage routes define routes to the underlying web
    // application embedded in the binary. As there are many
    // routes, and the fact that they're likely to change in
    // the future, this is represented as a simple mapping of
    // strings.
    //
    // Note: Use the generate config subcommand under server
    // to generate randomized values.
    LandingPage map[string]string `nonzero:"" json:"landing_page" yaml:"landing_page" mapstructure:"landing_page"`
}

// FileServerOptions provides configuration values for the file server
// component of Skyhook.
type FileServerOptions struct {
    ServerOptions `mapstructure:",squash" yaml:",inline"`
    // RootDir is where files will be served from.
    RootDir string `nonzero:"skyhook_webroot" mapstructure:"root_directory" yaml:"root_directory"`
    // Obfuscators is a slice of objects used to obfuscate and deobfuscate data.
    Obfuscators        []obfs.ObfuscatorConfig
    UploadOptions      FileServerUploadOptions      `nonzero:"" yaml:"upload_options" json:"upload_options" mapstructure:"upload_options"`
    Routes             FileServerRouteOptions       `nonzero:"" yaml:"routes" json:"routes" mapstructure:"routes"`
    EncryptedLoader    LandingFileEncryptionOptions `nonzero:"" yaml:"encrypted_loader" json:"encrypted_loader" mapstructure:"encrypted_loader"`
    LinkFqdns          []string                     `nonzero:"" yaml:"link_fqdns" json:"link_fqdns" mapstructure:"link_fqdns"`
    RangeHeaderOptions FileServerRangeHeaderOptions `nonzero:"" yaml:"range_header_options" json:"range_header_options" mapstructure:"range_header_options"`
}

// Validate FileServerOptions.
func (fs *FileServerOptions) Validate() (err error) {

    // Validate the options.
    if err = fs.ServerOptions.Validate(); err != nil {
        return err
    }

    if _, iErr := os.Stat(fs.RootDir); iErr != nil {

        //===================
        // HANDLE THE WEBROOT
        //===================

        log.WARN.Printf("Webroot doesn't exist: %s", fs.RootDir)
        log.INFO.Println("Attempting to create webroot directory")
        if err = os.Mkdir(fs.RootDir, 0700); err != nil {
            log.ERR.Printf("Failed to create webroot directory: %v", err)
            return err
        }

    }

    if len(fs.Obfuscators) == 0 {
        log.WARN.Println("No obfuscators have been configured")
        log.WARN.Println("Only Base64 encoding of artifacts will occur")
    }

    return nil

}

// LandingFileEncryptionOptions provide parameters that enable
// generation of an encrypted JavaScript function that, once
// decrypted, loads each file in an encrypted format and loads
// it into the browser to produce the web interface.
type LandingFileEncryptionOptions struct {
    // Key used to decrypt the loader.
    Key string `nonzero:"" json:"key" yaml:"key" mapstructure:"key"`
    // UriParam is a parameter that determines if the target
    // resource should be encrypted using Key before delivery to
    // the client.
    UriParam string `nonzero:"crypt" json:"uri_param" yaml:"uri_param" mapstructure:"uri_param"`
    // RootElementId is the root HTML element in the HTML landing
    // page that will be referenced to render the interface. This
    // is configurable to mitigate fingerprinting.
    RootElementId string `nonzero:"root" json:"root_element_id" yaml:"root_element_id" mapstructure:"root_element_id"`
}

// ServerOptions has all options related to the
// file server.
type ServerOptions struct {
    // AddtlCorsUrls additionally accepted CORS headers.
    AddtlCorsUrls []string `yaml:"additional_cors_urls" mapstructure:"additional_cors_urls"`
    // Interface is the string name of the network interface.
    Interface string `nonzero:"lo"`
    // Port is the port number the server will listen on.
    Port uint16
    // ip address of Interface.
    //
    // Validate must be called for this value to be populated.
    //
    // Use IP to retrieve this value.
    ip string
}

// Validate ServerOptions.
func (s *ServerOptions) Validate() (err error) {
    s.ip, err = FindInterface(s.Interface)
    return err
}

// IP gets the ServerOptions' validated ip value.
func (s *ServerOptions) IP() string {
    return s.ip
}

// Socket returns the socket the server targets.
func (s *ServerOptions) Socket() string {
    return fmt.Sprintf("%s:%v", s.ip, s.Port)
}

// AdminAuthHeaderOptions provides options related to JWT header
// authentication.
type AdminAuthHeaderOptions struct {
    Name   string `nonzero:"Authorization" yaml:"name" json:"name"`
    Scheme string `nonzero:"Bearer" yaml:"scheme" json:"scheme"`
}

type AuthOptions struct {
    Header AdminAuthHeaderOptions `nonzero:"" yaml:"header" mapstructure:"header" json:"header"`
    Jwt    JwtOptions             `nonzero:"" mapstructure:"jwt" yaml:"jwt" json:"jwt"`
}

// JwtOptions provides options related to JWT header
// authentication.
type JwtOptions struct {
    SafeJwtOptions `mapstructure:",squash" yaml:",inline"`
    SigningKey     string `nonzero:"" yaml:"signing_key" mapstructure:"signing_key" json:"signing_key"`
}

type SafeJwtOptions struct {
    Realm     string       `nonzero:"skyhook" yaml:"realm" mapstructure:"realm" json:"realm"`
    FieldKeys JwtFieldKeys `nonzero:"" yaml:"field_names" mapstructure:"field_names" json:"field_keys"`
}

type JwtFieldKeys struct {
    Username string `nonzero:"user" yaml:"username" mapstructure:"username" json:"username"`
    Admin    string `nonzero:"is_admin" yaml:"admin" mapstructure:"admin" json:"admin"`
    Config   string `nonzero:"config" yaml:"config" json:"config" mapstructure:"config"`
}

type SafeAuthOptions struct {
    Header AdminAuthHeaderOptions `yaml:"header" json:"header"`
    Jwt    SafeJwtOptions         `json:"jwt" yaml:"jwt"`
}

// AdminServerOptions are options related to the admin
// server used to control various attributes of the file
// server.
type AdminServerOptions struct {
    ServerOptions `yaml:",inline" mapstructure:",squash"`
}

// Validate AdminServerOptions.
func (as *AdminServerOptions) Validate() (err error) {
    as.ip, err = FindInterface(as.Interface)
    return err
}

// Credential objects represent a set of login credentials.
type Credential struct {
    // Username value
    Username string `nonzero:"" mapstructure:"username" yaml:"username" json:"username"`
    // Password value
    Password string `nonzero:"" mapstructure:"password" yaml:"password" json:"password"`
    // IsAdmin determines if the user is an admin, i.e., if
    // the user should have access to the admin panel.
    IsAdmin bool   `mapstructure:"is_admin" yaml:"is_admin" json:"is_admin"`
    Token   string `nonzero:"" yaml:"token" json:"token" mapstructure:"token"`
}

// SkyhookConfig holds all options related to a Skyhook
// configuration.
type SkyhookConfig struct {
    // Tls has configs related to TLS settings.
    Tls ManualTlsOptions `yaml:"tls_config" mapstructure:"tls_config"`
    // AdminServer is the configuration for the administrative
    // server.
    AdminServer AdminServerOptions `nonzero:"" mapstructure:"admin_server_config" yaml:"admin_server_config"`
    // FileServer is the configuration for the file server.
    FileServer FileServerOptions `nonzero:"" mapstructure:"file_server_config" yaml:"file_server_config"`
    // Users has credentials for user accounts.
    Users []Credential `nonzero:""`
    Auth  AuthOptions  `nonzero:"" mapstructure:"auth_config" yaml:"auth_config"`
}

// Validate SkyhookConfig.
func (sc *SkyhookConfig) Validate() (err error) {

    // Recursively check all configurations for zero values.
    if err = CheckNonZeroFormat(sc); err != nil {
        return err
    }

    if err = sc.AdminServer.Validate(); err != nil {
        log.ERR.Println("Validation of admin server config failed")
        return err
    }

    if err = sc.FileServer.Validate(); err != nil {
        log.ERR.Println("Validation of file server config failed")
        return err
    }

    return nil
}
