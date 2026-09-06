import Foundation

enum TemperaturePreset: Double, Codable, CaseIterable, Identifiable, Sendable {
    case precise = 0
    case balanced = 0.7
    case creative = 1.2

    var id: Self { self }
}

struct TemperatureAttempt: Codable, Equatable, Sendable {
    let temperature: TemperaturePreset
    let answer: String
    let model: String
    let finishReason: String
    let usage: ReasoningUsage
}

struct TemperatureScore: Codable, Equatable, Sendable {
    let temperature: TemperaturePreset
    let accuracy: Int
    let creativity: Int
    let diversity: Int
    let feedback: String

    var average: Double {
        Double(accuracy + creativity + diversity) / 3
    }
}

struct TemperatureRecommendation: Codable, Equatable, Sendable {
    let temperature: TemperaturePreset
    let bestFor: [String]
    let caution: String
}

struct TemperatureReview: Codable, Equatable, Sendable {
    let summary: String
    let bestAccuracy: TemperaturePreset
    let bestCreativity: TemperaturePreset
    let bestDiversity: TemperaturePreset
    let differences: [String]
    let scores: [TemperatureScore]
    let recommendations: [TemperatureRecommendation]
    let model: String
    let usage: ReasoningUsage
}

struct TemperatureComparison: Equatable, Sendable {
    let prompt: String
    let attempts: [TemperatureAttempt]
    let review: TemperatureReview

    func attempt(for temperature: TemperaturePreset) -> TemperatureAttempt? {
        attempts.first { $0.temperature == temperature }
    }
}
