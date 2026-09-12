import Foundation
import Observation

@MainActor
@Observable
final class AIAgentViewModel {
    var input = ""
    private(set) var profile: AIAgentProfile?
    private(set) var messages: [AIAgentMessage] = []
    private(set) var lastExchange: AIAgentExchange?
    private(set) var isLoading = false
    var errorMessage: String?

    private let talkToAgent: TalkToAgentUseCase

    init(talkToAgent: TalkToAgentUseCase) { self.talkToAgent = talkToAgent }

    var canSend: Bool {
        !input.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isLoading
    }

    func load() async {
        guard profile == nil else { return }
        do { profile = try await talkToAgent.loadProfile() }
        catch { errorMessage = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription }
    }

    func send() async {
        let message = input.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !message.isEmpty, !isLoading else { return }
        input = ""
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }
        do {
            let exchange = try await talkToAgent.execute(message: message)
            profile = exchange.agent
            messages.append(exchange.userMessage)
            messages.append(exchange.reply)
            lastExchange = exchange
        } catch {
            input = message
            errorMessage = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
        }
    }

    func useExample(_ value: String) { input = value }
    func clear() { guard !isLoading else { return }; messages = []; lastExchange = nil; errorMessage = nil }
}
