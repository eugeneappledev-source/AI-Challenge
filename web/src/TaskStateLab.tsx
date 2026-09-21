import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import { APIError, actOnTask, createTask, deleteTask, loadTaskState } from "./api";
import type { TaskAction, TaskExchange, TaskPhase, TaskState } from "./types";

const defaultGoal = "Подготовить ТЗ и архитектурный план экрана недельной статистики PulsePlan для iOS 17+ с offline-first и поддержкой VoiceOver.";
const phases: Array<{ id: TaskPhase; index: string; label: string }> = [
  { id: "planning", index: "01", label: "Planning" },
  { id: "execution", index: "02", label: "Execution" },
  { id: "validation", index: "03", label: "Validation" },
  { id: "done", index: "04", label: "Done" },
];

function TaskStateLab() {
  const userId = useMemo(getUserId, []);
  const taskId = `${userId}:pulse-plan`;
  const [goal, setGoal] = useState(defaultGoal);
  const [state, setState] = useState<TaskState | null>(null);
  const [exchange, setExchange] = useState<TaskExchange | null>(null);
  const [isBooting, setIsBooting] = useState(true);
  const [pending, setPending] = useState<TaskAction | "create" | "reset" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const abortController = useRef<AbortController | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    loadTaskState(taskId, controller.signal)
      .then((loaded) => {
        setState(loaded);
        if (loaded) setGoal(loaded.goal);
      })
      .catch((caught) => {
        if (!(caught instanceof DOMException && caught.name === "AbortError")) setError(errorMessage(caught));
      })
      .finally(() => setIsBooting(false));
    return () => controller.abort();
  }, [taskId]);

  async function startTask(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalized = goal.trim();
    if (!normalized || pending) return;
    await run("create", (signal) => createTask(taskId, userId, "engineer", normalized, signal));
  }

  async function perform(action: TaskAction) {
    if (pending) return;
    await run(action, (signal) => actOnTask(taskId, action, signal));
  }

  async function run(kind: TaskAction | "create", operation: (signal: AbortSignal) => Promise<TaskExchange>) {
    abortController.current?.abort();
    const controller = new AbortController();
    abortController.current = controller;
    setPending(kind);
    setError(null);
    try {
      const result = await operation(controller.signal);
      setExchange(result);
      setState(result.state);
    } catch (caught) {
      if (!(caught instanceof DOMException && caught.name === "AbortError")) setError(errorMessage(caught));
    } finally {
      if (abortController.current === controller) setPending(null);
    }
  }

  async function reset() {
    if (pending) return;
    setPending("reset");
    setError(null);
    try {
      await deleteTask(taskId);
      setState(null);
      setExchange(null);
      setGoal(defaultGoal);
    } catch (caught) {
      setError(errorMessage(caught));
    } finally {
      setPending(null);
    }
  }

  const displayedAnswer = exchange?.answer || state?.artifacts.at(-1)?.content || "";
  const isPaused = state?.status === "paused";
  const isDone = state?.phase === "done";

  return (
    <main className="page-shell task-page">
      <nav className="topbar" aria-label="Навигация">
        <a className="brand" href="#day-13"><span className="brand-mark">AI</span><span>Challenge</span></a>
        <div className="day-switcher" aria-label="Выбор задания">
          <a href="#day-02">День 2</a><a href="#day-11">День 11</a><a href="#day-12">День 12</a><a className="selected" href="#day-13">День 13</a><a href="#day-14">День 14</a><a href="#day-15">День 15</a>
        </div>
        <a className="github-link" href="https://github.com/eugeneappledev-source/AI-Challenge" target="_blank" rel="noreferrer">GitHub <span>↗</span></a>
      </nav>

      <section className="task-hero">
        <div>
          <p className="eyebrow">День 13 · Task State Machine</p>
          <h1>Задача помнит,<br /><em>где остановилась.</em></h1>
          <p className="hero-description">Этап, текущий шаг и ожидаемое действие формализованы в конечном автомате и переживают паузу, обновление страницы и перезапуск агента.</p>
        </div>
        <aside className="task-machine-mini" aria-label="Схема автомата">
          {phases.map((phase, index) => <div key={phase.id}><span>{phase.index}</span><strong>{phase.label}</strong>{index < phases.length - 1 && <i>→</i>}</div>)}
        </aside>
      </section>

      <section className="task-workspace">
        <div className="section-heading">
          <div><p className="eyebrow">Жизненный цикл задачи</p><h2>Управление состоянием</h2></div>
          <p>Переход выбирает backend. Модель получает сохранённое состояние и выполняет только текущий этап.</p>
        </div>

        {isBooting ? <TaskLoading text="Загружаю состояние задачи из SQLite…" /> : !state ? (
          <form className="task-create-card" onSubmit={startTask}>
            <header><div><span>Новая задача</span><h3>Опишите конечную цель</h3></div><b>Профиль: Senior iOS Engineer</b></header>
            <textarea value={goal} onChange={(event) => setGoal(event.target.value)} rows={5} maxLength={4000} disabled={pending !== null} />
            <footer><p>После запуска агент сам сформирует planning-артефакт и сохранит формальное состояние.</p><button type="submit" disabled={!goal.trim() || pending !== null}>{pending === "create" ? <><span className="spinner" /> Формирую план…</> : <>Создать задачу <span>→</span></>}</button></footer>
          </form>
        ) : (
          <>
            <TaskTimeline state={state} />

            <div className="task-control-grid">
              <article className="task-state-card">
                <header><div><span className={`task-status-dot ${state.status}`} /><div><small>Состояние</small><strong>{isPaused ? "На паузе" : isDone ? "Завершена" : "В работе"}</strong></div></div><b>revision {state.revision}</b></header>
                <div className="task-goal"><small>Цель задачи</small><p>{state.goal}</p></div>
                <dl>
                  <div><dt>Этап</dt><dd>{phaseLabel(state.phase)}</dd></div>
                  <div><dt>Текущий шаг</dt><dd>{state.currentStep}</dd></div>
                  <div className="expected"><dt>Ожидаемое действие</dt><dd>{state.expectedAction}</dd></div>
                </dl>
                <div className="task-actions">
                  {!isDone && !isPaused && <button className="primary" type="button" onClick={() => perform("advance")} disabled={pending !== null}>{pending === "advance" ? <><span className="spinner" /> Выполняю этап…</> : <>{advanceLabel(state.phase)} <span>→</span></>}</button>}
                  {!isDone && !isPaused && <button type="button" onClick={() => perform("pause")} disabled={pending !== null}>Поставить на паузу</button>}
                  {isPaused && <button className="resume" type="button" onClick={() => perform("resume")} disabled={pending !== null}>{pending === "resume" ? <><span className="spinner" /> Восстанавливаю…</> : <>Продолжить без объяснений <span>↗</span></>}</button>}
                  <button className="reset" type="button" onClick={reset} disabled={pending !== null}>{pending === "reset" ? "Сбрасываю…" : "Новая задача"}</button>
                </div>
              </article>

              <aside className="task-proof-card">
                <p className="eyebrow">Что сохраняется</p>
                <ul><li><span>01</span><div><strong>Phase</strong><small>{state.phase}</small></div></li><li><span>02</span><div><strong>Current step</strong><small>{state.currentStep}</small></div></li><li><span>03</span><div><strong>Expected action</strong><small>{state.expectedAction}</small></div></li><li><span>04</span><div><strong>Artifacts</strong><small>{state.artifacts.length} сохранено</small></div></li></ul>
                <p className="persistence-note"><span>SQLite</span> Обновите страницу на паузе: состояние останется тем же.</p>
              </aside>
            </div>

            {pending && pending !== "reset" && <TaskLoading text={loadingText(pending)} />}

            {displayedAnswer && !pending && <section className="task-output">
              <header><div><span className="agent-avatar">C</span><div><strong>{exchange?.answer ? "Ответ текущей команды" : "Последний сохранённый артефакт"}</strong><small>Compass · {phaseLabel(state.phase)}</small></div></div>{exchange?.usage.totalTokens ? <b>{exchange.usage.totalTokens} токенов</b> : <b>из SQLite</b>}</header>
              <div className="markdown-answer"><ReactMarkdown>{displayedAnswer}</ReactMarkdown></div>
              {exchange?.trace && <details className="task-trace"><summary>Как агент обработал переход</summary><ol>{exchange.trace.map((item) => <li key={item}>{item}</li>)}</ol></details>}
            </section>}

            <section className="task-history">
              <div><p className="eyebrow">Журнал автомата</p><h2>Переходы и артефакты</h2></div>
              <div className="task-history-grid">
                <article><h3>История переходов</h3><ol>{[...state.transitions].reverse().map((transition, index) => <li key={`${transition.createdAt}-${index}`}><span>{transition.action}</span><div><strong>{transition.from ? `${transition.from} → ${transition.to}` : transition.to}</strong><p>{transition.summary}</p></div></li>)}</ol></article>
                <article><h3>Артефакты этапов</h3><div className="task-artifacts">{state.artifacts.map((artifact) => <details key={`${artifact.phase}-${artifact.createdAt}`}><summary><span>{phaseShort(artifact.phase)}</span><div><strong>{artifact.title}</strong><small>{phaseLabel(artifact.phase)}</small></div><i>⌄</i></summary><div className="markdown-answer"><ReactMarkdown>{artifact.content}</ReactMarkdown></div></details>)}</div></article>
              </div>
            </section>
          </>
        )}

        {error && <div className="error-card" role="alert"><span>!</span><div><strong>Операция не выполнена</strong><p>{error}</p></div></div>}
      </section>
      <footer><p>AI Advent Challenge #9 · День 13</p><p>Finite State Machine · SQLite · DeepSeek</p></footer>
    </main>
  );
}

function TaskTimeline({ state }: { state: TaskState }) {
  const current = phases.findIndex((phase) => phase.id === state.phase);
  return <div className={`task-timeline ${state.status}`}><div className="task-timeline-line" />{phases.map((phase, index) => <div className={`task-phase ${index < current || state.phase === "done" ? "complete" : ""} ${index === current ? "current" : ""}`} key={phase.id}><span>{index < current || state.phase === "done" ? "✓" : phase.index}</span><div><strong>{phase.label}</strong><small>{index === current ? (state.status === "paused" ? "пауза" : "текущий этап") : index < current ? "завершён" : "ожидает"}</small></div></div>)}</div>;
}

function TaskLoading({ text }: { text: string }) { return <div className="task-loading" aria-live="polite"><span className="spinner" /><div><strong>{text}</strong><p>Состояние изменится только после успешного ответа backend.</p></div></div>; }
function getUserId(): string { const key = "ai-challenge-memory-user"; let value = localStorage.getItem(key); if (!value) { value = crypto.randomUUID(); localStorage.setItem(key, value); } return value; }
function phaseLabel(phase: TaskPhase): string { return ({ planning: "Planning · планирование", execution: "Execution · выполнение", validation: "Validation · проверка", done: "Done · завершено" })[phase]; }
function phaseShort(phase: TaskPhase): string { return ({ planning: "PL", execution: "EX", validation: "VA", done: "OK" })[phase]; }
function advanceLabel(phase: TaskPhase): string { return phase === "planning" ? "Перейти к выполнению" : phase === "execution" ? "Перейти к валидации" : "Завершить задачу"; }
function loadingText(action: TaskAction | "create"): string { return action === "pause" ? "Сохраняю точку остановки…" : action === "resume" ? "Восстанавливаю цель, этап и артефакты…" : action === "create" ? "Создаю задачу…" : "Агент выполняет следующий этап…"; }
function errorMessage(error: unknown): string { return error instanceof APIError || error instanceof Error ? error.message : "Произошла неизвестная ошибка."; }

export default TaskStateLab;
