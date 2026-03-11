import {
  cleanupTodosByTitlePrefix,
  cleanupLoadArtifacts,
  createOptionalCounter,
  createOptionalTrend,
  createLoadOptions,
  deleteTodo,
  graphqlRequest,
  isoDatePlusDays,
  jsonParams,
  loadSuffix,
  restBaseUrl,
  expectJsonStatus,
  waitUntil,
} from "./helpers.ts";
import http from "k6/http";
import { fail } from "k6";

export const options = createLoadOptions("todoFlowRestGraphql");
const todoFlowIterationDuration = createOptionalTrend("todo_flow_rest_graphql_iteration_duration");
const todoFlowCreateDuration = createOptionalTrend("todo_flow_rest_graphql_create_duration");
const todoFlowGraphqlUpdateDuration = createOptionalTrend("todo_flow_rest_graphql_graphql_update_duration");
const todoFlowTimeToRestVisible = createOptionalTrend("todo_flow_rest_graphql_time_to_rest_visible");
const todoFlowTimeToGraphqlVisible = createOptionalTrend("todo_flow_rest_graphql_time_to_graphql_visible");
const todoFlowTodosCreated = createOptionalCounter("todo_flow_rest_graphql_todos_created");

/**
 * Validates a mixed REST and GraphQL todo lifecycle in one iteration.
 */
export function todoFlowRestGraphql(): void {
  const iterationStartedAt = Date.now();
  const todoTitlePrefix = `Load Todo ${loadSuffix()}`;
  const todoTitle = todoTitlePrefix;
  const dueDate = isoDatePlusDays(1);
  let todoId = "";

  try {
    const createTodoResp = http.post(
      `${restBaseUrl}/api/v1/todos`,
      JSON.stringify({ title: todoTitle, due_date: dueDate }),
      jsonParams,
    );
    todoFlowCreateDuration.add(Number(createTodoResp.timings?.duration || 0));
    const createdTodo = expectJsonStatus(createTodoResp, 201, "create todo");
    todoId = String(createdTodo.id || "");
    if (!todoId) {
      fail("create todo: missing id");
    }
    todoFlowTodosCreated.add(1);

    const restVisibleStartedAt = Date.now();
    const restVisible = waitUntil(
      () => {
        const createdFromRest = findTodoInRestByTitle(todoTitle, todoId);
        return Boolean(createdFromRest);
      },
      10_000,
      500,
    );
    todoFlowTimeToRestVisible.add(Date.now() - restVisibleStartedAt);
    if (!restVisible) {
      fail("load todo flow: created todo not found in REST search results");
    }

    const graphqlVisibleStartedAt = Date.now();
    const graphqlVisible = waitUntil(
      () => {
        const createdFromGraphql = findTodoInGraphqlByTitle(todoTitle, todoId);
        return Boolean(createdFromGraphql);
      },
      10_000,
      500,
    );
    todoFlowTimeToGraphqlVisible.add(Date.now() - graphqlVisibleStartedAt);
    if (!graphqlVisible) {
      fail("load todo flow: created todo not found in GraphQL search results");
    }

    const updateStartedAt = Date.now();
    const updateTodoData = graphqlRequest(
      `
        mutation UpdateTodo($params: updateTodoParams!) {
          updateTodo(params: $params) {
            id
            status
          }
        }
      `,
      { params: { id: todoId, status: "DONE" } },
      "graphql update todo",
    );
    todoFlowGraphqlUpdateDuration.add(Date.now() - updateStartedAt);
    if (updateTodoData.updateTodo?.status !== "DONE") {
      fail("load todo flow: GraphQL update did not return DONE");
    }

    const restDone = waitUntil(
      () => {
        const updatedFromRest = findTodoInRestByTitle(todoTitle, todoId);
        if (!updatedFromRest) {
          return false;
        }
        return String(updatedFromRest.status || "") === "DONE";
      },
      10_000,
      500,
    );
    if (!restDone) {
      fail("load todo flow: updated todo did not reach DONE in REST search results");
    }
  } finally {
    deleteTodo(todoId);
    cleanupTodosByTitlePrefix(todoTitlePrefix);
    todoFlowIterationDuration.add(Date.now() - iterationStartedAt);
  }
}

/**
 * k6 default entrypoint.
 */
export default function (): void {
  todoFlowRestGraphql();
}

/**
 * Cleans up load-test artifacts created by this scenario.
 */
export function teardown(): void {
  cleanupLoadArtifacts();
}

/**
 * Finds one todo by exact title and id via REST list filtering.
 */
function findTodoInRestByTitle(title: string, todoId: string): Record<string, any> | null {
  const search = encodeURIComponent(title);
  const resp = http.get(
    `${restBaseUrl}/api/v1/todos?page=1&pageSize=20&search=${search}&searchType=TITLE&sort=createdAtDesc`,
    { tags: { name: "GET /api/v1/todos by-title" } },
  );
  const payload = expectJsonStatus(resp, 200, "rest list todos by title");
  const items = payload.items || [];
  const match = items.find((item: Record<string, any>) =>
    String(item.id || "") === todoId && String(item.title || "") === title,
  );
  return match || null;
}

/**
 * Finds one todo by exact title and id via GraphQL list filtering.
 */
function findTodoInGraphqlByTitle(title: string, todoId: string): Record<string, any> | null {
  const data = graphqlRequest(
    `
      query ListTodos($status: TodoStatus, $search: String, $searchType: SearchType, $page: Int!, $pageSize: Int!, $sortBy: TodoSortBy) {
        listTodos(status: $status, search: $search, searchType: $searchType, page: $page, pageSize: $pageSize, sortBy: $sortBy) {
          items { id title status }
        }
      }
    `,
    {
      status: null,
      search: title,
      searchType: "TITLE",
      page: 1,
      pageSize: 20,
      sortBy: "createdAtDesc",
    },
    "graphql list todos by title",
  );
  const items = data.listTodos?.items || [];
  const match = items.find((item: Record<string, any>) =>
    String(item.id || "") === todoId && String(item.title || "") === title,
  );
  return match || null;
}
