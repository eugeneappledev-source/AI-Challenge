import Foundation

enum ModelTier: String, Codable, CaseIterable, Identifiable, Sendable {
    case basic
    case extended
    case strong

    var id: Self { self }
}

struct ModelUsage: Codable, Equatable, Sendable {
    let promptTokens: Int
    let completionTokens: Int
    let totalTokens: Int
    let promptCacheHitTokens: Int
    let promptCacheMissTokens: Int

    init(
        promptTokens: Int,
        completionTokens: Int,
        totalTokens: Int,
        promptCacheHitTokens: Int = 0,
        promptCacheMissTokens: Int = 0
    ) {
        self.promptTokens = promptTokens
        self.completionTokens = completionTokens
        self.totalTokens = totalTokens
        self.promptCacheHitTokens = promptCacheHitTokens
        self.promptCacheMissTokens = promptCacheMissTokens
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        promptTokens = try container.decode(Int.self, forKey: .promptTokens)
        completionTokens = try container.decode(Int.self, forKey: .completionTokens)
        totalTokens = try container.decode(Int.self, forKey: .totalTokens)
        promptCacheHitTokens = try container.decodeIfPresent(Int.self, forKey: .promptCacheHitTokens) ?? 0
        promptCacheMissTokens = try container.decodeIfPresent(Int.self, forKey: .promptCacheMissTokens) ?? 0
    }
}

struct ModelBenchmarkAttempt: Codable, Equatable, Sendable {
    let tier: ModelTier
    let model: String
    let answer: String
    let latencyMilliseconds: Int
    let usage: ModelUsage
    let inputCostUSD: Double
    let outputCostUSD: Double
    let estimatedCostUSD: Double
    let pricingPeriod: String
    let finishReason: String
}

struct ModelQualityScore: Codable, Equatable, Sendable {
    let tier: ModelTier
    let accuracy: Int
    let completeness: Int
    let clarity: Int
    let feedback: String
}

struct ModelRecommendation: Codable, Equatable, Sendable {
    let tier: ModelTier
    let bestFor: [String]
    let tradeoff: String
}

struct ModelBenchmarkReview: Codable, Equatable, Sendable {
    let qualityWinner: ModelTier
    let fastest: ModelTier
    let cheapest: ModelTier
    let summary: String
    let differences: [String]
    let scores: [ModelQualityScore]
    let recommendations: [ModelRecommendation]
    let reviewerModel: String
    let reviewerUsage: ModelUsage
}

struct ModelComparison: Equatable, Sendable {
    let prompt: String
    let attempts: [ModelBenchmarkAttempt]
    let review: ModelBenchmarkReview

    func attempt(for tier: ModelTier) -> ModelBenchmarkAttempt? {
        attempts.first { $0.tier == tier }
    }
}
