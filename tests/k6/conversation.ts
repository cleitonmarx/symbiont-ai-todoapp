import http from "k6/http";
import { fail, sleep } from "k6";
import {
  chatParams,
  cleanupConversationsByMessageToken,
  cleanupLoadArtifacts,
  createOptionalCounter,
  createOptionalTrend,
  createLoadOptions,
  deleteConversation,
  expectJsonStatus,
  fetchChatModel,
  loadChatTimeout,
  loadSuffix,
  restBaseUrl,
  waitUntil,
} from "./helpers.ts";

export const options = createLoadOptions("conversation");
const conversationIterationDuration = createOptionalTrend("conversation_iteration_duration");
const conversationChatFlowDuration = createOptionalTrend("conversation_chat_flow_duration");
const conversationTimeToTitleConverged = createOptionalTrend("conversation_time_to_title_converged");
const conversationConversationsCreated = createOptionalCounter("conversation_conversations_created");
const conversationChatMessagesObserved = createOptionalCounter("conversation_chat_messages_observed");
const chatRetryCount = Number(__ENV.K6_LOAD_CHAT_RETRY_COUNT || "2");
const chatRetryBackoffSeconds = Number(__ENV.K6_LOAD_CHAT_RETRY_BACKOFF_SECONDS || "1");

/**
 * Exercises multi-turn chat and waits for asynchronous conversation title retitling.
 */
export function conversation(): void {
  const iterationStartedAt = Date.now();
  let conversationId = "";
  let memoryTokenA = "";

  try {
    const model = fetchChatModel();
    memoryTokenA = `tokena-${loadSuffix()}`;
    const firstPrompt = `Memory token A is ${memoryTokenA}. Reply with acknowledged.`;
    const chatFlowStartedAt = Date.now();
    const firstTurn = runChatTurn(model, null, firstPrompt);
    conversationId = firstTurn.conversationId;
    conversationConversationsCreated.add(1);

    if (!firstTurn.reply.trim()) {
      fail("load chat: empty assistant reply for initial prompt");
    }

    const prompts = [
      "Memory token B is nebula-7. Acknowledge.",
      "Memory token C is quasar-2. Acknowledge.",
      "Memory token D is pulsar-5. Acknowledge.",
      "Memory token E is comet-9. Acknowledge.",
      "Memory token F is asteroid-4. Acknowledge.",
      "Memory token G is galaxy-8. Acknowledge.",
    ];

    for (const prompt of prompts) {
      const turn = runChatTurn(model, conversationId, prompt);
      if (turn.conversationId !== conversationId) {
        fail("load chat: conversation id changed unexpectedly");
      }
      if (!turn.reply.trim()) {
        fail(`load chat: empty assistant reply for prompt: ${prompt}`);
      }
    }
    conversationChatFlowDuration.add(Date.now() - chatFlowStartedAt);

    const messagesResp = http.get(
      http.url`${restBaseUrl}/api/v1/chat/messages?conversation_id=${conversationId}&page=1&pageSize=200`,
      { tags: { name: "GET /api/v1/chat/messages by-conversation" } },
    );
    const messagesPayload = expectJsonStatus(messagesResp, 200, "list conversation messages");
    const messages = messagesPayload.messages || [];
    if (messages.length !== 14) {
      fail(`load chat: expected 14 messages, got ${messages.length}`);
    }
    conversationChatMessagesObserved.add(messages.length);
    const hasTokenA = messages.some((msg: Record<string, any>) =>
      String(msg.content || "")
        .toLowerCase()
        .includes(memoryTokenA.toLowerCase()),
    );
    if (!hasTokenA) {
      fail("load chat: conversation history missing memory token A");
    }

    let titlePollCount = 0;
    let lastTitleObservation = "";
    const titleWaitStartedAt = Date.now();
    const titleConverged = waitUntil(
      () => {
        titlePollCount += 1;
        const conversation = getConversationById(conversationId);
        if (!conversation) {
          return false;
        }
        const source = String(conversation.title_source || "");
        const title = String(conversation.title || "").trim();
        lastTitleObservation = `${source}|${title}`;
        if (source !== "llm") {
          return false;
        }
        return true;
      },
      60_000,
      2_000,
    );
    conversationTimeToTitleConverged.add(Date.now() - titleWaitStartedAt);
    if (!titleConverged) {
      const [lastSource = "", lastTitle = ""] = lastTitleObservation.split("|", 2);
      fail(
        `load chat: conversation title did not transition to LLM source (conversationId=${conversationId}, polls=${titlePollCount}, lastSource=${lastSource}, lastTitle=${lastTitle})`,
      );
    }
  } finally {
    deleteConversation(conversationId);
    cleanupConversationsByMessageToken(memoryTokenA);
    conversationIterationDuration.add(Date.now() - iterationStartedAt);
  }
}

/**
 * k6 default entrypoint.
 */
export default function (): void {
  conversation();
}

/**
 * Cleans up load-test artifacts created by this scenario.
 */
export function teardown(): void {
  cleanupLoadArtifacts();
}

/**
 * Runs one synchronous chat turn and parses streamed SSE output.
 */
function runChatTurn(
  model: string,
  conversationId: string | null,
  message: string,
): { conversationId: string; reply: string } {
  const requestBody: Record<string, any> = { model, message };
  if (conversationId) {
    requestBody.conversation_id = conversationId;
  }

  let lastStatus = 0;
  for (let attempt = 0; attempt <= chatRetryCount; attempt += 1) {
    const streamResp = http.post(
      `${restBaseUrl}/api/v1/chat`,
      JSON.stringify(requestBody),
      { ...chatParams, timeout: loadChatTimeout },
    );
    lastStatus = streamResp.status;

    if (streamResp.status === 200) {
      return readTurnStartedAndDelta(toTextBody(streamResp.body));
    }

    const shouldRetry = streamResp.status === 0 && attempt < chatRetryCount;
    if (shouldRetry) {
      sleep(chatRetryBackoffSeconds * (attempt + 1));
      continue;
    }

    fail(`stream chat: expected 200, got ${streamResp.status}`);
  }

  fail(`stream chat: expected 200, got ${lastStatus}`);
}

/**
 * Looks up one conversation by ID from the conversation list response.
 */
function getConversationById(conversationId: string): Record<string, any> | undefined {
  const resp = http.get(
    `${restBaseUrl}/api/v1/conversations?page=1&pageSize=200`,
  );
  const payload = expectJsonStatus(resp, 200, "list conversations");
  const conversations = payload.conversations || [];
  return conversations.find(
    (conversation: Record<string, any>) => conversation.id === conversationId,
  );
}

/**
 * Parses streamed chat SSE payload and extracts conversation ID and assistant text.
 */
function readTurnStartedAndDelta(streamBody: string): { conversationId: string; reply: string } {
  let currentEvent = "";
  let conversationId = "";
  let assistantReply = "";
  let turnCompleted = false;
  const lines = String(streamBody).split(/\r?\n/);

  for (const line of lines) {
    if (!line) {
      continue;
    }

    if (line.startsWith("event:")) {
      currentEvent = line.slice(6).trim();
      continue;
    }

    if (!line.startsWith("data:")) {
      continue;
    }

    const rawPayload = line.slice(5).trim();
    const payload = safeJson(rawPayload, `chat SSE event ${currentEvent}`);

    if (currentEvent === "turn_started") {
      if (payload.conversation_id) {
        conversationId = String(payload.conversation_id);
      }
      continue;
    }

    if (currentEvent === "message_delta") {
      assistantReply += String(payload.text || "");
      continue;
    }

    if (currentEvent === "action_approval_required") {
      fail("load chat: unexpected action_approval_required event");
    }

    if (currentEvent === "turn_completed") {
      turnCompleted = true;
    }
  }

  if (!conversationId) {
    fail("load chat: missing conversation_id in turn_started event");
  }
  if (!turnCompleted) {
    fail("load chat: missing turn_completed event");
  }

  return {
    conversationId,
    reply: assistantReply,
  };
}

/**
 * Converts k6 response body variants to text.
 */
function toTextBody(body: string | ArrayBuffer | null): string {
  if (typeof body === "string") {
    return body;
  }
  if (!body) {
    return "";
  }
  return new TextDecoder("utf-8").decode(body);
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
