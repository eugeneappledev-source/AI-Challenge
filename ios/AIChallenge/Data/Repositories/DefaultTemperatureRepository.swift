struct DefaultTemperatureRepository: TemperatureRepository {
    private let api: TemperatureAPI

    init(api: TemperatureAPI) {
        self.api = api
    }

    func run(prompt: String, temperature: TemperaturePreset) async throws -> TemperatureAttempt {
        try await api.run(prompt: prompt, temperature: temperature)
    }

    func review(prompt: String, attempts: [TemperatureAttempt]) async throws -> TemperatureReview {
        try await api.review(prompt: prompt, attempts: attempts)
    }
}
