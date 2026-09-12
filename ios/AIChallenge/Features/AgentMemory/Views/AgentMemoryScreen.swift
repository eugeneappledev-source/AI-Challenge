import SwiftUI

struct AgentMemoryScreen: View {
    @State var viewModel: AgentMemoryViewModel
    @FocusState private var inputFocused: Bool

    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 18) {
                    header
                    persistenceCard
                    testCard
                    if viewModel.messages.isEmpty { emptyState } else { messages }
                    Color.clear.frame(height: 92).id("memory-bottom")
                }
                .padding()
            }
            .background(Color(.systemGroupedBackground))
            .safeAreaInset(edge: .bottom) { composer }
            .onChange(of: viewModel.messages.count) { withAnimation { proxy.scrollTo("memory-bottom", anchor: .bottom) } }
        }
        .navigationTitle("День 7")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Очистить", role: .destructive) { Task { await viewModel.clear() } }
                    .disabled(viewModel.messages.isEmpty || viewModel.isLoading)
            }
        }
        .task { await viewModel.load() }
        .alert("Ошибка памяти", isPresented: Binding(get: { viewModel.errorMessage != nil }, set: { if !$0 { viewModel.errorMessage = nil } })) {
            Button("Понятно", role: .cancel) {}
        } message: { Text(viewModel.errorMessage ?? "Неизвестная ошибка") }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("ПОСТОЯННЫЙ КОНТЕКСТ").font(.caption.bold()).tracking(1).foregroundStyle(Color.aiPurple)
            Text("Compass помнит разговор.").font(.largeTitle.bold())
            Text("История хранится на сервере и снова подставляется в messages после перезапуска приложения или backend.")
                .font(.subheadline).foregroundStyle(.secondary).lineSpacing(3)
        }.padding(.vertical, 8)
    }

    private var persistenceCard: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Label(viewModel.didRestore ? "История восстановлена" : "Загрузка истории", systemImage: viewModel.didRestore ? "checkmark.icloud.fill" : "icloud.and.arrow.down")
                    .font(.headline).foregroundStyle(viewModel.didRestore ? Color.aiSuccess : Color.aiPurple)
                Spacer()
                if viewModel.isLoading { ProgressView().tint(Color.aiPurple) }
            }
            Divider()
            memoryRow(icon: "externaldrive.fill", title: "Хранилище", value: "SQLite · backend")
            memoryRow(icon: "bubble.left.and.bubble.right.fill", title: "Сообщений", value: "\(viewModel.messages.count)")
            memoryRow(icon: "number", title: "Conversation ID", value: String(viewModel.conversationID.prefix(13)) + "…")
        }
        .padding(17).background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22))
        .overlay { RoundedRectangle(cornerRadius: 22).stroke(Color.aiPurple.opacity(0.2)) }
    }

    private var testCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            Label("Проверка для видео", systemImage: "record.circle").font(.headline)
            Text("1. Напишите: «Меня зовут Женя, мой любимый язык — Swift».\n2. Закройте и заново откройте приложение.\n3. Спросите: «Как меня зовут и какой язык я люблю?»")
                .font(.caption).foregroundStyle(.secondary).lineSpacing(4)
        }
        .padding(16).background(Color.aiPurple.opacity(0.08), in: RoundedRectangle(cornerRadius: 18))
    }

    private var emptyState: some View {
        VStack(spacing: 11) {
            Image(systemName: "memorychip.fill").font(.largeTitle).foregroundStyle(Color.aiPurple)
            Text("Память пока пуста").font(.headline)
            Text("Начните диалог — каждая завершённая пара сообщений сохранится в SQLite.")
                .font(.caption).foregroundStyle(.secondary).multilineTextAlignment(.center)
        }.frame(maxWidth: .infinity).padding(.vertical, 28)
    }

    private var messages: some View {
        VStack(spacing: 12) {
            ForEach(viewModel.messages) { message in
                HStack {
                    if message.role == .user { Spacer(minLength: 45) }
                    VStack(alignment: .leading, spacing: 6) {
                        Text(message.role == .user ? "ВЫ" : "COMPASS").font(.system(size: 9, weight: .bold))
                            .foregroundStyle(message.role == .user ? .white.opacity(0.8) : Color.aiPurple)
                        Text(message.content).font(.body).lineSpacing(3).textSelection(.enabled)
                    }
                    .foregroundStyle(message.role == .user ? .white : .primary).padding(14)
                    .background(message.role == .user ? Color.aiForest : Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 18))
                    if message.role != .user { Spacer(minLength: 30) }
                }
            }
        }
    }

    private var composer: some View {
        HStack(alignment: .bottom, spacing: 10) {
            TextField("Продолжить разговор…", text: $viewModel.input, axis: .vertical)
                .lineLimit(1...5).padding(.horizontal, 14).padding(.vertical, 12)
                .background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 16))
                .focused($inputFocused).disabled(viewModel.isLoading)
            Button { inputFocused = false; Task { await viewModel.send() } } label: {
                Group { if viewModel.isLoading { ProgressView().tint(.white) } else { Image(systemName: "arrow.up").font(.headline.bold()) } }
                    .frame(width: 46, height: 46).foregroundStyle(.white)
                    .background(viewModel.canSend ? Color.aiPurple : Color(.tertiarySystemFill), in: Circle())
            }.buttonStyle(.plain).disabled(!viewModel.canSend)
        }.padding(.horizontal).padding(.vertical, 10).background(.ultraThinMaterial)
    }

    private func memoryRow(icon: String, title: String, value: String) -> some View {
        HStack { Label(title, systemImage: icon).foregroundStyle(.secondary); Spacer(); Text(value).font(.caption.bold().monospaced()) }.font(.caption)
    }
}
