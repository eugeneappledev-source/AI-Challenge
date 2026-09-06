import Foundation
import Observation

@MainActor
@Observable
final class ReasoningComparisonViewModel {
    var problem = examples[0]
    var selectedMethod: ReasoningMethod = .direct
    private(set) var submittedProblem: String?
    private(set) var attemptsByMethod: [ReasoningMethod: ReasoningAttempt] = [:]
    private(set) var comparison: ReasoningComparison?
    private(set) var isRunning = false
    private(set) var isReviewing = false
    var errorMessage: String?

    private let compareStrategies: CompareReasoningStrategiesUseCase

    init(compareStrategies: CompareReasoningStrategiesUseCase) {
        self.compareStrategies = compareStrategies
    }

    var canRun: Bool {
        !problem.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isRunning
    }

    var selectedAttempt: ReasoningAttempt? {
        comparison?.attempt(for: selectedMethod) ?? attemptsByMethod[selectedMethod]
    }

    var completedCount: Int {
        attemptsByMethod.count
    }

    func runComparison() async {
        let normalized = problem.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalized.isEmpty, !isRunning else { return }

        submittedProblem = normalized
        attemptsByMethod = [:]
        comparison = nil
        errorMessage = nil
        selectedMethod = .direct
        isRunning = true
        isReviewing = false
        defer {
            isRunning = false
            isReviewing = false
        }

        do {
            let result = try await compareStrategies.execute(problem: normalized) { [weak self] attempt in
                self?.attemptsByMethod[attempt.method] = attempt
                if self?.attemptsByMethod.count == ReasoningMethod.allCases.count {
                    self?.isReviewing = true
                }
            }
            comparison = result
            attemptsByMethod = Dictionary(uniqueKeysWithValues: result.attempts.map { ($0.method, $0) })
            selectedMethod = result.review.winner
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
        }
    }

    func useNextExample() {
        guard !isRunning else { return }
        let normalized = problem.trimmingCharacters(in: .whitespacesAndNewlines)
        let currentIndex = Self.examples.firstIndex(of: normalized) ?? -1
        problem = Self.examples[(currentIndex + 1) % Self.examples.count]
        clearResult()
    }

    func clearProblem() {
        guard !isRunning else { return }
        problem = ""
        clearResult()
    }

    func clearResult() {
        guard !isRunning else { return }
        submittedProblem = nil
        attemptsByMethod = [:]
        comparison = nil
        errorMessage = nil
        selectedMethod = .direct
    }

    static let examples = [
        "Между числами 1 2 3 4 5 6 7 8 9 поставь только знаки + или −, не меняя порядок и не объединяя числа, чтобы получить 100. Найди решение или докажи, что оно невозможно.",
        "Есть 8 одинаковых на вид монет. Одна из них тяжелее остальных. Как гарантированно найти её за два взвешивания на чашечных весах без гирь?",
        "Четырём людям нужно перейти мост ночью. Их скорости: 1, 2, 7 и 10 минут. Фонарь один, мост выдерживает двоих, пара идёт со скоростью медленного. Какое минимальное общее время и почему?",
    ]
}
