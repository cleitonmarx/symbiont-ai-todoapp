import http from "k6/http";
import { fail } from "k6";
import sse from "k6/x/sse";
import type { Options } from "k6/options";
import {
  chatParams,
  cleanupTodosByTitlePrefix,
  createOptionalCounter,
  createOptionalTrend,
  createLoadScenario,
  deleteConversation,
  deleteTodo,
  expectJsonStatus,
  fetchChatModel,
  isoDatePlusDays,
  jsonParams,
  loadChatTimeout,
  loadSuffix,
  restBaseUrl,
} from "./helpers.ts";

const approvalTokenPrefix = "approval-load-flow-token-";
const scenarioDefaults = { targetVUsDefault: 3, startVUsDefault: 1 };
const approvalFlowIterationDuration = createOptionalTrend("approval_flow_iteration_duration");
const approvalFlowApprovalHttpDuration = createOptionalTrend("approval_flow_approval_http_duration");
const approvalFlowTimeToApprovalRequired = createOptionalTrend("approval_flow_time_to_approval_required");
const approvalFlowTimeToApprovalResolved = createOptionalTrend("approval_flow_time_to_approval_resolved");
const approvalFlowTodosCreated = createOptionalCounter("approval_flow_todos_created");
const approvalFlowConversationsCreated = createOptionalCounter("approval_flow_conversations_created");
const approvalFlowChatMessagesCreated = createOptionalCounter("approval_flow_chat_messages_created");

export const options: Options = {
  scenarios: {
    assistantActionApprovalFlow: createLoadScenario(
      "assistantActionApprovalFlow",
      "0s",
      scenarioDefaults,
    ),
  },
  thresholds: {
    checks: ["rate==1.0"],
    http_req_failed: ["rate<0.01"],
  },
};

/**
 * Runs a full assistant approval roundtrip using live SSE events and approval submission.
 */
export function assistantActionApprovalFlow(): void {
  const iterationStartedAt = Date.now();
  let conversationId = "";
  let todoId = "";

  const todoTitlePrefix = `Load Approval Flow Todo ${loadSuffix()}`;
  const todoTitle = `${todoTitlePrefix} target`;
  const token = `${approvalTokenPrefix}${loadSuffix()}`;
  const chatTimeout = loadChatTimeout;

  try {
    const createResp = http.post(
      `${restBaseUrl}/api/v1/todos`,
      JSON.stringify({
        title: todoTitle,
        due_date: isoDatePlusDays(2),
      }),
      jsonParams,
    );
    const createdTodo = expectJsonStatus(createResp, 201, "approval flow create todo");
    todoId = String(createdTodo.id || "");
    if (!todoId) {
      fail("approval flow create todo: missing id");
    }
    approvalFlowTodosCreated.add(1);

    const model = fetchChatModel();

    const message = [
      "/todo-update ",
      `Approval flow token: ${token}.`,
      "Strict instruction: execute exactly one tool call named update_todos.",
      "Never call delete_todos.",
      `Use this exact payload: {"todos":[{"id":"${todoId}","status":"DONE"}]}.`,
      "Do not skip the tool call. Do not only explain. Wait for approval and tool result, then confirm done.",
    ].join(" ");

    let streamError = "";
    let totalEvents = 0;
    let approvalRequiredCount = 0;
    let approvalResolvedCount = 0;
    let approvalsAccepted = 0;
    let turnCompleted = false;
    let conversationCounted = false;
    let chatMessagesCreatedInTurn = 0;
    const resolvedStatuses: string[] = [];
    const actionCompletedPayloads: Array<Record<string, any>> = [];
    const approvedActionCalls: Record<string, boolean> = {};
    const streamStartedAt = Date.now();
    let firstApprovalRequiredAt = 0;
    let firstApprovalResolvedAt = 0;

    const streamResp = sse.open(
      `${restBaseUrl}/api/v1/chat`,
      {
        ...chatParams,
        method: "POST",
        body: JSON.stringify({ model, message }),
        timeout: chatTimeout,
      },
      (client) => {
        client.on("error", (event) => {
          const errorText = String(event?.data || "unknown SSE error");
          if (!streamError) {
            streamError = `approval flow SSE error: ${errorText}`;
          }
          client.close();
        });

        client.on("event", (event) => {
          const eventType = String(event?.name || "").trim();
          if (!eventType) {
            return;
          }

          let payload: Record<string, any> = {};
          const rawData = String(event?.data || "");
          if (rawData) {
            try {
              payload = JSON.parse(rawData);
            } catch (error) {
              return;
            }
          }

          totalEvents += 1;

          if (eventType === "turn_started") {
            if (payload.conversation_id) {
              conversationId = String(payload.conversation_id);
            }
            if (!conversationCounted && conversationId) {
              approvalFlowConversationsCreated.add(1);
              conversationCounted = true;
            }
            if (payload.user_message_id) {
              chatMessagesCreatedInTurn += 1;
            }
            if (payload.assistant_message_id) {
              chatMessagesCreatedInTurn += 1;
            }
            return;
          }

          if (eventType === "action_completed") {
            actionCompletedPayloads.push({
              id: String(payload.id || ""),
              name: String(payload.name || ""),
              success: payload.success,
              approval_status: String(payload.approval_status || ""),
              action_executed: payload.action_executed,
              error: String(payload.error || ""),
            });
            return;
          }

          if (eventType === "action_approval_required") {
            approvalRequiredCount += 1;
            if (firstApprovalRequiredAt === 0) {
              firstApprovalRequiredAt = Date.now();
              approvalFlowTimeToApprovalRequired.add(firstApprovalRequiredAt - streamStartedAt);
            }

            const eventConversationId = String(payload.conversation_id || conversationId || "").trim();
            const turnId = String(payload.turn_id || "").trim();
            const actionCallId = String(payload.action_call_id || "").trim();
            const actionName = String(payload.name || "").trim();

            if (!eventConversationId || !turnId || !actionCallId) {
              if (!streamError) {
                streamError = "approval flow: action_approval_required missing conversation_id, turn_id, or action_call_id";
              }
              client.close();
              return;
            }

            if (approvedActionCalls[actionCallId]) {
              return;
            }
            approvedActionCalls[actionCallId] = true;

            const approvalBody: Record<string, any> = {
              conversation_id: eventConversationId,
              turn_id: turnId,
              action_call_id: actionCallId,
              status: "APPROVED",
              reason: "approved by k6 xk6-sse flow",
            };
            if (actionName) {
              approvalBody.action_name = actionName;
            }

            const approvalResp = http.post(
              `${restBaseUrl}/api/v1/chat/approvals`,
              JSON.stringify(approvalBody),
              jsonParams,
            );
            approvalFlowApprovalHttpDuration.add(Number(approvalResp.timings?.duration || 0));

            if (approvalResp.status !== 202) {
              if (!streamError) {
                streamError = `approval flow submit approval: expected 202, got ${approvalResp.status}`;
              }
              client.close();
              return;
            }

            approvalsAccepted += 1;
            return;
          }

          if (eventType === "action_approval_resolved") {
            approvalResolvedCount += 1;
            if (firstApprovalResolvedAt === 0) {
              firstApprovalResolvedAt = Date.now();
              if (firstApprovalRequiredAt > 0) {
                approvalFlowTimeToApprovalResolved.add(firstApprovalResolvedAt - firstApprovalRequiredAt);
              } else {
                approvalFlowTimeToApprovalResolved.add(firstApprovalResolvedAt - streamStartedAt);
              }
            }
            const status = String(payload.status || "").trim().toUpperCase();
            if (status) {
              resolvedStatuses.push(status);
            }
            return;
          }

          if (eventType === "turn_completed") {
            turnCompleted = true;
            client.close();
          }
        });
      },
    );

    if (streamResp.status !== 200) {
      fail(`approval flow stream chat: expected 200, got ${streamResp.status}`);
    }
    if (streamError) {
      fail(streamError);
    }
    const completedActionNames = actionCompletedPayloads.map((action) =>
      String(action.name || "").trim().toLowerCase(),
    );
    if (completedActionNames.includes("delete_todos")) {
      fail(
        `approval flow: model called delete_todos unexpectedly; action_completed=${JSON.stringify(actionCompletedPayloads)}`,
      );
    }
    if (!completedActionNames.includes("update_todos")) {
      fail(
        `approval flow: expected update_todos action_completed; got ${JSON.stringify(actionCompletedPayloads)}`,
      );
    }
    const updateActionResult = actionCompletedPayloads.find(
      (action) => String(action.name || "").trim().toLowerCase() === "update_todos",
    );
    if (!updateActionResult) {
      fail("approval flow: missing update_todos action_completed payload");
    }
    if (updateActionResult.success !== true) {
      fail(`approval flow: expected update_todos success=true, got ${JSON.stringify(updateActionResult)}`);
    }
    if (updateActionResult.action_executed !== true) {
      fail(
        `approval flow: expected update_todos action_executed=true, got ${JSON.stringify(updateActionResult)}`,
      );
    }
    if (String(updateActionResult.approval_status || "").trim().toUpperCase() !== "APPROVED") {
      fail(
        `approval flow: expected update_todos approval_status=APPROVED, got ${JSON.stringify(updateActionResult)}`,
      );
    }

    if (approvalRequiredCount === 0) {
      fail("approval flow: missing action_approval_required event");
    }
    if (approvalsAccepted === 0) {
      fail("approval flow: no approvals were submitted");
    }
    if (approvalResolvedCount === 0) {
      fail("approval flow: missing action_approval_resolved event");
    }
    if (!turnCompleted) {
      fail("approval flow: missing turn_completed event");
    }
    if (!resolvedStatuses.includes("APPROVED")) {
      fail(`approval flow: expected APPROVED status, got ${resolvedStatuses.join(",") || "none"}`);
    }
    if (chatMessagesCreatedInTurn > 0) {
      approvalFlowChatMessagesCreated.add(chatMessagesCreatedInTurn);
    }
  } finally {
    approvalFlowIterationDuration.add(Date.now() - iterationStartedAt);
    deleteTodo(todoId);
    deleteConversation(conversationId);
    cleanupTodosByTitlePrefix(todoTitlePrefix);
  }
}

/**
 * k6 default entrypoint.
 */
export default function (): void {
  assistantActionApprovalFlow();
}

/**
 * Cleans up load-test artifacts created by this scenario.
 */
export function teardown(): void {
  cleanupTodosByTitlePrefix("Load Approval Flow Todo ");
}
