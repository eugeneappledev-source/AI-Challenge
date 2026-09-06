struct CompareTemperaturesUseCase: Sendable {
    private let repository: any TemperatureRepository

    init(repository: any TemperatureRepository) {
        self.repository = repository
    }

    func execute(
        prompt: String,
        onAttempt: @escaping @MainActor @Sendable (TemperatureAttempt) -> Void
    ) async throws -> TemperatureComparison {
        let attempts = try await withThrowingTaskGroup(
            of: TemperatureAttempt.self,
            returning: [TemperatureAttempt].self
        ) { group in
            for temperature in TemperaturePreset.allCases {
                group.addTask {
                    try await repository.run(prompt: prompt, temperature: temperature)
                }
            }

            var results: [TemperatureAttempt] = []
            for try await attempt in group {
                results.append(attempt)
                await onAttempt(attempt)
            }
            return TemperaturePreset.allCases.compactMap { temperature in
                results.first { $0.temperature == temperature }
            }
        }

        let review = try await repository.review(prompt: prompt, attempts: attempts)
        return TemperatureComparison(prompt: prompt, attempts: attempts, review: review)
    }
}
