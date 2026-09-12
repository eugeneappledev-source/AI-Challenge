import Foundation
import Observation

@MainActor
@Observable
final class TokenLabViewModel {
    var input = ""
    private(set) var lastAnswer: String?
    private(set) var metrics: AgentTokenMetrics?
    private(set) var isLoading = false
    var errorMessage: String?

    let conversationID: String
    private let useCase: InspectAgentTokensUseCase

    init(useCase: InspectAgentTokensUseCase, conversationID: String) {
        self.useCase = useCase
        self.conversationID = conversationID
    }

    var canSend: Bool { !input.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isLoading }

    func load() async {
        guard metrics == nil else { return }
        do { metrics = try await useCase.load(conversationID: conversationID) }
        catch { show(error) }
    }

    func send() async {
        let message = input.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !message.isEmpty, !isLoading else { return }
        input = ""
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }
        do {
            let result = try await useCase.send(message: message, conversationID: conversationID)
            lastAnswer = result.0.reply.content
            metrics = result.1
        } catch {
            input = message
            show(error)
        }
    }

    func clear() async {
        guard !isLoading else { return }
        do {
            try await useCase.clear(conversationID: conversationID)
            lastAnswer = nil
            metrics = try await useCase.load(conversationID: conversationID)
        } catch { show(error) }
    }

    func useExample(_ text: String) { input = text }

    private func show(_ error: Error) {
        errorMessage = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
    }
}
