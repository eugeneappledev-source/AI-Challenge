protocol AgentMemoryRepository: Sendable {
    func profile() async throws -> AIAgentProfile
    func history(conversationID: String) async throws -> AIAgentConversation
    func send(message: String, conversationID: String) async throws -> AIAgentExchange
    func clearHistory(conversationID: String) async throws
}
