import Foundation
import Observation

@MainActor
@Observable
final class ChatViewModel {
    var input = ""
    private(set) var messages: [ChatMessage] = []
    private(set) var isSending = false
    var errorMessage: String?

    private let sendMessage: SendMessageUseCase

    init(sendMessage: SendMessageUseCase) {
        self.sendMessage = sendMessage
    }

    var canSend: Bool {
        !input.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isSending
    }

    func send() async {
        let message = input.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !message.isEmpty, !isSending else { return }

        input = ""
        messages.append(ChatMessage(role: .user, text: message))
        isSending = true
        defer { isSending = false }

        do {
            let reply = try await sendMessage.execute(message: message)
            messages.append(ChatMessage(role: .assistant, text: reply.answer))
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
        }
    }
}
