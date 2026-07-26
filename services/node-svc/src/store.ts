import { randomUUID } from "node:crypto";
import { mkdir, readFile, rename, writeFile } from "node:fs/promises";
import { dirname } from "node:path";

import type { Notification, Profile, User } from "./types.js";

export class UserStore {
  private readonly usersByEmail = new Map<string, User>();
  private readonly profilesByID = new Map<string, Profile>();

  constructor(users: User[]) {
    for (const user of users) {
      this.usersByEmail.set(user.email.toLowerCase(), user);
      this.profilesByID.set(user.id, {
        id: user.id,
        email: user.email,
        displayName: user.displayName,
        currency: user.currency,
      });
    }
  }

  findByEmail(email: string): User | undefined {
    return this.usersByEmail.get(email.toLowerCase());
  }

  findProfile(id: string): Profile | undefined {
    return this.profilesByID.get(id);
  }
}

export class NotificationStore {
  private readonly notifications = new Map<string, Notification>();
  private readonly processedEvents = new Set<string>();
  private writeChain = Promise.resolve();

  constructor(notifications: Notification[], private readonly statePath?: string, processedEventIDs: string[] = []) {
    for (const notification of notifications) {
      this.notifications.set(notification.id, notification);
    }
    for (const eventID of processedEventIDs) {
      this.processedEvents.add(eventID);
    }
  }

  static async open(notifications: Notification[], statePath: string): Promise<NotificationStore> {
    try {
      const content = await readFile(statePath, "utf8");
      const state = JSON.parse(content) as { notifications?: Notification[]; processedEventIDs?: string[] };
      return new NotificationStore(state.notifications ?? notifications, statePath, state.processedEventIDs ?? []);
    } catch (error: unknown) {
      if (isMissingFile(error)) {
        return new NotificationStore(notifications, statePath);
      }
      throw error;
    }
  }

  async save(notification: Notification): Promise<void> {
    this.notifications.set(notification.id, notification);
    try {
      await this.persist();
    } catch (error) {
      this.notifications.delete(notification.id);
      throw error;
    }
  }

  async saveSettled(eventID: string, notification: Notification): Promise<boolean> {
    if (this.processedEvents.has(eventID)) {
      return false;
    }

    this.notifications.set(notification.id, notification);
    this.processedEvents.add(eventID);
    try {
      await this.persist();
      return true;
    } catch (error) {
      this.notifications.delete(notification.id);
      this.processedEvents.delete(eventID);
      throw error;
    }
  }

  listForUser(userID: string, limit: number): Notification[] {
    return [...this.notifications.values()]
      .filter((notification) => notification.userId === userID)
      .sort((left, right) => right.createdAt.localeCompare(left.createdAt))
      .slice(0, limit);
  }

  async markRead(userID: string, notificationID: string, readAt: string): Promise<boolean> {
    const notification = this.notifications.get(notificationID);
    if (!notification || notification.userId !== userID) {
      return false;
    }

    if (notification.readAt) {
      return true;
    }
    notification.readAt = readAt;
    try {
      await this.persist();
    } catch (error) {
      notification.readAt = null;
      throw error;
    }
    return true;
  }

  private async persist(): Promise<void> {
    if (!this.statePath) {
      return;
    }
    const statePath = this.statePath;

    this.writeChain = this.writeChain.catch(() => undefined).then(async () => {
      const state = JSON.stringify({
        notifications: [...this.notifications.values()],
        processedEventIDs: [...this.processedEvents],
      });
      await mkdir(dirname(statePath), { recursive: true });
      const temporaryPath = `${statePath}.${randomUUID()}.tmp`;
      await writeFile(temporaryPath, state, "utf8");
      await rename(temporaryPath, statePath);
    });
    return this.writeChain;
  }
}

function isMissingFile(error: unknown): error is NodeJS.ErrnoException {
  return typeof error === "object" && error !== null && "code" in error && error.code === "ENOENT";
}
