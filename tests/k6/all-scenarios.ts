import type { Options, Scenario } from "k6/options";
import { cleanupLoadArtifacts, createLoadScenario } from "./helpers.ts";
import { todoFlowRestGraphql } from "./todo-flow-rest-graphql.ts";
import { fetchBoardSummaryUpdates } from "./fetch-board-summary-updates.ts";
import { boardSummaryUnderEventBurst } from "./board-summary-under-event-burst.ts";
import {
  conversation,
} from "./conversation.ts";
import { todoRestCreateUpdateDelete } from "./todo-rest-create-update-delete.ts";
import {
  assistantActionApprovalFlow,
} from "./assistant-actions-approval-flow.ts";

const allScenariosDefaults = { targetVUsDefault: 3, startVUsDefault: 1 };
type LoadScenarioDefaults = {
  targetVUsDefault?: number;
  startVUsDefault?: number;
};
type ScenarioDefinition = {
  scenarioName: string;
  execName: string;
  defaults?: LoadScenarioDefaults;
};
type ScenarioConfig = {
  scenarios: ScenarioDefinition[];
};

const scenarioConfigs: Record<string, ScenarioConfig> = {
  "todo-flow-rest-graphql": {
    scenarios: [
      {
        scenarioName: "todoFlowRestGraphql",
        execName: "loadTodoFlowRestGraphql",
      },
    ],
  },
  "fetch-board-summary-updates": {
    scenarios: [
      {
        scenarioName: "fetchBoardSummaryUpdates",
        execName: "loadFetchBoardSummaryUpdates",
      },
    ],
  },
  "board-summary-under-event-burst": {
    scenarios: [
      {
        scenarioName: "boardSummaryUnderEventBurst",
        execName: "loadBoardSummaryUnderEventBurst",
      },
    ],
  },
  conversation: {
    scenarios: [
      {
        scenarioName: "conversation",
        execName: "loadConversation",
      },
    ],
  },
  "todo-rest-create-update-delete": {
    scenarios: [
      {
        scenarioName: "todoRestCreateUpdateDelete",
        execName: "loadTodoRestCreateUpdateDelete",
      },
    ],
  },
  "assistant-actions-approval-flow": {
    scenarios: [
      {
        scenarioName: "assistantActionApprovalFlow",
        execName: "loadAssistantActionApprovalFlow",
      },
    ],
  },
};

const regularScenarioKeys = [
  "todo-flow-rest-graphql",
  "fetch-board-summary-updates",
  "board-summary-under-event-burst",
  "conversation",
  "todo-rest-create-update-delete",
  "assistant-actions-approval-flow",
];
const runAllScenariosInParallel = parseBoolEnv(__ENV.K6_LOAD_RUN_ALL_SCENARIOS_PARALLEL || "false");

/**
 * Builds the default all-scenarios set, either sequential or fully parallel.
 */
function createRegularScenarios(): NonNullable<Options["scenarios"]> {
  const scenarios: NonNullable<Options["scenarios"]> = {};

  if (runAllScenariosInParallel) {
    for (const scenarioKey of regularScenarioKeys) {
      const config = scenarioConfigs[scenarioKey];
      const created = createScenarioSet(config, "0s");
      for (const [scenarioName, scenario] of Object.entries(created.scenarios)) {
        scenarios[scenarioName] = scenario;
      }
    }
    return scenarios;
  }

  let startOffsetSeconds = 0;

  for (const scenarioKey of regularScenarioKeys) {
    const config = scenarioConfigs[scenarioKey];
    const created = createScenarioSet(config, `${startOffsetSeconds}s`);
    for (const [scenarioName, scenario] of Object.entries(created.scenarios)) {
      scenarios[scenarioName] = scenario;
    }
    startOffsetSeconds += created.durationSeconds;
  }

  return scenarios;
}

/**
 * Selects scenarios to run based on `K6_LOAD_TEST_SCENARIO`.
 */
function selectScenarios(): NonNullable<Options["scenarios"]> {
  const selectedScenarioRaw = String(
    __ENV.K6_LOAD_TEST_SCENARIO || "regular",
  )
    .trim()
    .toLowerCase();

  if (selectedScenarioRaw === "regular") {
    return createRegularScenarios();
  }

  const scenarioConfig = scenarioConfigs[selectedScenarioRaw];
  if (!scenarioConfig) {
    console.warn(
      `Unknown K6_LOAD_TEST_SCENARIO='${selectedScenarioRaw}', falling back to 'regular'`,
    );
    return createRegularScenarios();
  }

  return createScenarioSet(scenarioConfig, "0s").scenarios;
}

/**
 * Creates one scenario map and returns both config and estimated duration.
 */
function createScenarioSet(
  config: ScenarioConfig,
  startTime: string,
): {
  scenarios: NonNullable<Options["scenarios"]>;
  durationSeconds: number;
} {
  const scenarios: NonNullable<Options["scenarios"]> = {};
  let durationSeconds = 0;

  for (const definition of config.scenarios) {
      const defaults: LoadScenarioDefaults = {
      targetVUsDefault: definition.defaults?.targetVUsDefault ??
        allScenariosDefaults.targetVUsDefault,
      startVUsDefault: definition.defaults?.startVUsDefault ??
        allScenariosDefaults.startVUsDefault,
    };

    const scenario = createLoadScenario(
      definition.execName,
      startTime,
      defaults,
    );
    scenarios[definition.scenarioName] = scenario;
    durationSeconds = Math.max(durationSeconds, estimateScenarioDurationSeconds(scenario));
  }

  return { scenarios, durationSeconds };
}

/**
 * Estimates scenario runtime from executor-specific duration fields.
 */
function estimateScenarioDurationSeconds(scenario: Scenario): number {
  switch (scenario.executor) {
    case "constant-vus":
      return parseDurationToSeconds(scenario.duration);
    case "ramping-vus":
      return sumStageDurationsInSeconds(scenario.stages);
    case "constant-arrival-rate":
      return parseDurationToSeconds(scenario.duration);
    case "ramping-arrival-rate":
      return sumStageDurationsInSeconds(scenario.stages);
    case "externally-controlled":
      return parseDurationToSeconds(scenario.duration);
    case "shared-iterations":
    case "per-vu-iterations":
      return parseDurationToSeconds(scenario.maxDuration, 600);
    default:
      return 0;
  }
}

/**
 * Sums all stage durations in seconds.
 */
function sumStageDurationsInSeconds(stages: Array<{ duration: string }>): number {
  let total = 0;
  for (const stage of stages || []) {
    total += parseDurationToSeconds(stage.duration);
  }
  return total;
}

/**
 * Parses a k6 duration expression (for example `1m30s`) into seconds.
 */
function parseDurationToSeconds(raw: string | undefined, fallbackSeconds = 0): number {
  const value = String(raw || "").trim().replace(/\s+/g, "");
  if (!value) {
    return fallbackSeconds;
  }

  const durationRegex = /(\d+(?:\.\d+)?)(ms|s|m|h)/g;
  let total = 0;
  let parsedLength = 0;
  let match: RegExpExecArray | null = null;

  while (true) {
    match = durationRegex.exec(value);
    if (!match) {
      break;
    }

    const amount = Number(match[1]);
    const unit = match[2];
    parsedLength += match[0].length;

    if (unit === "h") {
      total += amount * 3600;
      continue;
    }
    if (unit === "m") {
      total += amount * 60;
      continue;
    }
    if (unit === "s") {
      total += amount;
      continue;
    }
    if (unit === "ms") {
      total += amount / 1000;
    }
  }

  if (parsedLength !== value.length) {
    return fallbackSeconds;
  }

  return total;
}

/**
 * Parses common boolean env values (for example `true`, `1`, `yes`, `on`).
 */
function parseBoolEnv(value: string | undefined): boolean {
  const normalized = String(value || "").trim().toLowerCase();
  return normalized === "1" || normalized === "true" || normalized === "yes" || normalized === "on";
}

export const options: Options = {
  scenarios: selectScenarios(),
  thresholds: {
    checks: ["rate==1.0"],
    http_req_failed: ["rate<0.01"],
  },
};

/**
 * k6 exec entrypoint for the REST+GraphQL mixed todo flow.
 */
export function loadTodoFlowRestGraphql(): void {
  todoFlowRestGraphql();
}

/**
 * k6 exec entrypoint for board-summary update propagation checks.
 */
export function loadFetchBoardSummaryUpdates(): void {
  fetchBoardSummaryUpdates();
}

/**
 * k6 exec entrypoint for board-summary convergence under event burst.
 */
export function loadBoardSummaryUnderEventBurst(): void {
  boardSummaryUnderEventBurst();
}

/**
 * k6 exec entrypoint for conversation scenario checks.
 */
export function loadConversation(): void {
  conversation();
}

/**
 * k6 exec entrypoint for REST-only todo create/update/delete throughput.
 */
export function loadTodoRestCreateUpdateDelete(): void {
  todoRestCreateUpdateDelete();
}

/**
 * k6 exec entrypoint for assistant action approval flow under load.
 */
export async function loadAssistantActionApprovalFlow(): Promise<void> {
  await assistantActionApprovalFlow();
}

/**
 * Cleans up residual artifacts after all-scenarios execution.
 */
export function teardown(): void {
  cleanupLoadArtifacts();
}
