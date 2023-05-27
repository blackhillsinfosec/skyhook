package middleware

import (
    jwt "github.com/appleboy/gin-jwt/v2"
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
func JwtPayloadFunc(conf *config.SkyhookConfig) func(data interface{}) jwt.MapClaims {
    return func(data interface{}) jwt.MapClaims {

        //==========================================
        // CONSTRUCT AND RETURN JWT WITH CONFIG DATA
        //==========================================

        if v, ok := data.(*config.Credential); ok {

            if oc, err := structs.NewOperatingConfigData(*conf).JsonCryptMarshal(v.Token); err != nil {
                panic("failed to generate JWT response data while authenticating user")
            } else {
                return jwt.MapClaims{
                    conf.Auth.Jwt.FieldKeys.Username: v.Username,
                    conf.Auth.Jwt.FieldKeys.Admin:    v.IsAdmin,
                    conf.Auth.Jwt.FieldKeys.Config:   oc,
                }
            }
        }

        return jwt.MapClaims{}

    }
}
