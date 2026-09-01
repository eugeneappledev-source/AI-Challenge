import SwiftUI

struct ChatScreen: View {
    @State var viewModel: ChatViewModel

    var body: some View {
        NavigationStack {
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 20) {
                    introduction
                    AssistantGuideCard()

                    MessageInputView(
                        text: $viewModel.input,
                        isSending: viewModel.isSending,
                        canSend: viewModel.canSend
                    ) {
                        Task { await viewModel.compare() }
                    }

                    resultSection
                }
                .padding()
            }
            .background(Color(.systemGroupedBackground))
            .navigationTitle("День 2")
            .navigationBarTitleDisplayMode(.inline)
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

    private var introduction: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("ФОРМАТ ОТВЕТА")
                .font(.caption.weight(.bold))
                .foregroundStyle(.tint)
                .tracking(1.2)

            Text("Один запрос.\nДва уровня контроля.")
                .font(.largeTitle.bold())

            Text("Пищевой ассистент отвечает только по теме еды. Сравни свободный ответ с результатом в заданном формате, ограниченной длины и с явным завершением.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
        .padding(.vertical, 8)
    }

    @ViewBuilder
    private var resultSection: some View {
        if viewModel.isSending {
            LoadingComparisonCard()
        } else if let comparison = viewModel.comparison, let reply = viewModel.selectedReply {
            VStack(alignment: .leading, spacing: 14) {
                Text("Результаты")
                    .font(.title2.bold())

                Picker("Режим ответа", selection: $viewModel.selectedMode) {
                    Text("Без контроля").tag(ResponseControlMode.unrestricted)
                    Text("С контролем").tag(ResponseControlMode.controlled)
                }
                .pickerStyle(.segmented)

                ComparisonResultCard(
                    prompt: comparison.prompt,
                    reply: reply
                )
                .id(viewModel.selectedMode)
                .transition(.opacity.combined(with: .scale(scale: 0.98)))
            }
            .animation(.easeInOut(duration: 0.2), value: viewModel.selectedMode)
        } else {
            ContentUnavailableView {
                Label("Готово к сравнению", systemImage: "rectangle.2.swap")
            } description: {
                Text("Задай вопрос о еде — приложение одновременно получит оба варианта ответа.")
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 28)
        }
    }
}

private struct LoadingComparisonCard: View {
    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("DeepSeek готовит два ответа")
                .font(.headline)

            ForEach(ResponseControlMode.allCases) { mode in
                HStack(spacing: 12) {
                    ProgressView()
                    Text(mode.title)
                    Spacer()
                }
            }
        }
        .padding(18)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 20, style: .continuous))
    }
}

private struct ComparisonResultCard: View {
    let prompt: String
    let reply: ChatReply

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack(alignment: .firstTextBaseline, spacing: 12) {
                Label(reply.mode.title, systemImage: reply.mode.systemImage)
                    .font(.headline)
                Spacer()
                Text(reply.model)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }

            ScrollView(.horizontal) {
                HStack(spacing: 8) {
                    ForEach(reply.mode.badges, id: \.self) { badge in
                        Text(badge)
                            .font(.caption.weight(.semibold))
                            .padding(.horizontal, 10)
                            .padding(.vertical, 6)
                            .foregroundStyle(reply.mode.tint)
                            .background(reply.mode.tint.opacity(0.12), in: Capsule())
                    }
                }
            }
            .scrollIndicators(.hidden)

            VStack(alignment: .leading, spacing: 5) {
                Text("ИСХОДНЫЙ ЗАПРОС")
                    .font(.caption2.weight(.bold))
                    .foregroundStyle(.secondary)
                Text(prompt)
                    .font(.subheadline)
            }

            Divider()

            if let controlledAnswer = reply.controlledAnswer {
                ControlledAnswerView(answer: controlledAnswer)
            } else {
                Text(reply.answer)
                    .font(.body)
                    .textSelection(.enabled)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }

            if reply.mode == .controlled {
                RawModelAnswerView(json: reply.answer)
            }

            Divider()

            HStack(spacing: 0) {
                MetadataItem(title: "Ответ", value: "\(reply.completionTokens) токенов")
                Divider().frame(height: 34)
                MetadataItem(title: "Всего", value: "\(reply.totalTokens) токенов")
                Divider().frame(height: 34)
                MetadataItem(title: "Завершение", value: reply.finishReason ?? "—")
            }
        }
        .padding(18)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 20, style: .continuous))
    }
}

private struct AssistantGuideCard: View {
    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Label("Что умеет ассистент", systemImage: "fork.knife")
                .font(.headline)

            Text("На один вопрос он готовит два ответа: свободный текст и контролируемый JSON для разбора приложением.")
                .font(.subheadline)
                .foregroundStyle(.secondary)

            GuideRow(
                icon: "checkmark.circle.fill",
                color: .green,
                title: "Лучше спрашивать",
                text: "рецепты, ингредиенты, замены продуктов, кухонные техники и безопасность еды"
            )

            GuideRow(
                icon: "xmark.circle.fill",
                color: .orange,
                title: "Не по теме",
                text: "кино, политика, технологии и личные вопросы — controlled-режим вернёт out_of_scope"
            )
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 20, style: .continuous))
    }
}

private struct GuideRow: View {
    let icon: String
    let color: Color
    let title: String
    let text: String

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            Image(systemName: icon)
                .foregroundStyle(color)

            Text("**\(title):** \(text)")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }
}

private struct ControlledAnswerView: View {
    let answer: ControlledFoodAnswer

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack(spacing: 8) {
                Image(systemName: answer.status == .ok ? "checkmark.seal.fill" : "exclamationmark.circle.fill")
                Text(answer.status == .ok ? "JSON успешно распарсен" : "Запрос вне темы")
                    .font(.subheadline.weight(.semibold))
            }
            .foregroundStyle(answer.status == .ok ? Color.green : Color.orange)

            Text(answer.answer)
                .font(.body)
                .textSelection(.enabled)

            if !answer.ingredients.isEmpty {
                StructuredListSection(
                    title: "Ингредиенты",
                    systemImage: "basket.fill",
                    items: answer.ingredients,
                    numbered: false
                )
            }

            if !answer.steps.isEmpty {
                StructuredListSection(
                    title: "Приготовление",
                    systemImage: "list.number",
                    items: answer.steps,
                    numbered: true
                )
            }
        }
        .padding(14)
        .background(Color.purple.opacity(0.08), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
    }
}

private struct StructuredListSection: View {
    let title: String
    let systemImage: String
    let items: [String]
    let numbered: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 9) {
            Label(title, systemImage: systemImage)
                .font(.subheadline.weight(.semibold))

            ForEach(Array(items.enumerated()), id: \.offset) { index, item in
                HStack(alignment: .top, spacing: 9) {
                    Text(numbered ? "\(index + 1)." : "•")
                        .fontWeight(.semibold)
                        .foregroundStyle(.purple)
                        .frame(minWidth: 18, alignment: .trailing)

                    Text(item)
                        .font(.subheadline)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
        }
    }
}

private struct RawModelAnswerView: View {
    let json: String

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Label("P.S. Исходный ответ модели", systemImage: "curlybraces")
                .font(.subheadline.weight(.semibold))

            Text("Приложение распарсило этот JSON и собрало карточку выше.")
                .font(.caption)
                .foregroundStyle(.secondary)

            ScrollView(.horizontal) {
                Text(json)
                    .font(.system(.caption, design: .monospaced))
                    .textSelection(.enabled)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
            .scrollIndicators(.hidden)
            .padding(12)
            .background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 10, style: .continuous))
        }
    }
}

private struct MetadataItem: View {
    let title: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(value)
                .font(.caption.weight(.semibold))
                .lineLimit(1)
                .minimumScaleFactor(0.75)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 8)
    }
}

private extension ResponseControlMode {
    var title: String {
        switch self {
        case .unrestricted: "Без контроля"
        case .controlled: "С контролем"
        }
    }

    var systemImage: String {
        switch self {
        case .unrestricted: "text.alignleft"
        case .controlled: "slider.horizontal.3"
        }
    }

    var tint: Color {
        switch self {
        case .unrestricted: .blue
        case .controlled: .purple
        }
    }

    var badges: [String] {
        switch self {
        case .unrestricted:
            ["Тема: еда", "Свободный формат", "Без лимита"]
        case .controlled:
            ["Тема: еда", "JSON", "≤ 80 слов", "Явное завершение"]
        }
    }
}
