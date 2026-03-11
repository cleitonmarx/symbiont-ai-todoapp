import http, { RefinedResponse } from "k6/http";
import { check, fail, sleep } from "k6";
import { Counter, Trend } from "k6/metrics";
import type { Options, Scenario } from "k6/options";
import exec from "k6/execution";

export const restBaseUrl = trimTrailingSlash(
  __ENV.K6_REST_BASE_URL || "http://localhost:8080",
);

export const graphqlEndpoint =
  __ENV.K6_GRAPHQL_ENDPOINT || "http://localhost:8085/v1/query";

export const jsonParams = {
  headers: {
    "Content-Type": "application/json",
    Accept: "application/json",
  },
};

export const chatParams = {
  headers: {
    "Content-Type": "application/json",
    Accept: "text/event-stream",
  },
};
const cleanupDeleteResponseCallback = http.expectedStatuses(204, 404);
const cleanupMessagesResponseCallback = http.expectedStatuses(200, 404);
export const loadChatTimeout = __ENV.K6_LOAD_CHAT_TIMEOUT || "120s";
export const loadCustomMetricsEnabled = parseBoolEnv(__ENV.K6_LOAD_CUSTOM_METRICS || "false");
type MetricAdder = {
  add: (value: number, tags?: Record<string, string>) => void;
};
const noopMetric: MetricAdder = {
  add: (_value: number, _tags?: Record<string, string>): void => {},
};

type LoadExecutionStrategy = "regular" | "smoke" | "spike" | "stress";
type LoadScenarioDefaults = {
  targetVUsDefault?: number;
  startVUsDefault?: number;
};

/**
 * Builds a k6 scenario configuration using the selected load execution strategy.
 */
export function createLoadScenario(
  execName: string,
  startTime = "0s",
  defaults: LoadScenarioDefaults = {},
): Scenario {
  const targetVUs = Number(__ENV.K6_LOAD_TARGET_VUS || String(defaults.targetVUsDefault ?? 10));
  const startVUs = Number(__ENV.K6_LOAD_START_VUS || String(defaults.startVUsDefault ?? 1));
  const gracefulRampDown = __ENV.K6_LOAD_GRACEFUL_RAMP_DOWN || "30s";
  const gracefulStop = __ENV.K6_LOAD_GRACEFUL_STOP || "3m";
  const strategyRaw = String(__ENV.K6_LOAD_EXECUTION_STRATEGY || "regular")
    .trim()
    .toLowerCase();
  const strategy = (
    ["regular", "smoke", "spike", "stress"].includes(strategyRaw)
      ? strategyRaw
      : "regular"
  ) as LoadExecutionStrategy;

  if (strategyRaw !== strategy) {
    console.warn(
      `Unknown K6_LOAD_EXECUTION_STRATEGY='${strategyRaw}', falling back to 'regular'`,
    );
  }

  if (strategy === "smoke") {
    return {
      executor: "constant-vus" as const,
      exec: execName,
      startTime,
      vus: Number(__ENV.K6_LOAD_SMOKE_VUS || "1"),
      duration: __ENV.K6_LOAD_SMOKE_DURATION || "30s",
      gracefulStop,
    };
  }

  if (strategy === "spike") {
    const spikeTarget = Number(
      __ENV.K6_LOAD_SPIKE_TARGET_VUS || String(Math.max(targetVUs * 3, targetVUs + 1)),
    );
    return {
      executor: "ramping-vus" as const,
      exec: execName,
      startVUs,
      startTime,
      gracefulRampDown,
      stages: [
        { duration: __ENV.K6_LOAD_SPIKE_RAMP_UP || "20s", target: spikeTarget },
        { duration: __ENV.K6_LOAD_SPIKE_HOLD || "40s", target: spikeTarget },
        { duration: __ENV.K6_LOAD_SPIKE_RAMP_DOWN || "20s", target: 0 },
      ],
    };
  }

  if (strategy === "stress") {
    const stage2Target = Number(__ENV.K6_LOAD_STRESS_STAGE2_TARGET_VUS || String(targetVUs * 2));
    const stage3Target = Number(__ENV.K6_LOAD_STRESS_STAGE3_TARGET_VUS || String(targetVUs * 3));
    return {
      executor: "ramping-vus" as const,
      exec: execName,
      startVUs,
      startTime,
      gracefulRampDown,
      stages: [
        { duration: __ENV.K6_LOAD_STRESS_STAGE1_DURATION || "1m", target: targetVUs },
        { duration: __ENV.K6_LOAD_STRESS_STAGE2_DURATION || "1m", target: stage2Target },
        { duration: __ENV.K6_LOAD_STRESS_STAGE3_DURATION || "1m", target: stage3Target },
        { duration: __ENV.K6_LOAD_STRESS_RAMP_DOWN || "1m", target: 0 },
      ],
    };
  }

  return {
    executor: "ramping-vus" as const,
    exec: execName,
    startVUs,
    startTime,
    gracefulRampDown,
    stages: [
      { duration: __ENV.K6_LOAD_RAMP_UP || "1m", target: targetVUs },
      { duration: __ENV.K6_LOAD_STEADY || "1m", target: targetVUs },
      { duration: __ENV.K6_LOAD_RAMP_DOWN || "1m", target: 0 },
    ],
  };
}

/**
 * Creates default k6 options for a single exported exec function.
 */
export function createLoadOptions(execName: string): Options {
  return {
    scenarios: {
      [execName]: createLoadScenario(execName),
    },
    thresholds: {
      checks: ["rate==1.0"],
      http_req_failed: ["rate<0.01"],
    },
  };
}

/**
 * Generates a unique suffix for test entities across VUs and iterations.
 */
export function loadSuffix(): string {
  return `${Date.now()}-vu${exec.vu.idInTest}-it${exec.vu.iterationInScenario}-r${Math.floor(Math.random() * 1_000_000)}`;
}

/**
 * Returns the preferred chat model ID, favoring `qwen3` when available.
 */
export function fetchChatModel(): string {
  const modelsResp = http.get(`${restBaseUrl}/api/v1/models`);
  const payload = expectJsonStatus(modelsResp, 200, "list available models");
  const models = payload.models || [];
  if (models.length === 0) {
    fail("list available models: expected at least one model");
  }

  return String(models[0].id);
}

/**
 * Executes a GraphQL request and returns the `data` object.
 */
export function graphqlRequest(
  query: string,
  variables: Record<string, any>,
  label: string,
): Record<string, any> {
  const resp = http.post(
    graphqlEndpoint,
    JSON.stringify({ query, variables }),
    jsonParams,
  );

  const payload = expectJsonStatus(resp, 200, label);
  if (payload.errors && payload.errors.length > 0) {
    fail(`${label}: GraphQL returned errors: ${JSON.stringify(payload.errors)}`);
  }
  if (!payload.data) {
    fail(`${label}: GraphQL response missing data`);
  }

  return payload.data;
}

/**
 * Asserts HTTP status and parses JSON response body.
 */
export function expectJsonStatus(
  response: RefinedResponse<"text">,
  expectedStatus: number,
  label: string,
): Record<string, any> {
  const ok = check(response, {
    [`${label}: status ${expectedStatus}`]: (r) => r.status === expectedStatus,
  });
  if (!ok) {
    fail(`${label}: expected status ${expectedStatus}, got ${response.status}`);
  }

  return parseJson(response, label);
}

/**
 * Parses a non-empty JSON response body.
 */
export function parseJson(
  response: RefinedResponse<"text">,
  label: string,
): Record<string, any> {
  const body = String(response.body || "").trim();
  if (!body) {
    fail(`${label}: expected JSON body, got empty response`);
  }
  return safeJson(body, label);
}

/**
 * Safely parses a JSON string or fails with a labeled error.
 */
function safeJson(raw: string, label: string): Record<string, any> {
  try {
    return JSON.parse(raw);
  } catch (error) {
    fail(`${label}: invalid JSON payload (${error})`);
  }
}

/**
 * Polls a predicate until success or timeout.
 */
export function waitUntil(
  predicate: () => boolean,
  timeoutMs: number,
  intervalMs: number,
): boolean {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() <= deadline) {
    if (predicate()) {
      return true;
    }
    sleep(intervalMs / 1000);
  }
  return false;
}

/**
 * Returns an ISO date string in UTC offset by the given number of days.
 */
export function isoDatePlusDays(days: number): string {
  const date = new Date();
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString().slice(0, 10);
}

/**
 * Deletes a todo and tolerates 204/404 cleanup responses.
 */
export function deleteTodo(todoId: string): void {
  if (!todoId) {
    return;
  }

  const deleteResp = http.del(
    http.url`${restBaseUrl}/api/v1/todos/${todoId}`,
    null,
    {
      tags: { name: "DELETE /api/v1/todos/:id" },
      responseCallback: cleanupDeleteResponseCallback,
    },
  );
  check(deleteResp, {
    "cleanup todo returns 204/404": (r) => r.status === 204 || r.status === 404,
  });
}

/**
 * Deletes a conversation and tolerates 204/404 cleanup responses.
 */
export function deleteConversation(conversationId: string): void {
  if (!conversationId) {
    return;
  }

  const deleteResp = http.del(
    http.url`${restBaseUrl}/api/v1/conversations/${conversationId}`,
    null,
    {
      tags: { name: "DELETE /api/v1/conversations/:id" },
      responseCallback: cleanupDeleteResponseCallback,
    },
  );
  check(deleteResp, {
    "cleanup conversation returns 204/404": (r) => r.status === 204 || r.status === 404,
  });
}

/**
 * Finds and deletes all todos whose title starts with the provided prefix.
 */
export function cleanupTodosByTitlePrefix(titlePrefix: string): void {
  const prefix = String(titlePrefix || "").trim();
  if (!prefix) {
    return;
  }
  const encodedSearch = encodeURIComponent(prefix);
  // Keep cleanup page size conservative to avoid request-validator mismatches across stacks.
  const pageSize = 200;
  const maxPages = 2000;

  for (let attempt = 0; attempt < 3; attempt += 1) {
    const todoIdsToDelete: string[] = [];
    const seenTodoIds = new Set<string>();
    let page = 1;
    let pagesRead = 0;
    let failed = false;

    while (pagesRead < maxPages) {
      const listURL =
        `${restBaseUrl}/api/v1/todos?page=${page}&pageSize=${pageSize}` +
        `&search=${encodedSearch}&searchType=TITLE&sort=createdAtDesc`;
      const listResp = http.get(
        listURL,
        { tags: { name: "GET /api/v1/todos cleanup-by-title" } },
      );
      if (listResp.status !== 200) {
        failed = true;
        break;
      }

      const payload = tryParseResponseJson(listResp);
      if (!payload || !Array.isArray(payload.items)) {
        failed = true;
        break;
      }

      for (const todo of payload.items) {
        const todoId = String(todo?.id || "");
        const title = String(todo?.title || "");
        if (!todoId || seenTodoIds.has(todoId)) {
          continue;
        }
        seenTodoIds.add(todoId);
        if (title.startsWith(prefix)) {
          todoIdsToDelete.push(todoId);
        }
      }

      const nextPageRaw = payload.next_page;
      if (nextPageRaw === null || nextPageRaw === undefined) {
        break;
      }

      const nextPage = Number(nextPageRaw);
      if (!Number.isInteger(nextPage) || nextPage < 1) {
        break;
      }

      page = nextPage;
      pagesRead += 1;
    }

    if (failed) {
      sleep(1 + attempt);
      continue;
    }

    for (const todoId of todoIdsToDelete) {
      deleteTodo(todoId);
    }

    return;
  }
}

/**
 * Deletes conversations whose message content contains the provided marker token.
 */
export function cleanupConversationsByMessageToken(token: string): void {
  const marker = String(token || "").trim().toLowerCase();
  if (!marker) {
    return;
  }

  for (let attempt = 0; attempt < 3; attempt += 1) {
    const listResp = http.get(`${restBaseUrl}/api/v1/conversations?page=1&pageSize=500`);
    if (listResp.status !== 200) {
      sleep(1 + attempt);
      continue;
    }

    const payload = tryParseResponseJson(listResp);
    if (!payload || !Array.isArray(payload.conversations)) {
      sleep(1 + attempt);
      continue;
    }

    for (const conversation of payload.conversations) {
      const conversationId = String(conversation?.id || "");
      if (!conversationId) {
        continue;
      }

      const messagesResp = http.get(
        http.url`${restBaseUrl}/api/v1/chat/messages?conversation_id=${conversationId}&page=1&pageSize=500`,
        {
          tags: { name: "GET /api/v1/chat/messages by-conversation" },
          responseCallback: cleanupMessagesResponseCallback,
        },
      );
      if (messagesResp.status !== 200) {
        continue;
      }

      const messagesPayload = tryParseResponseJson(messagesResp);
      if (!messagesPayload || !Array.isArray(messagesPayload.messages)) {
        continue;
      }

      const hasToken = messagesPayload.messages.some((msg: Record<string, any>) =>
        String(msg?.content || "")
          .toLowerCase()
          .includes(marker),
      );
      if (hasToken) {
        deleteConversation(conversationId);
      }
    }

    return;
  }
}

/**
 * Cleans all known load-test todo and conversation artifacts.
 */
export function cleanupLoadArtifacts(): void {
  cleanupTodosByTitlePrefix("Load Todo ");
  cleanupTodosByTitlePrefix("Load Board Summary Todo ");
  cleanupTodosByTitlePrefix("Load Burst Todo ");
  cleanupTodosByTitlePrefix("Load Approval Flow Todo ");
  cleanupConversationsByMessageToken("tokena-");
  cleanupConversationsByMessageToken("approval-load-flow-token-");
}

/**
 * Creates a Trend metric only when custom load metrics are enabled.
 */
export function createOptionalTrend(name: string, isTime = true): Pick<Trend, "add"> {
  if (!loadCustomMetricsEnabled) {
    return noopMetric;
  }
  return new Trend(name, isTime);
}

/**
 * Creates a Counter metric only when custom load metrics are enabled.
 */
export function createOptionalCounter(name: string): MetricAdder {
  if (!loadCustomMetricsEnabled) {
    return noopMetric;
  }
  return new Counter(name);
}

/**
 * Parses JSON response body and returns `null` on invalid payloads.
 */
function tryParseResponseJson(
  response: RefinedResponse<"text">,
): Record<string, any> | null {
  const body = String(response.body || "").trim();
  if (!body) {
    return null;
  }
  try {
    return JSON.parse(body);
  } catch {
    return null;
  }
}

/**
 * Removes trailing slashes from a base URL.
 */
function trimTrailingSlash(value: string): string {
  return String(value).replace(/\/+$/, "");
}

/**
 * Parses common boolean env values (for example `true`, `1`, `yes`, `on`).
 */
function parseBoolEnv(value: string | undefined): boolean {
  const normalized = String(value || "").trim().toLowerCase();
  return normalized === "1" || normalized === "true" || normalized === "yes" || normalized === "on";
}
