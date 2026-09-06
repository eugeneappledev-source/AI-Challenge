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

    @Test @MainActor
    func dayOneSendsTrimmedPromptInUnrestrictedMode() async {
        let repository = RecordingChatRepository()
        let useCase = SendMessageUseCase(repository: repository)
        let viewModel = DayOneViewModel(sendMessage: useCase)
        viewModel.input = "  Дай рецепт супа  "

        await viewModel.send()
        let requests = await repository.requests

        #expect(requests.count == 1)
        #expect(requests.first?.message == "Дай рецепт супа")
        #expect(requests.first?.mode == .unrestricted)
        #expect(viewModel.submittedPrompt == "Дай рецепт супа")
        #expect(viewModel.reply?.answer == "unrestricted")
    }

    @Test @MainActor
    func comparesAllReasoningMethodsThenRequestsReview() async throws {
        let repository = RecordingReasoningRepository()
        let useCase = CompareReasoningStrategiesUseCase(repository: repository)
        var completedMethods: Set<ReasoningMethod> = []

        let comparison = try await useCase.execute(problem: "Одна задача") { attempt in
            completedMethods.insert(attempt.method)
        }
        let runRequests = await repository.runRequests
        let reviewRequest = await repository.reviewRequest

        #expect(runRequests.count == ReasoningMethod.allCases.count)
        #expect(runRequests.allSatisfy { $0.problem == "Одна задача" })
        #expect(Set(runRequests.map(\.method)) == Set(ReasoningMethod.allCases))
        #expect(completedMethods == Set(ReasoningMethod.allCases))
        #expect(reviewRequest?.problem == "Одна задача")
        #expect(reviewRequest?.attempts.count == 4)
        #expect(comparison.review.winner == .stepByStep)
    }

    @Test
    func decodesReasoningAttemptWhenOptionalFieldsAreMissing() throws {
        let json = """
        {
          "method": "direct",
          "answer": "Ответ",
          "model": "test-model",
          "finishReason": "stop",
          "usage": {
            "promptTokens": 4,
            "completionTokens": 5,
            "totalTokens": 9
          }
        }
        """

        let attempt = try JSONDecoder().decode(
            ReasoningAttempt.self,
            from: Data(json.utf8)
        )

        #expect(attempt.method == .direct)
        #expect(attempt.generatedPrompt == nil)
        #expect(attempt.experts.isEmpty)
        #expect(attempt.usage.totalTokens == 9)
    }

    @Test @MainActor
    func comparesExactPromptAtAllTemperaturesThenRequestsReview() async throws {
        let repository = RecordingTemperatureRepository()
        let useCase = CompareTemperaturesUseCase(repository: repository)
        var completedTemperatures: Set<TemperaturePreset> = []

        let comparison = try await useCase.execute(prompt: "Один и тот же запрос") { attempt in
            completedTemperatures.insert(attempt.temperature)
        }
        let runRequests = await repository.runRequests
        let reviewRequest = await repository.reviewRequest

        #expect(runRequests.count == TemperaturePreset.allCases.count)
        #expect(runRequests.allSatisfy { $0.prompt == "Один и тот же запрос" })
        #expect(Set(runRequests.map(\.temperature)) == Set(TemperaturePreset.allCases))
        #expect(completedTemperatures == Set(TemperaturePreset.allCases))
        #expect(reviewRequest?.prompt == "Один и тот же запрос")
        #expect(reviewRequest?.attempts.count == 3)
        #expect(comparison.review.bestAccuracy == .precise)
    }

    @Test
    func encodesTemperatureZeroAsExplicitNumber() throws {
        let request = RunTemperatureRequestDTO(
            prompt: "Запрос",
            temperature: .precise
        )

        let data = try JSONEncoder().encode(request)
        let json = try #require(JSONSerialization.jsonObject(with: data) as? [String: Any])

        #expect(json["temperature"] as? Double == 0)
        #expect(json["prompt"] as? String == "Запрос")
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

private actor RecordingReasoningRepository: ReasoningRepository {
    struct RunRequest: Sendable {
        let problem: String
        let method: ReasoningMethod
    }

    struct ReviewRequest: Sendable {
        let problem: String
        let attempts: [ReasoningAttempt]
    }

    private(set) var runRequests: [RunRequest] = []
    private(set) var reviewRequest: ReviewRequest?

    func run(problem: String, method: ReasoningMethod) async throws -> ReasoningAttempt {
        runRequests.append(RunRequest(problem: problem, method: method))
        return ReasoningAttempt(
            method: method,
            answer: method.rawValue,
            generatedPrompt: method == .metaPrompt ? "generated" : nil,
            experts: [],
            model: "test-model",
            finishReason: "stop",
            usage: ReasoningUsage(promptTokens: 2, completionTokens: 3, totalTokens: 5)
        )
    }

    func review(problem: String, attempts: [ReasoningAttempt]) async throws -> ReasoningReview {
        reviewRequest = ReviewRequest(problem: problem, attempts: attempts)
        return ReasoningReview(
            winner: .stepByStep,
            verdict: "Пошаговый ответ лучше проверен.",
            referenceAnswer: "Эталон",
            differences: ["Глубина проверки"],
            scores: [],
            model: "test-model",
            usage: ReasoningUsage(promptTokens: 10, completionTokens: 10, totalTokens: 20)
        )
    }
}

private actor RecordingTemperatureRepository: TemperatureRepository {
    struct RunRequest: Sendable {
        let prompt: String
        let temperature: TemperaturePreset
    }

    struct ReviewRequest: Sendable {
        let prompt: String
        let attempts: [TemperatureAttempt]
    }

    private(set) var runRequests: [RunRequest] = []
    private(set) var reviewRequest: ReviewRequest?

    func run(prompt: String, temperature: TemperaturePreset) async throws -> TemperatureAttempt {
        runRequests.append(RunRequest(prompt: prompt, temperature: temperature))
        return TemperatureAttempt(
            temperature: temperature,
            answer: String(temperature.rawValue),
            model: "test-model",
            finishReason: "stop",
            usage: ReasoningUsage(promptTokens: 2, completionTokens: 3, totalTokens: 5)
        )
    }

    func review(prompt: String, attempts: [TemperatureAttempt]) async throws -> TemperatureReview {
        reviewRequest = ReviewRequest(prompt: prompt, attempts: attempts)
        return TemperatureReview(
            summary: "Итог",
            bestAccuracy: .precise,
            bestCreativity: .creative,
            bestDiversity: .creative,
            differences: ["Различие"],
            scores: [],
            recommendations: [],
            model: "test-model",
            usage: ReasoningUsage(promptTokens: 10, completionTokens: 10, totalTokens: 20)
        )
    }
}
