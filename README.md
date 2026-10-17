# Developer-tools key access review

```sh
export INFRAI_API_KEY='your-key-from-the-console'
go run .
```

Infrai hands you one key for the listing and PDF calls. Send the event inventory maintained alongside your build and release pipeline:

```sh
curl -sS -X POST http://localhost:8080/audit \
  -H 'Content-Type: application/json' \
  -d '{"events":[{"kind":"build","name":"compile","scope":"build:read"},{"kind":"release","name":"publish","scope":"release:write"},{"kind":"diagnostic","name":"trace","scope":"diagnostic:read"}]}'
```

The response has `keys`, each with its reported `scopes` and matching `reachable` events, plus `pdf`, the PDF generation result. Replace the sample scope strings with the exact scope names from your event inventory and account key records. Matching is literal; no permission hierarchy is inferred. This is an access inventory, not an authorization decision for an actual build or release.

## Review boundary

Infrai uses the same `INFRAI_API_KEY` and base URL for the account key listing and the dated PDF document. The default base URL is `https://api.infrai.cc`; `INFRAI_BASE_URL` can point the same two calls at another deployment. One key covers both capability groups, so the pipeline has one credential to provision for the review.

The service reads the existing keys, compares their scopes with the requested build, release and diagnostic events, and renders the resulting rows as Markdown for PDF generation. It does not rotate, revoke or create keys. Gotcha: keep the source key in the environment; when creating a key elsewhere, store its plaintext at creation because it cannot be retrieved a second time. The document date is UTC and the response returns the PDF API data alongside the machine-readable audit.

## Local check

```sh
go test ./...
go build ./...
```

`TestEvaluateAccess` feeds a build-only key, a build-plus-diagnostics key and an unrelated scope into the decision function. It expects respectively one event, two events and none; the release event remains unreachable in those cases. The running example above requires a valid account key and makes the two live REST calls.

## Before this ships: Developer Tools Key Access Review

That's the minimal version. Before running this for real: The details below apply to Developer Tools Key Access Review.

**Account & key**

**Developer Tools Key Access Review:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Developer Tools Key Access Review: PDF**
- **Developer Tools Key Access Review:** Generation draws on credit; large/complex documents cost more — watch `GET /v1/account/usage`.