package jwtx

import (
	"testing"
	"time"
)

const testSecret = "test-secret-at-least-32-characters!!"

func TestGenerateAndParse(t *testing.T) {
	token, exp, err := Generate(42, "admin", testSecret, time.Hour)
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	if exp.IsZero() {
		t.Fatal("过期时间为零值")
	}
	claims, err := Parse(token, testSecret)
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	if claims.UserID != 42 || claims.Role != "admin" {
		t.Fatalf("claims 不匹配: uid=%d role=%s", claims.UserID, claims.Role)
	}
}

func TestParseWrongSecret(t *testing.T) {
	token, _, err := Generate(1, "user", testSecret, time.Hour)
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	if _, err := Parse(token, "another-secret-at-least-32-characters"); err == nil {
		t.Fatal("错误密钥应解析失败")
	}
}

func TestParseExpired(t *testing.T) {
	token, _, err := Generate(1, "user", testSecret, -time.Minute)
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	if _, err := Parse(token, testSecret); err != ErrInvalid {
		t.Fatalf("过期 token 应返回 ErrInvalid，实际: %v", err)
	}
}

func TestParseGarbage(t *testing.T) {
	if _, err := Parse("not.a.token", testSecret); err != ErrInvalid {
		t.Fatalf("垃圾字符串应返回 ErrInvalid，实际: %v", err)
	}
}
