# Ayumi Link

**A self-hosted web UI for your local [Claude Code](https://docs.claude.com/en/docs/claude-code) sessions.**
Read, continue and start Claude Code conversations from any browser — including your phone — with
your own login (password + 2FA). Claude keeps running on *your* machine; nothing is sent to a third
party.

> **Unofficial project.** Ayumi Link is not affiliated with, endorsed by, or sponsored by Anthropic.
> "Claude" is a trademark of Anthropic. It drives the official `claude` command-line tool that you
> install and log in to yourself.

🇮🇩 [Baca dalam Bahasa Indonesia](README.id.md)

---

## Why

Claude Code lives in your terminal or desktop app. Ayumi Link puts a chat UI in front of it so you
can keep working from a phone or another computer **without** giving that device your Claude login
and without a VPN client: you log in to *Ayumi Link* (password + authenticator code), and the `claude`
CLI on your own machine does the work.

## Features

- **Browse every session** from `~/.claude`, grouped like Claude Desktop's folders, with live badges.
- **Continue chats** with streaming replies; replies keep running in the background while you browse.
- **Start new sessions** in a folder you pick from a locked-down folder browser.
- **Attach files to a message** (button, drag-and-drop, or paste): images/photos, PDF, Excel (`.xlsx`, `.csv`) and Word (`.docx`) — Claude reads them. Every other file type is rejected.
- **Per-message model / effort / tool-permission** menu (Fable, Opus, Sonnet, Haiku and older models).
- **Rename, move, hide or delete** sessions and folders (local to Ayumi Link; never edits Claude's data).
- **Browser tab:** a headless Chromium on the server (Docker image or your own Chrome/Chromium when
  run natively) streamed into the UI — open sites, click, type, scroll. Site logins persist (cookies +
  profile live in the data dir); it shuts itself down after `AYUMI_BROWSER_IDLE_MIN` idle minutes.
  Disable with `AYUMI_BROWSER_DISABLED=true`, or skip Chromium in the image with
  `--build-arg WITH_BROWSER=false`. Only http/https; it browses from the **server's** network.
- **Real authentication:** argon2id password, TOTP 2FA, one-time recovery codes, account lockout,
  device list with remote sign-out, step-up re-verification for sensitive actions.
- **Access control from `.env`:** client **IP allowlist**, trusted-proxy support, cookie hardening.
- **Deep links:** `/s/<session-id>` reopens a session after reload; skeleton loading states; clear
  error dialogs; responsive layout (phone drawer sidebar).
- **One image, one command:** `docker compose up -d --build`.

## How it works

```
 Browser ──HTTPS──▶  Ayumi Link (Go + Fiber API, serves the React app)
 (phone/laptop)            │  spawns   `claude -p --resume <id> --output-format stream-json`
                           ▼
                     claude CLI  ──▶  Anthropic API
                           │
                    ~/.claude  (sessions are plain JSONL transcripts, read directly)
```

Replies stream to the browser over Server-Sent Events. Ayumi Link stores only its own small state
(account, 2FA, UI preferences) in its data directory.

## Quick start — Docker (one command)

Requires Docker with Compose v2.24+.

```bash
git clone https://github.com/bustomii/ayumi-link.git && cd ayumi-link
docker compose up -d --build          # or: make up
```

The first build downloads the base images and the Claude Code CLI (roughly 1 GB), so it can take
several minutes on a slow connection; later builds are cached.

1. Open <http://localhost:8081>.
2. Create your account. The one-time **setup token** is printed in the logs:
   `docker compose logs ayumi-link | grep -A1 SETUP` (or `make token`). Scan the QR with an
   authenticator app and **store the recovery codes**.
3. Log the Claude CLI in **inside the container** (credentials live in a Docker volume):
   `docker compose exec -it ayumi-link claude auth login` (or `make claude-login`).
   Alternatives: set `CLAUDE_CODE_OAUTH_TOKEN` (from `claude setup-token`) or `ANTHROPIC_API_KEY` in `.env`.
4. Put the projects Claude may work in under `./workspace` (or set `WORKSPACE_DIR` in `.env`).

By default the port is published on **127.0.0.1 only**. See [Remote access](#remote-access-phone) before
changing that.

> **Docker vs. running directly.** A Linux container cannot read your macOS Keychain, Claude Desktop's
> folder groups, or host paths such as `/Users/you/...`. Docker is ideal for a server/NAS/VPS and for
> trying the app. To mirror an **existing desktop setup** (all your current sessions and Claude Desktop
> groups) run it directly on that machine.

## Quick start — run directly (macOS / Linux)

Requirements: Go ≥ 1.24, Node ≥ 22, and the `claude` CLI installed and logged in.

```bash
cp .env.example .env                       # optional; the backend also finds it from backend/ (../.env)
# development (two terminals)
cd backend  && go run .
cd frontend && npm install && npm run dev  # http://localhost:5173

# or a single production process
cd frontend && npm install && npm run build
cd ../backend && AYUMI_STATIC_DIR=../frontend/dist go run .   # http://localhost:8081
```

Direct runs use your home directory as the folder-picker root by default and prefer the newest
`claude` bundled with Claude Desktop (so new models work), falling back to `claude` in `PATH`.

## Configuration

Everything is optional and set through environment variables or a `.env` file
(see [`.env.example`](.env.example) — every option is documented there). Real environment
variables always win over `.env`.

| Variable | Default | Purpose |
|---|---|---|
| `AYUMI_ALLOWED_IPS` | *(empty = all)* | **IP allowlist**: comma-separated IPs / CIDRs allowed to reach the app at all |
| `AYUMI_ALLOWED_HOSTS` | *(empty = any)* | **Host allowlist**: only these domain names may be used to reach the app (`*.example.com` wildcards ok) |
| `AYUMI_TRUST_PROXY` | `false` | Trust the proxy's client-IP header and `X-Forwarded-Proto` |
| `AYUMI_PROXY_HEADER` | `X-Forwarded-For` | Header holding the real client IP (`CF-Connecting-IP`, `X-Real-IP`, …) |
| `AYUMI_TRUSTED_PROXIES` | – | Only these proxies are believed (IPs/CIDRs) |
| `AYUMI_COOKIE_SECURE` | `false` | Mark the session cookie `Secure` (**set `true` on HTTPS**) |
| `ALLOWED_ORIGINS` | – | Extra allowed origins (custom domain) |
| `HOST` / `PORT` | `127.0.0.1` / `8081` | Listen address (Docker listens on all interfaces inside the container) |
| `AYUMI_ROOTS` | home dir | Folders selectable for new sessions (`:`-separated) |
| `AYUMI_PERMISSION_MODE` | `auto` | Default tool-permission mode: `auto`, `acceptEdits`, `manual`, `plan` |
| `AYUMI_AUTH_RATE_LIMIT` | `10` | Auth requests / minute / IP |
| `AYUMI_MAX_UPLOAD_MB` / `AYUMI_UPLOAD_MAX_TOTAL_MB` / `AYUMI_UPLOAD_TTL_DAYS` | `20` / `2048` / `30` | Attachment size per file, total storage, and retention |
| `AYUMI_DATA_DIR` | OS config dir | Where account, 2FA and state are stored (mode `0600`) |
| `CLAUDE_BIN`, `CLAUDE_HOME`, `CLAUDE_DESKTOP_DIR` | auto | Locations of the CLI and Claude's data |
| `AYUMI_STATIC_DIR` | – | Serve a built frontend from this folder (set automatically in Docker) |
| `AYUMI_BIND`, `AYUMI_PORT`, `WORKSPACE_DIR`, `CLAUDE_CODE_VERSION` | see file | Docker-only options |

### IP allowlist example

```env
AYUMI_ALLOWED_IPS=127.0.0.1, ::1, 192.168.1.0/24, 100.64.0.0/10
```

Requests from any other address get `403` before anything else is processed (the login page
included). Include your own address, or you will lock yourself out of the browser (the app itself
keeps working; edit `.env` and restart).

## Remote access (phone)

Ayumi Link can run commands on your machine through Claude, so treat exposing it like exposing SSH.
**Never publish it over plain HTTP.** Good options:

- **Tailscale** (or another private mesh VPN): set `AYUMI_BIND` to the Tailscale IP (Docker) or
  `HOST` to it (direct run), and add the tailnet range to `AYUMI_ALLOWED_IPS`.
- **Cloudflare Tunnel / reverse proxy (HTTPS):** keep the app on loopback, terminate TLS at the proxy and set
  `AYUMI_COOKIE_SECURE=true`, `AYUMI_TRUST_PROXY=true`, `AYUMI_TRUSTED_PROXIES=<proxy ip>` and a header
  your proxy *sets* (`CF-Connecting-IP`, `X-Real-IP`). `X-Forwarded-For` can be spoofed by clients if
  your proxy merely appends to it.

Checklist before going remote: HTTPS ✔ · 2FA enabled ✔ · recovery codes stored ✔ · allowlist or private
network ✔ · `AYUMI_PERMISSION_MODE` you are comfortable with ✔.

## Security model

- **Authentication:** single owner account; argon2id; TOTP 2FA is mandatory and cannot be turned off;
  replayed TOTP codes are rejected; 8 one-time recovery codes; 5 failures lock the account for 15 min;
  per-IP rate limiting; sessions idle out after 7 days (30 days absolute) and can be revoked.
- **Sensitive actions** (change password, regenerate recovery codes, replace the authenticator) require
  re-entering password + a 2FA code. Changing them signs out all other devices.
- **Web hardening:** origin check + JSON-only mutations + `SameSite=Strict` HttpOnly cookie, strict
  CSP, `no-store` API responses, no external requests (fonts are bundled), no credentials in logs.
- **Filesystem:** the folder picker lists directories only, inside `AYUMI_ROOTS`, resolves symlinks,
  hides dot-folders, and the server re-validates the folder when a session is created.
- **Attachments:** only images (PNG/JPG/GIF/WEBP), PDF, `.xlsx`, `.csv` and `.docx`. The type is verified from the file *contents* (magic bytes and document structure), not the extension, so a renamed `.exe` is refused; SVG/HTML/scripts/archives/legacy `.doc`/`.xls` and Office files with macros are rejected. File names are sanitised, ids are random, files live under the data directory with mode `0600`, are served with `nosniff` + a sandbox CSP (only images inline), count against a storage quota and expire automatically. Word/Excel are converted to text on the server (size-capped, zip-bomb checked) before Claude reads them; Claude can only read the attachment folders you send, via `--add-dir`.
- **Permissions:** the UI never offers `bypassPermissions`. Tools that need approval are denied in
  `manual` mode because the non-interactive CLI cannot ask.
- **Known limits:** one user only; the TOTP secret is stored unencrypted in `auth.json` (mode `0600`, so
  protect the data directory); Ayumi Link cannot make a chat appear in Claude Desktop.

Found a vulnerability? See [SECURITY.md](SECURITY.md).

## Known limitations

- Sessions **created in Ayumi Link do not show up in Claude Desktop** (Desktop ignores externally created
  index entries). Sessions started in Desktop can be continued from Ayumi Link.
- Renaming/moving/hiding is local to Ayumi Link.
- Attachments: legacy `.doc`/`.xls` are not supported (save as `.docx`/`.xlsx`). Excel dates appear as serial numbers and very large sheets are truncated (3,000 rows per sheet) in the text Claude receives.
- The UI is currently Indonesian only.
- Docker mode cannot mirror Claude Desktop groups (see the note above).

## Development

```bash
make test                              # go vet + go test + frontend type check
cd backend  && go test ./...           # auth, API, session parsing, folder browser, config
cd frontend && npx tsc -b && npm run lint
```

```
backend/internal/  config/ api/ auth/ claude/ fsbrowse/ store/     (Go, Fiber v2)
frontend/src/      api/ hooks/ components/ lib/                     (React + TypeScript)
```

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE). Bundled font: IBM Plex Sans (SIL Open Font License 1.1).

## Disclaimer

Provided "as is", without warranty. You are responsible for how you expose it and for complying with
Anthropic's terms of service and usage policies for the plan or API key you use with Claude.
