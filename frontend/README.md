# Ayumi Link — frontend

React 19 + TypeScript + Vite + Tailwind CSS 4. See the [root README](../README.md) for what the
project is and how to run all of it.

```bash
npm install
npm run dev      # http://localhost:5173, proxies /api to the backend on 127.0.0.1:8081
npm run build    # type-check + production build into dist/
npm run lint     # oxlint
```

## Layout

```
src/
  api/          HTTP client (http.ts turns 401 into a login redirect), auth, sessions, chat (SSE)
  hooks/        useAuth, useSessionList, useSessionDetail, useChats, usePrefs, useSessionRoute, …
  components/
    auth/       first-run setup, 2-step login, AuthGate, skeletons
    chat/       chat panel, composer, model/effort/permission menu, markdown, code blocks
    sessions/   sidebar, new-session folder picker, move/rename/delete dialogs
    profile/    account, 2FA settings, login devices, defaults, server info
    ui/         button, input, modal, popover menu, skeleton, error dialog, …
  lib/          error translation, device parsing, chat option tables, helpers
```

Conventions: the session open in the URL is `/s/<id>`; the profile view is `#/profile`.
UI strings are currently in Indonesian (an i18n layer would be a welcome contribution).
