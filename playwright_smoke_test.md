# Tinychat manual smoke test (3 tabs)

No Playwright automation — use this checklist for a quick multi-client sanity check.

## Prerequisites

1. From the repo root, start the server:
   ```bash
   go run ./cmd/tinychat
   ```
2. Confirm it listens on `:8080` (or your chosen `-addr`).

## Steps

1. Open **three** browser tabs to `http://localhost:8080/?name=alice`, `?name=bob`, and `?name=carol` (adjust host/port if needed).
2. In each tab, confirm the header **Online** list shows all three names after everyone has connected.
3. In tab 1 (alice), send a normal message, e.g. `hello from alice`.
4. Verify tabs 2 and 3 show alice’s message in the chat log.
5. In tab 2, run `/who` and confirm the system line lists alice, bob, and carol.
6. In tab 3, run `/name dana` and confirm the online list updates to show **dana** instead of carol.
7. Close tab 3; in tabs 1 and 2 confirm **dana** disappears from the online list and a leave system message appears.
8. Switch to tab 2 while tab 1 sends a message; confirm the browser tab title shows an unread count badge until you focus tab 2.

## Pass criteria

- All three tabs see the same presence list while connected.
- Chat messages broadcast to every tab.
- `/who`, `/clear`, and `/name` behave as documented in `web/app.js`.
- Disconnect/reconnect shows backoff messaging and restores presence after the server is reachable again.
