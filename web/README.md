# ClarityIT Web Frontend

React 19 + TypeScript + Tailwind CSS 4 + shadcn/ui frontend for ClarityIT.

## Quick Start

```bash
# Install dependencies
npm install

# Development server (proxies API to localhost:8765)
npm run dev

# Run tests
npm test

# Build for production
npm run build
```

## Architecture

```
src/
├── api/client.ts          # Typed API client (in-memory tokens, auto-refresh, idempotency)
├── auth/context.tsx       # AuthProvider (login, register, logout, switchTeam, permissions)
├── components/
│   ├── layout/AppLayout.tsx  # Sidebar layout, team switcher, WS indicator
│   └── ui/                   # shadcn/ui primitives (button, card, input, dialog, etc.)
├── features/
│   ├── auth/             # Login, Register, Forgot/Reset Password
│   ├── bootstrap/        # First-user bootstrap flow
│   ├── dashboard/        # Dashboard with live counts + WS refetch
│   ├── queue/            # Work item table with status filter
│   ├── board/            # Kanban-style board
│   ├── objects/          # Object detail + edit form + comments + links
│   ├── work-items/       # New work item form
│   ├── incidents/        # Incident list + detail + timeline + resolve
│   ├── team/             # Team settings, members, invitations
│   └── admin/            # Users, Teams, Audit (owner-only)
├── hooks/
│   ├── usePermissions.ts    # Permission check hook
│   ├── useRefetch.tsx       # RefetchProvider (WS invalidation → version bump)
│   └── useWebSocket.ts      # WebSocket hook with reconnect
├── lib/utils.ts             # shadcn/ui utilities
├── main.tsx                 # Router + providers
└── test/
    ├── setup.ts             # Vitest setup
    └── app.test.tsx         # 15 frontend tests
```

## Key Design Decisions

- **Access tokens in memory only** — not stored in localStorage
- **Auto-refresh on 401** — transparent token rotation via httpOnly cookie
- **Idempotency keys** — all mutations send `Idempotency-Key` header (UUID v4)
- **WS events as invalidation** — WebSocket events trigger `useRefetch` version bump, not direct state mutations
- **Optimistic locking** — object edits send `expected_version`, 409 handled with auto-refetch
- **No email enumeration** — forgot-password always shows success
- **Permission-aware UI** — nav items hidden based on `hasPermission()` checks

## Routes

| Path | Feature | Auth |
|------|---------|------|
| `/bootstrap` | First-user setup | Public |
| `/login` | Sign in / Register | Public |
| `/forgot-password` | Password reset request | Public |
| `/reset-password?token=...` | Password reset | Public |
| `/` | Dashboard | Protected |
| `/queue` | Work item queue | Protected |
| `/board` | Kanban board | Protected |
| `/objects/:id` | Object detail | Protected |
| `/work-items/new` | New work item | Protected |
| `/incidents` | Incident list | Protected |
| `/incidents/:id` | Incident detail | Protected |
| `/settings/team` | Team settings | Protected |
| `/admin/users` | User management | Owner |
| `/admin/teams` | Team management | Owner |
| `/admin/audit` | Audit log | Owner |

## Deployment

Frontend is built as a Docker container:

```bash
# Build and deploy
cd /opt/clarityit && docker compose up -d --build clarityit-web
```

The container uses a multi-stage build:
1. **Stage 1**: Node.js 22 builds the Vite production bundle
2. **Stage 2**: nginx:alpine serves the static files + proxies `/api/` to the Go backend

Frontend accessible at `http://192.168.3.20:3000`.

## Tests

```bash
npm test          # Run once
npm run test:watch # Watch mode
```

15 tests covering:
- Login/register form submissions
- Bootstrap 409 redirect
- Auth refresh flow
- Logout state clearing
- Permission checks (granted/denied)
- Work item mutation patterns
- 409 stale version handling
- Board/object/comment component existence
- Team switch API
