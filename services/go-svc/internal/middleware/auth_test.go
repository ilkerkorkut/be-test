package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestAuthMiddlewareValidatesHS256TokenAndSubClaim(t *testing.T) {
	app := fiber.New()
	auth := NewAuthMiddleware("shared-secret")
	app.Use(auth.RequireJWT)
	app.Get("/protected", func(c fiber.Ctx) error {
		return c.SendString(auth.UserID(c))
	})

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set(fiber.HeaderAuthorization, "Bearer "+testJWT("shared-secret", "user-1", time.Now().Add(time.Hour).Unix()))
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	invalidRequest := httptest.NewRequest(http.MethodGet, "/protected", nil)
	invalidRequest.Header.Set(fiber.HeaderAuthorization, "Bearer "+testJWT("other-secret", "user-1", time.Now().Add(time.Hour).Unix()))
	invalidResponse, err := app.Test(invalidRequest)
	if err != nil {
		t.Fatalf("test invalid request: %v", err)
	}
	if invalidResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("invalid token status = %d, want %d", invalidResponse.StatusCode, http.StatusUnauthorized)
	}
}

func testJWT(secret string, subject string, expiresAt int64) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"` + subject + `","exp":` + strconv.FormatInt(expiresAt, 10) + `}`))
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(header + "." + payload))
	return header + "." + payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
