struct SendMessageUseCase: Sendable {
    private let repository: any ChatRepository

    init(repository: any ChatRepository) {
        self.repository = repository
    }

    func execute(message: String, mode: ResponseControlMode) async throws -> ChatReply {
        try await repository.send(message: message, mode: mode)
    }
}
