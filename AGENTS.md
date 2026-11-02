# StreamShop — agent notes

This file is the **contract** for anyone (human or agent) changing this repo via the software factory. Read it before coding.

`CLAUDE.md` is swamp-managed tooling guidance. **This file** owns factory scope, tickets, and the quality gate.

## Product

**StreamShop** — local-first distributed systems showcase: polyglot microservices, event streaming (Redpanda), and multi-store data (Postgres, Mongo, Redis, ClickHouse, MinIO). HTTP entry is Traefik on `:8080`.

Day-to-day stack: `just up && just seed && just smoke`.

## Services

| Service | Lang | Role |
|---------|------|------|
| `order-api` | Go | Orders, outbox, idempotency |
| `catalog-api` | TypeScript | Catalog + Redis cache + MinIO images |
| `analytics-api` | Python | Order detail + ClickHouse aggregates |
| `event-processor` | Rust | Consume order events → Postgres/ClickHouse |
| `agent-gateway` | Rust | Ops agent tools (admin) |
| `storefront-ui` | TypeScript | Customer UI |
| `admin-ui` | TypeScript | Admin / agent UI |

Do **not** add services, public routes, or infra outside the active ticket.

## Tickets (factory input)

Work orders live in [GitHub Project 10 — StreamShop Distributed System](https://github.com/users/ianckc/projects/10).

There is **no** local `tickets/` folder.

| Project Status | Meaning |
|----------------|---------|
| Backlog | Not factory-ready |
| **Ready** | Pickable on the factory line |
| **In progress** | A `run-ticket` job is implementing it |
| In review | Human / PR review |
| **Done** | Shipped + bookkeeping finished |

File new work with the **Factory ticket** GitHub issue form (`.github/ISSUE_TEMPLATE/factory-ticket.yml`), add the issue to Project 10, then set Status = **Ready** when it is pickable.

## Hard rules

1. **Work only from Project tickets** that are **Ready** or **In progress**. Implement only Goal / Acceptance; respect Out of scope. Do not invent scope.
2. **Every behavior change** ships with tests in the **Affected services** listed on the issue (same bar as that service’s CI).
3. **`just gate`** must stay green for those services (`just gate order-api,catalog-api`). Red gate = not done.
4. **CI is the merge inspector** — path-filtered workflows under `.github/workflows/`. Red CI = do not merge.
5. When a ticket is finished: set Project Status = **Done**, check off acceptance on the issue, append a short note to [`factory/loop-notes.md`](factory/loop-notes.md).

## How to run one factory job (kickoff)

**Preferred (Swamp line):**

```bash
swamp model method run list-ready-tickets execute
swamp workflow run run-ticket --input issue_number=N --input services=order-api,catalog-api
# At the coding pause: implement the issue, then:
swamp workflow approve run-ticket implement-in-cursor --run <run-id>
swamp workflow resume run-ticket --run <run-id>
# After green gate, finish bookkeeping and approve bookkeeping-reminder.
```

**DIY fallback (Cursor paste):**

```text
Implement only GitHub issue #N from Project 10 (Status Ready / In progress).
Follow AGENTS.md strictly. Do not invent scope.
When done: set Project Status to Done, check acceptance criteria on the issue,
append factory/loop-notes.md, and leave just gate green for the affected services.
```

### Agent workflow checklist

1. Confirm the issue is on Project 10 with Status **Ready** (or already **In progress**).
2. Read Goal, Affected services, Acceptance, Out of scope, Test plan on the issue body.
3. Set Status to **In progress** when you start.
4. Implement **only** what the ticket asks; add or extend tests in those services.
5. Run the gate: `just gate <affected>` **or** `swamp model method run factory-gate execute`.
6. For cross-service / pipeline tickets, also run `just smoke` (or `swamp model method run smoke-gate execute`) when the stack is up.
7. Bookkeeping per hard rule 5.

## Quality gate

- Local: `just gate` / `just gate order-api,catalog-api` (see `scripts/factory-gate.sh`)
- Swamp model (versioned output): `swamp model method run factory-gate execute`
- Optional elevated check (Compose up): `just smoke` / `swamp model method run smoke-gate execute`
- CI: per-service workflows under `.github/workflows/`

## Agent control panel (Swamp)

Prefer these **approved** Swamp operations for factory ops (also mirrored in `.cursor/rules/factory-control-panel.mdc` once present):

| Intent | Command |
|--------|---------|
| List Ready tickets | `swamp model method run list-ready-tickets execute` |
| Show this contract | `swamp model method run show-agents-contract execute` |
| Check one ticket | `swamp model method run check-ticket-ready execute` (env `ISSUE_NUMBER`) |
| Run quality gate | `swamp model method run factory-gate execute` (env `SERVICES`) |
| Optional smoke gate | `swamp model method run smoke-gate execute` |
| Run one factory job | `swamp workflow run run-ticket --input issue_number=N --input services=…` |

Cursor remains the coding worker. Swamp is the control panel + line.

## Layout (factory-relevant)

- `factory/` — factory meta (loop notes, how-to); not the Docusaurus site
- `factory/loop-notes.md` — what each factory loop taught us
- `services/*` — product code and unit tests
- `scripts/smoke-test.sh` — end-to-end happy path
- `justfile` — local recipes (`up`, `seed`, `smoke`, `gate`, …)
- `scripts/factory-gate.sh` — CI-parity local gate
- `.github/ISSUE_TEMPLATE/factory-ticket.yml` — issue form for new tickets
- `.github/workflows/*` — automatic inspectors per service
- `factory/scripts/` — Project ticket helpers used by swamp models
- `models/command/shell/*` — Swamp factory control-panel models
- `workflows/` — `run-ticket` factory line (when present)
- `.swamp.yaml` — Swamp repo metadata (`.swamp/` runtime data is gitignored)
- `docs/` — Docusaurus product docs site (do not put factory notes there)
