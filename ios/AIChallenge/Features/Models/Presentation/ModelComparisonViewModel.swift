import Foundation
import Observation

@MainActor
@Observable
final class ModelComparisonViewModel {
    var prompt = examples[0]
    var selectedTier: ModelTier = .basic
    private(set) var attemptsByTier: [ModelTier: ModelBenchmarkAttempt] = [:]
    private(set) var comparison: ModelComparison?
    private(set) var isRunning = false
    private(set) var isReviewing = false
    var errorMessage: String?

    private let compareModels: CompareModelsUseCase

    init(compareModels: CompareModelsUseCase) { self.compareModels = compareModels }

    var canRun: Bool { !prompt.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isRunning }
    var selectedAttempt: ModelBenchmarkAttempt? {
        comparison?.attempt(for: selectedTier) ?? attemptsByTier[selectedTier]
    }

    func runComparison() async {
        let normalized = prompt.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalized.isEmpty, !isRunning else { return }
        attemptsByTier = [:]
        comparison = nil
        errorMessage = nil
        selectedTier = .basic
        isRunning = true
        isReviewing = false
        defer { isRunning = false; isReviewing = false }

        do {
            let result = try await compareModels.execute(prompt: normalized) { [weak self] attempt in
                self?.attemptsByTier[attempt.tier] = attempt
                if self?.attemptsByTier.count == ModelTier.allCases.count { self?.isReviewing = true }
            }
            comparison = result
            attemptsByTier = Dictionary(uniqueKeysWithValues: result.attempts.map { ($0.tier, $0) })
            selectedTier = result.review.qualityWinner
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
        }
    }

    func useNextExample() {
        guard !isRunning else { return }
        let current = prompt.trimmingCharacters(in: .whitespacesAndNewlines)
        let index = Self.examples.firstIndex(of: current) ?? -1
        prompt = Self.examples[(index + 1) % Self.examples.count]
        clearResult()
    }

    func clearPrompt() { guard !isRunning else { return }; prompt = ""; clearResult() }
    func clearResult() { guard !isRunning else { return }; attemptsByTier = [:]; comparison = nil; errorMessage = nil; selectedTier = .basic }

    static let examples = [
        "Объясни владельцу небольшого интернет-магазина, как уменьшить количество брошенных корзин. Дай ровно 5 практических шагов, расставь их по приоритету и для каждого кратко объясни ожидаемый эффект. Не более 180 слов.",
        "Объясни начинающему iOS-разработчику разницу между actor и классом с NSLock. Приведи один короткий пример выбора и назови главный риск каждого подхода. Не более 180 слов.",
        "Составь план запуска маленькой кофейни за 30 дней: 5 приоритетных действий, один риск для каждого и измеримый критерий успеха. Не более 180 слов.",
    ]
}
