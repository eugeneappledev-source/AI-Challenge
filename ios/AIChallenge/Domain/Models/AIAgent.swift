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
    var conversationId: String? = nil
    var historyCount: Int? = nil
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

struct ContextState: Codable, Equatable, Sendable {
    let conversationId: String
    let summary: String
    let summaryCoveredMessages: Int
    let recentMessages: Int
    let fullHistoryMessages: Int
    let fullEstimatedTokens: Int
    let compressedEstimatedTokens: Int
    let estimatedSavedTokens: Int
    let compressionActive: Bool
}

struct ContextAnswer: Codable, Equatable, Sendable {
    let mode: String
    let answer: String
    let usage: ModelUsage
}

struct ContextReview: Codable, Equatable, Sendable {
    let qualityPreserved: Bool
    let verdict: String
    let differences: [String]
    let recommendation: String
}

struct ContextComparison: Codable, Equatable, Sendable {
    let question: String
    let summary: String
    let full: ContextAnswer
    let compressed: ContextAnswer
    let promptTokensSaved: Int
    let savingsPercent: Double
    let review: ContextReview
}
