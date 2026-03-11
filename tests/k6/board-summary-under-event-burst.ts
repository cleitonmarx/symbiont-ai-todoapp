import http from "k6/http";
import { fail } from "k6";
import {
  cleanupLoadArtifacts,
  createOptionalCounter,
  createOptionalTrend,
  createLoadOptions,
  expectJsonStatus,
  graphqlRequest,
  isoDatePlusDays,
  jsonParams,
  loadSuffix,
  parseJson,
  restBaseUrl,
  waitUntil,
} from "./helpers.ts";

export const options = createLoadOptions("boardSummaryUnderEventBurst");
const boardSummaryBurstIterationDuration = createOptionalTrend("board_summary_under_event_burst_iteration_duration");
const boardSummaryBurstCreateBatchDuration = createOptionalTrend("board_summary_under_event_burst_create_batch_duration");
const boardSummaryBurstConvergenceDuration = createOptionalTrend("board_summary_under_event_burst_convergence_duration");
const boardSummaryBurstTodosCreated = createOptionalCounter("board_summary_under_event_burst_todos_created");

/**
 * Generates a burst of todo events and validates board summary convergence.
 */
export function boardSummaryUnderEventBurst(): void {
  const iterationStartedAt = Date.now();
  try {
    const totalCreated = 200;
    const createdIds: string[] = [];
    const prefix = `Load Burst Todo ${loadSuffix()}`;
    const countsBefore = readCurrentTodoCounts();

    const createRequests = [];
    for (let i = 0; i < totalCreated; i += 1) {
      createRequests.push({
        method: "POST",
        url: `${restBaseUrl}/api/v1/todos`,
        body: JSON.stringify({
          title: `${prefix} #${String(i + 1).padStart(2, "0")}`,
          due_date: isoDatePlusDays(1),
        }),
        params: jsonParams,
      });
    }

    const createBatchStartedAt = Date.now();
    const createResponses = http.batch(createRequests);
    boardSummaryBurstCreateBatchDuration.add(Date.now() - createBatchStartedAt);
    for (let i = 0; i < createResponses.length; i += 1) {
      const resp = createResponses[i];
      if (resp.status !== 201) {
        fail(`burst create todo ${i + 1}: expected 201, got ${resp.status}`);
      }
      const body = parseJson(resp, `burst create todo ${i + 1}`);
      if (!body.id) {
        fail(`burst create todo ${i + 1}: missing id`);
      }
      createdIds.push(String(body.id));
    }
    boardSummaryBurstTodosCreated.add(createdIds.length);

    const toDoneCount = Math.floor(totalCreated / 2);
    const toUpdate = createdIds.slice(0, toDoneCount);
    const updateResult = graphqlUpdateTodosToDone(toUpdate);
    if (updateResult !== toDoneCount) {
      fail(`burst update todos: expected ${toDoneCount} updated todos, got ${updateResult}`);
    }

    const expectedOpen = countsBefore.open + (totalCreated - toDoneCount);
    const expectedDone = countsBefore.done + toDoneCount;

    const convergenceStartedAt = Date.now();
    const converged = waitUntil(
      () => {
        const resp = http.get(`${restBaseUrl}/api/v1/board/summary`);
        if (resp.status !== 200) {
          return false;
        }
        const summary = parseJson(resp, "board summary burst convergence");
        return (
          summary.counts?.OPEN >= expectedOpen &&
          summary.counts?.DONE >= expectedDone
        );
      },
      60_000,
      2_000,
    );
    boardSummaryBurstConvergenceDuration.add(Date.now() - convergenceStartedAt);

    if (!converged) {
      const counts = readCurrentTodoCounts();
      fail(
        `board summary did not converge under burst load expected: (open=${expectedOpen} done=${expectedDone}), got: (open=${counts.open} done=${counts.done})`,
      );
    }
  } finally {
    boardSummaryBurstIterationDuration.add(Date.now() - iterationStartedAt);
  }
}

/**
 * k6 default entrypoint.
 */
export default function (): void {
  boardSummaryUnderEventBurst();
}

/**
 * Cleans up load-test artifacts created by this scenario.
 */
export function teardown(): void {
  cleanupLoadArtifacts();
}

/**
 * Reads current OPEN and DONE todo counts from the list endpoint.
 */
function readCurrentTodoCounts(): { open: number; done: number } {
  const todoList = listTodos(1000);
  let open = 0;
  let done = 0;

  for (const item of todoList.items || []) {
    if (item.status === "OPEN") {
      open += 1;
    }
    if (item.status === "DONE") {
      done += 1;
    }
  }

  return { open, done };
}

/**
 * Lists the first page of todos with a configurable page size.
 */
function listTodos(pageSize: number): Record<string, any> {
  const listResp = http.get(
    `${restBaseUrl}/api/v1/todos?page=1&pageSize=${pageSize}`,
  );
  return expectJsonStatus(listResp, 200, "list todos");
}

/**
 * Executes a batched GraphQL mutation to set multiple todos to DONE.
 */
function graphqlUpdateTodosToDone(todoIds: string[]): number {
  if (todoIds.length === 0) {
    return 0;
  }

  const variables: Record<string, any> = {};
  const variableDefs: string[] = [];
  const mutationFields: string[] = [];

  for (let i = 0; i < todoIds.length; i += 1) {
    const varName = `params${i}`;
    variableDefs.push(`$${varName}: updateTodoParams!`);
    variables[varName] = {
      id: todoIds[i],
      status: "DONE",
    };
    mutationFields.push(
      `updateTodo${i}: updateTodo(params: $${varName}) { id status }`,
    );
  }

  const data = graphqlRequest(
    `mutation UpdateTodos(${variableDefs.join(", ")}) { ${mutationFields.join(" ")} }`,
    variables,
    "graphql burst update todos",
  );

  let updated = 0;
  for (let i = 0; i < todoIds.length; i += 1) {
    const item = data[`updateTodo${i}`];
    if (!item || item.status !== "DONE") {
      fail(`graphql burst update todos: todo index ${i} was not updated to DONE`);
    }
    updated += 1;
  }
  return updated;
}
