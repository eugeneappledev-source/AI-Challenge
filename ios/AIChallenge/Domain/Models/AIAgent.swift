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
    let userMessage: AIAgentMessage
    let reply: AIAgentMessage
    let model: String
    let finishReason: String
    let usage: ModelUsage
    let trace: [String]
}
