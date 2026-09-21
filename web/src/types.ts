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

export type MemoryLayer = "auto" | "short_term" | "working" | "long_term";

export interface AgentMessage {
  id: string;
  role: "user" | "assistant" | "system";
  content: string;
  createdAt: string;
  usage?: Usage;
}

export interface MemoryItem {
  key: string;
  value: string;
  source: string;
  updatedAt: string;
}

export interface MemoryRoute {
  requestedLayer: MemoryLayer;
  selectedLayer: Exclude<MemoryLayer, "auto">;
  key: string;
  value: string;
  reason: string;
  automatic: boolean;
}

export interface LayeredMemoryState {
  sessionId: string;
  taskId: string;
  userId: string;
  shortTerm: AgentMessage[];
  working: MemoryItem[];
  longTerm: MemoryItem[];
  updatedAt: string;
}

export interface LayeredMemoryExchange {
  message: string;
  answer: string;
  model: string;
  finishReason: string;
  usage: Usage;
  route: MemoryRoute;
  state: LayeredMemoryState;
  contextPreview: string[];
  trace: string[];
}

export interface MemoryScope {
  sessionId: string;
  taskId: string;
  userId: string;
}

export interface UserProfile {
  id: string;
  userId: string;
  name: string;
  address: string;
  style: string;
  responseFormat: string;
  constraints: string[];
  pipelineId: "engineering_review" | "product_discovery";
  updatedAt: string;
}

export interface ProfileSkill {
  id: string;
  name: string;
  description: string;
}

export interface ProfileSkillRun {
  skill: ProfileSkill;
  output: string;
  usage: Usage;
}

export interface PersonalizedExchange {
  profile: UserProfile;
  skills: ProfileSkillRun[];
  message: string;
  answer: string;
  model: string;
  finishReason: string;
  usage: Usage;
  route: MemoryRoute;
  memory: LayeredMemoryState;
  trace: string[];
}
