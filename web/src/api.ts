import type {
  ChatReply,
  Comparison,
  ControlledFoodAnswer,
  ResponseMode,
  LayeredMemoryExchange,
  LayeredMemoryState,
  MemoryLayer,
  MemoryScope,
  PersonalizedExchange,
  UserProfile,
  TaskAction,
  TaskExchange,
  TaskState,
} from "./types";

interface APIErrorPayload {
  error?: {
    code?: string;
    message?: string;
  };
}

export class APIError extends Error {
  readonly status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "APIError";
    this.status = status;
  }
}

export async function compareAnswers(
  message: string,
  signal?: AbortSignal,
): Promise<Comparison> {
  const [unrestricted, controlled] = await Promise.all([
    sendMessage(message, "unrestricted", signal),
    sendMessage(message, "controlled", signal),
  ]);

  return { unrestricted, controlled };
}

async function sendMessage(
  message: string,
  mode: ResponseMode,
  signal?: AbortSignal,
): Promise<ChatReply> {
  const response = await fetch("/web-api/chat", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message, mode }),
    signal,
  });

  if (!response.ok) {
    let payload: APIErrorPayload | undefined;
    try {
      payload = (await response.json()) as APIErrorPayload;
    } catch {
      payload = undefined;
    }

    const localizedMessage =
      response.status === 429
        ? "Лимит демо-запросов исчерпан. Попробуйте немного позже."
        : response.status === 502
          ? "Модель временно недоступна. Попробуйте ещё раз через минуту."
          : payload?.error?.message ?? "Не удалось получить ответ. Попробуйте ещё раз.";
    throw new APIError(localizedMessage, response.status);
  }

  return (await response.json()) as ChatReply;
}

export function parseControlledAnswer(raw: string): ControlledFoodAnswer {
  let value: unknown;
  try {
    value = JSON.parse(raw);
  } catch {
    throw new Error("Модель вернула невалидный JSON.");
  }

  if (!isControlledFoodAnswer(value)) {
    throw new Error("JSON модели не соответствует ожидаемой структуре.");
  }
  return value;
}

function isControlledFoodAnswer(value: unknown): value is ControlledFoodAnswer {
  if (typeof value !== "object" || value === null) return false;

  const candidate = value as Record<string, unknown>;
  return (
    (candidate.status === "ok" || candidate.status === "out_of_scope") &&
    typeof candidate.answer === "string" &&
    Array.isArray(candidate.ingredients) &&
    candidate.ingredients.every((item) => typeof item === "string") &&
    Array.isArray(candidate.steps) &&
    candidate.steps.every((item) => typeof item === "string")
  );
}

export async function sendLayeredMemoryMessage(
  scope: MemoryScope,
  message: string,
  layer: MemoryLayer,
  signal?: AbortSignal,
): Promise<LayeredMemoryExchange> {
  return requestJSON<LayeredMemoryExchange>("/web-api/agent/memory/message", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ...scope, message, layer }),
    signal,
  });
}

export async function loadLayeredMemoryState(
  scope: MemoryScope,
  signal?: AbortSignal,
): Promise<LayeredMemoryState> {
  const query = memoryScopeQuery(scope);
  return requestJSON<LayeredMemoryState>(`/web-api/agent/memory/state?${query}`, { signal });
}

export async function clearLayeredMemory(scope: MemoryScope): Promise<void> {
  const query = memoryScopeQuery(scope);
  const response = await fetch(`/web-api/agent/memory?${query}`, { method: "DELETE" });
  if (!response.ok) {
    throw await responseError(response);
  }
}

function memoryScopeQuery(scope: MemoryScope): URLSearchParams {
  return new URLSearchParams({
    sessionId: scope.sessionId,
    taskId: scope.taskId,
    userId: scope.userId,
  });
}

async function requestJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, init);
  if (!response.ok) {
    throw await responseError(response);
  }
  return (await response.json()) as T;
}

async function responseError(response: Response): Promise<APIError> {
  let payload: APIErrorPayload | undefined;
  try {
    payload = (await response.json()) as APIErrorPayload;
  } catch {
    payload = undefined;
  }
  const message =
    response.status === 429
      ? "Лимит демо-запросов исчерпан. Попробуйте немного позже."
      : response.status === 502
        ? "Агент или модель временно недоступны. Попробуйте ещё раз."
        : payload?.error?.message ?? "Не удалось выполнить запрос.";
  return new APIError(message, response.status);
}

export async function loadProfiles(userId: string, signal?: AbortSignal): Promise<UserProfile[]> {
  const query = new URLSearchParams({ userId });
  return requestJSON<UserProfile[]>(`/web-api/agent/profiles?${query}`, { signal });
}

export async function saveProfile(profile: UserProfile): Promise<UserProfile> {
  return requestJSON<UserProfile>("/web-api/agent/profiles", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(profile),
  });
}

export async function sendPersonalizedMessage(
  scope: MemoryScope,
  profileId: string,
  message: string,
  signal?: AbortSignal,
): Promise<PersonalizedExchange> {
  return requestJSON<PersonalizedExchange>("/web-api/agent/personalized/message", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ...scope, profileId, message }),
    signal,
  });
}

export async function createTask(
  taskId: string,
  userId: string,
  profileId: string,
  goal: string,
  signal?: AbortSignal,
): Promise<TaskExchange> {
  return requestJSON<TaskExchange>("/web-api/agent/tasks", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ taskId, userId, profileId, goal }),
    signal,
  });
}

export async function loadTaskState(taskId: string, signal?: AbortSignal): Promise<TaskState | null> {
  const query = new URLSearchParams({ taskId });
  try {
    return await requestJSON<TaskState>(`/web-api/agent/tasks/state?${query}`, { signal });
  } catch (error) {
    if (error instanceof APIError && error.status === 404) return null;
    throw error;
  }
}

export async function actOnTask(taskId: string, action: TaskAction, signal?: AbortSignal): Promise<TaskExchange> {
  return requestJSON<TaskExchange>("/web-api/agent/tasks/action", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ taskId, action }),
    signal,
  });
}

export async function deleteTask(taskId: string): Promise<void> {
  const query = new URLSearchParams({ taskId });
  const response = await fetch(`/web-api/agent/tasks?${query}`, { method: "DELETE" });
  if (!response.ok) throw await responseError(response);
}
