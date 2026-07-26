import assert from "node:assert/strict";
import { once } from "node:events";
import { mkdtemp, rm } from "node:fs/promises";
import { createServer, type Server } from "node:http";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import jwt from "jsonwebtoken";

import { createApp } from "../src/app.js";
import { AuthenticationService } from "../src/auth.js";
import { EmailNotificationAdapter, InAppNotificationAdapter, type EmailTransport, NotificationService } from "../src/notifications.js";
import { NotificationStore, UserStore } from "../src/store.js";
import type { Notification, User } from "../src/types.js";

const user: User = {
  id: "user-1",
  email: "mia@example.com",
  pinHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
  displayName: "Mia Thomas",
  currency: "GBP",
};

const seededNotifications: Notification[] = [
  {
    id: "notification-older",
    userId: user.id,
    title: "Older",
    body: "Older notification",
    channel: "IN_APP",
    readAt: null,
    createdAt: "2026-01-01T00:00:00.000Z",
  },
  {
    id: "notification-newer",
    userId: user.id,
    title: "Newer",
    body: "Newer notification",
    channel: "IN_APP",
    readAt: null,
    createdAt: "2026-01-02T00:00:00.000Z",
  },
];

async function runningServer(): Promise<{ server: Server; url: string }> {
  const users = new UserStore([user]);
  const notifications = new NotificationStore(seededNotifications);
  const app = createApp({
    auth: new AuthenticationService(users, "shared-secret", "1234"),
    notifications,
  });
  const server = createServer(app).listen(0);
  await once(server, "listening");
  const address = server.address();
  assert.ok(address && typeof address !== "string");
  return { server, url: `http://127.0.0.1:${address.port}` };
}

test("profile endpoints validate HS256 JWTs and expose the authenticated profile", async (context) => {
  const { server, url } = await runningServer();
  context.after(() => server.close());

  const unauthorized = await fetch(`${url}/me`);
  assert.equal(unauthorized.status, 401);

  const login = await fetch(`${url}/auth/login`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ email: user.email, pin: "1234" }),
  });
  assert.equal(login.status, 200);
  const session = await login.json() as { token: string; userId: string };
  assert.equal(session.userId, user.id);
  const decoded = jwt.decode(session.token, { complete: true });
  assert.ok(decoded && typeof decoded.payload === "object");
  assert.equal(decoded.header.alg, "HS256");
  assert.equal(decoded.payload.sub, user.id);
  assert.equal(decoded.payload.exp! - decoded.payload.iat!, 24 * 60 * 60);

  const profile = await fetch(`${url}/me`, {
    headers: { authorization: `Bearer ${session.token}` },
  });
  assert.deepEqual(await profile.json(), {
    id: user.id,
    email: user.email,
    displayName: user.displayName,
    currency: user.currency,
  });

  const wrongSecret = session.token.slice(0, -1) + (session.token.endsWith("a") ? "b" : "a");
  const invalid = await fetch(`${url}/me`, {
    headers: { authorization: `Bearer ${wrongSecret}` },
  });
  assert.equal(invalid.status, 401);

  const wrongAlgorithm = jwt.sign({}, "shared-secret", {
    algorithm: "HS384",
    subject: user.id,
  });
  const unsupportedAlgorithm = await fetch(`${url}/me`, {
    headers: { authorization: `Bearer ${wrongAlgorithm}` },
  });
  assert.equal(unsupportedAlgorithm.status, 401);
});

test("CORS allows the configured UI origin and answers login preflight", async (context) => {
  const users = new UserStore([user]);
  const app = createApp({
    auth: new AuthenticationService(users, "shared-secret", "1234"),
    notifications: new NotificationStore([]),
    allowedOrigins: ["http://localhost:13000"],
  });
  const server = createServer(app).listen(0);
  await once(server, "listening");
  context.after(() => server.close());
  const address = server.address();
  assert.ok(address && typeof address !== "string");

  const preflight = await fetch(`http://127.0.0.1:${address.port}/auth/login`, {
    method: "OPTIONS",
    headers: {
      origin: "http://localhost:13000",
      "access-control-request-method": "POST",
      "access-control-request-headers": "content-type",
    },
  });
  assert.equal(preflight.status, 204);
  assert.equal(preflight.headers.get("access-control-allow-origin"), "http://localhost:13000");
});

test("notifications are newest first, scoped to the JWT user, and can be marked read", async (context) => {
  const { server, url } = await runningServer();
  context.after(() => server.close());
  const login = await fetch(`${url}/auth/login`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ email: user.email, pin: "1234" }),
  });
  const { token } = await login.json() as { token: string };
  const headers = { authorization: `Bearer ${token}` };

  const list = await fetch(`${url}/notifications?limit=1`, { headers });
  assert.equal(list.status, 200);
  const notifications = await list.json() as Notification[];
  assert.equal(notifications.length, 1);
  assert.equal(notifications[0].id, "notification-newer");

  const markRead = await fetch(`${url}/notifications/notification-newer/read`, {
    method: "POST",
    headers,
  });
  assert.deepEqual(await markRead.json(), { ok: true });

  const afterRead = await fetch(`${url}/notifications?limit=1`, { headers });
  const [notification] = await afterRead.json() as Notification[];
  assert.ok(notification.readAt);
});

test("notification adapters dispatch each supported channel", async () => {
  const store = new NotificationStore([]);
  const sentEmails: string[] = [];
  const emailTransport: EmailTransport = {
    async send(input) {
      sentEmails.push(input.title);
    },
  };
  const notifications = new NotificationService([
    new InAppNotificationAdapter(store),
    new EmailNotificationAdapter(store, emailTransport),
  ]);

  await notifications.send({ userId: user.id, channel: "IN_APP", title: "In app", body: "Message" });
  await notifications.send({ userId: user.id, channel: "EMAIL", title: "Email", body: "Message" });

  assert.equal(store.listForUser(user.id, 10).length, 2);
  assert.deepEqual(sentEmails, ["Email"]);
});

test("notification state and processed settlement events survive a restart", async (context) => {
  const directory = await mkdtemp(join(tmpdir(), "node-svc-notifications-"));
  context.after(async () => rm(directory, { recursive: true, force: true }));
  const statePath = join(directory, "notifications.json");
  const notification: Notification = {
    id: "settlement-notification",
    userId: user.id,
    title: "Bet settled",
    body: "Your bet settled.",
    channel: "IN_APP",
    readAt: null,
    createdAt: "2026-01-03T00:00:00.000Z",
  };

  const firstStore = await NotificationStore.open([], statePath);
  assert.equal(await firstStore.saveSettled("settlement-event", notification), true);
  assert.equal(await firstStore.markRead(user.id, notification.id, "2026-01-03T01:00:00.000Z"), true);

  const restartedStore = await NotificationStore.open([], statePath);
  const [persisted] = restartedStore.listForUser(user.id, 10);
  assert.equal(persisted.id, notification.id);
  assert.equal(persisted.readAt, "2026-01-03T01:00:00.000Z");
  assert.equal(await restartedStore.saveSettled("settlement-event", { ...notification, id: "duplicate" }), false);
});
