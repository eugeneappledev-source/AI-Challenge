struct DefaultAIAgentRepository: AIAgentRepository, AgentMemoryRepository {
    private let api: AIAgentAPI

    init(api: AIAgentAPI) { self.api = api }

    func profile() async throws -> AIAgentProfile { try await api.profile() }
    func send(message: String) async throws -> AIAgentExchange { try await api.send(message: message) }
    func history(conversationID: String) async throws -> AIAgentConversation { try await api.history(conversationID: conversationID) }
    func send(message: String, conversationID: String) async throws -> AIAgentExchange {
        try await api.send(message: message, conversationID: conversationID)
    }
    func clearHistory(conversationID: String) async throws { try await api.clearHistory(conversationID: conversationID) }
}
