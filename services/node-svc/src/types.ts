export type User = {
  id: string;
  email: string;
  pinHash: string;
  displayName: string;
  currency: string;
};

export type Profile = Pick<User, "id" | "email" | "displayName" | "currency">;

export type NotificationChannel = "IN_APP" | "EMAIL";

export type Notification = {
  id: string;
  userId: string;
  title: string;
  body: string;
  channel: NotificationChannel;
  readAt: string | null;
  createdAt: string;
};

export type NotificationDraft = Omit<Notification, "id" | "readAt" | "createdAt">;

export type BetSettledEvent = {
  eventId: string;
  betId: string;
  userId: string;
  status: "WON" | "LOST";
  payout: number;
  settledAt: string;
};
