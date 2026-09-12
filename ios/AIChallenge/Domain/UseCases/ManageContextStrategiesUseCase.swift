struct ManageContextStrategiesUseCase: Sendable {
    private let repository: any ContextStrategyRepository

    init(repository: any ContextStrategyRepository) {
        self.repository = repository
    }

    func state(sessionID: String, strategy: ContextStrategy, branchID: String? = nil, windowSize: Int) async throws -> ContextStrategyState {
        try await repository.state(sessionID: sessionID, strategy: strategy, branchID: branchID, windowSize: windowSize)
    }

    func send(message: String, sessionID: String, strategy: ContextStrategy, branchID: String? = nil, windowSize: Int) async throws -> ContextStrategyExchange {
        try await repository.send(message: message, sessionID: sessionID, strategy: strategy, branchID: branchID, windowSize: windowSize)
    }

    func createBranches(sessionID: String) async throws -> ContextStrategyState {
        try await repository.createBranches(sessionID: sessionID)
    }

    func compare(sessionID: String, windowSize: Int) async throws -> ContextStrategyComparison {
        try await repository.compare(sessionID: sessionID, windowSize: windowSize)
    }

    func clear(sessionID: String) async throws {
        try await repository.clear(sessionID: sessionID)
    }
}
