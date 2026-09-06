import Foundation
import Observation

@MainActor
@Observable
final class TemperatureComparisonViewModel {
    var prompt = examples[0]
    var selectedTemperature: TemperaturePreset = .precise
    private(set) var submittedPrompt: String?
    private(set) var attemptsByTemperature: [TemperaturePreset: TemperatureAttempt] = [:]
    private(set) var comparison: TemperatureComparison?
    private(set) var isRunning = false
    private(set) var isReviewing = false
    var errorMessage: String?

    private let compareTemperatures: CompareTemperaturesUseCase

    init(compareTemperatures: CompareTemperaturesUseCase) {
        self.compareTemperatures = compareTemperatures
    }

    var canRun: Bool {
        !prompt.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isRunning
    }

    var selectedAttempt: TemperatureAttempt? {
        comparison?.attempt(for: selectedTemperature) ?? attemptsByTemperature[selectedTemperature]
    }

    func runComparison() async {
        let normalized = prompt.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalized.isEmpty, !isRunning else { return }

        submittedPrompt = normalized
        attemptsByTemperature = [:]
        comparison = nil
        errorMessage = nil
        selectedTemperature = .precise
        isRunning = true
        isReviewing = false
        defer {
            isRunning = false
            isReviewing = false
        }

        do {
            let result = try await compareTemperatures.execute(prompt: normalized) { [weak self] attempt in
                self?.attemptsByTemperature[attempt.temperature] = attempt
                if self?.attemptsByTemperature.count == TemperaturePreset.allCases.count {
                    self?.isReviewing = true
                }
            }
            comparison = result
            attemptsByTemperature = Dictionary(
                uniqueKeysWithValues: result.attempts.map { ($0.temperature, $0) }
            )
            selectedTemperature = result.review.bestCreativity
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
        }
    }

    func useNextExample() {
        guard !isRunning else { return }
        let normalized = prompt.trimmingCharacters(in: .whitespacesAndNewlines)
        let currentIndex = Self.examples.firstIndex(of: normalized) ?? -1
        prompt = Self.examples[(currentIndex + 1) % Self.examples.count]
        clearResult()
    }

    func clearPrompt() {
        guard !isRunning else { return }
        prompt = ""
        clearResult()
    }

    func clearResult() {
        guard !isRunning else { return }
        submittedPrompt = nil
        attemptsByTemperature = [:]
        comparison = nil
        errorMessage = nil
        selectedTemperature = .precise
    }

    static let examples = [
        "Объясни двенадцатилетнему, почему небо кажется голубым. Используй одну запоминающуюся метафору, но сохрани научную точность. Ответ — не более 120 слов.",
        "Придумай три названия для мобильного приложения, которое помогает не забывать пить воду. Для каждого добавь короткий слоган и объясни идею одним предложением.",
        "Опиши умный будильник будущего в пяти предложениях: функции должны быть технически правдоподобными, а подача — яркой и запоминающейся.",
    ]
}
