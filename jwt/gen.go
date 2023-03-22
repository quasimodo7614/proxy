package jwt

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt"
)

var JwtSecret string

func init() {
	if s := os.Getenv("JWT_Secret"); s != "" {
		JwtSecret = s
	} else {
		JwtSecret = "123456"
	}
}

func GenJwt(d time.Duration) (string, error) {
	hmacSampleSecret := []byte(JwtSecret)
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = jwt.StandardClaims{
		ExpiresAt: time.Now().Add(d).Unix(),
	}
	return token.SignedString(hmacSampleSecret)
}
