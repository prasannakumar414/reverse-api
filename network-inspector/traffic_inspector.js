const { chromium } = require("playwright");
const fs = require("fs");
const fsp = require("fs/promises");
const path = require("path");
const readline = require("readline/promises");

const ENV_FILE = ".env";
const AUTH_STATE_FILE = "auth-state.json";
const API_OUTPUT_FILE = "apis.json";
const HTML_API_OUTPUT_FILE = "html_apis.json";

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
const MAX_DATA_CHARS = 4000;
const MAX_REQUEST_BODY_CHARS = 250000;
const EXCLUDED_CONTENT_TYPES = ["text/css", "text/javascript"];

const apiEntries = [];
const htmlApiEntries = [];
const pendingCaptures = new Set();
let writeQueue = Promise.resolve();

function truncateData(data) {
  if (data == null) return null;

  const text = typeof data === "string" ? data : JSON.stringify(data);
  if (text.length <= MAX_DATA_CHARS) return text;

  return `${text.slice(0, MAX_DATA_CHARS)}... [truncated]`;
}

function truncateRequestBody(data) {
  if (data == null) {
    return {
      requestPostData: null,
      requestPostDataTruncated: false,
      requestPostDataLength: 0,
    };
  }

  if (data.length <= MAX_REQUEST_BODY_CHARS) {
    return {
      requestPostData: data,
      requestPostDataTruncated: false,
      requestPostDataLength: data.length,
    };
  }

  return {
    requestPostData: `${data.slice(0, MAX_REQUEST_BODY_CHARS)}... [truncated]`,
    requestPostDataTruncated: true,
    requestPostDataLength: data.length,
  };
}

function shouldCaptureRequestBody(request) {
  return (
    request.method() === "POST" &&
    request.url().includes("/flagship-web/rsc-action/")
  );
}

function buildRequestCapture(request) {
  if (!shouldCaptureRequestBody(request)) return {};

  const headers = request.headers();
  return {
    requestContentType: headers["content-type"] || null,
    ...truncateRequestBody(request.postData()),
  };
}

function isExcludedContentType(response) {
  const contentType = (response.headers()["content-type"] || "").toLowerCase();

  return EXCLUDED_CONTENT_TYPES.some((excludedType) =>
    contentType.startsWith(excludedType)
  );
}

function isHtmlContentType(response) {
  const contentType = (response.headers()["content-type"] || "").toLowerCase();

  return contentType.includes("text/html");
}

async function readTruncatedResponseData(response) {
  const contentType = (response.headers()["content-type"] || "").toLowerCase();

  if (contentType.includes("json")) {
    return truncateData(await response.json());
  }

  if (
    contentType.startsWith("text/") ||
    contentType.includes("javascript") ||
    contentType.includes("xml") ||
    contentType.includes("html") ||
    contentType.includes("form-urlencoded")
  ) {
    return truncateData(await response.text());
  }

  const body = await response.body();
  if (!body.length) return null;

  return truncateData(body.toString("base64"));
}

async function readHtmlResponseData(response) {
  return response.text();
}

function buildEntry(response, data, dataField = "truncatedData") {
  const request = response.request();
  const headers = response.headers();
  const now = new Date().toISOString();

  return {
    api: response.url(),
    type: request.resourceType(),
    statusCode: response.status(),
    [dataField]: data,
    method: request.method(),
    contentType: headers["content-type"] || null,
    ...buildRequestCapture(request),
    capturedAt: now,
  };
}

function recordApi(response, truncatedData) {
  const entry = buildEntry(response, truncatedData);
  apiEntries.push(entry);
  return entry;
}

function recordHtmlApi(response, truncatedData) {
  const entry = buildEntry(response, truncatedData, "data");
  htmlApiEntries.push(entry);
  return entry;
}

function writeEntriesFile(outputFile, entries) {
  const payload = {
    generatedAt: new Date().toISOString(),
    targetUrl: TARGET_URL,
    count: entries.length,
    apis: entries,
  };
  const outputPath = path.resolve(outputFile);
  const tempPath = `${outputPath}.tmp`;

  writeQueue = writeQueue.then(() =>
    fsp
      .writeFile(tempPath, JSON.stringify(payload, null, 2) + "\n", "utf8")
      .then(() => fsp.rename(tempPath, outputPath))
  );

  return writeQueue;
}

function writeApisFile() {
  return writeEntriesFile(API_OUTPUT_FILE, apiEntries);
}

function writeHtmlApisFile() {
  return writeEntriesFile(HTML_API_OUTPUT_FILE, htmlApiEntries);
}

function writeOutputFiles() {
  writeApisFile();
  writeHtmlApisFile();
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
      `Scroll pass ${pass}/${MAX_SCROLL_PASSES}: ${apiEntries.length} responses, ${htmlApiEntries.length} HTML responses captured`
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
    if (isExcludedContentType(response)) return;

    const capture = (async () => {
      try {
        const isHtml = isHtmlContentType(response);
        const data = isHtml
          ? await readHtmlResponseData(response)
          : await readTruncatedResponseData(response);
        const entry = isHtml
          ? recordHtmlApi(response, data)
          : recordApi(response, data);
        const label = isHtml ? "HTML" : "RESPONSE";
        console.log(
          `${label} ${entry.method} ${entry.statusCode} ${entry.type}: ${entry.api}`
        );
        await writeOutputFiles();
      } catch (err) {
        const isHtml = isHtmlContentType(response);
        const entry = isHtml ? recordHtmlApi(response, null) : recordApi(response, null);
        console.warn(`Captured response without body data: ${entry.api}`);
        await writeOutputFiles();
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
  await writeOutputFiles();
  await writeQueue;
  await browser.close();
})().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
