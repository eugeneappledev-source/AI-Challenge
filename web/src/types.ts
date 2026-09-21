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

export type TaskPhase = "planning" | "execution" | "validation" | "done";
export type TaskStatus = "active" | "paused";
export type TaskAction = "advance" | "pause" | "resume";

export interface TaskArtifact {
  phase: TaskPhase;
  title: string;
  content: string;
  createdAt: string;
}

export interface TaskTransition {
  action: string;
  from?: TaskPhase;
  to: TaskPhase;
  summary: string;
  createdAt: string;
}

export interface TaskState {
  id: string;
  userId: string;
  profileId: string;
  goal: string;
  phase: TaskPhase;
  status: TaskStatus;
  currentStep: string;
  expectedAction: string;
  artifacts: TaskArtifact[];
  transitions: TaskTransition[];
  revision: number;
  createdAt: string;
  updatedAt: string;
}

export interface TaskExchange {
  state: TaskState;
  answer: string;
  model?: string;
  finishReason?: string;
  usage: Usage;
  trace: string[];
}

export type InvariantCategory = "architecture" | "technical_decision" | "stack" | "business_rule";
export type InvariantProtection = "hard_check" | "semantic_guard";
export type InvariantVerdict = "allowed" | "rejected";

export interface Invariant {
  id: string;
  taskId: string;
  category: InvariantCategory;
  title: string;
  rule: string;
  rationale: string;
  protection: InvariantProtection;
  forbiddenTerms: string[];
  createdAt: string;
}

export interface InvariantAssessment {
  invariantId: string;
  verdict: "compliant" | "conflict" | "pending_semantic_check";
  note: string;
}

export interface InvariantConflict {
  invariantId: string;
  title: string;
  reason: string;
  detectedBy: "hard_check" | "semantic_guard";
}

export interface InvariantExchange {
  request: string;
  verdict: InvariantVerdict;
  explanation: string;
  safeAlternative?: string;
  answer?: string;
  invariants: Invariant[];
  assessments: InvariantAssessment[];
  conflicts: InvariantConflict[];
  taskState?: TaskState;
  model?: string;
  finishReason?: string;
  usage: Usage;
  trace: string[];
}
