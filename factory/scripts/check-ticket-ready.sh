#!/usr/bin/env sh
# Verify a GitHub issue is on Project 10 with Status Ready or In progress.
# Requires env ISSUE_NUMBER.
set -eu

PROJECT_NUMBER=10
OWNER="${FACTORY_PROJECT_OWNER:-@me}"
REPO="${FACTORY_REPO:-ianckc/streamshop-distributed-system}"

if [ -z "${ISSUE_NUMBER:-}" ]; then
  echo "ISSUE_NUMBER env is required" >&2
  exit 1
fi

ISSUE_NUMBER="$(printf '%s' "$ISSUE_NUMBER" | sed 's/^#//')"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI not found — install GitHub CLI" >&2
  exit 1
fi

if ! gh api user -q .login >/dev/null 2>&1; then
  echo "gh auth failed — run: gh auth refresh -h github.com" >&2
  exit 1
fi

FACTORY_PROJECT_JSON="$(gh project item-list "$PROJECT_NUMBER" --owner "$OWNER" --limit 100 --format json)" || {
  echo "failed to list project items — check project access / gh auth" >&2
  exit 1
}

export ISSUE_NUMBER REPO FACTORY_PROJECT_JSON
match="$(python3 <<'PY'
import json, os, sys

want = int(os.environ["ISSUE_NUMBER"])
repo = os.environ.get("REPO", "ianckc/streamshop-distributed-system")
data = json.loads(os.environ["FACTORY_PROJECT_JSON"])
for it in data.get("items") or []:
    content = it.get("content") or {}
    num = content.get("number")
    if num == want:
        print(json.dumps({
            "number": num,
            "title": it.get("title") or content.get("title") or "",
            "status": it.get("status") or "",
            "url": content.get("url") or f"https://github.com/{repo}/issues/{num}",
        }))
        sys.exit(0)
sys.exit(2)
PY
)" || {
  echo "issue #${ISSUE_NUMBER} is not on Project ${PROJECT_NUMBER}" >&2
  exit 1
}

status="$(printf '%s' "$match" | python3 -c 'import json,sys; print(json.load(sys.stdin)["status"])')"
case "$status" in
  Ready|"In progress") ;;
  *)
    echo "issue #${ISSUE_NUMBER} Status is '${status}' — need Ready or In progress" >&2
    exit 1
    ;;
esac

issue_json="$(gh issue view "$ISSUE_NUMBER" --repo "$REPO" --json number,title,body,url,state)" || {
  echo "failed to load issue #${ISSUE_NUMBER} from ${REPO}" >&2
  exit 1
}

export FACTORY_PROJECT_MATCH="$match" FACTORY_ISSUE_JSON="$issue_json"
python3 <<'PY'
import json, os, re

project = json.loads(os.environ["FACTORY_PROJECT_MATCH"])
issue = json.loads(os.environ["FACTORY_ISSUE_JSON"])
body = issue.get("body") or ""

services = []
for m in re.finditer(r"^- \[[xX]\]\s+(\S+)\s*$", body, re.M):
    services.append(m.group(1))
if not services:
    m = re.search(r"(?is)Affected services\s*\n+([^\n#]+(?:\n(?![#\-]).+)*)", body)
    if m:
        chunk = m.group(1)
        for line in chunk.splitlines():
            line = line.strip().lstrip("-").strip()
            if not line:
                continue
            for part in re.split(r"[,]", line):
                part = part.strip()
                if part and re.match(r"^[a-z0-9][a-z0-9_/-]*$", part):
                    services.append(part)

seen = set()
uniq = []
for s in services:
    if s not in seen:
        seen.add(s)
        uniq.append(s)
services = uniq

print(f"ok: #{issue['number']} Status={project['status']!r}")
print(f"title: {issue.get('title')}")
print(f"url: {issue.get('url')}")
print(f"state: {issue.get('state')}")
if services:
    print("affected_services: " + ",".join(services))
else:
    print("affected_services: (not parsed — pass services=… to run-ticket / just gate)")
print()
print("=== body ===")
print(body)
PY
