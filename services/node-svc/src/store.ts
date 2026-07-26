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

  constructor(notifications: Notification[]) {
    for (const notification of notifications) {
      this.notifications.set(notification.id, notification);
    }
  }

  save(notification: Notification): void {
    this.notifications.set(notification.id, notification);
  }

  listForUser(userID: string, limit: number): Notification[] {
    return [...this.notifications.values()]
      .filter((notification) => notification.userId === userID)
      .sort((left, right) => right.createdAt.localeCompare(left.createdAt))
      .slice(0, limit);
  }

  markRead(userID: string, notificationID: string, readAt: string): boolean {
    const notification = this.notifications.get(notificationID);
    if (!notification || notification.userId !== userID) {
      return false;
    }

    if (!notification.readAt) {
      notification.readAt = readAt;
    }
    return true;
  }
}
