export type ResponseMode = "unrestricted" | "controlled";

export interface Usage {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
}

export interface ChatReply {
  answer: string;
  model: string;
  mode: ResponseMode;
  finishReason: string;
  usage: Usage;
}

export interface ControlledFoodAnswer {
  status: "ok" | "out_of_scope";
  answer: string;
  ingredients: string[];
  steps: string[];
}

export interface Comparison {
  unrestricted: ChatReply;
  controlled: ChatReply;
}
