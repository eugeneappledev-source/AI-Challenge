import Foundation
import Testing
@testable import AIChallenge

struct SendMessageUseCaseTests {
    @Test
    func forwardsMessageAndReturnsReply() async throws {
        let expected = ChatReply(
            answer: "Ответ",
            model: "test-model",
            mode: .controlled,
            finishReason: "stop",
            promptTokens: 7,
            completionTokens: 5,
            totalTokens: 12
        )
        let repository = ChatRepositoryStub(reply: expected)
        let useCase = SendMessageUseCase(repository: repository)

        let reply = try await useCase.execute(message: "Вопрос", mode: .controlled)

        #expect(reply == expected)
    }

    @Test
    func comparesSameMessageInBothModes() async throws {
        let repository = RecordingChatRepository()
        let sendMessage = SendMessageUseCase(repository: repository)
        let useCase = CompareResponsesUseCase(sendMessage: sendMessage)

        let comparison = try await useCase.execute(message: "Дай рецепт салата")
        let requests = await repository.requests

        #expect(comparison.prompt == "Дай рецепт салата")
        #expect(comparison.unrestricted.mode == .unrestricted)
        #expect(comparison.controlled.mode == .controlled)
        #expect(requests.count == 2)
        #expect(requests.allSatisfy { $0.message == "Дай рецепт салата" })
        #expect(Set(requests.map(\.mode)) == Set(ResponseControlMode.allCases))
    }

    @Test
    func decodesControlledAnswerForPresentation() throws {
        let json = """
        {
          "status": "ok",
          "answer": "Готово",
          "ingredients": ["Помидоры", "Фета"],
          "steps": ["Нарежьте продукты", "Смешайте"]
        }
        """

        let answer = try JSONDecoder().decode(
            ControlledFoodAnswer.self,
            from: Data(json.utf8)
        )

        #expect(answer.status == .ok)
        #expect(answer.answer == "Готово")
        #expect(answer.ingredients == ["Помидоры", "Фета"])
        #expect(answer.steps == ["Нарежьте продукты", "Смешайте"])
    }
}

private struct ChatRepositoryStub: ChatRepository {
    let reply: ChatReply

    func send(message: String, mode: ResponseControlMode) async throws -> ChatReply {
        reply
    }
}

private actor RecordingChatRepository: ChatRepository {
    struct Request: Sendable {
        let message: String
        let mode: ResponseControlMode
    }

    private(set) var requests: [Request] = []

    func send(message: String, mode: ResponseControlMode) async throws -> ChatReply {
        requests.append(Request(message: message, mode: mode))
        return ChatReply(
            answer: mode.rawValue,
            model: "test-model",
            mode: mode,
            finishReason: "stop",
            promptTokens: 10,
            completionTokens: 5,
            totalTokens: 15
        )
    }
}
