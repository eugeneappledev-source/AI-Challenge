import Foundation

struct AIAgentProfile: Codable, Equatable, Sendable {
    let id: String
    let name: String
    let role: String
    let instructions: String
    let model: String
    let temperature: Double
    let maxOutputTokens: Int
}

enum AIAgentMessageRole: String, Codable, Sendable {
    case user
    case assistant
    case system
}

struct AIAgentMessage: Codable, Equatable, Identifiable, Sendable {
    let id: String
    let role: AIAgentMessageRole
    let content: String
    let createdAt: Date
    let usage: ModelUsage?
}

struct AIAgentExchange: Codable, Equatable, Sendable {
    let agent: AIAgentProfile
    let conversationId: String? = nil
    let historyCount: Int? = nil
    let userMessage: AIAgentMessage
    let reply: AIAgentMessage
    let model: String
    let finishReason: String
    let usage: ModelUsage
    let trace: [String]
}

struct AIAgentConversation: Codable, Equatable, Sendable {
    let id: String
    let agentId: String
    let messages: [AIAgentMessage]
    let updatedAt: Date?
}

struct AgentTokenMetrics: Codable, Equatable, Sendable {
    let conversationId: String
    let model: String
    let messageCount: Int
    let currentMessageEstimatedTokens: Int
    let lastContextPromptTokens: Int
    let lastResponseTokens: Int
    let historyEstimatedTokens: Int
    let cumulativePromptTokens: Int
    let cumulativeResponseTokens: Int
    let cumulativeTotalTokens: Int
    let estimatedCostUSD: Double
    let contextWindowTokens: Int
    let estimatedRemainingTokens: Int
    let scenarios: [TokenScenario]
}

struct TokenScenario: Codable, Equatable, Identifiable, Sendable {
    let id: String
    let title: String
    let messageCount: Int
    let estimatedTokens: Int
    let accepted: Bool
    let outcome: String
}
