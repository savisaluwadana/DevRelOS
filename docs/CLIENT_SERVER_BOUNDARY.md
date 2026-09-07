# Client/server API boundary

DevRelOS web code keeps server credentials out of browser bundles by separating shared UI contracts from server-only data loaders.

- `*-shared.ts` modules contain browser-safe types, labels and constants.
- `server-api.ts` is server-only and is the only shared helper that reads `DEVRELOS_API_URL` / `DEVRELOS_API_TOKEN`.
- `*-api.ts` loader modules may import `server-api.ts` and must only be imported by Server Components.
- Client Components should import browser-safe contracts from `*-shared.ts` and send mutations through the same-origin `/api/devrelos/...` proxy.

This split prevents a client component from pulling the private backend client into its module graph and lets Next.js enforce the server-only boundary at build time.
