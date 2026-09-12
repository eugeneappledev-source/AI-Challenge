struct TalkToAgentUseCase: Sendable {
    private let repository: any AIAgentRepository

    init(repository: any AIAgentRepository) {
        self.repository = repository
    }

    func loadProfile() async throws -> AIAgentProfile {
        try await repository.profile()
    }

    func execute(message: String) async throws -> AIAgentExchange {
        try await repository.send(message: message)
    }
}
