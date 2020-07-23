package jwt

import (
	"fmt"
	"go-jwt/db"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

var hmacSampleSecret = "RANDOM_SECRET"

type Payload struct {
	Device string `json:"device,omitempty"`
	Scp    string `json:"scp,omitempty"`
	jwt.StandardClaims
}

func GenPayload(device, scp, sub string) Payload {
	now := time.Now()
	return Payload{
		Device: device,
		Scp:    scp,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: now.Add(1 * time.Hour).Unix(),
			Id:        uuid.New().String(),
			NotBefore: now.Unix(),
			IssuedAt:  now.Unix(),
			Subject:   sub,
		},
	}
}

func JwtRevoked(payload Payload) bool {
	return db.RDBExists(fmt.Sprintf("user_blacklist:%s:%s:%s", payload.Subject, payload.Device, payload.Id))
}

func RevokeJwt(payload Payload) {
	expiration := payload.ExpiresAt - payload.IssuedAt
	db.RDBSet(fmt.Sprintf("user_blacklist:%s:%s:%s", payload.Subject, payload.Device, payload.Id), payload.Id, time.Duration(expiration)*time.Second)
}

func RevokeLastJwt(payload Payload) {
	lastJwt, err := db.RDBGet(fmt.Sprintf("user_device_jwt:%s:%s", payload.Subject, payload.Device))
	if err != nil && err != redis.Nil {
		fmt.Println("redis err:", err)
	}
	if lastJwt != "" {
		arr := strings.Split(lastJwt, ":")
		jti, expStr := arr[0], arr[len(arr)-1]
		exp, err := strconv.ParseInt(expStr, 10, 64)
		if err != nil {
			exp = time.Now().Unix()
		}
		payload.Id = jti
		payload.IssuedAt = time.Now().Unix()
		payload.ExpiresAt = exp
		RevokeJwt(payload)
	}
}

func OnJwtDispatch(payload Payload) {
	RevokeLastJwt(payload)
	db.RDBSet(fmt.Sprintf("user_device_jwt:%s:%s", payload.Subject, payload.Device), fmt.Sprintf("%s:%d", payload.Id, payload.ExpiresAt), time.Unix(payload.ExpiresAt, 0).Sub(time.Now()))
}

func Encoder(payload Payload) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	tokenString, err := token.SignedString([]byte(hmacSampleSecret))
	if err != nil {
		fmt.Println(err)
	}
	return tokenString
}

func Decoder(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Payload{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(hmacSampleSecret), nil
	})
	if err != nil {
		return "", err
	}

	if payload, ok := token.Claims.(*Payload); ok && token.Valid {
		sub := (*payload).Subject
		if sub != "" && !JwtRevoked(*payload) {
			return sub, nil
		} else {
			return "", fmt.Errorf("token is expired")
		}
	}

	return "", fmt.Errorf("invalid token")
}
