# Network Inspector

Small Playwright utility for opening a target page in Chromium and capturing JSON or GraphQL network responses into `apis.json`.

## Setup

```sh
npm install
```

If Playwright browsers are not installed yet, run:

```sh
npx playwright install chromium
```

## Usage

Create a local `.env` file:

```sh
cp .env.example .env
```

Set `TARGET_URL` in `.env`, then run:

```sh
node traffic_inspector.js
```

The script opens Chromium with `headless: false`, loads `TARGET_URL` from the environment or `.env`, scrolls the page to trigger lazy-loaded requests, and writes captured API responses to `apis.json`.

For `POST /flagship-web/rsc-action/*` requests, `apis.json` also includes:

- `requestContentType`
- `requestPostData`
- `requestPostDataTruncated`
- `requestPostDataLength`

If `auth-state.json` exists, the browser context uses it as Playwright storage state.

## Generated Files

- `apis.json`: captured API response output.
- `auth-state.json`: optional local browser auth/session state.
