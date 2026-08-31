import SwiftUI

struct MessageInputView: View {
    @Binding var text: String
    let isSending: Bool
    let canSend: Bool
    let onSend: () -> Void

    var body: some View {
        HStack(alignment: .bottom, spacing: 12) {
            TextField("Введите сообщение", text: $text, axis: .vertical)
                .lineLimit(1...5)
                .textFieldStyle(.plain)
                .padding(.horizontal, 14)
                .padding(.vertical, 11)
                .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 18))
                .submitLabel(.send)
                .onSubmit {
                    if canSend { onSend() }
                }

            Button(action: onSend) {
                Image(systemName: "arrow.up")
                    .font(.headline.bold())
                    .frame(width: 42, height: 42)
                    .foregroundStyle(.white)
                    .background(canSend ? Color.accentColor : Color.secondary, in: Circle())
            }
            .disabled(!canSend)
            .accessibilityLabel(isSending ? "Отправка сообщения" : "Отправить сообщение")
        }
        .padding(.horizontal)
        .padding(.vertical, 10)
        .background(.ultraThinMaterial)
    }
}
