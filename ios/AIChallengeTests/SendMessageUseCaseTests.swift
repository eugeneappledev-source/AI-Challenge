import Testing
@testable import AIChallenge

struct SendMessageUseCaseTests {
    @Test
    func forwardsMessageAndReturnsReply() async throws {
        let expected = ChatReply(answer: "Ответ", model: "test-model", totalTokens: 12)
        let repository = ChatRepositoryStub(reply: expected)
        let useCase = SendMessageUseCase(repository: repository)

        let reply = try await useCase.execute(message: "Вопрос")

        #expect(reply == expected)
    }
}

private struct ChatRepositoryStub: ChatRepository {
    let reply: ChatReply

    func send(message: String) async throws -> ChatReply {
        reply
    }
}
