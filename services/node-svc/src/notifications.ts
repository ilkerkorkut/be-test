import { randomUUID } from "node:crypto";
import type { Logger } from "pino";

import { NotificationStore } from "./store.js";
import type { BetSettledEvent, Notification, NotificationChannel, NotificationDraft } from "./types.js";

export interface NotificationAdapter {
  readonly channel: NotificationChannel;
  send(notification: Notification): Promise<void>;
}

export class InAppNotificationAdapter implements NotificationAdapter {
  readonly channel = "IN_APP" as const;

  constructor(private readonly store: NotificationStore) {}

  async send(notification: Notification): Promise<void> {
    await this.store.save(notification);
  }
}

export type EmailTransport = {
  send(input: Pick<Notification, "userId" | "title" | "body">): Promise<void>;
};

export class LogEmailTransport implements EmailTransport {
  constructor(private readonly logger?: Logger) {}

  async send(input: Pick<Notification, "userId" | "title" | "body">): Promise<void> {
    this.logger?.info({ user_id: input.userId, title: input.title }, "email notification queued");
  }
}

export class EmailNotificationAdapter implements NotificationAdapter {
  readonly channel = "EMAIL" as const;

  constructor(
    private readonly store: NotificationStore,
    private readonly transport: EmailTransport,
  ) {}

  async send(notification: Notification): Promise<void> {
    await this.transport.send(notification);
    await this.store.save(notification);
  }
}

export class NotificationService {
  private readonly adapters: Map<NotificationChannel, NotificationAdapter>;
  private readonly processedEvents = new Set<string>();

  constructor(
    adapters: NotificationAdapter[],
    private readonly logger?: Logger,
    private readonly store?: NotificationStore,
  ) {
    this.adapters = new Map(adapters.map((adapter) => [adapter.channel, adapter]));
  }

  async send(draft: NotificationDraft): Promise<Notification> {
    const adapter = this.adapters.get(draft.channel);
    if (!adapter) {
      throw new Error(`no notification adapter for channel ${draft.channel}`);
    }

    const notification: Notification = {
      ...draft,
      id: randomUUID(),
      readAt: null,
      createdAt: new Date().toISOString(),
    };
    await adapter.send(notification);
    this.logger?.info({ notification_id: notification.id, user_id: notification.userId, channel: notification.channel }, "notification delivered");
    return notification;
  }

  async handleBetSettled(event: BetSettledEvent): Promise<void> {
    if (!this.store && this.processedEvents.has(event.eventId)) {
      this.logger?.debug({ event_id: event.eventId, bet_id: event.betId }, "duplicate bet settled event ignored");
      return;
    }
    this.logger?.info({ event_id: event.eventId, bet_id: event.betId, user_id: event.userId, status: event.status }, "processing bet settled notification");

    const won = event.status === "WON";
    const notification = {
      id: randomUUID(),
      userId: event.userId,
      channel: "IN_APP" as const,
      title: won ? "Big win!" : "Bet settled",
      body: won
        ? `Your bet ${event.betId} won ${event.payout.toFixed(2)}.`
        : `Your bet ${event.betId} has settled without a payout.`,
      readAt: null,
      createdAt: new Date().toISOString(),
    };
    if (this.store) {
      if (!await this.store.saveSettled(event.eventId, notification)) {
        this.logger?.debug({ event_id: event.eventId, bet_id: event.betId }, "duplicate bet settled event ignored");
        return;
      }
    } else {
      await this.send(notification);
      this.processedEvents.add(event.eventId);
    }
  }
}
