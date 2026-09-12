struct ContinueAgentConversationUseCase: Sendable {
    private let repository: any AgentMemoryRepository

    init(repository: any AgentMemoryRepository) { self.repository = repository }

    func loadProfile() async throws -> AIAgentProfile { try await repository.profile() }
    func loadHistory(conversationID: String) async throws -> AIAgentConversation {
        try await repository.history(conversationID: conversationID)
    }
    func send(message: String, conversationID: String) async throws -> AIAgentExchange {
        try await repository.send(message: message, conversationID: conversationID)
    }
    func clear(conversationID: String) async throws {
        try await repository.clearHistory(conversationID: conversationID)
    }
}
