# Resource Guide

Use official documentation first, then study mature open-source projects for design ideas. Do not copy code without checking its license and attribution requirements.

## Similar download manager projects

- [aria2](https://github.com/aria2/aria2) — segmented downloads, retries, RPC, protocol handling.
- [Motrix](https://github.com/agalwood/Motrix) — desktop UX reference.
- [Persepolis Download Manager](https://github.com/persepolisdm/persepolis) — queue and task workflows.
- [AriaNg](https://github.com/mayswind/AriaNg) — presenting task state and controls.
- [uGet](https://github.com/ugetdm/uget) — feature and workflow comparison.

## Core technology documentation

- [Go documentation](https://go.dev/doc/)
- [Go modules](https://go.dev/ref/mod)
- [net/http](https://pkg.go.dev/net/http)
- [Wails v3 documentation](https://v3.wails.io/)
- [Wails v3 installation](https://v3.wails.io/getting-started/installation/)
- [SQLite documentation](https://www.sqlite.org/docs.html)
- [React documentation](https://react.dev/)
- [TypeScript handbook](https://www.typescriptlang.org/docs/)
- [Vite](https://vite.dev/guide/)
- [Tailwind CSS](https://tailwindcss.com/docs)
- [pnpm workspaces](https://pnpm.io/workspaces)

## Browser extension documentation

- [Chrome Extensions](https://developer.chrome.com/docs/extensions/)
- [MDN WebExtensions](https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions)
- [Native Messaging — Chrome](https://developer.chrome.com/docs/extensions/develop/concepts/native-messaging)
- [Native Messaging — MDN](https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions/Native_messaging)

## HTTP references

- [HTTP Semantics — RFC 9110](https://www.rfc-editor.org/rfc/rfc9110)
- [HTTP Range Requests](https://www.rfc-editor.org/rfc/rfc9110.html#name-range-requests)
- [Content-Disposition — RFC 6266](https://www.rfc-editor.org/rfc/rfc6266)

## Learning order

1. Go fundamentals: packages, interfaces, concurrency, context, errors.
2. `net/http`: clients, transports, redirects, timeouts, proxies.
3. HTTP Range and conditional request semantics.
4. SQLite and migrations.
5. Wails v3: services, events, assets, packaging.
6. WebExtensions and Native Messaging.

## How to study reference repositories

- Read architecture and license files first.
- Trace one complete download lifecycle.
- Inspect retry and resume behavior.
- Inspect how partial files are named and finalized.
- Inspect tests for malformed/failed servers.
- Compare documented behavior with actual code.
- Record design lessons; do not copy large blocks blindly.
