import pino, { type Logger } from "pino";

export function createLogger(): Logger {
  return pino({
    level: process.env.LOG_LEVEL ?? "info",
    base: { service: "node-svc" },
    redact: {
      paths: ["req.headers.authorization", "req.body.pin", "token"],
      censor: "[REDACTED]",
    },
  });
}
