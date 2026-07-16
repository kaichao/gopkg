package security

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// ── 辅助 ────────────────────────────────────────────────────

func generateTestKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return pub, priv
}

type testClaims struct {
	Sub      string   `json:"sub"`
	Iss      string   `json:"iss"`
	Iat      int64    `json:"iat"`
	Exp      int64    `json:"exp"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

func signJWT(t *testing.T, priv ed25519.PrivateKey, claims testClaims) string {
	t.Helper()
	header := `{"alg":"EdDSA","typ":"JWT"}`
	claimBytes, _ := json.Marshal(claims)
	headerB64 := b64URLEncode([]byte(header))
	claimsB64 := b64URLEncode(claimBytes)
	message := headerB64 + "." + claimsB64
	sig := ed25519.Sign(priv, []byte(message))
	return message + "." + b64URLEncode(sig)
}

// ── 测试 ────────────────────────────────────────────────────

func TestJWTVerifierValidToken(t *testing.T) {
	pub, priv := generateTestKey(t)
	keyFile := writeTempKey(t, pub)
	defer os.Remove(keyFile)

	v, err := NewJWTVerifier(JWTVerifierConfig{
		PublicKeyFile: keyFile,
		Issuer:        "test-issuer",
	})
	if err != nil {
		t.Fatalf("NewJWTVerifier: %v", err)
	}

	now := time.Now()
	token := signJWT(t, priv, testClaims{
		Sub:      "user-1",
		Iss:      "test-issuer",
		Iat:      now.Unix(),
		Exp:      now.Add(1 * time.Hour).Unix(),
		Username: "alice",
		Roles:    []string{"admin"},
	})

	id, err := v.AuthenticateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("AuthenticateToken: %v", err)
	}
	if id.Subject() != "user-1" {
		t.Errorf("Subject = %q, want user-1", id.Subject())
	}
	if id.Name() != "alice" {
		t.Errorf("Name = %q, want alice", id.Name())
	}
}

func TestJWTVerifierExpiredToken(t *testing.T) {
	pub, priv := generateTestKey(t)
	keyFile := writeTempKey(t, pub)
	defer os.Remove(keyFile)

	v, _ := NewJWTVerifier(JWTVerifierConfig{PublicKeyFile: keyFile})

	now := time.Now()
	token := signJWT(t, priv, testClaims{
		Sub: "user-1",
		Iss: "test-issuer",
		Iat: now.Add(-2 * time.Hour).Unix(),
		Exp: now.Add(-1 * time.Hour).Unix(), // 已过期
	})

	_, err := v.AuthenticateToken(context.Background(), token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestJWTVerifierWrongIssuer(t *testing.T) {
	pub, priv := generateTestKey(t)
	keyFile := writeTempKey(t, pub)
	defer os.Remove(keyFile)

	v, _ := NewJWTVerifier(JWTVerifierConfig{
		PublicKeyFile: keyFile,
		Issuer:        "expected-issuer",
	})

	now := time.Now()
	token := signJWT(t, priv, testClaims{
		Sub: "user-1",
		Iss: "wrong-issuer",
		Iat: now.Unix(),
		Exp: now.Add(1 * time.Hour).Unix(),
	})

	_, err := v.AuthenticateToken(context.Background(), token)
	if err == nil {
		t.Fatal("expected error for wrong issuer")
	}
}

func TestJWTVerifierWrongSignature(t *testing.T) {
	pub, _ := generateTestKey(t)
	_, wrongPriv := generateTestKey(t) // 另一对密钥
	keyFile := writeTempKey(t, pub)
	defer os.Remove(keyFile)

	v, _ := NewJWTVerifier(JWTVerifierConfig{PublicKeyFile: keyFile})

	now := time.Now()
	token := signJWT(t, wrongPriv, testClaims{ // 用错私钥签名
		Sub: "user-1",
		Iss: "test-issuer",
		Iat: now.Unix(),
		Exp: now.Add(1 * time.Hour).Unix(),
	})

	_, err := v.AuthenticateToken(context.Background(), token)
	if err == nil {
		t.Fatal("expected error for wrong signature")
	}
}

func TestJWTVerifierCache(t *testing.T) {
	pub, priv := generateTestKey(t)
	keyFile := writeTempKey(t, pub)
	defer os.Remove(keyFile)

	v, _ := NewJWTVerifier(JWTVerifierConfig{PublicKeyFile: keyFile})

	token := signJWT(t, priv, testClaims{
		Sub:      "user-1",
		Iss:      "test-issuer",
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(1 * time.Hour).Unix(),
		Username: "cached-user",
		Roles:    []string{"viewer"},
	})

	// 第一次：验签、缓存
	id1, err := v.AuthenticateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("first auth: %v", err)
	}
	// 第二次：缓存命中
	id2, err := v.AuthenticateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("second auth: %v", err)
	}

	if id1.Subject() != id2.Subject() {
		t.Error("cached result should match")
	}
}

func TestJWTVerifierMissingKeyConfig(t *testing.T) {
	_, err := NewJWTVerifier(JWTVerifierConfig{})
	if err == nil {
		t.Fatal("expected error for missing key config")
	}
}

func TestJWTClaimsUnmarshal(t *testing.T) {
	now := time.Now()
	jsonStr := `{"sub":"user-1","iss":"test","exp":` +
		formatUnix(now.Add(1*time.Hour).Unix()) + `,"iat":` +
		formatUnix(now.Unix()) + `}`

	var c JWTClaims
	if err := json.Unmarshal([]byte(jsonStr), &c); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if c.Subject != "user-1" {
		t.Errorf("Subject = %q", c.Subject)
	}
	if c.Issuer != "test" {
		t.Errorf("Issuer = %q", c.Issuer)
	}
}

func TestJWTClaimsToPrincipal(t *testing.T) {
	c := &JWTClaims{
		Subject:  "user-42",
		Username: "bob",
		Roles:    []string{"admin", "viewer"},
		Attrs:    map[string]string{"project_id": "prj-1"},
	}
	p := c.ToPrincipal()
	if p.ID != "user-42" {
		t.Errorf("ID = %q", p.ID)
	}
	if len(p.AllowedClusters) == 0 || p.AllowedClusters[0] != "*" {
		t.Errorf("AllowedClusters should default to [*], got %v", p.AllowedClusters)
	}
	if v, ok := p.Attr("project_id"); !ok || v != "prj-1" {
		t.Errorf("Attr project_id = %q", v)
	}
}

// ── 辅助 ────────────────────────────────────────────────────

func writeTempKey(t *testing.T, pub ed25519.PublicKey) string {
	t.Helper()
	f, err := os.CreateTemp("", "jwt-test-pub-*.pem")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.Write(pub); err != nil {
		t.Fatalf("write key: %v", err)
	}
	f.Close()
	return f.Name()
}

func formatUnix(ts int64) string {
	// json.Number 需要数字字符串
	s := ""
	n := ts
	if n == 0 {
		return "0"
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
