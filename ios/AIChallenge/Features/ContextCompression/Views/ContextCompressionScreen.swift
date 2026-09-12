import SwiftUI

struct ContextCompressionScreen: View {
    @State var viewModel: ContextCompressionViewModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                header
                if let state = viewModel.state { contextDiagram(state) }
                controls
                if !viewModel.messages.isEmpty { recentConversation }
                comparisonControls
                if let comparison = viewModel.comparison { comparisonResult(comparison) }
            }.padding()
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle("День 9")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar { ToolbarItem(placement: .topBarTrailing) { Button("Очистить", role: .destructive) { Task { await viewModel.clear() } }.disabled(viewModel.isLoading) } }
        .task { await viewModel.load() }
        .alert("Context Lab", isPresented: Binding(get: { viewModel.errorMessage != nil }, set: { if !$0 { viewModel.errorMessage = nil } })) {
            Button("Понятно", role: .cancel) {}
        } message: { Text(viewModel.errorMessage ?? "Неизвестная ошибка") }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("CONTEXT LAB").font(.caption.bold()).tracking(1).foregroundStyle(Color.aiForest)
            Text("Помнить смысл, а не всё.").font(.largeTitle.bold())
            Text("Полная история остаётся в SQLite. Старые сообщения сворачиваются пакетами по 10, а минимум 6 последних сохраняются дословно.")
                .font(.subheadline).foregroundStyle(.secondary).lineSpacing(3)
        }.padding(.vertical, 8)
    }

    private func contextDiagram(_ state: ContextState) -> some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack {
                Label(state.compressionActive ? "Сжатие активно" : "Нужно 10 старых + 6 последних", systemImage: state.compressionActive ? "archivebox.fill" : "hourglass")
                    .font(.headline).foregroundStyle(state.compressionActive ? Color.aiSuccess : Color.aiCoral)
                Spacer(); Text("\(state.fullHistoryMessages) msg").font(.caption.bold().monospaced())
            }
            HStack(spacing: 8) {
                contextBlock("SUMMARY", "\(state.summaryCoveredMessages)", Color.aiForest)
                Image(systemName: "plus").foregroundStyle(.secondary)
                contextBlock("RECENT", "\(state.recentMessages)", Color.aiBlue)
                Image(systemName: "arrow.right").foregroundStyle(.secondary)
                contextBlock("LLM", "\(state.compressedEstimatedTokens)", Color.aiPurple)
            }
            if state.compressionActive {
                VStack(alignment: .leading, spacing: 5) {
                    Text("СОХРАНЁННАЯ СВОДКА").font(.system(size: 9, weight: .bold)).foregroundStyle(.secondary)
                    Text(state.summary).font(.caption).lineSpacing(3).textSelection(.enabled)
                }
                Divider()
                HStack {
                    Text("Полный ≈ \(state.fullEstimatedTokens)")
                    Spacer()
                    Text("Сжатый ≈ \(state.compressedEstimatedTokens)")
                    Spacer()
                    Text(estimatedDelta(state)).foregroundStyle(state.estimatedSavedTokens >= 0 ? Color.aiSuccess : Color.aiCoral)
                }
                    .font(.caption.bold().monospaced())
            }
        }.padding(16).background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 22))
    }

    private var controls: some View {
        VStack(alignment: .leading, spacing: 11) {
            TextField("Новое сообщение в сжатый диалог…", text: $viewModel.input, axis: .vertical)
                .lineLimit(2...5).padding(13).background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 14))
            HStack {
                Button { Task { await viewModel.buildDemoContext() } } label: { Label("Демо-контекст", systemImage: "wand.and.stars") }
                    .disabled(viewModel.isLoading)
                Spacer()
                Button { Task { await viewModel.send() } } label: { Label("Отправить", systemImage: "paperplane.fill") }
                    .buttonStyle(.borderedProminent).tint(Color.aiForest).disabled(!viewModel.canSend)
            }.font(.caption.bold())
            if viewModel.isLoading { HStack { ProgressView(); Text(viewModel.progressText).font(.caption).foregroundStyle(.secondary) } }
        }.padding(16).background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 20))
    }

    private var recentConversation: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Архив диалога").font(.headline)
            ForEach(viewModel.messages.suffix(6)) { message in
                VStack(alignment: .leading, spacing: 3) {
                    Text(message.role == .user ? "ВЫ" : "COMPASS").font(.system(size: 9, weight: .bold)).foregroundStyle(message.role == .user ? Color.aiCoral : Color.aiForest)
                    Text(message.content).font(.caption).lineLimit(4)
                }.frame(maxWidth: .infinity, alignment: .leading).padding(11)
                    .background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 13))
            }
        }
    }

    private var comparisonControls: some View {
        VStack(alignment: .leading, spacing: 10) {
            Label("Честное A/B-сравнение", systemImage: "arrow.left.arrow.right").font(.headline)
            Text("Один вопрос, одна модель и настройки. Отличается только контекст: полная история против summary + recent.")
                .font(.caption).foregroundStyle(.secondary)
            TextField("Контрольный вопрос", text: $viewModel.comparisonQuestion, axis: .vertical)
                .lineLimit(2...4).padding(12).background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 13))
            Button { Task { await viewModel.compare() } } label: { Label("Сравнить и получить AI-фидбек", systemImage: "scale.3d") .frame(maxWidth: .infinity) }
                .buttonStyle(.borderedProminent).tint(Color.aiPurple).disabled(!viewModel.canCompare)
            if viewModel.state?.compressionActive != true { Text("Сравнение откроется после первого summary. Нажмите «Демо-контекст» или продолжите разговор.").font(.caption2).foregroundStyle(Color.aiCoral) }
        }.padding(16).background(Color.aiPurple.opacity(0.07), in: RoundedRectangle(cornerRadius: 20))
    }

    private func comparisonResult(_ result: ContextComparison) -> some View {
        VStack(alignment: .leading, spacing: 13) {
            Text("Результат сравнения").font(.title3.bold())
            answer("Без сжатия", result.full, Color.aiCoral)
            answer("Со сжатием", result.compressed, Color.aiForest)
            HStack {
                Label(result.promptTokensSaved >= 0 ? "Сэкономлено" : "Контекст пока дороже", systemImage: result.promptTokensSaved >= 0 ? "leaf.fill" : "exclamationmark.triangle.fill")
                Spacer()
                Text("\(abs(result.promptTokensSaved)) токенов · \(abs(result.savingsPercent).formatted(.number.precision(.fractionLength(1))))%")
            }
            .font(.subheadline.bold())
            .foregroundStyle(result.promptTokensSaved >= 0 ? Color.aiSuccess : Color.aiCoral)
            .padding(13)
            .background((result.promptTokensSaved >= 0 ? Color.aiSuccess : Color.aiCoral).opacity(0.09), in: RoundedRectangle(cornerRadius: 14))
            VStack(alignment: .leading, spacing: 8) {
                HStack { Label("Независимый рецензент", systemImage: "checkmark.seal.fill").font(.headline); Spacer(); Text(result.review.qualityPreserved ? "СМЫСЛ СОХРАНЁН" : "ЕСТЬ ПОТЕРИ").font(.system(size: 9, weight: .bold)).foregroundStyle(result.review.qualityPreserved ? Color.aiSuccess : Color.red) }
                Text(result.review.verdict).font(.subheadline)
                ForEach(result.review.differences, id: \.self) { Text("• " + $0).font(.caption).foregroundStyle(.secondary) }
                Text(result.review.recommendation).font(.caption.bold()).foregroundStyle(Color.aiPurple)
            }.padding(15).background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 18))
        }
    }

    private func answer(_ title: String, _ answer: ContextAnswer, _ color: Color) -> some View {
        VStack(alignment: .leading, spacing: 7) {
            HStack { Text(title).font(.subheadline.bold()).foregroundStyle(color); Spacer(); Text("prompt \(answer.usage.promptTokens)").font(.caption2.monospaced()).foregroundStyle(.secondary) }
            Text(answer.answer).font(.caption).lineSpacing(3).textSelection(.enabled)
        }.padding(14).background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 16))
    }

    private func contextBlock(_ title: String, _ value: String, _ color: Color) -> some View {
        VStack(spacing: 3) { Text(title).font(.system(size: 8, weight: .bold)); Text(value).font(.headline.monospaced()) }
            .foregroundStyle(color).frame(maxWidth: .infinity).padding(.vertical, 10).background(color.opacity(0.09), in: RoundedRectangle(cornerRadius: 12))
    }

    private func estimatedDelta(_ state: ContextState) -> String {
        state.estimatedSavedTokens >= 0 ? "−\(state.estimatedSavedTokens)" : "+\(abs(state.estimatedSavedTokens))"
    }
}
