protocol ChatRepository: Sendable {
    func send(message: String) async throws -> ChatReply
}
