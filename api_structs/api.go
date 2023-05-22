package api_structs

import (
    obfs "github.com/blackhillsinfosec/skyhook-obfuscation"
    "github.com/blackhillsinfosec/skyhook/config"
    "github.com/blackhillsinfosec/skyhook/server/upload"
)

var (
    // TODO uncomment these as more algorithms become available
    Obfuscators = map[string]interface{}{
        "base64":   obfs.Base64{},
        "xor":      obfs.XOR{},
        "aes":      obfs.AES{},
        "blowfish": obfs.Blowfish{},
        "twofish":  obfs.Twofish{},
    }
)

// BaseResponse provides a base foundation for response objects.
type BaseResponse struct {
    Success bool   `json:"success"`
    Message string `json:"msg"`
}

// BaseSuccessResponse returns a BaseResponse with BaseResponse.Success
// set to true.
func BaseSuccessResponse() BaseResponse {
    return BaseResponse{
        Success: true,
    }
}

// PingResponse is the response structure for PingHandler.
type PingResponse struct {
    BaseResponse `mapstructure:",squash"`
    Response     string `json:"response"`
}

// LoginResponse is the JSON response returned from
// authentication requests.
type LoginResponse struct {
    BaseResponse `mapstructure:",squash"`
    Outcome      bool `json:"outcome"`
}

// LoginPayload is the JSON body expected from authentication.
type LoginPayload struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

// CredList is a list of user credentials.
//
// Warning: this _does_ include the cleartext password.
type CredList struct {
    Users []config.Credential `json:"users" binding:"required"`
}

// CredListResponse is used as the response object to various
// handler functions.
type CredListResponse struct {
    BaseResponse `mapstructure:",squash"`
    CredList     `mapstructure:",squash"`
    //ConfigUri    string `yaml:"config_uri" json:"config_uri" mapstructure:"config_uri"`
}

// DuplicateUsernameResponse is used as the response object to various
// handler functions.
type DuplicateUsernameResponse struct {
    BaseResponse `mapstructure:",squash"`
    Usernames    []string `json:"usernames"`
}

// ObfuscatorsPayload is the request payload for various handler
// functions.
type ObfuscatorsPayload struct {
    Obfuscators []obfs.ObfuscatorConfig `json:"obfuscators"`
}

// SaveObfuscatorsResponse is used as the response object to various
// handler functions.
type SaveObfuscatorsResponse struct {
    BaseResponse       `mapstructure:",squash"`
    ObfuscatorsPayload `mapstructure:",squash"`
}

// ListObfuscatorsResponse is used as the response object to various
// handler functions.
type ListObfuscatorsResponse struct {
    BaseResponse `mapstructure:",squash"`
    Obfuscators  map[string]interface{} `json:"obfuscators" yaml:"obfuscators"`
}

// GetObfuscatorsResponse is used as the response object to various
// handler functions.
type GetObfuscatorsResponse struct {
    BaseResponse `mapstructure:",squash"`
    Obfuscators  []obfs.ObfuscatorConfig `json:"obfuscators"`
}

//// obfuscators lists all supported obfuscators.
//type obfuscators struct {
//	Aes      obfs.AES      `json:"aes"`
//	Base64   obfs.Base64   `json:"base64"`
//	Blowfish obfs.Blowfish `json:"blowfish"`
//	Twofish  obfs.Twofish  `json:"twofish"`
//	Xor      obfs.XOR      `json:"xor"`
//}

type RegisterUploadRequest struct {
    Path string `json:"path" yaml:"path"`
}

type RegisterUploadResponse struct {
    BaseResponse          `mapstructure:",squash"`
    RegisterUploadRequest `mapstructure:",squash"`
    //   Id                    string `json:"id" yaml:"id"`
}

type ListUploadsResponse struct {
    BaseResponse `mapstructure:",squash"`
    Uploads      []upload.Upload `json:"uploads" yaml:"uploads"`
}

type AdvancedConfigResponse struct {
    BaseResponse `mapstructure:",squash"`
    ApiRoutes    config.FileServerApiRoutes `json:"api_routes" yaml:"api_routes"`
    Obfuscators  []obfs.ObfuscatorConfig    `json:"obfuscators" yaml:"obfuscators"`
    AuthConfig   config.SafeAuthOptions     `json:"auth_config" yaml:"auth_config"`
}

type ConfigRequest struct {
    Token string `json:"token" yaml:"token"`
}

type LinksResponse map[string]Links

type Links struct {
    Standard  StandardLinks        `json:"standard" yaml:"standard" mapstructure:"standard"`
    Encrypted EncryptedLoaderLinks `yaml:"encrypted" json:"encrypted" mapstructure:"encrypted"`
}

type EncryptedLoaderLinks struct {
    Js           string `yaml:"js" json:"js" mapstructure:"js"`
    Html         string `yaml:"html" json:"html" mapstructure:"html"`
    AutoloadHtml string `yaml:"autoload_html" json:"autoload_html" mapstructure:"autoload_html"`
}

type StandardLinks struct {
    Html string `yaml:"html" json:"html" mapstructure:"html"`
}

type EncryptedJsResponse struct {
    BaseResponse `mapstructure:",squash"`
    EncryptedJs  string `yaml:"encrypted_js" json:"encrypted_js" mapstructure:"encrypted_js"`
}
