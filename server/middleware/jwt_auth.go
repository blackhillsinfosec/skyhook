package middleware

import (
	"encoding/json"
	jwt "github.com/appleboy/gin-jwt/v2"
	obfuscate "github.com/blackhillsinfosec/skyhook-obfuscation"
	structs "github.com/blackhillsinfosec/skyhook/api_structs"
	"github.com/blackhillsinfosec/skyhook/config"
	"github.com/gin-gonic/gin"
	"net/http"
)

// JwtIsUnauthorized handles JwtIsUnauthorized requests.
func JwtIsUnauthorized(c *gin.Context, code int, message string) {
    //c.JSON(code, structs.BaseResponse{})
    c.Status(http.StatusUnauthorized)
}

// JwtIsCredAdmin handles request authorization.
func JwtIsCredAdmin(cred interface{}, c *gin.Context) bool {
    if v, ok := cred.(*config.Credential); ok && v.IsAdmin {
        return true
    }
    return false
}

func JwtIdentityHandler(usernameField, adminField *string) func(c *gin.Context) interface{} {
    return func(c *gin.Context) interface{} {
        return JwtExtractCtxClaims(*usernameField, *adminField, c)
    }
}

// JwtExtractCtxClaims extract claims from the current request's
// authentication context, i.e., fields are parsed and returned from
// the authenticated user's JWT.
func JwtExtractCtxClaims(usernameField, adminField string, c *gin.Context) interface{} {
    claims := jwt.ExtractClaims(c)
    return &config.Credential{
        Username: claims[usernameField].(string),
        IsAdmin:  claims[adminField].(bool),
    }
}

func JwtLoginHandler(users *[]config.Credential, adminRequired bool) func(c *gin.Context) (interface{}, error) {
    return func(c *gin.Context) (interface{}, error) {
        p := structs.LoginPayload{}
        if err := c.BindJSON(&p); err != nil {
            c.Status(http.StatusUnauthorized)
            return nil, nil
        }
        for _, cred := range *users {
            if cred.Username == p.Username && cred.Password == p.Password {
                if !adminRequired || (adminRequired && cred.IsAdmin) {
                    return &cred, nil
                }
            }
        }
        return nil, jwt.ErrFailedAuthentication
    }
}

// JwtPayloadFunc returns a function that generates the JWT payload.
func JwtPayloadFunc(conf *config.SkyhookConfig, obfs *[]obfuscate.Obfuscator) func(data interface{}) jwt.MapClaims {
    return func(data interface{}) jwt.MapClaims {
        if v, ok := data.(*config.Credential); ok {
            var addtl []byte
            var err error
            if addtl, err = json.Marshal(JwtConfigData{
                ApiRoutes:   conf.FileServer.Routes.Api,
                Obfuscators: *obfuscate.UnparseObfuscators(obfs),
                UploadConfig: UploadConfigData{
                    RangeHeaderName: conf.FileServer.RangeHeaderOptions.Name,
                    RangePrefix:     conf.FileServer.RangeHeaderOptions.RangePrefix,
                },
                AuthConfig: config.SafeAuthOptions{
                    Header: conf.Auth.Header,
                    Jwt:    conf.Auth.Jwt.SafeJwtOptions}}); err != nil {
                panic("failed to generate JWT response data while authenticating user")
            } else {
                // Encrypt the data with the user's token
                x := obfuscate.XOR{Key: v.Token}
                addtl, _ = x.Obfuscate(addtl)
                addtl = obfuscate.Base64Encode(addtl)
            }

            return jwt.MapClaims{
                conf.Auth.Jwt.FieldKeys.Username: v.Username,
                conf.Auth.Jwt.FieldKeys.Admin:    v.IsAdmin,
                conf.Auth.Jwt.FieldKeys.Config:   string(addtl),
            }
        }
        return jwt.MapClaims{}
    }
}

type JwtConfigData struct {
    ApiRoutes    config.FileServerApiRoutes   `json:"api_routes" yaml:"api_routes"`
    Obfuscators  []obfuscate.ObfuscatorConfig `json:"obfuscators" yaml:"obfuscators"`
    AuthConfig   config.SafeAuthOptions       `json:"auth_config" yaml:"auth_config"`
    UploadConfig UploadConfigData             `json:"upload_config" yaml:"upload_config"`
}

type UploadConfigData struct {
    RangeHeaderName string `json:"range_header_name" yaml:"range_header_name"`
    RangePrefix     string `json:"range_prefix" yaml:"range_prefix"`
}
