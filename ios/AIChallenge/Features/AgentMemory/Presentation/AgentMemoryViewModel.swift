import Foundation
import Observation

@MainActor
@Observable
final class AgentMemoryViewModel {
    var input = ""
    private(set) var profile: AIAgentProfile?
    private(set) var messages: [AIAgentMessage] = []
    private(set) var isLoading = false
    private(set) var didRestore = false
    var errorMessage: String?

    let conversationID: String
    private let useCase: ContinueAgentConversationUseCase

    init(useCase: ContinueAgentConversationUseCase, conversationID: String) {
        self.useCase = useCase
        self.conversationID = conversationID
    }

    var canSend: Bool { !input.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isLoading }

    func load() async {
        guard !isLoading else { return }
        isLoading = true
        defer { isLoading = false }
        do {
            async let loadedProfile = useCase.loadProfile()
            async let history = useCase.loadHistory(conversationID: conversationID)
            profile = try await loadedProfile
            messages = try await history.messages
            didRestore = true
        } catch { show(error) }
    }

    func send() async {
        let message = input.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !message.isEmpty, !isLoading else { return }
        input = ""
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }
        do {
            let exchange = try await useCase.send(message: message, conversationID: conversationID)
            profile = exchange.agent
            messages.append(exchange.userMessage)
            messages.append(exchange.reply)
        } catch {
            input = message
            show(error)
        }
    }

    func clear() async {
        guard !isLoading else { return }
        isLoading = true
        defer { isLoading = false }
        do {
            try await useCase.clear(conversationID: conversationID)
            messages = []
            didRestore = true
        } catch { show(error) }
    }

    private func show(_ error: Error) {
        errorMessage = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
    }
}
