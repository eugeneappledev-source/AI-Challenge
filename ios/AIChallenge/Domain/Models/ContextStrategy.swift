import Foundation

enum ContextStrategy: String, Codable, CaseIterable, Identifiable, Sendable {
    case slidingWindow = "sliding_window"
    case stickyFacts = "sticky_facts"
    case branching

    var id: String { rawValue }

    var title: String {
        switch self {
        case .slidingWindow: "Window"
        case .stickyFacts: "Facts"
        case .branching: "Ветки"
        }
    }

    var fullTitle: String {
        switch self {
        case .slidingWindow: "Sliding Window"
        case .stickyFacts: "Sticky Facts"
        case .branching: "Branching"
        }
    }

    var explanation: String {
        switch self {
        case .slidingWindow: "Только 6 последних сообщений. Дёшево, но ранние решения исчезают безвозвратно."
        case .stickyFacts: "Важные данные живут в key-value памяти, рядом остаются 6 свежих сообщений."
        case .branching: "Checkpoint клонирует общий контекст в две независимые версии: MVP и Growth."
        }
    }
}

struct MemoryFact: Codable, Equatable, Identifiable, Sendable {
    let key: String
    let value: String
    var id: String { key }
}

struct StrategyBranch: Codable, Equatable, Identifiable, Sendable {
    let id: String
    let title: String
    let messageCount: Int
    let checkpointMessages: Int
}

struct ContextStrategyState: Codable, Equatable, Sendable {
    let sessionId: String
    let strategy: ContextStrategy
    let activeBranchId: String?
    let windowSize: Int
    let messages: [AIAgentMessage]
    let facts: [MemoryFact]
    let branches: [StrategyBranch]
    let lastUsage: ModelUsage
}

struct ContextStrategyExchange: Codable, Equatable, Sendable {
    let strategy: ContextStrategy
    let branchId: String?
    let exchange: AIAgentExchange
    let state: ContextStrategyState
}

struct StrategyBranchAnswer: Codable, Equatable, Identifiable, Sendable {
    let branchId: String
    let title: String
    let answer: String
    let usage: ModelUsage
    var id: String { branchId }
}

struct ContextStrategyResult: Codable, Equatable, Identifiable, Sendable {
    let strategy: ContextStrategy
    let answer: String
    let usage: ModelUsage
    let messagesKept: Int
    let factsKept: Int
    let branches: [StrategyBranchAnswer]
    let behavior: String
    var id: String { strategy.id }
}

struct ContextStrategyScore: Codable, Equatable, Identifiable, Sendable {
    let strategy: ContextStrategy
    let quality: Int
    let stability: Int
    let tokenEfficiency: Int
    let usability: Int
    let feedback: String
    var id: String { strategy.id }
}

struct ContextStrategyReview: Codable, Equatable, Sendable {
    let winner: ContextStrategy
    let verdict: String
    let differences: [String]
    let scores: [ContextStrategyScore]
    let recommendations: [String]
    let model: String
    let usage: ModelUsage
}

struct ContextStrategyComparison: Codable, Equatable, Sendable {
    let sessionId: String
    let scenario: [String]
    let question: String
    let results: [ContextStrategyResult]
    let review: ContextStrategyReview
}
