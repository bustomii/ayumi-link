# Contributing to Ayumi Link

Thanks for helping! Bug reports, docs, translations and code are all welcome.

## Setup

```bash
git clone https://github.com/bustomii/ayumi-link.git && cd ayumi-link
cd backend  && go test ./...                 # Go >= 1.24
cd ../frontend && npm install && npx tsc -b  # Node >= 22
```

Run it: `cd backend && go run .` and `cd frontend && npm run dev` (open <http://localhost:5173>).
The first start prints a one-time setup token in the backend log. A real `claude` CLI login is
only needed to actually chat; everything else (auth, sessions list, UI) works without it.

## Before you open a PR

- `cd backend && go vet ./... && go test ./...` passes.
- `cd frontend && npx tsc -b && npm run lint` passes (no new warnings).
- `gofmt -l backend` prints nothing.
- Behaviour changes come with tests. **Anything touching authentication, the folder browser, the
  IP allowlist or the static/CSP layer must have tests** — these are the security boundaries.
- Keep PRs focused; describe *why*, not only *what*.

## Project principles

- **Never write to Claude's or Claude Desktop's data.** Ayumi Link only reads them; its own state
  lives in its data directory. Renaming/moving/hiding sessions is local on purpose.
- **Secure by default.** New options should default to the safer behaviour, and anything that widens
  access must be opt-in and documented in `.env.example`.
- **No new external requests at runtime** (fonts, analytics, CDNs). The CSP forbids them.
- **Errors are for humans.** Add friendly messages in `frontend/src/lib/errors.ts` instead of showing
  raw server text.

## Code style

- Go: standard `gofmt`; small packages under `backend/internal/`; table-driven tests.
- TypeScript: functional components + hooks; data fetching in `src/api`, state in `src/hooks`.
- Code comments and UI strings are currently in Indonesian; English comments are fine in new code.
  A proper i18n layer for the UI would be a great contribution.

## Reporting bugs

Include: how you run it (Docker / direct, OS), the version (`ayumi-link version`), what you did, what you
expected, and the relevant log lines. **Do not paste tokens, recovery codes or your `.env`.**
For security issues follow [SECURITY.md](SECURITY.md) instead of opening a public issue.
