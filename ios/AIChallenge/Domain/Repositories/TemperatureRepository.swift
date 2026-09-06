protocol TemperatureRepository: Sendable {
    func run(prompt: String, temperature: TemperaturePreset) async throws -> TemperatureAttempt
    func review(prompt: String, attempts: [TemperatureAttempt]) async throws -> TemperatureReview
}
