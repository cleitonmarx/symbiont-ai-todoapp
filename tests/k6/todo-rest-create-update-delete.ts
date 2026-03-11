import http from "k6/http";
import { fail } from "k6";
import {
  createOptionalCounter,
  createOptionalTrend,
  createLoadOptions,
  expectJsonStatus,
  isoDatePlusDays,
  jsonParams,
  loadSuffix,
  restBaseUrl,
} from "./helpers.ts";

export const options = createLoadOptions("todoRestCreateUpdateDelete");
const todoRestCudIterationDuration = createOptionalTrend("todo_rest_create_update_delete_iteration_duration");
const todoRestCudCreateDuration = createOptionalTrend("todo_rest_create_update_delete_create_duration");
const todoRestCudUpdateDuration = createOptionalTrend("todo_rest_create_update_delete_update_duration");
const todoRestCudDeleteDuration = createOptionalTrend("todo_rest_create_update_delete_delete_duration");
const todoRestCudTodosCreated = createOptionalCounter("todo_rest_create_update_delete_todos_created");

/**
 * Measures REST CRUD throughput for todo create, update, and delete only.
 */
export function todoRestCreateUpdateDelete(): void {
  const iterationStartedAt = Date.now();
  const createResp = http.post(
    `${restBaseUrl}/api/v1/todos`,
    JSON.stringify({
      title: `Load Todo CUD ${loadSuffix()}`,
      due_date: isoDatePlusDays(1),
    }),
    jsonParams,
  );
  todoRestCudCreateDuration.add(Number(createResp.timings?.duration || 0));
  const createdTodo = expectJsonStatus(createResp, 201, "rest create todo");
  const todoId = String(createdTodo.id || "");
  if (!todoId) {
    fail("rest create todo: missing id");
  }
  todoRestCudTodosCreated.add(1);

  const updateResp = http.patch(
    http.url`${restBaseUrl}/api/v1/todos/${todoId}`,
    JSON.stringify({ status: "DONE" }),
    { ...jsonParams, tags: { name: "PATCH /api/v1/todos/:id" } },
  );
  todoRestCudUpdateDuration.add(Number(updateResp.timings?.duration || 0));
  const updatedTodo = expectJsonStatus(updateResp, 200, "rest update todo");
  if (String(updatedTodo.id || "") !== todoId) {
    fail("rest update todo: response id mismatch");
  }
  if (updatedTodo.status !== "DONE") {
    fail(`rest update todo: expected DONE, got ${updatedTodo.status}`);
  }

  const deleteResp = http.del(
    http.url`${restBaseUrl}/api/v1/todos/${todoId}`,
    null,
    { tags: { name: "DELETE /api/v1/todos/:id" } },
  );
  todoRestCudDeleteDuration.add(Number(deleteResp.timings?.duration || 0));
  if (deleteResp.status !== 204) {
    fail(`rest delete todo: expected status 204, got ${deleteResp.status}`);
  }

  todoRestCudIterationDuration.add(Date.now() - iterationStartedAt);
}

/**
 * k6 default entrypoint.
 */
export default function (): void {
  todoRestCreateUpdateDelete();
}
