struct DefaultAIAgentRepository: AIAgentRepository {
    private let api: AIAgentAPI

    init(api: AIAgentAPI) { self.api = api }

    func profile() async throws -> AIAgentProfile { try await api.profile() }
    func send(message: String) async throws -> AIAgentExchange { try await api.send(message: message) }
}
