package auth

import (
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)



func HashedPassword(password string) (string, error){
	hash_pw, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hash_pw, nil
}

func CheckPasswordHash(password string, hash string) (bool, error){
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err!= nil {
		return false, err
	}
	return match, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error){
	jwt := jwt.NewWithClaims()
}