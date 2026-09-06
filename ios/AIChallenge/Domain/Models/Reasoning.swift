import Foundation

enum ReasoningMethod: String, Codable, CaseIterable, Identifiable, Sendable {
    case direct
    case stepByStep = "step_by_step"
    case metaPrompt = "meta_prompt"
    case expertPanel = "expert_panel"

    var id: Self { self }
}

struct ReasoningUsage: Codable, Equatable, Sendable {
    let promptTokens: Int
    let completionTokens: Int
    let totalTokens: Int
}

struct ExpertSolution: Codable, Equatable, Sendable {
    let role: String
    let answer: String
}

struct ReasoningAttempt: Codable, Equatable, Sendable {
    let method: ReasoningMethod
    let answer: String
    let generatedPrompt: String?
    let experts: [ExpertSolution]
    let model: String
    let finishReason: String
    let usage: ReasoningUsage

    init(
        method: ReasoningMethod,
        answer: String,
        generatedPrompt: String? = nil,
        experts: [ExpertSolution] = [],
        model: String,
        finishReason: String,
        usage: ReasoningUsage
    ) {
        self.method = method
        self.answer = answer
        self.generatedPrompt = generatedPrompt
        self.experts = experts
        self.model = model
        self.finishReason = finishReason
        self.usage = usage
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        method = try container.decode(ReasoningMethod.self, forKey: .method)
        answer = try container.decode(String.self, forKey: .answer)
        generatedPrompt = try container.decodeIfPresent(String.self, forKey: .generatedPrompt)
        experts = try container.decodeIfPresent([ExpertSolution].self, forKey: .experts) ?? []
        model = try container.decode(String.self, forKey: .model)
        finishReason = try container.decode(String.self, forKey: .finishReason)
        usage = try container.decode(ReasoningUsage.self, forKey: .usage)
    }
}

struct ReasoningScore: Codable, Equatable, Sendable {
    let method: ReasoningMethod
    let correctness: Int
    let clarity: Int
    let verification: Int
    let feedback: String

    var average: Double {
        Double(correctness + clarity + verification) / 3
    }
}

struct ReasoningReview: Codable, Equatable, Sendable {
    let winner: ReasoningMethod
    let verdict: String
    let referenceAnswer: String
    let differences: [String]
    let scores: [ReasoningScore]
    let model: String
    let usage: ReasoningUsage
}

struct ReasoningComparison: Equatable, Sendable {
    let problem: String
    let attempts: [ReasoningAttempt]
    let review: ReasoningReview

    func attempt(for method: ReasoningMethod) -> ReasoningAttempt? {
        attempts.first { $0.method == method }
    }
}
