package jwt

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Jwt struct {
	Key                 []byte
	TokenExpireDuration time.Duration
}

type UserClaims struct {
	UserId uint `json:"user_id"`
	jwt.RegisteredClaims
}

func NewJwt(key string, tokenExpireDuration time.Duration) *Jwt {
	return &Jwt{
		Key:                 []byte(key),
		TokenExpireDuration: tokenExpireDuration,
	}
}

func (s *Jwt) GenerateToken(userId uint) string {
	if len(s.Key) == 0 {
		fmt.Println("jwt key is nil")
		return ""
	}
	nonce := make([]byte, 24)
	if _, err := rand.Read(nonce); err != nil {
		panic(fmt.Errorf("cannot generate unique login token: %w", err))
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256,
		UserClaims{
			UserId: userId,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.TokenExpireDuration)),
				ID:        base64.RawURLEncoding.EncodeToString(nonce),
			},
		})
	token, err := t.SignedString(s.Key)
	if err != nil {
		fmt.Printf("jwt token generate error: %v", err)
		return ""
	}
	return token
}

func (s *Jwt) ParseToken(tokenString string) (uint, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.Key, nil
	})
	if err != nil {
		return 0, err
	}
	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims.UserId, nil
	}
	return 0, err
}
