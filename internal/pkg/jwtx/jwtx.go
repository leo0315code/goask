// Package jwtx 封装 JWT（HS256）的签发与解析。
package jwtx

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalid 统一表示 token 缺失、篡改、过期等所有无效情况。
var ErrInvalid = errors.New("jwtx: invalid token")

type Claims struct {
	UserID uint64 `json:"uid"`
	Role   string `json:"role"`
	Ver    uint64 `json:"ver"` // 用户 token_version，改密码/全端登出后旧 token 失效
	jwt.RegisteredClaims
}

// Generate 签发 token，返回 token 字符串与过期时间。角色冗余在 claims 中，
// 用于免查库的权限判断（角色变更在令牌过期后生效）。
func Generate(userID uint64, role string, ver uint64, secret string, ttl time.Duration) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(ttl)
	claims := Claims{
		UserID: userID,
		Role:   role,
		Ver:    ver,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "goask",
			Subject:   strconv.FormatUint(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return signed, exp, err
}

// Parse 校验并解析 token；任何失败都返回 ErrInvalid。
func Parse(tokenStr, secret string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer("goask"))
	if err != nil || !parsed.Valid {
		return nil, ErrInvalid
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || claims.UserID == 0 {
		return nil, ErrInvalid
	}
	return claims, nil
}
