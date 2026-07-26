import { Kafka, logLevel } from "kafkajs";
import type { Logger } from "pino";

import { NotificationService } from "./notifications.js";
import type { BetSettledEvent } from "./types.js";

const betSettledTopic = "bets.settled.v1";

export async function startBetSettledConsumer(
  brokers: string[],
  notifications: NotificationService,
  logger: Logger,
): Promise<() => Promise<void>> {
  const kafka = new Kafka({ brokers, logLevel: logLevel.NOTHING });
  const consumer = kafka.consumer({ groupId: "node-svc-notifications-v1" });
  try {
    await consumer.connect();
    logger.info({ brokers, group_id: "node-svc-notifications-v1" }, "kafka notification consumer connected");
    await consumer.subscribe({ topic: betSettledTopic, fromBeginning: true });
    logger.info({ topic: betSettledTopic }, "kafka notification consumer subscribed");
    await consumer.run({
      eachMessage: async ({ message }) => {
        if (!message.value) {
          logger.warn({ topic: betSettledTopic }, "empty kafka notification message ignored");
          return;
        }
        let event: BetSettledEvent;
        try {
          event = JSON.parse(message.value.toString()) as BetSettledEvent;
        } catch (error) {
          logger.error({ err: error, topic: betSettledTopic }, "invalid kafka notification payload");
          throw error;
        }
        if (!event.eventId || !event.betId || !event.userId || !event.settledAt) {
          logger.error({ topic: betSettledTopic }, "invalid bet settled event");
          throw new Error("invalid bets.settled.v1 event");
        }
        logger.info({ event_id: event.eventId, bet_id: event.betId, user_id: event.userId, status: event.status }, "bet settled event received");
        await notifications.handleBetSettled(event);
      },
    });
  } catch (error) {
    await consumer.disconnect().catch(() => undefined);
    throw error;
  }

  return async () => {
    await consumer.disconnect();
    logger.info({ group_id: "node-svc-notifications-v1" }, "kafka notification consumer disconnected");
  };
}
