package jwt

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

var DisAbleJwt bool

func init() {
	if s := os.Getenv("Disable_Jwt"); s == "true" {
		DisAbleJwt = true
	}
}
func JwtCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if DisAbleJwt {
			return
		}
		if c.Request.Method == http.MethodOptions {
			return
		}

		tokenString := c.GetHeader("Authorization")
		var hmacSampleSecret = []byte(JwtSecret)
		//前面例子生成的token
		token, err := jwt.ParseWithClaims(tokenString, &jwt.StandardClaims{}, func(t *jwt.Token) (interface{}, error) {
			return hmacSampleSecret, nil
		})

		if err != nil {
			log.Println("ParseWithClaims err: ", err)
			c.AbortWithStatusJSON(400, err.Error())
			return
		}
		if !token.Valid {
			c.AbortWithStatusJSON(400, err.Error())
			return
		}
		log.Println("token valid")
		c.Next()
	}
}
