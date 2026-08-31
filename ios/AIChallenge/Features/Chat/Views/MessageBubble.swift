import SwiftUI

struct MessageBubble: View {
    let message: ChatMessage

    var body: some View {
        HStack {
            if message.role == .user {
                Spacer(minLength: 44)
            }

            Text(message.text)
                .textSelection(.enabled)
                .padding(.horizontal, 14)
                .padding(.vertical, 10)
                .foregroundStyle(message.role == .user ? Color.white : Color.primary)
                .background(
                    message.role == .user ? Color.accentColor : Color(.secondarySystemGroupedBackground),
                    in: RoundedRectangle(cornerRadius: 18, style: .continuous)
                )

            if message.role == .assistant {
                Spacer(minLength: 44)
            }
        }
        .padding(.horizontal)
    }
}
