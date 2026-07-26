import { readFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import { createApp } from "./app.js";
import { AuthenticationService } from "./auth.js";
import { startBetSettledConsumer } from "./kafka.js";
import { createLogger } from "./logger.js";
import { EmailNotificationAdapter, InAppNotificationAdapter, LogEmailTransport, NotificationService } from "./notifications.js";
import { NotificationStore, UserStore } from "./store.js";
import type { Notification, User } from "./types.js";

const currentDirectory = dirname(fileURLToPath(import.meta.url));

function dataDirectory(): string {
  return process.env.DATA_DIRECTORY ?? resolve(currentDirectory, "../../../data");
}

function allowedOrigins(): string[] {
  return (process.env.CORS_ORIGINS ?? "http://localhost:3000,http://127.0.0.1:3000")
    .split(",")
    .map((origin) => origin.trim())
    .filter(Boolean);
}

async function loadJSON<T>(fileName: string): Promise<T> {
  return JSON.parse(await readFile(resolve(dataDirectory(), fileName), "utf8")) as T;
}

export async function start(): Promise<void> {
  const logger = createLogger();
  const jwtSecret = process.env.JWT_SECRET;
  if (!jwtSecret) {
    throw new Error("JWT_SECRET is required");
  }

  const [users, seededNotifications] = await Promise.all([
    loadJSON<User[]>("users.json"),
    loadJSON<Notification[]>("notifications.json"),
  ]);
  logger.info({ users: users.length, notifications: seededNotifications.length }, "service data loaded");
  const userStore = new UserStore(users);
  const notificationStore = new NotificationStore(seededNotifications);
  const notificationService = new NotificationService([
    new InAppNotificationAdapter(notificationStore),
    new EmailNotificationAdapter(notificationStore, new LogEmailTransport(logger)),
  ], logger);
  const app = createApp({
    auth: new AuthenticationService(userStore, jwtSecret, process.env.SEEDED_USER_PIN ?? "1234"),
    notifications: notificationStore,
    allowedOrigins: allowedOrigins(),
    logger,
  });

  const port = Number(process.env.PORT ?? 8082);
  const server = app.listen(port, () => logger.info({ port }, "node service listening"));
  let stopConsumer: (() => Promise<void>) | undefined;
  const brokers = process.env.KAFKA_BROKERS?.split(",").filter(Boolean) ?? [];
  if (brokers.length > 0) {
    stopConsumer = await startBetSettledConsumer(brokers, notificationService, logger);
  } else {
    logger.warn("kafka notification consumer disabled because no brokers were configured");
  }

  const shutdown = async () => {
    logger.info("node service shutting down");
    await stopConsumer?.();
    server.close();
  };
  process.once("SIGINT", shutdown);
  process.once("SIGTERM", shutdown);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  start().catch((error: unknown) => {
    createLogger().error({ err: error }, "node service failed to start");
    process.exitCode = 1;
  });
}
