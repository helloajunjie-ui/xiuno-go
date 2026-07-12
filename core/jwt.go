// xiuno-go v2.1.0-beta 尼克修改版
package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWTClaims JWT 载荷
type JWTClaims struct {
	UID uint32 `json:"uid"`
	GID uint16 `json:"gid"`
	Exp int64  `json:"exp"`
}

// jwtHeader JWT 头部
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// SignJWT 签发 JWT token
func SignJWT(uid uint32, gid uint16, secret string, expireHour int) (string, error) {
	header := jwtHeader{Alg: "HS256", Typ: "JWT"}
	headerJSON, _ := json.Marshal(header)

	claims := JWTClaims{
		UID: uid,
		GID: gid,
		Exp: time.Now().Add(time.Duration(expireHour) * time.Hour).Unix(),
	}
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	payload := headerB64 + "." + claimsB64
	sig := jwtSign(payload, secret)

	return payload + "." + sig, nil
}

// ParseJWT 解析并验证 JWT token
func ParseJWT(token string, secret string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	payload := parts[0] + "." + parts[1]
	expectedSig := jwtSign(payload, secret)

	// 恒定时间比较签名
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid signature")
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid claims encoding: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("invalid claims json: %w", err)
	}

	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

func jwtSign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
