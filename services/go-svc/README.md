# go-svc

This service owns the Odds proxy and wallet/bet engine. See `/README.md` (top of repo) for the full contract.

## Architecture decisions

Go owns the Odds proxy, wallet and bets because wallet mutations need a small, explicit concurrency boundary. Wallet balances, bets, and ledger values are stored as integer cents in one lock-protected JSON state document. A bet debit, bet insert, and outbox insert commit together, preventing overspend and preserving the settlement command. Wallets start with a zero USD balance and expose atomic `Credit` and `Debit` operations through the wallet service interface.

The Node service owns login/profile and notifications, which are primarily JSON and I/O workloads. It issues HS256 JWTs using the shared `JWT_SECRET`; this service validates the signature and uses the `sub` claim as the user ID for every protected request.

Odds cache policy lives in the sports service: sports for five minutes, events for 30 seconds, and event odds for 15 seconds. Local development defaults to an in-memory TTL cache; setting `REDIS_URL` uses Redis so replicas share cached Odds API responses. The client logs the upstream quota headers for every cache miss. The API publishes `bets.placed.v1` via a transactional outbox; the worker settles it after 30 seconds with a 45% win chance, then publishes `bets.settled.v1` for Node notifications. Consumers use event IDs for idempotency.

- The endpoints this service must expose (depends on the split you chose).
- The concurrent-safety requirements for the wallet.
- The Odds API caching expectations.

## Run locally

```
go run ./cmd/api
# → http://localhost:8081/health
```

## Notes

- Copy the root `.env.example` to `.env` and set a shared `JWT_SECRET` and `ODDS_API_KEY`.
- `make go-run`, `make go-test-race`, `make go-build`, and `make go-prod` cover the Go development and production workflows.
- `make go-worker-run` and `make go-worker-build` run and build the separate settlement worker binary.
- Set `REDIS_URL=redis://localhost:6379/0` to use Redis locally. Docker Compose configures the shared Redis cache automatically.
- API and worker share `data/go-svc.json`. Its file lock and atomic replace make each wallet/bet/outbox transition indivisible across both binaries.
- Wallets are created lazily at a zero USD balance. Credits are intentionally available in the wallet service interface for a future deposit/admin flow; no uncontracted public deposit endpoint is exposed.

## Kafka contracts

`bets.placed.v1` carries `{ eventId, betId, userId, placedAt, settleAt }` and is keyed by `betId`. The API writes it to the JSON outbox in the same transaction as the wallet debit and bet creation. The worker commits its Kafka offset only after the file transaction settles the bet.

`bets.settled.v1` carries `{ eventId, betId, userId, status, payout, settledAt }` and is keyed by `userId`. Node should use `eventId` to deduplicate notifications. Outbox publication is at-least-once; consumers must be idempotent.
