import { randomUUID } from "node:crypto";

import express, { type NextFunction, type Request, type Response } from "express";
import pino, { type Logger } from "pino";

import { AuthenticationService } from "./auth.js";
import { NotificationStore } from "./store.js";

export type AppDependencies = {
  auth: AuthenticationService;
  notifications: NotificationStore;
  allowedOrigins?: string[];
  logger?: Logger;
};

function parseLimit(value: unknown): number | undefined {
  if (typeof value !== "string" || !/^\d+$/.test(value)) {
    return undefined;
  }
  const limit = Number(value);
  return limit >= 1 && limit <= 100 ? limit : undefined;
}

export function createApp(dependencies: AppDependencies) {
  const app = express();
  const logger = dependencies.logger ?? pino({ enabled: false });
  const allowedOrigins = new Set(dependencies.allowedOrigins ?? [
    "http://localhost:3000",
    "http://127.0.0.1:3000",
  ]);
  app.use(express.json());
  app.use((request, response, next) => {
    const requestID = request.header("x-request-id") || randomUUID();
    const startedAt = process.hrtime.bigint();
    response.setHeader("X-Request-ID", requestID);
    response.locals.requestID = requestID;
    response.on("finish", () => {
      const durationMs = Number(process.hrtime.bigint() - startedAt) / 1_000_000;
      const fields = {
        request_id: requestID,
        method: request.method,
        path: request.path,
        status: response.statusCode,
        duration_ms: Number(durationMs.toFixed(3)),
      };
      if (response.statusCode >= 500) {
        logger.error(fields, "http request completed");
      } else if (response.statusCode >= 400) {
        logger.warn(fields, "http request completed");
      } else {
        logger.info(fields, "http request completed");
      }
    });
    next();
  });
  app.use((request, response, next) => {
    const origin = request.header("origin");
    if (origin && allowedOrigins.has(origin)) {
      response.setHeader("Access-Control-Allow-Origin", origin);
      response.setHeader("Vary", "Origin");
      response.setHeader("Access-Control-Allow-Headers", "Authorization, Content-Type");
      response.setHeader("Access-Control-Allow-Methods", "GET, POST, OPTIONS");
    } else if (origin) {
      logger.warn({ request_id: response.locals.requestID, origin }, "cors origin rejected");
    }
    if (request.method === "OPTIONS") {
      response.sendStatus(204);
      return;
    }
    next();
  });

  app.get("/health", (_request, response) => {
    response.json({ status: "ok", service: "node-svc" });
  });

  app.post("/auth/login", async (request, response) => {
    const { email, pin } = request.body as { email?: unknown; pin?: unknown };
    if (typeof email !== "string" || typeof pin !== "string" || !email || !pin) {
      logger.warn({ request_id: response.locals.requestID }, "login rejected due to invalid request");
      response.status(400).json({ error: "email and pin are required" });
      return;
    }

    const session = await dependencies.auth.login(email, pin);
    if (!session) {
      logger.warn({ request_id: response.locals.requestID }, "login rejected");
      response.status(401).json({ error: "invalid email or pin" });
      return;
    }
    logger.info({ request_id: response.locals.requestID, user_id: session.userId }, "login succeeded");
    response.json(session);
  });

  const requireUser = (request: Request, response: Response, next: NextFunction) => {
    const userID = dependencies.auth.authenticate(request.header("authorization"));
    if (!userID) {
      logger.warn({ request_id: response.locals.requestID }, "authenticated request rejected");
      response.status(401).json({ error: "unauthorized" });
      return;
    }
    response.locals.userID = userID;
    next();
  };

  app.get("/me", requireUser, (_request, response) => {
    const profile = dependencies.auth.profile(response.locals.userID as string);
    if (!profile) {
      logger.warn({ request_id: response.locals.requestID, user_id: response.locals.userID }, "profile missing for authenticated user");
      response.status(401).json({ error: "unauthorized" });
      return;
    }
    response.json(profile);
  });

  app.get("/notifications", requireUser, (request, response) => {
    const limit = request.query.limit === undefined ? 20 : parseLimit(request.query.limit);
    if (!limit) {
      response.status(400).json({ error: "limit must be an integer between 1 and 100" });
      return;
    }

    const userID = response.locals.userID as string;
    response.json(dependencies.notifications.listForUser(userID, limit).map((notification) => ({
      id: notification.id,
      title: notification.title,
      body: notification.body,
      readAt: notification.readAt,
      createdAt: notification.createdAt,
    })));
  });

  app.post("/notifications/:id/read", requireUser, (request, response) => {
    const userID = response.locals.userID as string;
    if (!dependencies.notifications.markRead(userID, request.params.id, new Date().toISOString())) {
      logger.warn({ request_id: response.locals.requestID, user_id: userID, notification_id: request.params.id }, "notification read rejected");
      response.status(404).json({ error: "notification not found" });
      return;
    }
    logger.info({ request_id: response.locals.requestID, user_id: userID, notification_id: request.params.id }, "notification marked read");
    response.json({ ok: true });
  });

  app.use((error: unknown, request: Request, response: Response, _next: NextFunction) => {
    logger.error({ err: error, request_id: response.locals.requestID, method: request.method, path: request.path }, "unhandled http error");
    response.status(500).json({ error: "internal server error" });
  });

  return app;
}
