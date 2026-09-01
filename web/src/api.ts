import type {
  ChatReply,
  Comparison,
  ControlledFoodAnswer,
  ResponseMode,
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
