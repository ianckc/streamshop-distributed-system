# StreamShop software factory

Production line for scoped changes: **GitHub Project ticket → Cursor implements → automated gate → bookkeeping**.

Tickets live in [Project 10 — StreamShop Distributed System](https://github.com/users/ianckc/projects/10). File new work with the **Factory ticket** issue form, add it to the project, set **Status = Ready** when pickable.

## Pieces

| Piece | Where |
|-------|--------|
| Contract | [`AGENTS.md`](../AGENTS.md) |
| Loop notes | [`factory/loop-notes.md`](loop-notes.md) |
| Local gate | `just gate` / `just gate order-api,catalog-api` |
| Swamp control panel | models under `models/command/shell/` (see `AGENTS.md`) |
| Factory line | workflow `run-ticket` |

## One job (shape)

1. Pick a **Ready** ticket on Project 10.
2. Start `run-ticket` with that issue number and the affected `services`.
3. At the coding pause: implement only the ticket; set Status to **In progress**; keep `just gate` green for those services.
4. Approve → swamp runs `factory-gate` and asserts exit 0.
5. Bookkeeping pause: Status **Done**, acceptance checked, append [`loop-notes.md`](loop-notes.md).

Exact swamp kickoff / approve / resume commands are in [`AGENTS.md`](../AGENTS.md).

## DIY without the workflow

Paste the DIY kickoff from `AGENTS.md` into Cursor Agent mode, then close out the ticket the same way (Done + loop notes + green gate).
