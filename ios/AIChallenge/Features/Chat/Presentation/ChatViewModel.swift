import Foundation
import Observation

@MainActor
@Observable
final class ChatViewModel {
    var input = ""
    var selectedMode: ResponseControlMode = .unrestricted
    private(set) var comparison: ResponseComparison?
    private(set) var isSending = false
    var errorMessage: String?

    private let compareResponses: CompareResponsesUseCase

    init(compareResponses: CompareResponsesUseCase) {
        self.compareResponses = compareResponses
    }

    var canSend: Bool {
        !input.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isSending
    }

    var selectedReply: ChatReply? {
        comparison?.reply(for: selectedMode)
    }

    func compare() async {
        let message = input.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !message.isEmpty, !isSending else { return }

        isSending = true
        errorMessage = nil
        defer { isSending = false }

        do {
            comparison = try await compareResponses.execute(message: message)
            selectedMode = .unrestricted
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
        }
    }
}
