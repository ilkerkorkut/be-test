package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const requestIDHeader = "X-Request-ID"

func RequestLogger(logger *zap.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		requestID := c.Get(requestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set(requestIDHeader, requestID)

		startedAt := time.Now()
		err := c.Next()
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("duration", time.Since(startedAt)),
		}
		if err != nil {
			logger.Error("http request failed", append(fields, zap.Error(err))...)
			return err
		}

		if c.Response().StatusCode() >= fiber.StatusInternalServerError {
			logger.Error("http request completed", fields...)
		} else if c.Response().StatusCode() >= fiber.StatusBadRequest {
			logger.Warn("http request completed", fields...)
		} else {
			logger.Info("http request completed", fields...)
		}
		return nil
	}
}
