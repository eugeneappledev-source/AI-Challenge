import Foundation
import Observation

@MainActor
@Observable
final class DayOneViewModel {
    var input = ""
    private(set) var reply: ChatReply?
    private(set) var submittedPrompt: String?
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

        isSending = true
        errorMessage = nil
        defer { isSending = false }

        do {
            reply = try await sendMessage.execute(message: message, mode: .unrestricted)
            submittedPrompt = message
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
        }
    }
}
