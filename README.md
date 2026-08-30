# Reverse API

- A utility tool for LinkedIn profile retrieval.
- Objective was to Reverse engineer LinkedIn's web and API endpoints to retrieve profile data programmatically.
- Instead of scraping LinkedIn profiles manually or using browser automation, this tool leverages captured web and API traffic to programmatically retrieve profile data.

## Approach
- The approach I have followed is similar to Capture The Flag (CTF) style web security challenges, where understanding and manipulating web traffic is key.
- The repository currently contains two independent tools:

- `network-inspector`: a Playwright script that opens a target page in Chromium and captures response payloads.
- `profile-retrieval-service`: a Go HTTP service that retrieves LinkedIn profile data from LinkedIn web/API sources.
- I have used network inspector to capture LinkedIn web and API traffic and write to a file locally, I have used the captured data to understand the structure and behavior of LinkedIn's web and API endpoints. Then I have used this understanding to implement the profile-retrieval-service for programmatically retrieving profile data.

## Detours/Notes
- Intially I have tried to use graphQL and data apis for profile retrieval, but accept for certifications all the other data returned 410 Gone status code. Therefore, I had to rely on understanding the web endpoints and implementing the profile-retrieval-service accordingly.

## Prerequisites

- Go `1.26.4` or newer for `profile-retrieval-service`
- Node.js and npm for `network-inspector`
- Chromium installed through Playwright for browser capture

## Repository Structure

```text
.
├── Makefile
├── README.md
├── network-inspector/
│   ├── README.md
│   ├── package.json
│   ├── package-lock.json
│   └── traffic_inspector.js
└── profile-retrieval-service/
    ├── .env.example
    ├── cmd/
    │   └── main.go
    ├── datasource/
    ├── docs/
    │   ├── openapi.go
    │   └── openapi.yaml
    ├── http/
    │   ├── controller/
    │   ├── helper/
    │   ├── server.go
    │   └── server_test.go
    └── services/
```

## Local Setup

### Profile Retrieval Service

From the repository root:

```sh
cd profile-retrieval-service
cp .env.example .env
go test ./...
go run ./cmd
```

Or use the root Makefile:

```sh
make up
```

The service reads `.env` from `profile-retrieval-service/` and listens on `PORT`. The default address is `:8080`.

Useful environment variables:

```sh
PORT=8080
PROFILE_RETRIEVE_RATE_LIMIT_REQUESTS=1
PROFILE_RETRIEVE_RATE_LIMIT_WINDOW=2m
LINKEDIN_REQUEST_MIN_INTERVAL=3s
LINKEDIN_COOKIE_HEADER=
LINKEDIN_CSRF_TOKEN=
LINKEDIN_LI_AT=
LINKEDIN_JSESSIONID=
```

LinkedIn profile retrieval depends on valid LinkedIn session credentials. Prefer `LINKEDIN_COOKIE_HEADER` and `LINKEDIN_CSRF_TOKEN` copied from an authenticated browser request. As an alternative, set `LINKEDIN_LI_AT` and `LINKEDIN_JSESSIONID`.

### Network Inspector

From the repository root:

```sh
cd network-inspector
npm install
npx playwright install chromium
cp .env.example .env
```

Set `TARGET_URL` in `network-inspector/.env`, then run:

```sh
node traffic_inspector.js
```

Or use the root Makefile:

```sh
make network-inspector-up
```

The script launches Chromium in headed mode, opens `TARGET_URL`, scrolls to trigger lazy-loaded requests, lets you interact with the page, and writes captured responses locally.

Generated files:

- `network-inspector/apis.json`: non-HTML captured responses, excluding CSS and JavaScript responses.
- `network-inspector/html_apis.json`: captured HTML responses.
- `network-inspector/auth-state.json`: optional Playwright storage state file if you want to reuse browser auth.

## Endpoints

The Go service runs at `http://localhost:8080` by default.

### `GET /healthz`

Health check.

```sh
curl -i http://localhost:8080/healthz
```

Response:

```json
{
  "status": "ok"
}
```

### `GET /docs`

Swagger UI for the API.

```sh
open http://localhost:8080/docs
```

### `GET /docs/openapi.yaml`

Raw OpenAPI specification.

```sh
curl -i http://localhost:8080/docs/openapi.yaml
```

### `GET /profiles/retrieve`

Retrieve a LinkedIn profile using a query parameter.

```sh
curl -i "http://localhost:8080/profiles/retrieve?profile_url=https://www.linkedin.com/in/example-profile/"
```

The service also accepts `url` as an alias for `profile_url`.

### `POST /profiles/retrieve`

Retrieve a LinkedIn profile using a JSON body.

```sh
curl -i http://localhost:8080/profiles/retrieve \
  -H "Content-Type: application/json" \
  -d '{"profile_url":"https://www.linkedin.com/in/example-profile/"}'
```

The JSON body also accepts `url` as an alias for `profile_url`.

Common responses:

- `200`: profile data was retrieved.
- `400`: missing, invalid, or non-LinkedIn `/in/` profile URL.
- `429`: inbound rate limit exceeded.
- `502`: LinkedIn upstream fetch failed.

## Development Commands

Run all Go tests:

```sh
cd profile-retrieval-service
go test ./...
```

Check the network inspector script syntax:

```sh
cd network-inspector
node --check traffic_inspector.js
```

Run the profile service on a different port:

```sh
cd profile-retrieval-service
PORT=18081 go run ./cmd
```

Then test it:

```sh
curl -i http://localhost:18081/healthz
```
