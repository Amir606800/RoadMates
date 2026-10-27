# RoadMates frontend

React mobile UI for 360–430px screens. Accounts, roles, driver applications,
admin review, district pickers, and ride publishing/search/details use the Go
API. The map is illustrative and reviews remain a UI preview; reservations and
payments are not implemented. See the [root README](../README.md) for Docker,
environment settings, tests, and the [API contract](../backend/api/openapi/v1.yaml).

```powershell
npm.cmd ci
npm.cmd run dev
```

Open <http://localhost:5173>. Vite proxies `/api`, `/healthz`, and `/readyz` to
`VITE_PROXY_TARGET` (default `http://localhost:8080`). Configure the backend's
`ALLOWED_ORIGIN` to match Vite's exact origin. Copy `.env.example` to `.env.local`
for optional public build settings; never put secrets in `VITE_*` variables.
On macOS/Linux use `npm` instead of `npm.cmd`.

| Page | Route |
| --- | --- |
| Register/sign in | `/register`, `/login` |
| District/date/seat search and real ride results | `/drives` |
| Saved ride details | `/drives/:driveId` |
| Identity, travel role, verification status, logout | `/profile` |
| Driver application | `/verification` |
| Admin application queue and review | `/admin`, `/admin/driver-applications/:applicationId` |
| Publish or cancel/complete the driver's active ride | `/driver/drives` |
| Review UI preview | `/reviews` |
| Illustrative route preview | `/map` |

The auth provider keeps the access token only in state/ref. Startup uses the
HttpOnly refresh cookie to restore the session. Parallel 401 responses share
one refresh; each request retries once. Profile roles come from the server.
Approval refreshes the session to gain driver access, and logout revokes the
refresh family. An outdated app receives a full-screen update/reload message.

Both ride forms use the API-backed district dialog. ASCII searches work with
Azerbaijani names, permit districts show a hint, and the other selected stop is
disabled. Dates and departure times use Azerbaijan time. Search starts with
empty stops; there are no sample rides. The driver page shows a form only after
the active lookup returns 404. Existing rides show Cancel/Complete, and a
cross-tab publish conflict reloads the active ride.

PINs are masked and cleared after submission; only admin detail can reveal a
decrypted PIN. No tokens or PINs are persisted in browser storage. Reviews and
completed-trip review fixtures are isolated under `src/mocks`; the standalone
booking helper tests are retained without offering mock purchases in real rides.

`src/api` contains API clients and payload builders, `context` contains auth,
role/search/application state, and `components`/`pages` contain mobile screens.
Shared design tokens live in `src/styles/tokens.css`. `AdSlot` is hidden by
default; `VITE_ADS_ENABLED=true` previews labelled placeholders.

Run `npm.cmd test` and `npm.cmd run build`. The build produces `dist/` and
`version.json`. Production hosting needs an SPA fallback; the included nginx
configuration provides it. Check keyboard navigation and phone widths of 360,
390, and 430px when changing screens.
