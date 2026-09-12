struct ManageContextStrategiesUseCase: Sendable {
    private let repository: any ContextStrategyRepository

    init(repository: any ContextStrategyRepository) {
        self.repository = repository
    }

    func state(sessionID: String, strategy: ContextStrategy, branchID: String? = nil) async throws -> ContextStrategyState {
        try await repository.state(sessionID: sessionID, strategy: strategy, branchID: branchID)
    }

    func send(message: String, sessionID: String, strategy: ContextStrategy, branchID: String? = nil) async throws -> ContextStrategyExchange {
        try await repository.send(message: message, sessionID: sessionID, strategy: strategy, branchID: branchID)
    }

    func createBranches(sessionID: String) async throws -> ContextStrategyState {
        try await repository.createBranches(sessionID: sessionID)
    }

    func compare(sessionID: String) async throws -> ContextStrategyComparison {
        try await repository.compare(sessionID: sessionID)
    }

    func clear(sessionID: String) async throws {
        try await repository.clear(sessionID: sessionID)
    }
}
