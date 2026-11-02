#!/usr/bin/env sh
# List GitHub Project 10 items with Status = Ready.
# Project: https://github.com/users/ianckc/projects/10
set -eu

PROJECT_NUMBER=10
OWNER="${FACTORY_PROJECT_OWNER:-@me}"
REPO="${FACTORY_REPO:-ianckc/streamshop-distributed-system}"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI not found — install GitHub CLI" >&2
  exit 1
fi

if ! gh api user -q .login >/dev/null 2>&1; then
  echo "gh auth failed — run: gh auth refresh -h github.com" >&2
  exit 1
fi

echo "Ready tickets (Project ${PROJECT_NUMBER} — StreamShop Distributed System):"
echo

FACTORY_PROJECT_JSON="$(gh project item-list "$PROJECT_NUMBER" --owner "$OWNER" --limit 100 --format json)" || {
  echo "failed to list project items — check project access / gh auth (needs 'project' scope)" >&2
  exit 1
}

export FACTORY_PROJECT_JSON REPO
python3 <<'PY'
import json, os

data = json.loads(os.environ["FACTORY_PROJECT_JSON"])
items = data.get("items") or []
repo = os.environ.get("REPO", "ianckc/streamshop-distributed-system")
ready = []
for it in items:
    if (it.get("status") or "") != "Ready":
        continue
    content = it.get("content") or {}
    num = content.get("number")
    title = it.get("title") or content.get("title") or "(no title)"
    url = content.get("url") or ""
    if not url and num:
        url = f"https://github.com/{repo}/issues/{num}"
    ready.append((num, title, url))
ready.sort(key=lambda x: (x[0] is None, x[0] or 0))
if not ready:
    print("  (none)")
else:
    for num, title, url in ready:
        label = f"#{num}" if num is not None else "(no issue)"
        print(f"  - {label} {title}")
        if url:
            print(f"    {url}")
print()
print(f"total: {len(ready)}")
PY
