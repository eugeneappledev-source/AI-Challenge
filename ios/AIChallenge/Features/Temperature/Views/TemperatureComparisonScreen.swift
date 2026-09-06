import SwiftUI

struct TemperatureComparisonScreen: View {
    @State var viewModel: TemperatureComparisonViewModel
    @FocusState private var isPromptFocused: Bool

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 20) {
                introduction
                experimentRule
                promptEditor

                if viewModel.isRunning {
                    TemperatureProgressCard(
                        attempts: viewModel.attemptsByTemperature,
                        isReviewing: viewModel.isReviewing
                    )
                } else if let comparison = viewModel.comparison {
                    resultSection(comparison)
                } else {
                    readyState
                }
            }
            .padding()
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle("День 4")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar(.visible, for: .navigationBar)
        .onChange(of: viewModel.prompt) {
            viewModel.clearResult()
        }
        .alert(
            "Эксперимент не завершён",
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
            Text("ТЕМПЕРАТУРА ГЕНЕРАЦИИ")
                .font(.caption.weight(.bold))
                .foregroundStyle(Color.aiPurple)
                .tracking(1.05)

            Text("Один запрос.\nТри степени свободы.")
                .font(.largeTitle.bold())

            Text("Сравни точность, креативность и разнообразие ответов при temperature 0, 0.7 и 1.2. Затем получи общую оценку AI-рецензента.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .lineSpacing(3)
        }
        .padding(.vertical, 8)
    }

    private var experimentRule: some View {
        HStack(alignment: .top, spacing: 12) {
            Image(systemName: "thermometer.variable.and.figure")
                .font(.title2)
                .foregroundStyle(Color.aiPurple)

            VStack(alignment: .leading, spacing: 4) {
                Text("Меняется только temperature")
                    .font(.headline)
                Text("Текст запроса фиксируется на время запуска. Модель, инструкции и лимит ответа одинаковы для всех трёх вариантов.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineSpacing(2)
            }
        }
        .padding(16)
        .background(Color.aiPurple.opacity(0.08), in: RoundedRectangle(cornerRadius: 20, style: .continuous))
    }

    private var promptEditor: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack {
                Label("Общий запрос", systemImage: "text.quote")
                    .font(.headline)

                Spacer()

                if viewModel.isRunning {
                    Text("ЗАФИКСИРОВАН")
                        .font(.system(size: 9, weight: .bold))
                        .foregroundStyle(Color.aiPurple)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 5)
                        .background(Color.aiPurple.opacity(0.1), in: Capsule())
                }
            }

            TextEditor(text: $viewModel.prompt)
                .font(.system(.body, design: .serif, weight: .medium))
                .lineSpacing(4)
                .frame(minHeight: 138)
                .scrollContentBackground(.hidden)
                .padding(12)
                .background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
                .focused($isPromptFocused)
                .disabled(viewModel.isRunning)

            HStack(spacing: 10) {
                Button {
                    isPromptFocused = false
                    viewModel.useNextExample()
                } label: {
                    Label("Другой пример", systemImage: "arrow.triangle.2.circlepath")
                }
                .buttonStyle(TemperatureSecondaryButtonStyle())

                Button {
                    viewModel.clearPrompt()
                    isPromptFocused = true
                } label: {
                    Label("Очистить", systemImage: "xmark")
                }
                .buttonStyle(TemperatureSecondaryButtonStyle())
            }
            .disabled(viewModel.isRunning)

            Button {
                isPromptFocused = false
                Task { await viewModel.runComparison() }
            } label: {
                HStack(spacing: 10) {
                    Image(systemName: "thermometer.medium")
                    Text("Сравнить 3 температуры")
                        .fontWeight(.semibold)
                }
                .frame(maxWidth: .infinity)
                .frame(height: 50)
                .foregroundStyle(viewModel.canRun ? .white : Color(.tertiaryLabel))
                .background(
                    viewModel.canRun ? Color.aiPurple : Color(.quaternarySystemFill),
                    in: RoundedRectangle(cornerRadius: 15, style: .continuous)
                )
            }
            .buttonStyle(.plain)
            .disabled(!viewModel.canRun)
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: 22, style: .continuous)
                .stroke(Color.primary.opacity(0.05))
        }
    }

    private var readyState: some View {
        VStack(spacing: 13) {
            HStack(spacing: 13) {
                ForEach(TemperaturePreset.allCases) { temperature in
                    VStack(spacing: 6) {
                        Image(systemName: temperature.systemImage)
                            .font(.title2)
                        Text(temperature.formattedValue)
                            .font(.caption.bold().monospacedDigit())
                    }
                    .foregroundStyle(temperature.color)
                    .frame(maxWidth: .infinity)
                }
            }

            Text("Готово к эксперименту")
                .font(.headline)
            Text("Лучше использовать запрос, где одновременно можно оценить факты и оригинальность подачи.")
                .font(.caption)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 26)
        .padding(.horizontal, 20)
    }

    private func resultSection(_ comparison: TemperatureComparison) -> some View {
        VStack(alignment: .leading, spacing: 18) {
            HStack {
                Text("Ответы модели")
                    .font(.title2.bold())
                Spacer()
                Text("3 ИЗ 3")
                    .font(.caption2.weight(.bold))
                    .foregroundStyle(Color.aiSuccess)
            }

            TemperaturePicker(selection: $viewModel.selectedTemperature)

            if let attempt = viewModel.selectedAttempt {
                TemperatureAttemptCard(
                    attempt: attempt,
                    review: comparison.review
                )
                .id(attempt.temperature)
                .transition(.opacity.combined(with: .scale(scale: 0.98)))
            }

            TemperatureReviewCard(review: comparison.review)

            Button {
                viewModel.clearResult()
                isPromptFocused = true
            } label: {
                Label("Новый эксперимент", systemImage: "arrow.counterclockwise")
                    .fontWeight(.semibold)
                    .frame(maxWidth: .infinity)
                    .frame(height: 48)
            }
            .buttonStyle(.bordered)
            .tint(.aiPurple)
        }
        .animation(.easeInOut(duration: 0.2), value: viewModel.selectedTemperature)
    }
}

private struct TemperatureProgressCard: View {
    let attempts: [TemperaturePreset: TemperatureAttempt]
    let isReviewing: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack {
                VStack(alignment: .leading, spacing: 3) {
                    Text(isReviewing ? "Рецензент сравнивает ответы" : "DeepSeek генерирует ответы")
                        .font(.headline)
                    Text(isReviewing ? "Формируются оценки и рекомендации" : "Готово вариантов: \(attempts.count) из 3")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                ProgressView()
                    .tint(.aiPurple)
            }

            ForEach(TemperaturePreset.allCases) { temperature in
                HStack(spacing: 11) {
                    Image(systemName: attempts[temperature] == nil ? temperature.systemImage : "checkmark.circle.fill")
                        .foregroundStyle(attempts[temperature] == nil ? temperature.color : Color.aiSuccess)
                        .frame(width: 22)
                    Text("temperature = \(temperature.formattedValue)")
                        .font(.subheadline.weight(.medium).monospacedDigit())
                    Spacer()
                    Text(attempts[temperature] == nil ? "В работе" : "Готово")
                        .font(.caption2.weight(.bold))
                        .foregroundStyle(attempts[temperature] == nil ? .secondary : Color.aiSuccess)
                }
            }

            if isReviewing {
                Divider()
                Label("Четвёртый запрос: независимый разбор", systemImage: "checkmark.bubble")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(Color.aiPurple)
            }
        }
        .padding(18)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
    }
}

private struct TemperaturePicker: View {
    @Binding var selection: TemperaturePreset

    var body: some View {
        HStack(spacing: 8) {
            ForEach(TemperaturePreset.allCases) { temperature in
                Button {
                    selection = temperature
                } label: {
                    VStack(spacing: 4) {
                        Text(temperature.formattedValue)
                            .font(.subheadline.bold().monospacedDigit())
                        Text(temperature.shortTitle)
                            .font(.system(size: 9, weight: .semibold))
                    }
                    .foregroundStyle(selection == temperature ? .white : temperature.color)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .background(
                        selection == temperature ? temperature.color : temperature.color.opacity(0.1),
                        in: RoundedRectangle(cornerRadius: 13, style: .continuous)
                    )
                }
                .buttonStyle(.plain)
            }
        }
    }
}

private struct TemperatureAttemptCard: View {
    let attempt: TemperatureAttempt
    let review: TemperatureReview

    var body: some View {
        VStack(alignment: .leading, spacing: 17) {
            HStack(spacing: 12) {
                Image(systemName: attempt.temperature.systemImage)
                    .font(.headline)
                    .foregroundStyle(.white)
                    .frame(width: 44, height: 44)
                    .background(attempt.temperature.color, in: RoundedRectangle(cornerRadius: 14, style: .continuous))

                VStack(alignment: .leading, spacing: 2) {
                    Text("temperature = \(attempt.temperature.formattedValue)")
                        .font(.headline.monospacedDigit())
                    Text(attempt.temperature.description)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                Spacer()
            }

            TemperatureMarkdownText(attempt.answer)

            if let score = review.scores.first(where: { $0.temperature == attempt.temperature }) {
                Divider()
                HStack(spacing: 7) {
                    TemperatureMetric(title: "Точность", value: score.accuracy, highlighted: review.bestAccuracy == attempt.temperature)
                    TemperatureMetric(title: "Креативность", value: score.creativity, highlighted: review.bestCreativity == attempt.temperature)
                    TemperatureMetric(title: "Разнообразие", value: score.diversity, highlighted: review.bestDiversity == attempt.temperature)
                }

                Text(score.feedback)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineSpacing(2)
            }

            Divider()

            HStack {
                Label(attempt.model, systemImage: "cpu")
                Spacer()
                Text("\(attempt.usage.totalTokens) токенов")
                Text("·")
                Text(attempt.finishReason)
            }
            .font(.caption2)
            .foregroundStyle(.secondary)
        }
        .padding(18)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: 22, style: .continuous)
                .stroke(attempt.temperature.color.opacity(0.24))
        }
    }
}

private struct TemperatureMetric: View {
    let title: String
    let value: Int
    let highlighted: Bool

    var body: some View {
        VStack(spacing: 4) {
            HStack(spacing: 3) {
                if highlighted {
                    Image(systemName: "crown.fill")
                        .font(.system(size: 8))
                }
                Text("\(value)")
                    .font(.caption.weight(.bold).monospacedDigit())
            }
            Text(title)
                .font(.system(size: 8))
                .lineLimit(1)
                .minimumScaleFactor(0.75)
        }
        .foregroundStyle(highlighted ? Color.aiCoral : Color.primary)
        .frame(maxWidth: .infinity)
        .padding(.vertical, 8)
        .background(
            highlighted ? Color.aiCoral.opacity(0.1) : Color.primary.opacity(0.04),
            in: RoundedRectangle(cornerRadius: 9, style: .continuous)
        )
    }
}

private struct TemperatureReviewCard: View {
    let review: TemperatureReview

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            HStack(spacing: 13) {
                Image(systemName: "text.magnifyingglass")
                    .font(.title2)
                    .foregroundStyle(.white)
                    .frame(width: 48, height: 48)
                    .background(Color.aiPurple, in: RoundedRectangle(cornerRadius: 15, style: .continuous))

                VStack(alignment: .leading, spacing: 3) {
                    Text("Разбор AI-рецензента")
                        .font(.headline)
                    Text("Три независимых критерия")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(Color.aiPurple)
                }
            }

            TemperatureMarkdownText(review.summary)

            HStack(spacing: 8) {
                WinnerBadge(title: "ТОЧНОСТЬ", temperature: review.bestAccuracy)
                WinnerBadge(title: "КРЕАТИВ", temperature: review.bestCreativity)
                WinnerBadge(title: "РАЗНООБРАЗИЕ", temperature: review.bestDiversity)
            }

            VStack(alignment: .leading, spacing: 10) {
                Text("ЧТО ИЗМЕНИЛОСЬ")
                    .font(.caption2.weight(.bold))
                    .foregroundStyle(Color.aiPurple)
                    .tracking(0.7)

                ForEach(Array(review.differences.enumerated()), id: \.offset) { index, difference in
                    HStack(alignment: .top, spacing: 10) {
                        Text("\(index + 1)")
                            .font(.caption2.weight(.bold))
                            .foregroundStyle(.white)
                            .frame(width: 23, height: 23)
                            .background(Color.aiPurple, in: Circle())
                        Text(difference)
                            .font(.subheadline)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
            }

            VStack(alignment: .leading, spacing: 12) {
                Text("КОГДА КАКУЮ ИСПОЛЬЗОВАТЬ")
                    .font(.caption2.weight(.bold))
                    .foregroundStyle(Color.aiPurple)
                    .tracking(0.7)

                ForEach(TemperaturePreset.allCases) { temperature in
                    if let recommendation = review.recommendations.first(where: { $0.temperature == temperature }) {
                        TemperatureRecommendationCard(recommendation: recommendation)
                    }
                }
            }

            HStack {
                Label(review.model, systemImage: "cpu")
                Spacer()
                Text("\(review.usage.totalTokens) токенов")
            }
            .font(.caption2)
            .foregroundStyle(.secondary)

            Label("Один прогон показывает конкретные примеры, но не доказывает устойчивую закономерность. Для исследования эксперимент стоит повторять несколько раз.", systemImage: "info.circle")
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .padding(18)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: 22, style: .continuous)
                .stroke(Color.aiPurple.opacity(0.2))
        }
    }
}

private struct WinnerBadge: View {
    let title: String
    let temperature: TemperaturePreset

    var body: some View {
        VStack(spacing: 5) {
            Text(title)
                .font(.system(size: 7, weight: .bold))
                .minimumScaleFactor(0.7)
                .lineLimit(1)
            Text(temperature.formattedValue)
                .font(.headline.bold().monospacedDigit())
        }
        .foregroundStyle(temperature.color)
        .frame(maxWidth: .infinity)
        .padding(.vertical, 10)
        .background(temperature.color.opacity(0.09), in: RoundedRectangle(cornerRadius: 11, style: .continuous))
    }
}

private struct TemperatureRecommendationCard: View {
    let recommendation: TemperatureRecommendation

    var body: some View {
        VStack(alignment: .leading, spacing: 9) {
            HStack {
                Label(
                    "temperature = \(recommendation.temperature.formattedValue)",
                    systemImage: recommendation.temperature.systemImage
                )
                .font(.subheadline.weight(.bold).monospacedDigit())
                .foregroundStyle(recommendation.temperature.color)
                Spacer()
                Text(recommendation.temperature.shortTitle.uppercased())
                    .font(.system(size: 8, weight: .bold))
                    .foregroundStyle(recommendation.temperature.color)
            }

            ForEach(recommendation.bestFor, id: \.self) { useCase in
                Label(useCase, systemImage: "checkmark")
                    .font(.caption)
            }

            Label(recommendation.caution, systemImage: "exclamationmark.triangle")
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .padding(14)
        .background(
            recommendation.temperature.color.opacity(0.07),
            in: RoundedRectangle(cornerRadius: 14, style: .continuous)
        )
    }
}

private struct TemperatureMarkdownText: View {
    private let text: String

    init(_ text: String) {
        self.text = text
    }

    var body: some View {
        if let attributed = try? AttributedString(markdown: text) {
            Text(attributed)
                .font(.body)
                .lineSpacing(4)
                .textSelection(.enabled)
                .frame(maxWidth: .infinity, alignment: .leading)
        } else {
            Text(text)
                .font(.body)
                .lineSpacing(4)
                .textSelection(.enabled)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}

private struct TemperatureSecondaryButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.caption.weight(.semibold))
            .foregroundStyle(Color.aiPurple)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 10)
            .background(Color.aiPurple.opacity(configuration.isPressed ? 0.15 : 0.08), in: RoundedRectangle(cornerRadius: 11, style: .continuous))
    }
}

private extension TemperaturePreset {
    var formattedValue: String {
        switch self {
        case .precise: "0"
        case .balanced: "0.7"
        case .creative: "1.2"
        }
    }

    var shortTitle: String {
        switch self {
        case .precise: "Фокус"
        case .balanced: "Баланс"
        case .creative: "Свобода"
        }
    }

    var description: String {
        switch self {
        case .precise: "Минимальная случайность и более стабильный выбор слов"
        case .balanced: "Компромисс между предсказуемостью и вариативностью"
        case .creative: "Больше неожиданных формулировок и идей"
        }
    }

    var systemImage: String {
        switch self {
        case .precise: "scope"
        case .balanced: "scale.3d"
        case .creative: "sparkles"
        }
    }

    var color: Color {
        switch self {
        case .precise: .aiBlue
        case .balanced: .aiForest
        case .creative: .aiCoral
        }
    }
}
