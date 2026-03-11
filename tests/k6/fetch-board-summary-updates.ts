import http from "k6/http";
import { fail } from "k6";
import {
  cleanupLoadArtifacts,
  createOptionalCounter,
  createOptionalTrend,
  createLoadOptions,
  expectJsonStatus,
  isoDatePlusDays,
  jsonParams,
  loadSuffix,
  parseJson,
  restBaseUrl,
  waitUntil,
} from "./helpers.ts";

export const options = createLoadOptions("fetchBoardSummaryUpdates");
const fetchBoardSummaryIterationDuration = createOptionalTrend("fetch_board_summary_updates_iteration_duration");
const fetchBoardSummaryBaselineDuration = createOptionalTrend("fetch_board_summary_updates_baseline_duration");
const fetchBoardSummaryTimeToUpdate = createOptionalTrend("fetch_board_summary_updates_time_to_update");
const fetchBoardSummaryTodosCreated = createOptionalCounter("fetch_board_summary_updates_todos_created");

/**
 * Creates a todo and waits until board summary OPEN count increases.
 */
export function fetchBoardSummaryUpdates(): void {
  const iterationStartedAt = Date.now();
  try {
    let baselineOpen = 0;
    const todoTitlePrefix = `Load Board Summary Todo ${loadSuffix()}`;

    const baselineStartedAt = Date.now();
    const summaryResp = http.get(`${restBaseUrl}/api/v1/board/summary`);
    fetchBoardSummaryBaselineDuration.add(Date.now() - baselineStartedAt);
    if (summaryResp.status === 200) {
      const summary = parseJson(summaryResp, "board summary baseline");
      baselineOpen = summary.counts?.OPEN || 0;
    } else if (summaryResp.status !== 404) {
      fail(`board summary baseline: expected 200 or 404, got ${summaryResp.status}`);
    }

    const createResp = http.post(
      `${restBaseUrl}/api/v1/todos`,
      JSON.stringify({
        title: todoTitlePrefix,
        due_date: isoDatePlusDays(1),
      }),
      jsonParams,
    );
    const createdTodo = expectJsonStatus(createResp, 201, "create board summary todo");
    const todoId = String(createdTodo.id || "");
    if (!todoId) {
      fail("create board summary todo: missing id");
    }
    fetchBoardSummaryTodosCreated.add(1);

    const waitStartedAt = Date.now();
    const updated = waitUntil(
      () => {
        const resp = http.get(`${restBaseUrl}/api/v1/board/summary`);
        if (resp.status !== 200) {
          return false;
        }
        const data = parseJson(resp, "board summary eventual");
        return (data.counts?.OPEN || 0) > baselineOpen;
      },
      60_000,
      2_000,
    );
    if (!updated) {
      fail("board summary did not update in time after creating a new todo");
    }
    fetchBoardSummaryTimeToUpdate.add(Date.now() - waitStartedAt);
  } finally {
    fetchBoardSummaryIterationDuration.add(Date.now() - iterationStartedAt);
  }
}

/**
 * k6 default entrypoint.
 */
export default function (): void {
  fetchBoardSummaryUpdates();
}

/**
 * Cleans up load-test artifacts created by this scenario.
 */
export function teardown(): void {
  cleanupLoadArtifacts();
}
