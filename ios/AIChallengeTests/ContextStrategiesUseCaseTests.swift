import Foundation
import Testing
@testable import AIChallenge

struct ContextStrategiesUseCaseTests {
    @Test
    func forwardsSelectedStrategyBranchAndComparisonSession() async throws {
        let repository = RecordingContextStrategyRepository()
        let useCase = ManageContextStrategiesUseCase(repository: repository)

        _ = try await useCase.send(
            message: "Новая идея",
            sessionID: "session-10",
            strategy: .branching,
            branchID: "growth",
            windowSize: 10
        )
        _ = try await useCase.compare(sessionID: "session-10", windowSize: 10)

        let request = await repository.sendRequest
        let comparedSessionID = await repository.comparedSessionID
        let comparedWindowSize = await repository.comparedWindowSize
        #expect(request?.message == "Новая идея")
        #expect(request?.strategy == .branching)
        #expect(request?.branchID == "growth")
        #expect(request?.windowSize == 10)
        #expect(comparedSessionID == "session-10")
        #expect(comparedWindowSize == 10)
    }
}

private actor RecordingContextStrategyRepository: ContextStrategyRepository {
    struct SendRequest: Sendable {
        let message: String
        let strategy: ContextStrategy
        let branchID: String?
        let windowSize: Int
    }

    private(set) var sendRequest: SendRequest?
    private(set) var comparedSessionID: String?
    private(set) var comparedWindowSize: Int?

    func state(sessionID: String, strategy: ContextStrategy, branchID: String?, windowSize: Int) async throws -> ContextStrategyState {
        makeState(sessionID: sessionID, strategy: strategy, branchID: branchID, windowSize: windowSize)
    }

    func send(message: String, sessionID: String, strategy: ContextStrategy, branchID: String?, windowSize: Int) async throws -> ContextStrategyExchange {
        sendRequest = SendRequest(message: message, strategy: strategy, branchID: branchID, windowSize: windowSize)
        let profile = AIAgentProfile(
            id: "mentor", name: "Compass", role: "mentor", instructions: "help",
            model: "test-model", temperature: 0.3, maxOutputTokens: 500
        )
        let usage = ModelUsage(promptTokens: 5, completionTokens: 2, totalTokens: 7)
        let exchange = AIAgentExchange(
            agent: profile,
            userMessage: AIAgentMessage(id: "u", role: .user, content: message, createdAt: .now, usage: nil),
            reply: AIAgentMessage(id: "a", role: .assistant, content: "ok", createdAt: .now, usage: usage),
            model: "test-model", finishReason: "stop", usage: usage, trace: []
        )
        return ContextStrategyExchange(
            strategy: strategy, branchId: branchID, exchange: exchange,
            state: makeState(sessionID: sessionID, strategy: strategy, branchID: branchID, windowSize: windowSize)
        )
    }

    func createBranches(sessionID: String) async throws -> ContextStrategyState {
        makeState(sessionID: sessionID, strategy: .branching, branchID: "mvp", windowSize: 6)
    }

    func compare(sessionID: String, windowSize: Int) async throws -> ContextStrategyComparison {
        comparedSessionID = sessionID
        comparedWindowSize = windowSize
        return ContextStrategyComparison(
            sessionId: sessionID, windowSize: windowSize, scenario: [], question: "question", results: [],
            review: ContextStrategyReview(
                winner: .stickyFacts, verdict: "verdict", differences: [], scores: [], recommendations: [],
                model: "test-model", usage: ModelUsage(promptTokens: 1, completionTokens: 1, totalTokens: 2)
            )
        )
    }

    func clear(sessionID: String) async throws {}

    private func makeState(sessionID: String, strategy: ContextStrategy, branchID: String?, windowSize: Int) -> ContextStrategyState {
        ContextStrategyState(
            sessionId: sessionID, strategy: strategy, activeBranchId: branchID,
            windowSize: windowSize, messages: [], facts: [], branches: [],
            lastUsage: ModelUsage(promptTokens: 0, completionTokens: 0, totalTokens: 0)
        )
    }
}
