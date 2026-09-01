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
                .foregroundStyle(Color.aiCoral)
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
        VStack(alignment: .leading, spacing: 0) {
            VStack(alignment: .leading, spacing: 14) {
                HStack(alignment: .center, spacing: 12) {
                    Circle()
                        .fill(reply.mode.tint)
                        .frame(width: 10, height: 10)
                        .padding(5)
                        .background(Color(.tertiarySystemGroupedBackground), in: Circle())

                    VStack(alignment: .leading, spacing: 2) {
                        Text(reply.mode == .controlled ? "Контролируемый ответ" : "Свободный ответ")
                            .font(.subheadline.weight(.bold))
                        Text(reply.mode == .controlled ? "JSON · до 80 слов" : "Естественный текст · без лимита")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }

                    Spacer()

                    Image(systemName: reply.mode.systemImage)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(reply.mode.tint)
                }

                ScrollView(.horizontal) {
                    HStack(spacing: 8) {
                        ForEach(reply.mode.badges, id: \.self) { badge in
                            Text(badge)
                                .font(.caption2.weight(.semibold))
                                .padding(.horizontal, 10)
                                .padding(.vertical, 6)
                                .foregroundStyle(reply.mode.tint)
                                .background(reply.mode.tint.opacity(0.11), in: Capsule())
                        }
                    }
                }
                .scrollIndicators(.hidden)
            }
            .padding(18)

            Divider()

            VStack(alignment: .leading, spacing: 18) {
                VStack(alignment: .leading, spacing: 6) {
                    Text("ИСХОДНЫЙ ЗАПРОС")
                        .font(.caption2.weight(.bold))
                        .foregroundStyle(Color.aiCoral)
                        .tracking(0.8)
                    Text(prompt)
                        .font(.subheadline.weight(.medium))
                }

                Divider()

                if let controlledAnswer = reply.controlledAnswer {
                    ControlledAnswerView(answer: controlledAnswer)
                } else {
                    Text(reply.answer)
                        .font(.body)
                        .lineSpacing(4)
                        .textSelection(.enabled)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }

                if reply.mode == .controlled {
                    RawModelAnswerView(json: reply.answer)
                }

                Divider()

                HStack(spacing: 0) {
                    MetadataItem(title: "ОТВЕТ", value: "\(reply.completionTokens)")
                    Divider().frame(height: 34)
                    MetadataItem(title: "ВСЕГО", value: "\(reply.totalTokens)")
                    Divider().frame(height: 34)
                    MetadataItem(title: "ФИНИШ", value: reply.finishReason ?? "—")
                }

                HStack(spacing: 6) {
                    Image(systemName: "cpu")
                    Text(reply.model)
                }
                .font(.caption2.weight(.medium))
                .foregroundStyle(.secondary)
            }
            .padding(18)
        }
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 22, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: 22, style: .continuous)
                .stroke(Color.primary.opacity(0.06), lineWidth: 1)
        }
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
        VStack(alignment: .leading, spacing: 24) {
            HStack(spacing: 7) {
                Circle()
                    .fill(statusColor)
                    .frame(width: 6, height: 6)
                Text(answer.status == .ok ? "JSON РАСПАРСЕН" : "ВОПРОС ВНЕ ТЕМЫ")
                    .font(.caption2.weight(.bold))
                    .tracking(0.6)
            }
            .foregroundStyle(statusColor)
            .padding(.horizontal, 10)
            .padding(.vertical, 7)
            .background(statusColor.opacity(0.12), in: Capsule())

            Text(answer.answer)
                .font(.system(.title3, design: .serif, weight: .semibold))
                .italic()
                .lineSpacing(4)
                .textSelection(.enabled)

            if !answer.ingredients.isEmpty {
                RecipeSectionTitle(number: "01", title: "Ингредиенты")

                LazyVGrid(
                    columns: [GridItem(.adaptive(minimum: 118), spacing: 8)],
                    alignment: .leading,
                    spacing: 8
                ) {
                    ForEach(Array(answer.ingredients.enumerated()), id: \.offset) { _, ingredient in
                        HStack(spacing: 7) {
                            Circle()
                                .fill(Color.aiCoral)
                                .frame(width: 5, height: 5)
                            Text(ingredient)
                                .font(.caption.weight(.medium))
                                .lineLimit(2)
                        }
                        .frame(maxWidth: .infinity, minHeight: 34, alignment: .leading)
                        .padding(.horizontal, 11)
                        .padding(.vertical, 4)
                        .background(Color.aiForest.opacity(0.08), in: RoundedRectangle(cornerRadius: 10, style: .continuous))
                    }
                }
            }

            if !answer.steps.isEmpty {
                RecipeSectionTitle(number: "02", title: "Приготовление")

                VStack(alignment: .leading, spacing: 14) {
                    ForEach(Array(answer.steps.enumerated()), id: \.offset) { index, step in
                        HStack(alignment: .top, spacing: 12) {
                            Text("\(index + 1)")
                                .font(.caption2.weight(.bold))
                                .foregroundStyle(.white)
                                .frame(width: 28, height: 28)
                                .background(Color.aiForest, in: Circle())

                            Text(step)
                                .font(.subheadline)
                                .lineSpacing(3)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(.top, 4)
                        }
                    }
                }
            }
        }
    }

    private var statusColor: Color {
        answer.status == .ok ? .aiSuccess : .aiCoral
    }
}

private struct RecipeSectionTitle: View {
    let number: String
    let title: String

    var body: some View {
        HStack(spacing: 9) {
            Text(number)
                .font(.caption2.weight(.bold))
                .foregroundStyle(Color.aiCoral)
            Text(title)
                .font(.subheadline.weight(.bold))
            Spacer()
        }
        .padding(.top, 4)
    }
}

private struct RawModelAnswerView: View {
    let json: String
    @State private var isExpanded = false

    var body: some View {
        DisclosureGroup(isExpanded: $isExpanded) {
            VStack(alignment: .leading, spacing: 10) {
                Text("Приложение распарсило этот JSON и собрало карточку выше.")
                    .font(.caption)
                    .foregroundStyle(.secondary)

                ScrollView(.horizontal) {
                    Text(prettyJSON)
                        .font(.system(.caption2, design: .monospaced))
                        .foregroundStyle(Color(red: 0.82, green: 0.91, blue: 0.84))
                        .textSelection(.enabled)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
                .scrollIndicators(.hidden)
                .padding(14)
                .background(Color(red: 0.08, green: 0.12, blue: 0.09), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
            }
            .padding(.top, 12)
        } label: {
            HStack(spacing: 8) {
                Text("P.S.")
                    .foregroundStyle(Color.aiCoral)
                Label("Исходный JSON модели", systemImage: "curlybraces")
            }
            .font(.subheadline.weight(.semibold))
        }
        .tint(.primary)
        .padding(14)
        .background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
    }

    private var prettyJSON: String {
        guard
            let data = json.data(using: .utf8),
            let object = try? JSONSerialization.jsonObject(with: data),
            let formatted = try? JSONSerialization.data(withJSONObject: object, options: [.prettyPrinted, .sortedKeys]),
            let result = String(data: formatted, encoding: .utf8)
        else {
            return json
        }
        return result
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
                .font(.caption.weight(.bold))
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
        case .unrestricted: .aiCoral
        case .controlled: .aiForest
        }
    }

    var badges: [String] {
        switch self {
        case .unrestricted:
            ["Тема: еда", "Свободный формат", "Без лимита"]
        case .controlled:
            ["Тема: еда", "JSON", "≤ 80 слов", "≤ 8 ингредиентов", "≤ 4 шагов", "Без обрыва"]
        }
    }
}

extension Color {
    static let aiForest = Color(red: 23 / 255, green: 61 / 255, blue: 43 / 255)
    static let aiCoral = Color(red: 242 / 255, green: 110 / 255, blue: 63 / 255)
    static let aiSuccess = Color(red: 47 / 255, green: 122 / 255, blue: 76 / 255)
}
