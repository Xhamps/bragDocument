---
status: proposed
date: 2026-10-09
owner: Product
stakeholders: Engineering, Operations
related-adrs: [ADR-0008, ADR-0012]
---

# PRD-0008: Run all backend services with one command, with service-tagged JSON logs

## 1. Summary

Developers and operators can start the API, the Telegram bot and the export worker with a single command, and every log line those services write is a JSON object that names the service that wrote it. Running the backend gets simpler, and merged log output can be read and filtered by service.

## 2. Problem

- Running the full backend locally takes three commands in three terminals (`api`, `bot`, `worker`). Forgetting one shows up as a missing feature: Telegram messages go unanswered, or PDF exports sit queued forever.
- The three services write logs in the same shape and none of them says which service wrote a line. Once outputs are merged (one terminal, `docker compose logs`, a log collector), you cannot tell a bot error from an API error without guessing from the message text.
- Some lines are not JSON (for example the final error when a process exits), so log tooling that expects one JSON object per line drops or mangles them.

## 3. Goals

- One command starts api, bot and worker, and one Ctrl-C (or SIGTERM) stops all three cleanly.
- 100% of log lines written by the api, bot and worker identify their service.
- 100% of log lines are valid JSON when the JSON format is selected (the default).

## 4. Non-goals

- Replacing the per-service commands or the one-container-per-service Docker Compose setup ([ADR-0008](../adr/0008-docker-compose-for-local-provisioning.md)). Those stay for production-like runs.
- Running database migrations from the new command. Migrations stay a separate step.
- Metrics endpoints for the bot and the worker.
- Centralised log storage, dashboards or alerting.

## 5. Users and personas

- **Developer:** runs the backend on a laptop to build and test features end to end.
- **Operator:** runs the backend on a small host or a single container and reads logs to diagnose problems.

## 6. User stories

- As a developer, I want to start the whole backend with one command so that I don't keep three terminals open and don't forget a service.
- As a developer, I want one Ctrl-C to stop everything so that no process is left running in the background.
- As an operator, I want each log line to say which service wrote it so that I can filter the output to the bot, the worker or the API.
- As an operator, I want every log line to be JSON so that my log tooling parses all of it, including startup failures and exit errors.

## 7. Functional requirements

| ID | Requirement | Priority |
|---|---|---|
| FR-1 | The backend binary MUST provide a command that starts the api, the bot and the worker in one process. | Must |
| FR-2 | The existing `api`, `bot` and `worker` commands MUST keep working unchanged. | Must |
| FR-3 | A stop signal (SIGINT, SIGTERM) MUST stop all three services, each with the same graceful shutdown it has when run alone. | Must |
| FR-4 | If one service fails to start or stops with an error, the command MUST stop the other two and exit with a non-zero code that reports the failing service's error. | Must |
| FR-5 | Each service MUST keep its current behaviour for optional dependencies: without a Telegram token the bot idles, without an export key the worker idles, and the API still refuses to start without its required settings. | Must |
| FR-6 | Every log record written by the api, bot or worker MUST include a `service` field with the value `api`, `bot` or `worker`, both under the combined command and when a service runs alone. | Must |
| FR-7 | Logs written while handling an API request MUST carry `service` alongside the existing `request_id` and `tenant_id`. | Must |
| FR-8 | With the JSON log format (the default), every line written to the log stream, including the process exit error, MUST be a single valid JSON object. | Must |
| FR-9 | The plain-text log format MUST remain available for local use and SHOULD also show `service`. | Should |
| FR-10 | The command MUST NOT run database migrations. | Must |
| FR-11 | The README and the developer shortcuts (Makefile) SHOULD document the new command. | Should |

## 8. Non-functional requirements

- NFR-1: Startup and shutdown under the combined command are no slower than the slowest single service; shutdown completes within the configured shutdown timeout.
- NFR-2: Adding the `service` field adds no measurable latency to API requests.
- NFR-3: Logs MUST NOT gain new personal data; `service` is the only new field.
- NFR-4: Observability: the combined output can be filtered per service with a standard JSON tool (for example `jq 'select(.service=="bot")'`).

## 9. UX notes

No end-user screens change. Developer experience:

- `make run-all` (or the binary's new command) prints interleaved JSON lines from all three services, each tagged:

  ```json
  {"level":"INFO","msg":"api listening","service":"api"}
  {"level":"WARN","msg":"TELEGRAM_BOT_TOKEN not set; bot idle","service":"bot"}
  {"level":"INFO","msg":"worker started","service":"worker"}
  ```

- Error state: if the API cannot start (for example a missing required setting), its error is logged as JSON, the bot and worker stop, and the command exits non-zero.

## 10. Data

No business entities change. Log records gain one field, `service` (`api` | `bot` | `worker`).

## 11. Success metrics

| Metric | Target | Source |
|---|---|---|
| Log lines from api/bot/worker that carry `service` | 100% | `jq` over a local run with every feature exercised |
| Log lines that fail JSON parsing in JSON mode | 0 | Same run, `jq` exit status |
| Commands needed to run the full backend locally | 1 (plus migrations) | README |

## 12. Open questions

| Question | Owner | Due |
|---|---|---|
| Name of the command: `all`, `serve` or `dev`? | Engineering | Before implementation |
| Should the three services share one database pool and cache client in the combined mode, or keep one each? | Engineering | Before implementation |
| Should Docker Compose offer a single-container profile that uses the combined command? | Operations | Later |

## 13. Decisions log

| Date | Decision | Why |
|---|---|---|
| 2026-10-09 | Keep per-service commands and containers | Production-like runs scale and restart services independently |
| 2026-10-09 | Migrations stay out of the combined command | The app's database role must not need schema rights |
| 2026-10-09 | One failing service stops the whole command | A half-running backend hides failures; failing loudly is easier to notice |
