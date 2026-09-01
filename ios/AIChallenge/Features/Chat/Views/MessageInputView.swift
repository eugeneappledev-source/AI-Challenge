import SwiftUI

struct MessageInputView: View {
    @Binding var text: String
    let isSending: Bool
    let canSend: Bool
    let onSend: () -> Void

    @FocusState private var isInputFocused: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Label("Введи свой вопрос", systemImage: "text.bubble")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(.secondary)

            HStack(alignment: .top, spacing: 10) {
                TextField("Напиши вопрос о еде", text: $text, axis: .vertical)
                    .lineLimit(1...5)
                    .textFieldStyle(.plain)
                    .focused($isInputFocused)
                    .submitLabel(.go)
                    .onSubmit {
                        if canSend { onSend() }
                    }

                if !text.isEmpty {
                    Button {
                        text = ""
                        isInputFocused = true
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .font(.title3)
                            .foregroundStyle(.secondary)
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Очистить вопрос")
                }
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 13)
            .background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 14))

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
