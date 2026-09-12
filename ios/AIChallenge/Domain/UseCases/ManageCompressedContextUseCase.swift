struct ManageCompressedContextUseCase: Sendable {
    private let repository: any ContextCompressionRepository

    init(repository: any ContextCompressionRepository) { self.repository = repository }

    func load(conversationID: String) async throws -> (AIAgentConversation, ContextState) {
        async let history = repository.history(conversationID: conversationID)
        async let state = repository.contextState(conversationID: conversationID)
        return try await (history, state)
    }

    func send(message: String, conversationID: String) async throws -> (AIAgentExchange, ContextState) {
        let exchange = try await repository.sendCompressed(message: message, conversationID: conversationID)
        let state = try await repository.contextState(conversationID: conversationID)
        return (exchange, state)
    }

    func compare(question: String, conversationID: String) async throws -> ContextComparison {
        try await repository.compareContexts(conversationID: conversationID, question: question)
    }

    func clear(conversationID: String) async throws { try await repository.clearHistory(conversationID: conversationID) }
}
