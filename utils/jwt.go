package utils

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
    "os"
)

func GenerateJWT(email string) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "email": email,
        "exp":   time.Now().Add(time.Hour * 72).Unix(),
    })

    return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
