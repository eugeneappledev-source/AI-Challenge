import Foundation
import Observation

@MainActor
@Observable
final class ContextStrategiesViewModel {
    let windowSizeOptions = [4, 6, 10]
    var selectedStrategy: ContextStrategy = .slidingWindow
    var selectedWindowSize = 6
    var activeBranchID = "main"
    var input = ""
    private(set) var states: [ContextStrategy: ContextStrategyState] = [:]
    private(set) var comparison: ContextStrategyComparison?
    private(set) var isLoading = false
    private(set) var progressText = ""
    var errorMessage: String?

    let sessionID: String
    private let useCase: ManageContextStrategiesUseCase

    init(useCase: ManageContextStrategiesUseCase, sessionID: String) {
        self.useCase = useCase
        self.sessionID = sessionID
    }

    var currentState: ContextStrategyState? { states[selectedStrategy] }
    var canSend: Bool { !input.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isLoading }
    var canCreateCheckpoint: Bool {
        selectedStrategy == .branching && activeBranchID == "main" && !(currentState?.messages.isEmpty ?? true) && !isLoading
    }

    func loadSelected() async {
        guard !isLoading else { return }
        await load(strategy: selectedStrategy, branchID: selectedStrategy == .branching ? activeBranchID : nil)
    }

    func selectBranch(_ branchID: String) async {
        activeBranchID = branchID
        await load(strategy: .branching, branchID: branchID)
    }

    func windowSizeDidChange() async {
        comparison = nil
        await loadSelected()
    }

    func send() async {
        let value = input.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !value.isEmpty, !isLoading else { return }
        input = ""
        isLoading = true
        progressText = selectedStrategy == .stickyFacts ? "Обновляю facts и спрашиваю агента…" : "Агент применяет \(selectedStrategy.fullTitle)…"
        errorMessage = nil
        defer { isLoading = false; progressText = "" }
        do {
            let branchID = selectedStrategy == .branching ? activeBranchID : nil
            let result = try await useCase.send(
                message: value,
                sessionID: sessionID,
                strategy: selectedStrategy,
                branchID: branchID,
                windowSize: selectedWindowSize
            )
            states[selectedStrategy] = result.state
        } catch {
            input = value
            show(error)
        }
    }

    func createCheckpoint() async {
        guard canCreateCheckpoint else { return }
        isLoading = true
        progressText = "Создаю checkpoint и две независимые ветки…"
        errorMessage = nil
        defer { isLoading = false; progressText = "" }
        do {
            activeBranchID = "mvp"
            states[.branching] = try await useCase.createBranches(sessionID: sessionID)
        } catch { show(error) }
    }

    func runComparison() async {
        guard !isLoading else { return }
        isLoading = true
        progressText = "Прогоняю 12 сообщений при N = \(selectedWindowSize)…"
        errorMessage = nil
        comparison = nil
        defer { isLoading = false; progressText = "" }
        do {
            comparison = try await useCase.compare(sessionID: sessionID, windowSize: selectedWindowSize)
            states[.slidingWindow] = try await useCase.state(sessionID: sessionID, strategy: .slidingWindow, windowSize: selectedWindowSize)
            states[.stickyFacts] = try await useCase.state(sessionID: sessionID, strategy: .stickyFacts, windowSize: selectedWindowSize)
            activeBranchID = "mvp"
            states[.branching] = try await useCase.state(
                sessionID: sessionID,
                strategy: .branching,
                branchID: activeBranchID,
                windowSize: selectedWindowSize
            )
        } catch { show(error) }
    }

    func clear() async {
        guard !isLoading else { return }
        isLoading = true
        progressText = "Очищаю лабораторию…"
        errorMessage = nil
        defer { isLoading = false; progressText = "" }
        do {
            try await useCase.clear(sessionID: sessionID)
            states = [:]
            comparison = nil
            activeBranchID = "main"
            await load(strategy: selectedStrategy, branchID: selectedStrategy == .branching ? activeBranchID : nil)
        } catch { show(error) }
    }

    private func load(strategy: ContextStrategy, branchID: String? = nil) async {
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }
        do {
            states[strategy] = try await useCase.state(
                sessionID: sessionID,
                strategy: strategy,
                branchID: branchID,
                windowSize: selectedWindowSize
            )
        }
        catch { show(error) }
    }

    private func show(_ error: Error) {
        errorMessage = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
    }
}
