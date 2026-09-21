import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import {
  APIError,
  actOnTask,
  createTask,
  deleteTask,
  loadTaskLifecycleGraph,
  loadTaskState,
  transitionTask,
} from "./api";
import type {
  ControlledTransitionExchange,
  TaskAction,
  TaskLifecycleGraph,
  TaskPhase,
  TaskState,
} from "./types";

const defaultGoal = "Спроектировать и проверить экран недельной статистики PulsePlan для iOS 17+, сохранив offline-first и поддержку VoiceOver.";
const phaseOrder: TaskPhase[] = ["planning", "execution", "validation", "done"];
const phaseMeta: Record<TaskPhase, { index: string; title: string; description: string }> = {
  planning: { index: "01", title: "Planning", description: "план и критерии" },
  execution: { index: "02", title: "Execution", description: "реализация решения" },
  validation: { index: "03", title: "Validation", description: "проверка результата" },
  done: { index: "04", title: "Done", description: "принятый финал" },
};

function ControlledLifecycleLab() {
  const userId = useMemo(getUserId, []);
  const taskId = `${userId}:controlled-lifecycle`;
  const [graph, setGraph] = useState<TaskLifecycleGraph | null>(null);
  const [state, setState] = useState<TaskState | null>(null);
  const [goal, setGoal] = useState(defaultGoal);
  const [rollbackReason, setRollbackReason] = useState("Вернуться на этап назад и исправить замечание проверки.");
  const [exchange, setExchange] = useState<ControlledTransitionExchange | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [isBooting, setIsBooting] = useState(true);
  const [pending, setPending] = useState<"create" | "reset" | "pause" | "resume" | TaskPhase | null>(null);
  const [error, setError] = useState<string | null>(null);
  const abortController = useRef<AbortController | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    Promise.all([loadTaskLifecycleGraph(controller.signal), loadTaskState(taskId, controller.signal)])
      .then(([loadedGraph, loadedState]) => {
        setGraph(loadedGraph);
        setState(loadedState);
        if (loadedState) setGoal(loadedState.goal);
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
    setPending("create");
    setError(null);
    try {
      const result = await createTask(taskId, userId, "engineer", normalized);
      setState(result.state);
      setExchange(null);
      setNotice("Planning создан: обязательный план сохранён, теперь gateway может разрешить только соседний этап Execution.");
    } catch (caught) {
      setError(errorMessage(caught));
    } finally {
      setPending(null);
    }
  }

  async function requestTransition(target: TaskPhase, reason = "") {
    if (!state || pending) return;
    abortController.current?.abort();
    const controller = new AbortController();
    abortController.current = controller;
    setPending(target);
    setError(null);
    setNotice(null);
    try {
      const result = await transitionTask(taskId, target, reason, controller.signal);
      setExchange(result);
      setState(result.state);
    } catch (caught) {
      if (!(caught instanceof DOMException && caught.name === "AbortError")) setError(errorMessage(caught));
    } finally {
      if (abortController.current === controller) setPending(null);
    }
  }

  async function performAction(action: Extract<TaskAction, "pause" | "resume">) {
    if (!state || pending) return;
    setPending(action);
    setError(null);
    setExchange(null);
    try {
      const result = await actOnTask(taskId, action);
      setState(result.state);
      setNotice(action === "pause" ? "Пауза сохранена. Попробуйте выполнить переход: gateway отклонит его и не потеряет этап." : "Контекст восстановлен. Теперь разрешённый переход снова доступен без повторного описания задачи.");
    } catch (caught) {
      setError(errorMessage(caught));
    } finally {
      setPending(null);
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
      setNotice(null);
      setGoal(defaultGoal);
    } catch (caught) {
      setError(errorMessage(caught));
    } finally {
      setPending(null);
    }
  }

  const currentRules = graph && state ? graph.rules.filter((rule) => rule.from === state.phase) : [];
  const forwardRule = currentRules?.find((rule) => rule.direction === "forward");
  const rollbackRule = currentRules?.find((rule) => rule.direction === "rollback");
  const redTarget = state ? blockedTarget(state, forwardRule?.to) : "validation";

  return (
    <main className="page-shell lifecycle-page">
      <nav className="topbar" aria-label="Навигация">
        <a className="brand" href="#day-15"><span className="brand-mark">AI</span><span>Challenge</span></a>
        <div className="day-switcher" aria-label="Выбор задания">
          <a href="#day-02">День 2</a><a href="#day-11">День 11</a><a href="#day-12">День 12</a><a href="#day-13">День 13</a><a href="#day-14">День 14</a><a className="selected" href="#day-15">День 15</a>
        </div>
        <a className="github-link" href="https://github.com/eugeneappledev-source/AI-Challenge" target="_blank" rel="noreferrer">GitHub <span>↗</span></a>
      </nav>

      <section className="lifecycle-hero">
        <div>
          <p className="eyebrow">День 15 · Controlled Lifecycle</p>
          <h1>Happy path.<br /><em>И красный путь.</em></h1>
          <p className="hero-description">Модель больше не может выбрать состояние сама. Каждый переход проходит через явный граф, проверку предусловий и единый backend-gateway.</p>
        </div>
        <aside className="route-legend">
          <div><span className="green-line" /><strong>Разрешённое ребро</strong><small>проверка → LLM → сохранение</small></div>
          <div><span className="red-line" /><strong>Сход с маршрута</strong><small>отказ → 0 токенов → этап сохранён</small></div>
          <div><span className="blue-line" /><strong>Rollback</strong><small>только назад по графу и с причиной</small></div>
        </aside>
      </section>

      <section className="lifecycle-workspace">
        <div className="section-heading">
          <div><p className="eyebrow">Явный граф состояний</p><h2>Маршрут нельзя перепрыгнуть</h2></div>
          <p>Зелёные переходы ведут вперёд, синие возвращают на доработку. Любое другое ребро отсутствует в таблице и блокируется до LLM.</p>
        </div>

        <LifecycleGraph graph={graph} state={state} />

        {isBooting ? <LifecycleLoading text="Загружаю граф и состояние из SQLite…" /> : !state ? (
          <form className="lifecycle-create" onSubmit={startTask}>
            <header><div><span>Новая контролируемая задача</span><h3>Начинаем строго с Planning</h3></div><b>единая точка входа</b></header>
            <textarea value={goal} onChange={(event) => setGoal(event.target.value)} rows={5} maxLength={4000} disabled={pending !== null} />
            <footer><p>Compass создаст planning-артефакт. Следующее состояние выберет не модель, а transition gateway.</p><button type="submit" disabled={!goal.trim() || pending !== null}>{pending === "create" ? <><span className="spinner" /> Создаю Planning…</> : <>Создать задачу <span>→</span></>}</button></footer>
          </form>
        ) : (
          <>
            <section className="lifecycle-console">
              <article className="lifecycle-state">
                <header><div><span className={`lifecycle-status ${state.status}`} /><div><small>Текущее состояние</small><h3>{phaseMeta[state.phase].title}</h3></div></div><b>{state.status === "paused" ? "PAUSED" : `REVISION ${state.revision}`}</b></header>
                <div className="lifecycle-state-body">
                  <div><small>Текущий шаг</small><p>{state.currentStep}</p></div>
                  <div><small>Ожидаемое действие</small><p>{state.expectedAction}</p></div>
                  <div className="allowed-targets"><small>Разрешённые цели графа</small><p>{currentRules.length ? currentRules.map((rule) => `${rule.to} (${rule.direction})`).join(" · ") : "нет"}</p></div>
                </div>
                <footer><button type="button" onClick={reset} disabled={pending !== null}>{pending === "reset" ? "Сбрасываю…" : "Начать заново"}</button>{state.status === "active" && state.phase !== "done" ? <button type="button" onClick={() => performAction("pause")} disabled={pending !== null}>Пауза</button> : state.status === "paused" ? <button className="resume" type="button" onClick={() => performAction("resume")} disabled={pending !== null}>{pending === "resume" ? <><span className="spinner" /> Восстанавливаю…</> : "Возобновить"}</button> : null}</footer>
              </article>

              <div className="route-actions">
                {forwardRule && state.status === "active" && <article className="route-action allowed"><span>Разрешённый путь</span><h3>{state.phase} → {forwardRule.to}</h3><p>{forwardRule.requirement}</p><button type="button" onClick={() => requestTransition(forwardRule.to, "Предыдущий этап принят")} disabled={pending !== null}>{pending === forwardRule.to ? <><i className="spinner" /> Выполняю переход…</> : <>Перейти в {forwardRule.to} <b>→</b></>}</button></article>}
                <article className="route-action rejected"><span>{state.status === "paused" ? "Проверка паузы" : "Попытка схода"}</span><h3>{state.phase} → {redTarget}</h3><p>{state.status === "paused" ? "Даже разрешённое ребро заблокировано, пока задача на паузе." : "Такого ребра нет: этап будет сохранён, модель не вызовется."}</p><button type="button" onClick={() => requestTransition(redTarget, "Пользователь просит перескочить этап")} disabled={pending !== null}>{pending === redTarget ? <><i className="spinner" /> Проверяю gateway…</> : <>Попробовать запрещённый переход <b>×</b></>}</button></article>
              </div>
            </section>

            {rollbackRule && state.status === "active" && <section className="rollback-card">
              <div><span>Контролируемый rollback</span><h3>{state.phase} → {rollbackRule.to}</h3><p>{rollbackRule.requirement}. Без причины gateway отклонит откат.</p></div>
              <label><span>Причина отката</span><input value={rollbackReason} onChange={(event) => setRollbackReason(event.target.value)} disabled={pending !== null} /></label>
              <button type="button" onClick={() => requestTransition(rollbackRule.to, rollbackReason)} disabled={!rollbackReason.trim() || pending !== null}>{pending === rollbackRule.to ? <><i className="spinner" /> Возвращаю…</> : <>Вернуть в {rollbackRule.to} <b>↙</b></>}</button>
            </section>}

            {notice && <div className="lifecycle-notice"><span>i</span><p>{notice}</p></div>}
            {pending && pending !== "reset" && pending !== "pause" && pending !== "resume" && <LifecycleLoading text="Gateway проверяет ребро и предусловия…" />}
            {exchange && <TransitionResult exchange={exchange} />}

            <AuditLog state={state} />
          </>
        )}

        {error && <div className="error-card" role="alert"><span>!</span><div><strong>Операция не выполнена</strong><p>{error}</p></div></div>}
      </section>
      <footer><p>AI Advent Challenge #9 · День 15</p><p>Transition Graph · Guarded FSM · SQLite</p></footer>
    </main>
  );
}

function LifecycleGraph({ graph, state }: { graph: TaskLifecycleGraph | null; state: TaskState | null }) {
  return <div className="lifecycle-graph">{phaseOrder.map((phase, index) => {
    const meta = phaseMeta[phase];
    const current = state?.phase === phase;
    const completed = state ? phaseOrder.indexOf(state.phase) > index : false;
    return <div className="graph-node-wrap" key={phase}><article className={`graph-node ${current ? "current" : ""} ${completed ? "completed" : ""}`}><span>{completed ? "✓" : meta.index}</span><div><strong>{meta.title}</strong><small>{current ? state?.status === "paused" ? "пауза" : "текущее" : meta.description}</small></div></article>{index < phaseOrder.length - 1 && <div className="graph-edge"><i>→</i><small>{graph?.rules.find((rule) => rule.from === phase && rule.to === phaseOrder[index + 1]) ? "allowed" : "blocked"}</small><b>←</b></div>}</div>;
  })}</div>;
}

function TransitionResult({ exchange }: { exchange: ControlledTransitionExchange }) {
  return <section className={`transition-result ${exchange.allowed ? "allowed" : "rejected"}`}>
    <header><span>{exchange.allowed ? "✓" : "×"}</span><div><small>{exchange.allowed ? "Переход выполнен" : "Переход заблокирован"}</small><h2>{exchange.requestedFrom} → {exchange.requestedTo}</h2></div><b>{exchange.allowed ? "STATE CHANGED" : "STATE UNCHANGED"}</b></header>
    <div className="transition-explanation"><div><small>Решение gateway</small><p>{exchange.reason}</p></div><dl><div><dt>Код</dt><dd>{exchange.code}</dd></div><div><dt>Токены</dt><dd>{exchange.usage.totalTokens}</dd></div><div><dt>Finish</dt><dd>{exchange.finishReason || "blocked"}</dd></div></dl></div>
    <ol className="transition-trace">{exchange.trace.map((step, index) => <li key={`${index}-${step}`}><span>{String(index + 1).padStart(2, "0")}</span><p>{step}</p></li>)}</ol>
    {exchange.answer && <article className="transition-answer"><header><strong>Результат разрешённого этапа</strong><small>Compass получил состояние только после одобрения gateway</small></header><div className="markdown-answer"><ReactMarkdown>{exchange.answer}</ReactMarkdown></div></article>}
  </section>;
}

function AuditLog({ state }: { state: TaskState }) {
  const attempts = state.attempts ?? [];
  return <section className="lifecycle-audit"><div><p className="eyebrow">Persistence и аудит</p><h2>Каждая попытка оставляет след</h2></div><div className="audit-grid"><article><h3>Попытки переходов <span>{attempts.length}</span></h3>{attempts.length ? <ol>{[...attempts].reverse().map((attempt, index) => <li className={attempt.allowed ? "allowed" : "rejected"} key={`${attempt.createdAt}-${index}`}><span>{attempt.allowed ? "✓" : "×"}</span><div><strong>{attempt.from} → {attempt.to}</strong><p>{attempt.reason}</p></div><b>{attempt.code}</b></li>)}</ol> : <p className="empty-audit">Сделайте разрешённую или запрещённую попытку.</p>}</article><article><h3>Артефакты этапов <span>{state.artifacts.length}</span></h3><div className="audit-artifacts">{state.artifacts.map((artifact, index) => <details key={`${artifact.phase}-${artifact.createdAt}-${index}`}><summary><span>{phaseMeta[artifact.phase].index}</span><div><strong>{artifact.title}</strong><small>{artifact.phase}</small></div><i>⌄</i></summary><div className="markdown-answer"><ReactMarkdown>{artifact.content}</ReactMarkdown></div></details>)}</div></article></div></section>;
}

function LifecycleLoading({ text }: { text: string }) { return <div className="lifecycle-loading" aria-live="polite"><span className="spinner" /><div><strong>{text}</strong><p>Состояние изменится только после успешной проверки и ответа backend.</p></div></div>; }
function blockedTarget(state: TaskState, forwardTarget?: TaskPhase): TaskPhase { if (state.status === "paused" && forwardTarget) return forwardTarget; return ({ planning: "validation", execution: "done", validation: "planning", done: "planning" } as Record<TaskPhase, TaskPhase>)[state.phase]; }
function getUserId(): string { const key = "ai-challenge-memory-user"; let value = localStorage.getItem(key); if (!value) { value = crypto.randomUUID(); localStorage.setItem(key, value); } return value; }
function errorMessage(error: unknown): string { return error instanceof APIError || error instanceof Error ? error.message : "Произошла неизвестная ошибка."; }

export default ControlledLifecycleLab;
