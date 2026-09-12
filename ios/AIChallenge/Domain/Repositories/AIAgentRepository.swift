protocol AIAgentRepository: Sendable {
    func profile() async throws -> AIAgentProfile
    func send(message: String) async throws -> AIAgentExchange
}
