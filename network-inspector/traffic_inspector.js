const { chromium } = require("playwright");
const fs = require("fs");
const fsp = require("fs/promises");
const path = require("path");
const readline = require("readline/promises");

const ENV_FILE = ".env";
const AUTH_STATE_FILE = "auth-state.json";
const API_OUTPUT_FILE = "apis.json";

function loadEnvFile() {
  if (!fs.existsSync(ENV_FILE)) return;

  const env = fs.readFileSync(ENV_FILE, "utf8");

  for (const line of env.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;

    const separatorIndex = trimmed.indexOf("=");
    if (separatorIndex === -1) continue;

    const key = trimmed.slice(0, separatorIndex).trim();
    const value = trimmed.slice(separatorIndex + 1).trim();

    if (!key || process.env[key] !== undefined) continue;

    process.env[key] = value.replace(/^["']|["']$/g, "");
  }
}

loadEnvFile();

const TARGET_URL = process.env.TARGET_URL;
const MAX_SCROLL_PASSES = 8;
const SCROLL_DISTANCE_PX = 1200;
const SCROLL_SETTLE_MS = 2000;
const FINAL_SETTLE_MS = 5000;

const apis = new Map();
const pendingCaptures = new Set();
let writeQueue = Promise.resolve();

function isDataApiResponse(response) {
  const url = response.url();
  const headers = response.headers();
  const contentType = (headers["content-type"] || "").toLowerCase();

  let parsed;
  try {
    parsed = new URL(url);
  } catch {
    return false;
  }

  const pathname = parsed.pathname.toLowerCase();
  const isJsonResponse = contentType.includes("json");
  const isGraphqlRoute = pathname.includes("/graphql");

  return isJsonResponse || isGraphqlRoute;
}

async function readResponseData(response) {
  const contentType = (response.headers()["content-type"] || "").toLowerCase();

  if (contentType.includes("json")) {
    return response.json();
  }

  const text = await response.text();
  if (!text.trim()) return null;

  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

function upsertApi(response, data) {
  const request = response.request();
  const headers = response.headers();
  const url = response.url();
  const now = new Date().toISOString();
  const key = `${request.method()} ${url}`;
  const existing = apis.get(key);

  if (existing) {
    existing.count += 1;
    existing.lastSeenAt = now;
    existing.statuses = Array.from(new Set([...existing.statuses, response.status()]));
    existing.responses.push({
      capturedAt: now,
      status: response.status(),
      data,
    });
    return existing;
  }

  const api = {
    method: request.method(),
    url,
    resourceType: request.resourceType(),
    status: response.status(),
    statuses: [response.status()],
    contentType: headers["content-type"] || null,
    firstSeenAt: now,
    lastSeenAt: now,
    count: 1,
    responses: [
      {
        capturedAt: now,
        status: response.status(),
        data,
      },
    ],
  };

  apis.set(key, api);
  return api;
}

function writeApisFile() {
  const payload = {
    generatedAt: new Date().toISOString(),
    targetUrl: TARGET_URL,
    count: apis.size,
    apis: Array.from(apis.values()).sort((a, b) => a.url.localeCompare(b.url)),
  };

  writeQueue = writeQueue.then(() =>
    fsp.writeFile(
      path.resolve(API_OUTPUT_FILE),
      JSON.stringify(payload, null, 2) + "\n",
      "utf8"
    )
  );

  return writeQueue;
}

async function waitForUserToFinish() {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  try {
    await rl.question(
      "Browser is open. Interact with the page, then press Enter here to finish capture..."
    );
  } finally {
    rl.close();
  }
}

async function waitForQuietNetwork(page, timeout = 5000) {
  await page.waitForLoadState("networkidle", { timeout }).catch(() => {});
}

async function scrollForLazyLoadedData(page) {
  for (let pass = 1; pass <= MAX_SCROLL_PASSES; pass += 1) {
    await page.mouse.wheel(0, SCROLL_DISTANCE_PX);
    await page.waitForTimeout(SCROLL_SETTLE_MS);
    await waitForQuietNetwork(page, SCROLL_SETTLE_MS);

    const scrollState = await page.evaluate(() => ({
      scrollY: window.scrollY,
      innerHeight: window.innerHeight,
      scrollHeight: document.body.scrollHeight,
    }));
    const reachedBottom =
      scrollState.scrollY + scrollState.innerHeight >= scrollState.scrollHeight - 20;

    console.log(
      `Scroll pass ${pass}/${MAX_SCROLL_PASSES}: ${apis.size} data APIs captured`
    );

    if (reachedBottom) break;
  }
}

(async () => {
  if (!TARGET_URL) {
    throw new Error("TARGET_URL environment variable is required");
  }

  const browser = await chromium.launch({
    headless: false,
    slowMo: 50,
  });

  const contextOptions = fs.existsSync(AUTH_STATE_FILE)
    ? { storageState: AUTH_STATE_FILE }
    : {};

  const context = await browser.newContext(contextOptions);

  const page = await context.newPage();

  page.on("response", async (response) => {
    if (!isDataApiResponse(response)) return;

    const capture = (async () => {
      try {
        const data = await readResponseData(response);
        const api = upsertApi(response, data);
        console.log(`DATA API ${api.method} ${api.status}: ${api.url}`);
        await writeApisFile();
      } catch (err) {
        console.warn(`Skipped data API response: ${response.url()}`);
      }
    })();

    pendingCaptures.add(capture);
    capture.finally(() => pendingCaptures.delete(capture));
  });

  await page.goto(TARGET_URL, {
    waitUntil: "networkidle",
  });

  await scrollForLazyLoadedData(page);
  await page.waitForTimeout(FINAL_SETTLE_MS);
  await waitForQuietNetwork(page, FINAL_SETTLE_MS);
  await waitForUserToFinish();
  await Promise.allSettled(Array.from(pendingCaptures));
  await writeApisFile();
  await writeQueue;
  await browser.close();
})().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
