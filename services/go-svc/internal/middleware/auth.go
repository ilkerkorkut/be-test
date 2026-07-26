package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/flotaro/be-test/services/go-svc/internal/response"
	"github.com/gofiber/fiber/v3"
)

const userIDLocalKey = "user_id"

type AuthMiddleware interface {
	RequireJWT(c fiber.Ctx) error
	UserID(c fiber.Ctx) string
}

type authMiddleware struct {
	secret []byte
}

func NewAuthMiddleware(secret string) AuthMiddleware {
	return &authMiddleware{secret: []byte(secret)}
}

func (m *authMiddleware) RequireJWT(c fiber.Ctx) error {
	parts := strings.Fields(c.Get(fiber.HeaderAuthorization))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return unauthorized(c)
	}

	userID, ok := m.validate(parts[1])
	if !ok {
		return unauthorized(c)
	}

	c.Locals(userIDLocalKey, userID)
	return c.Next()
}

func (m *authMiddleware) UserID(c fiber.Ctx) string {
	userID, _ := c.Locals(userIDLocalKey).(string)
	return userID
}

func (m *authMiddleware) validate(token string) (string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", false
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", false
	}
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return "", false
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	var header struct {
		Algorithm string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Algorithm != "HS256" {
		return "", false
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}
	var claims struct {
		Subject   string  `json:"sub"`
		ExpiresAt float64 `json:"exp"`
		NotBefore float64 `json:"nbf"`
		IssuedAt  float64 `json:"iat"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil || claims.Subject == "" {
		return "", false
	}
	now := float64(time.Now().Unix())
	if claims.ExpiresAt <= 0 || now >= claims.ExpiresAt {
		return "", false
	}
	if claims.NotBefore > 0 && now < claims.NotBefore {
		return "", false
	}

	return claims.Subject, true
}

func unauthorized(c fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(response.NewErrorResponse("unauthorized", "valid bearer token required", nil))
}
