import SwiftUI

struct ModelComparisonScreen: View {
    @State var viewModel: ModelComparisonViewModel
    @FocusState private var isPromptFocused: Bool

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 20) {
                introduction
                experimentRule
                modelLegend
                promptEditor

                if viewModel.isRunning {
                    ModelProgressCard(attempts: viewModel.attemptsByTier, isReviewing: viewModel.isReviewing)
                } else if let comparison = viewModel.comparison {
                    resultSection(comparison)
                } else {
                    readyState
                }
            }
            .padding()
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle("День 5")
        .navigationBarTitleDisplayMode(.inline)
        .onChange(of: viewModel.prompt) { viewModel.clearResult() }
        .alert(
            "Сравнение не завершено",
            isPresented: Binding(
                get: { viewModel.errorMessage != nil },
                set: { if !$0 { viewModel.errorMessage = nil } }
            )
        ) { Button("Понятно", role: .cancel) {} } message: {
            Text(viewModel.errorMessage ?? "Неизвестная ошибка")
        }
    }

    private var introduction: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("ВЕРСИИ МОДЕЛЕЙ")
                .font(.caption.weight(.bold))
                .foregroundStyle(Color.aiBlue)
                .tracking(1.05)
            Text("Один запрос.\nТри модели.")
                .font(.largeTitle.bold())
            Text("Сравни реальное время ответа, токены, расчётную стоимость и качество. Затем получи независимый разбор результатов.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .lineSpacing(3)
        }
        .padding(.vertical, 8)
    }

    private var experimentRule: some View {
        HStack(alignment: .top, spacing: 12) {
            Image(systemName: "equal.circle.fill")
                .font(.title2)
                .foregroundStyle(Color.aiBlue)
            VStack(alignment: .leading, spacing: 4) {
                Text("Меняется только модель")
                    .font(.headline)
                Text("Запрос, system prompt, temperature = 0, thinking off и лимит ответа одинаковы. Запуски идут параллельно.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineSpacing(2)
            }
        }
        .padding(16)
        .background(Color.aiBlue.opacity(0.08), in: RoundedRectangle(cornerRadius: 20, style: .continuous))
    }

    private var modelLegend: some View {
        VStack(alignment: .leading, spacing: 12) {
            Label("Профили эксперимента", systemImage: "cpu")
                .font(.headline)
            ForEach(ModelTier.allCases) { tier in
                HStack(spacing: 11) {
                    Image(systemName: tier.systemImage)
                        .foregroundStyle(tier.color)
                        .frame(width: 25)
                    VStack(alignment: .leading, spacing: 2) {
                        Text("\(tier.title) · \(tier.modelName)")
                            .font(.subheadline.weight(.semibold))
                        Text(tier.legend)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
            }
            Label("Это учебные профили, а не официальные уровни интеллекта DeepSeek. Vision Exp — экспериментальная мультимодальная Flash-модель.", systemImage: "info.circle")
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 20, style: .continuous))
    }

    private var promptEditor: some View {
        VStack(alignment: .leading, spacing: 14) {
            Label("Одинаковый запрос", systemImage: "text.quote")
                .font(.headline)
            TextEditor(text: $viewModel.prompt)
                .font(.system(.body, design: .serif, weight: .medium))
                .lineSpacing(4)
                .frame(minHeight: 145)
                .scrollContentBackground(.hidden)
                .padding(12)
                .background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
                .focused($isPromptFocused)
                .disabled(viewModel.isRunning)

            HStack(spacing: 10) {
                Button { isPromptFocused = false; viewModel.useNextExample() } label: {
                    Label("Другой пример", systemImage: "arrow.triangle.2.circlepath")
                }
                .buttonStyle(ModelSecondaryButtonStyle())
                Button { viewModel.clearPrompt(); isPromptFocused = true } label: {
                    Label("Очистить", systemImage: "xmark")
                }
                .buttonStyle(ModelSecondaryButtonStyle())
            }
            .disabled(viewModel.isRunning)

            Button { isPromptFocused = false; Task { await viewModel.runComparison() } } label: {
                Label("Сравнить 3 модели", systemImage: "point.3.connected.trianglepath.dotted")
                    .fontWeight(.semibold)
                    .frame(maxWidth: .infinity)
                    .frame(height: 50)
                    .foregroundStyle(viewModel.canRun ? .white : Color(.tertiaryLabel))
                    .background(
                        viewModel.canRun ? Color.aiBlue : Color(.quaternarySystemFill),
                        in: RoundedRectangle(cornerRadius: 15, style: .continuous)
                    )
            }
            .buttonStyle(.plain)
            .disabled(!viewModel.canRun)
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
    }

    private var readyState: some View {
        VStack(spacing: 13) {
            HStack(spacing: 12) {
                ForEach(ModelTier.allCases) { tier in
                    VStack(spacing: 7) {
                        Image(systemName: tier.systemImage).font(.title2)
                        Text(tier.title).font(.caption.bold())
                    }
                    .foregroundStyle(tier.color)
                    .frame(maxWidth: .infinity)
                }
            }
            Text("Готово к сравнению").font(.headline)
            Text("Лучше выбирать запрос с проверяемыми ограничениями и достаточной сложностью.")
                .font(.caption).foregroundStyle(.secondary).multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 25)
    }

    private func resultSection(_ comparison: ModelComparison) -> some View {
        VStack(alignment: .leading, spacing: 18) {
            HStack {
                Text("Результаты").font(.title2.bold())
                Spacer()
                Text("3 ИЗ 3").font(.caption2.bold()).foregroundStyle(Color.aiSuccess)
            }
            ModelTierPicker(selection: $viewModel.selectedTier)
            if let attempt = viewModel.selectedAttempt {
                ModelAttemptCard(attempt: attempt, review: comparison.review)
                    .id(attempt.tier)
            }
            ModelReviewCard(review: comparison.review)
            methodology
            Button { viewModel.clearResult(); isPromptFocused = true } label: {
                Label("Новое сравнение", systemImage: "arrow.counterclockwise")
                    .fontWeight(.semibold).frame(maxWidth: .infinity).frame(height: 48)
            }
            .buttonStyle(.bordered)
            .tint(.aiBlue)
        }
        .animation(.easeInOut(duration: 0.2), value: viewModel.selectedTier)
    }

    private var methodology: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("КАК СЧИТАЮТСЯ МЕТРИКИ").font(.caption2.bold()).foregroundStyle(Color.aiBlue)
            Label("Время — от отправки до получения ответа провайдера.", systemImage: "timer")
            Label("Токены — поле usage из API; стоимость — расчёт по тарифу и cache hit/miss.", systemImage: "dollarsign.circle")
            Label("GPU и RAM провайдер не раскрывает, поэтому ресурсоёмкость сравнивается через время, токены и стоимость.", systemImage: "memorychip")
            Divider()
            Link("Официальные цены DeepSeek", destination: URL(string: "https://api-docs.deepseek.com/quick_start/pricing")!)
            Link("Официальный список моделей", destination: URL(string: "https://api-docs.deepseek.com/api/list-models")!)
        }
        .font(.caption)
        .foregroundStyle(.secondary)
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 20, style: .continuous))
    }
}

private struct ModelProgressCard: View {
    let attempts: [ModelTier: ModelBenchmarkAttempt]
    let isReviewing: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 15) {
            HStack {
                VStack(alignment: .leading, spacing: 3) {
                    Text(isReviewing ? "Рецензент изучает ответы" : "Модели отвечают параллельно").font(.headline)
                    Text(isReviewing ? "Названия и метрики от него скрыты" : "Готово: \(attempts.count) из 3").font(.caption).foregroundStyle(.secondary)
                }
                Spacer(); ProgressView().tint(.aiBlue)
            }
            ForEach(ModelTier.allCases) { tier in
                HStack {
                    Image(systemName: attempts[tier] == nil ? tier.systemImage : "checkmark.circle.fill")
                        .foregroundStyle(attempts[tier] == nil ? tier.color : Color.aiSuccess).frame(width: 24)
                    Text(tier.modelName).font(.subheadline.weight(.medium))
                    Spacer()
                    Text(attempts[tier] == nil ? "В работе" : "Готово").font(.caption2.bold()).foregroundStyle(.secondary)
                }
            }
            if isReviewing {
                Divider()
                Label("Четвёртый запрос: независимый feedback", systemImage: "checkmark.bubble")
                    .font(.caption.weight(.semibold)).foregroundStyle(Color.aiBlue)
            }
        }
        .padding(18)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
    }
}

private struct ModelTierPicker: View {
    @Binding var selection: ModelTier
    var body: some View {
        HStack(spacing: 8) {
            ForEach(ModelTier.allCases) { tier in
                Button { selection = tier } label: {
                    VStack(spacing: 4) {
                        Image(systemName: tier.systemImage)
                        Text(tier.title).font(.caption2.bold())
                    }
                    .foregroundStyle(selection == tier ? .white : tier.color)
                    .frame(maxWidth: .infinity).padding(.vertical, 10)
                    .background(selection == tier ? tier.color : tier.color.opacity(0.1), in: RoundedRectangle(cornerRadius: 13))
                }.buttonStyle(.plain)
            }
        }
    }
}

private struct ModelAttemptCard: View {
    let attempt: ModelBenchmarkAttempt
    let review: ModelBenchmarkReview

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack(spacing: 12) {
                Image(systemName: attempt.tier.systemImage).foregroundStyle(.white)
                    .frame(width: 44, height: 44).background(attempt.tier.color, in: RoundedRectangle(cornerRadius: 14))
                VStack(alignment: .leading, spacing: 2) {
                    Text(attempt.tier.title).font(.headline)
                    Text(attempt.model).font(.caption.monospaced()).foregroundStyle(.secondary)
                }
                Spacer()
                if review.qualityWinner == attempt.tier { Image(systemName: "crown.fill").foregroundStyle(Color.aiCoral) }
            }
            ModelMarkdownText(attempt.answer)
            HStack(spacing: 8) {
                ModelMetric(title: "Время", value: attempt.formattedLatency, highlighted: review.fastest == attempt.tier)
                ModelMetric(title: "Токены", value: "\(attempt.usage.totalTokens)", highlighted: false)
                ModelMetric(title: "Цена", value: attempt.formattedCost, highlighted: review.cheapest == attempt.tier)
            }
            if let score = review.scores.first(where: { $0.tier == attempt.tier }) {
                Divider()
                HStack(spacing: 8) {
                    ModelScore(title: "Точность", value: score.accuracy)
                    ModelScore(title: "Полнота", value: score.completeness)
                    ModelScore(title: "Ясность", value: score.clarity)
                }
                Text(score.feedback).font(.caption).foregroundStyle(.secondary)
            }
            DisclosureGroup("Детали расчёта") {
                VStack(alignment: .leading, spacing: 5) {
                    Text("Вход: \(attempt.usage.promptTokens) · выход: \(attempt.usage.completionTokens)")
                    Text("Cache hit: \(attempt.usage.promptCacheHitTokens) · miss: \(attempt.usage.promptCacheMissTokens)")
                    Text("Тарифный период: \(attempt.pricingPeriod == "peak" ? "peak" : "off-peak") · finish: \(attempt.finishReason)")
                }.font(.caption2).foregroundStyle(.secondary).padding(.top, 6)
            }.font(.caption.weight(.semibold))
        }
        .padding(18)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22))
        .overlay { RoundedRectangle(cornerRadius: 22).stroke(attempt.tier.color.opacity(0.25)) }
    }
}

private struct ModelMetric: View {
    let title: String
    let value: String
    let highlighted: Bool
    var body: some View {
        VStack(spacing: 4) {
            HStack(spacing: 3) {
                if highlighted { Image(systemName: "trophy.fill").font(.system(size: 8)) }
                Text(value).font(.caption.bold().monospacedDigit()).minimumScaleFactor(0.7)
            }
            Text(title).font(.system(size: 9))
        }
        .foregroundStyle(highlighted ? Color.aiSuccess : Color.primary)
        .frame(maxWidth: .infinity).padding(.vertical, 9)
        .background((highlighted ? Color.aiSuccess : Color.primary).opacity(0.07), in: RoundedRectangle(cornerRadius: 10))
    }
}

private struct ModelScore: View {
    let title: String
    let value: Int
    var body: some View {
        VStack(spacing: 3) { Text("\(value)/10").font(.caption.bold().monospacedDigit()); Text(title).font(.system(size: 9)) }
            .frame(maxWidth: .infinity).padding(.vertical, 8).background(Color.aiBlue.opacity(0.07), in: RoundedRectangle(cornerRadius: 9))
    }
}

private struct ModelReviewCard: View {
    let review: ModelBenchmarkReview
    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            HStack(spacing: 12) {
                Image(systemName: "text.magnifyingglass").font(.title2).foregroundStyle(.white)
                    .frame(width: 48, height: 48).background(Color.aiBlue, in: RoundedRectangle(cornerRadius: 15))
                VStack(alignment: .leading, spacing: 3) {
                    Text("Разбор AI-рецензента").font(.headline)
                    Text("Содержание оценено вслепую").font(.caption.weight(.semibold)).foregroundStyle(Color.aiBlue)
                }
            }
            ModelMarkdownText(review.summary)
            HStack(spacing: 8) {
                ModelWinnerBadge(title: "КАЧЕСТВО", tier: review.qualityWinner)
                ModelWinnerBadge(title: "СКОРОСТЬ", tier: review.fastest)
                ModelWinnerBadge(title: "ЦЕНА", tier: review.cheapest)
            }
            VStack(alignment: .leading, spacing: 10) {
                Text("НАБЛЮДАЕМЫЕ ОТЛИЧИЯ").font(.caption2.bold()).foregroundStyle(Color.aiBlue)
                ForEach(Array(review.differences.enumerated()), id: \.offset) { index, item in
                    HStack(alignment: .top, spacing: 9) {
                        Text("\(index + 1)").font(.caption2.bold()).foregroundStyle(.white).frame(width: 22, height: 22).background(Color.aiBlue, in: Circle())
                        Text(item).font(.subheadline)
                    }
                }
            }
            VStack(alignment: .leading, spacing: 10) {
                Text("КОГДА КАКУЮ ИСПОЛЬЗОВАТЬ").font(.caption2.bold()).foregroundStyle(Color.aiBlue)
                ForEach(ModelTier.allCases) { tier in
                    if let item = review.recommendations.first(where: { $0.tier == tier }) {
                        VStack(alignment: .leading, spacing: 7) {
                            Text(tier.title).font(.subheadline.bold()).foregroundStyle(tier.color)
                            Text(item.bestFor.joined(separator: " · ")).font(.caption)
                            Label(item.tradeoff, systemImage: "arrow.left.arrow.right").font(.caption2).foregroundStyle(.secondary)
                        }
                        .padding(12).background(tier.color.opacity(0.07), in: RoundedRectangle(cornerRadius: 13))
                    }
                }
            }
            Label("Один прогон — пример, а не абсолютный рейтинг. Для устойчивого вывода нужны повторения и набор разных задач.", systemImage: "info.circle")
                .font(.caption2).foregroundStyle(.secondary)
            HStack { Label(review.reviewerModel, systemImage: "cpu"); Spacer(); Text("\(review.reviewerUsage.totalTokens) токенов") }
                .font(.caption2).foregroundStyle(.secondary)
        }
        .padding(18)
        .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22))
        .overlay { RoundedRectangle(cornerRadius: 22).stroke(Color.aiBlue.opacity(0.2)) }
    }
}

private struct ModelWinnerBadge: View {
    let title: String
    let tier: ModelTier
    var body: some View {
        VStack(spacing: 4) { Text(title).font(.system(size: 7, weight: .bold)); Text(tier.title).font(.caption.bold()) }
            .foregroundStyle(tier.color).frame(maxWidth: .infinity).padding(.vertical, 10)
            .background(tier.color.opacity(0.09), in: RoundedRectangle(cornerRadius: 11))
    }
}

private struct ModelMarkdownText: View {
    let text: String
    init(_ text: String) { self.text = text }
    var body: some View {
        if let value = try? AttributedString(markdown: text) {
            Text(value).font(.body).lineSpacing(4).textSelection(.enabled).frame(maxWidth: .infinity, alignment: .leading)
        } else { Text(text).font(.body).lineSpacing(4).textSelection(.enabled) }
    }
}

private struct ModelSecondaryButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label.font(.caption.weight(.semibold)).foregroundStyle(Color.aiBlue)
            .frame(maxWidth: .infinity).padding(.vertical, 10)
            .background(Color.aiBlue.opacity(configuration.isPressed ? 0.15 : 0.08), in: RoundedRectangle(cornerRadius: 11))
    }
}

private extension ModelBenchmarkAttempt {
    var formattedLatency: String {
        latencyMilliseconds < 1000 ? "\(latencyMilliseconds) мс" : String(format: "%.2f c", Double(latencyMilliseconds) / 1000)
    }
    var formattedCost: String { String(format: "$%.6f", estimatedCostUSD) }
}

private extension ModelTier {
    var title: String {
        switch self { case .basic: "Базовая"; case .extended: "Расширенная"; case .strong: "Сильная" }
    }
    var modelName: String {
        switch self { case .basic: "V4 Flash"; case .extended: "V4 Flash Vision Exp"; case .strong: "V4 Pro" }
    }
    var legend: String {
        switch self {
        case .basic: "Быстрый и экономичный текстовый профиль"
        case .extended: "Экспериментальный Flash с мультимодальными возможностями"
        case .strong: "Более дорогой Pro-профиль для сложных задач"
        }
    }
    var systemImage: String {
        switch self { case .basic: "hare.fill"; case .extended: "eye.fill"; case .strong: "brain.fill" }
    }
    var color: Color {
        switch self { case .basic: .aiForest; case .extended: .aiPurple; case .strong: .aiCoral }
    }
}
