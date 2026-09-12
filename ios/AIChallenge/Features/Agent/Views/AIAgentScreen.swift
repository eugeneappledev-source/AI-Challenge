import SwiftUI

struct AIAgentScreen: View {
    @State var viewModel: AIAgentViewModel
    @FocusState private var inputFocused: Bool

    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 18) {
                    header
                    if let profile = viewModel.profile { AgentProfileCard(profile: profile) }
                    explanation
                    suggestions

                    if viewModel.messages.isEmpty && !viewModel.isLoading {
                        emptyState
                    } else {
                        messages
                        if let exchange = viewModel.lastExchange { AgentTraceCard(exchange: exchange) }
                        if viewModel.isLoading { pendingExchange }
                    }
                    Color.clear.frame(height: 92).id("bottom")
                }
                .padding()
            }
            .background(Color(.systemGroupedBackground))
            .safeAreaInset(edge: .bottom) { composer }
            .onChange(of: viewModel.messages.count) {
                withAnimation { proxy.scrollTo("bottom", anchor: .bottom) }
            }
            .onChange(of: viewModel.isLoading) {
                withAnimation { proxy.scrollTo("bottom", anchor: .bottom) }
            }
        }
        .navigationTitle("День 6")
        .navigationBarTitleDisplayMode(.inline)
        .task { await viewModel.load() }
        .alert(
            "Агент не ответил",
            isPresented: Binding(get: { viewModel.errorMessage != nil }, set: { if !$0 { viewModel.errorMessage = nil } })
        ) { Button("Понятно", role: .cancel) {} } message: { Text(viewModel.errorMessage ?? "Неизвестная ошибка") }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("ПЕРВЫЙ АГЕНТ").font(.caption.bold()).tracking(1).foregroundStyle(Color.aiCoral)
            Text("Знакомься — Compass.").font(.largeTitle.bold())
            Text("Он принимает сообщение, применяет собственную конфигурацию, вызывает LLM и возвращает результат в интерфейс.")
                .font(.subheadline).foregroundStyle(.secondary).lineSpacing(3)
        }.padding(.vertical, 8)
    }

    private var explanation: some View {
        VStack(alignment: .leading, spacing: 12) {
            Label("Почему это уже агент", systemImage: "shippingbox.fill").font(.headline)
            AgentStep(number: "01", title: "Принимает input", detail: "Валидирует пользовательское сообщение")
            AgentStep(number: "02", title: "Владеет настройками", detail: "Имя, роль, system prompt, модель и параметры")
            AgentStep(number: "03", title: "Управляет вызовом", detail: "Сам собирает запрос к LLMProvider и обрабатывает ответ")
        }
        .padding(16)
        .background(Color.aiCoral.opacity(0.07), in: RoundedRectangle(cornerRadius: 20))
    }

    private var suggestions: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 9) {
                ForEach(Self.examples, id: \.self) { value in
                    Button { viewModel.useExample(value); inputFocused = true } label: {
                        Text(value).font(.caption.weight(.medium)).lineLimit(1).padding(.horizontal, 12).padding(.vertical, 9)
                            .background(Color(.secondarySystemGroupedBackground), in: Capsule())
                    }.buttonStyle(.plain)
                }
            }
        }
    }

    private var emptyState: some View {
        VStack(spacing: 12) {
            Image(systemName: "bubble.left.and.bubble.right.fill").font(.largeTitle).foregroundStyle(Color.aiCoral)
            Text("Агент готов к работе").font(.headline)
            Text("Можно задать любой вопрос. В День 6 каждый ответ независим — постоянная память появится в следующем задании.")
                .font(.caption).foregroundStyle(.secondary).multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity).padding(.vertical, 30).padding(.horizontal, 24)
    }

    private var messages: some View {
        VStack(spacing: 12) {
            ForEach(viewModel.messages) { message in
                HStack {
                    if message.role == .user { Spacer(minLength: 48) }
                    VStack(alignment: .leading, spacing: 7) {
                        Text(message.role == .user ? "ВЫ" : (viewModel.profile?.name.uppercased() ?? "AGENT"))
                            .font(.system(size: 9, weight: .bold)).foregroundStyle(message.role == .user ? .white.opacity(0.8) : Color.aiCoral)
                        AgentMarkdownText(message.content)
                    }
                    .foregroundStyle(message.role == .user ? .white : .primary)
                    .padding(14)
                    .background(message.role == .user ? Color.aiForest : Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 18))
                    if message.role != .user { Spacer(minLength: 32) }
                }
            }
        }
    }

    private var pendingExchange: some View {
        VStack(spacing: 12) {
            if let pendingMessage = viewModel.pendingMessage {
                HStack {
                    Spacer(minLength: 48)
                    VStack(alignment: .leading, spacing: 7) {
                        Text("ВЫ")
                            .font(.system(size: 9, weight: .bold))
                            .foregroundStyle(.white.opacity(0.8))
                        Text(pendingMessage)
                            .font(.body)
                            .lineSpacing(3)
                    }
                    .foregroundStyle(.white)
                    .padding(14)
                    .background(Color.aiForest, in: RoundedRectangle(cornerRadius: 18))
                }
            }

            HStack(alignment: .top, spacing: 12) {
                ProgressView()
                    .tint(Color.aiCoral)
                    .controlSize(.regular)
                    .padding(.top, 2)
                VStack(alignment: .leading, spacing: 4) {
                    Text("COMPASS ФОРМИРУЕТ ОТВЕТ")
                        .font(.system(size: 9, weight: .bold))
                        .tracking(0.7)
                        .foregroundStyle(Color.aiCoral)
                    Text("Применяет настройки агента и ждёт ответ DeepSeek…")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer(minLength: 24)
            }
            .padding(14)
            .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 18))
            .overlay {
                RoundedRectangle(cornerRadius: 18)
                    .stroke(Color.aiCoral.opacity(0.18))
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Compass формирует ответ")
    }

    private var composer: some View {
        HStack(alignment: .bottom, spacing: 10) {
            TextField("Сообщение агенту…", text: $viewModel.input, axis: .vertical)
                .lineLimit(1...5).padding(.horizontal, 14).padding(.vertical, 12)
                .background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 16))
                .focused($inputFocused).disabled(viewModel.isLoading)
            Button { inputFocused = false; Task { await viewModel.send() } } label: {
                Group {
                    if viewModel.isLoading { ProgressView().tint(.white) }
                    else { Image(systemName: "arrow.up").font(.headline.bold()) }
                }
                .frame(width: 46, height: 46).foregroundStyle(.white)
                .background(viewModel.isLoading || viewModel.canSend ? Color.aiCoral : Color(.tertiarySystemFill), in: Circle())
            }.buttonStyle(.plain).disabled(!viewModel.canSend)
        }
        .padding(.horizontal).padding(.vertical, 10)
        .background(.ultraThinMaterial)
    }

    private static let examples = [
        "Объясни простыми словами, что делает AI-агент",
        "Составь план изучения LLM API на неделю",
        "Придумай идею полезного iOS-приложения с ИИ",
    ]
}

private struct AgentProfileCard: View {
    let profile: AIAgentProfile
    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack(spacing: 13) {
                Image(systemName: "location.north.circle.fill").font(.largeTitle).foregroundStyle(.white)
                    .frame(width: 58, height: 58).background(Color.aiCoral, in: RoundedRectangle(cornerRadius: 18))
                VStack(alignment: .leading, spacing: 3) {
                    Text(profile.name).font(.title3.bold())
                    Text(profile.role).font(.caption.weight(.semibold)).foregroundStyle(Color.aiCoral)
                }
                Spacer(); Text("AGENT").font(.system(size: 9, weight: .bold)).padding(.horizontal, 8).padding(.vertical, 5)
                    .background(Color.aiSuccess.opacity(0.12), in: Capsule()).foregroundStyle(Color.aiSuccess)
            }
            Divider()
            HStack {
                AgentProfileMetric(title: "Модель", value: profile.model)
                AgentProfileMetric(title: "Temperature", value: profile.temperature.formatted())
                AgentProfileMetric(title: "Max output", value: "\(profile.maxOutputTokens)")
            }
            DisclosureGroup("Системная инструкция") {
                Text(profile.instructions).font(.caption).foregroundStyle(.secondary).padding(.top, 6).textSelection(.enabled)
            }.font(.caption.weight(.semibold))
        }
        .padding(17).background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22))
        .overlay { RoundedRectangle(cornerRadius: 22).stroke(Color.aiCoral.opacity(0.2)) }
    }
}

private struct AgentProfileMetric: View {
    let title: String; let value: String
    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(title.uppercased()).font(.system(size: 8, weight: .bold)).foregroundStyle(.secondary)
            Text(value).font(.caption.bold().monospaced()).lineLimit(1).minimumScaleFactor(0.65)
        }.frame(maxWidth: .infinity, alignment: .leading)
    }
}

private struct AgentStep: View {
    let number: String; let title: String; let detail: String
    var body: some View {
        HStack(spacing: 11) {
            Text(number).font(.caption2.bold().monospaced()).foregroundStyle(.white).frame(width: 30, height: 30).background(Color.aiCoral, in: Circle())
            VStack(alignment: .leading, spacing: 2) { Text(title).font(.subheadline.bold()); Text(detail).font(.caption).foregroundStyle(.secondary) }
        }
    }
}

private struct AgentTraceCard: View {
    let exchange: AIAgentExchange
    var body: some View {
        DisclosureGroup {
            VStack(alignment: .leading, spacing: 10) {
                ForEach(Array(exchange.trace.enumerated()), id: \.offset) { index, item in
                    Label(item, systemImage: index == exchange.trace.count - 1 ? "checkmark.circle.fill" : "arrow.down.circle")
                        .font(.caption).foregroundStyle(index == exchange.trace.count - 1 ? Color.aiSuccess : .secondary)
                }
                Divider()
                HStack { Text(exchange.model); Spacer(); Text("\(exchange.usage.totalTokens) токенов · \(exchange.finishReason)") }
                    .font(.caption2).foregroundStyle(.secondary)
            }.padding(.top, 10)
        } label: { Label("Как агент обработал последний запрос", systemImage: "point.3.connected.trianglepath.dotted").font(.subheadline.bold()) }
        .padding(16).background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 18))
    }
}

private struct AgentMarkdownText: View {
    let value: String
    init(_ value: String) { self.value = value }
    var body: some View {
        if let text = try? AttributedString(markdown: value) { Text(text).font(.body).lineSpacing(3).textSelection(.enabled) }
        else { Text(value).font(.body).lineSpacing(3).textSelection(.enabled) }
    }
}
