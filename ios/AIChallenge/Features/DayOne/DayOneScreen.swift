import SwiftUI

struct DayOneScreen: View {
    @State var viewModel: DayOneViewModel

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 20) {
                introduction
                requestFlow
                inputCard
                result
            }
            .padding()
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle("День 1")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar(.visible, for: .navigationBar)
        .alert(
            "Не удалось получить ответ",
            isPresented: Binding(
                get: { viewModel.errorMessage != nil },
                set: { if !$0 { viewModel.errorMessage = nil } }
            )
        ) {
            Button("Понятно", role: .cancel) {
                viewModel.errorMessage = nil
            }
        } message: {
            Text(viewModel.errorMessage ?? "Неизвестная ошибка")
        }
    }

    private var introduction: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("ПЕРВЫЙ API-ЗАПРОС")
                .font(.caption.weight(.bold))
                .foregroundStyle(.blue)
                .tracking(1.2)

            Text("Сообщение уходит\nв облачную LLM")
                .font(.largeTitle.bold())

            Text("Минимальный сквозной сценарий: iPhone отправляет текст на Go backend, DeepSeek готовит ответ, а приложение показывает результат.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
        .padding(.vertical, 8)
    }

    private var requestFlow: some View {
        HStack(spacing: 8) {
            FlowBadge(icon: "iphone", title: "iOS")
            Image(systemName: "arrow.right")
            FlowBadge(icon: "server.rack", title: "Go API")
            Image(systemName: "arrow.right")
            FlowBadge(icon: "sparkles", title: "DeepSeek")
        }
        .font(.caption.weight(.semibold))
        .foregroundStyle(.secondary)
        .frame(maxWidth: .infinity)
        .padding(14)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
    }

    private var inputCard: some View {
        VStack(alignment: .leading, spacing: 14) {
            Label("Твой запрос", systemImage: "text.bubble")
                .font(.subheadline.weight(.semibold))

            TextField("Напиши вопрос о еде", text: $viewModel.input, axis: .vertical)
                .lineLimit(1...5)
                .padding(14)
                .background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 14))
                .submitLabel(.send)
                .onSubmit {
                    guard viewModel.canSend else { return }
                    Task { await viewModel.send() }
                }

            Button {
                Task { await viewModel.send() }
            } label: {
                HStack(spacing: 9) {
                    if viewModel.isSending {
                        ProgressView().tint(.white)
                    } else {
                        Image(systemName: "paperplane.fill")
                    }
                    Text(viewModel.isSending ? "Получаем ответ…" : "Отправить в LLM")
                        .fontWeight(.semibold)
                }
                .frame(maxWidth: .infinity)
                .frame(height: 48)
                .foregroundStyle(viewModel.canSend || viewModel.isSending ? .white : Color(.tertiaryLabel))
                .background(
                    viewModel.canSend || viewModel.isSending ? Color.blue : Color(.quaternarySystemFill),
                    in: RoundedRectangle(cornerRadius: 14, style: .continuous)
                )
            }
            .buttonStyle(.plain)
            .disabled(!viewModel.canSend)
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 20, style: .continuous))
    }

    @ViewBuilder
    private var result: some View {
        if let reply = viewModel.reply, let prompt = viewModel.submittedPrompt {
            VStack(alignment: .leading, spacing: 16) {
                Label("Ответ получен", systemImage: "checkmark.circle.fill")
                    .font(.headline)
                    .foregroundStyle(Color.aiSuccess)

                VStack(alignment: .leading, spacing: 5) {
                    Text("ЗАПРОС")
                        .font(.caption2.weight(.bold))
                        .foregroundStyle(.blue)
                    Text(prompt)
                        .font(.subheadline.weight(.medium))
                }

                Divider()

                Text(reply.answer)
                    .lineSpacing(4)
                    .textSelection(.enabled)

                Divider()

                HStack {
                    Label(reply.model, systemImage: "cpu")
                    Spacer()
                    Text("\(reply.totalTokens) токенов")
                }
                .font(.caption)
                .foregroundStyle(.secondary)
            }
            .padding(18)
            .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
        } else if !viewModel.isSending {
            ContentUnavailableView {
                Label("Готово к запросу", systemImage: "bubble.left.and.bubble.right")
            } description: {
                Text("Введи сообщение, чтобы увидеть полный путь запроса через API.")
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 24)
        }
    }
}

private struct FlowBadge: View {
    let icon: String
    let title: String

    var body: some View {
        Label(title, systemImage: icon)
            .padding(.horizontal, 9)
            .padding(.vertical, 7)
            .background(Color(.tertiarySystemGroupedBackground), in: Capsule())
    }
}
