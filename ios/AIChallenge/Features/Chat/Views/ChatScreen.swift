import SwiftUI

struct ChatScreen: View {
    @State var viewModel: ChatViewModel

    var body: some View {
        NavigationStack {
            Group {
                if viewModel.messages.isEmpty {
                    emptyState
                } else {
                    messages
                }
            }
            .background(Color(.systemGroupedBackground))
            .navigationTitle("AI Challenge")
            .safeAreaInset(edge: .bottom) {
                MessageInputView(
                    text: $viewModel.input,
                    isSending: viewModel.isSending,
                    canSend: viewModel.canSend
                ) {
                    Task { await viewModel.send() }
                }
            }
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
    }

    private var emptyState: some View {
        ContentUnavailableView {
            Label("Спроси DeepSeek", systemImage: "sparkles")
        } description: {
            Text("Сообщение уйдёт через твой Go-backend на VPS, а ответ появится здесь.")
        }
    }

    private var messages: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(spacing: 12) {
                    ForEach(viewModel.messages) { message in
                        MessageBubble(message: message)
                            .id(message.id)
                    }

                    if viewModel.isSending {
                        HStack {
                            ProgressView()
                            Text("DeepSeek отвечает…")
                                .foregroundStyle(.secondary)
                            Spacer()
                        }
                        .padding(.horizontal)
                    }
                }
                .padding(.vertical)
            }
            .onChange(of: viewModel.messages.count) {
                guard let lastID = viewModel.messages.last?.id else { return }
                withAnimation {
                    proxy.scrollTo(lastID, anchor: .bottom)
                }
            }
        }
    }
}
