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
            branchID: "growth"
        )
        _ = try await useCase.compare(sessionID: "session-10")

        let request = await repository.sendRequest
        let comparedSessionID = await repository.comparedSessionID
        #expect(request?.message == "Новая идея")
        #expect(request?.strategy == .branching)
        #expect(request?.branchID == "growth")
        #expect(comparedSessionID == "session-10")
    }
}

private actor RecordingContextStrategyRepository: ContextStrategyRepository {
    struct SendRequest: Sendable {
        let message: String
        let strategy: ContextStrategy
        let branchID: String?
    }

    private(set) var sendRequest: SendRequest?
    private(set) var comparedSessionID: String?

    func state(sessionID: String, strategy: ContextStrategy, branchID: String?) async throws -> ContextStrategyState {
        makeState(sessionID: sessionID, strategy: strategy, branchID: branchID)
    }

    func send(message: String, sessionID: String, strategy: ContextStrategy, branchID: String?) async throws -> ContextStrategyExchange {
        sendRequest = SendRequest(message: message, strategy: strategy, branchID: branchID)
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
            state: makeState(sessionID: sessionID, strategy: strategy, branchID: branchID)
        )
    }

    func createBranches(sessionID: String) async throws -> ContextStrategyState {
        makeState(sessionID: sessionID, strategy: .branching, branchID: "mvp")
    }

    func compare(sessionID: String) async throws -> ContextStrategyComparison {
        comparedSessionID = sessionID
        return ContextStrategyComparison(
            sessionId: sessionID, scenario: [], question: "question", results: [],
            review: ContextStrategyReview(
                winner: .stickyFacts, verdict: "verdict", differences: [], scores: [], recommendations: [],
                model: "test-model", usage: ModelUsage(promptTokens: 1, completionTokens: 1, totalTokens: 2)
            )
        )
    }

    func clear(sessionID: String) async throws {}

    private func makeState(sessionID: String, strategy: ContextStrategy, branchID: String?) -> ContextStrategyState {
        ContextStrategyState(
            sessionId: sessionID, strategy: strategy, activeBranchId: branchID,
            windowSize: 6, messages: [], facts: [], branches: [],
            lastUsage: ModelUsage(promptTokens: 0, completionTokens: 0, totalTokens: 0)
        )
    }
}
