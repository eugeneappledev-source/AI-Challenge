protocol ChatRepository: Sendable {
    func send(message: String, mode: ResponseControlMode) async throws -> ChatReply
}
