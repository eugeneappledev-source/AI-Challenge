import SwiftUI

struct MessageInputView: View {
    @Binding var text: String
    let isSending: Bool
    let canSend: Bool
    let onSend: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Label("Один запрос для двух режимов", systemImage: "text.bubble")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(.secondary)

            TextField("Введите запрос", text: $text, axis: .vertical)
                .lineLimit(1...5)
                .textFieldStyle(.plain)
                .padding(.horizontal, 14)
                .padding(.vertical, 13)
                .background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 14))
                .submitLabel(.go)
                .onSubmit {
                    if canSend { onSend() }
                }

            Button(action: onSend) {
                HStack(spacing: 10) {
                    if isSending {
                        ProgressView()
                            .tint(.white)
                    } else {
                        Image(systemName: "arrow.triangle.branch")
                    }
                    Text(isSending ? "Сравниваем…" : "Сравнить ответы")
                        .fontWeight(.semibold)
                }
                .frame(maxWidth: .infinity)
                .frame(height: 48)
            }
            .buttonStyle(.borderedProminent)
            .buttonBorderShape(.roundedRectangle(radius: 14))
            .disabled(!canSend)
            .accessibilityLabel(isSending ? "Сравнение ответов" : "Сравнить ответы")
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 20, style: .continuous))
    }
}
