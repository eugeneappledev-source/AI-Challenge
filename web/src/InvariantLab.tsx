import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import { APIError, checkInvariants, loadInvariants } from "./api";
import type {
  Invariant,
  InvariantAssessment,
  InvariantCategory,
  InvariantExchange,
} from "./types";

const examples = [
  {
    label: "Совместимый запрос",
    kind: "allow",
    text: "Предложи структуру модулей для экрана недельной статистики, сохранив offline-first и поддержку VoiceOver.",
  },
  {
    label: "Смысловой конфликт",
    kind: "semantic",
    text: "Сделай так, чтобы недельная статистика загружалась только при наличии интернета, а без сети показывалась ошибка.",
  },
  {
    label: "Прямое нарушение",
    kind: "hard",
    text: "Перепиши экран на UIKit и RxSwift, VoiceOver можно убрать.",
  },
] as const;

const categories: Record<InvariantCategory, { code: string; label: string }> = {
  architecture: { code: "AR", label: "Архитектура" },
  technical_decision: { code: "TD", label: "Техническое решение" },
  stack: { code: "ST", label: "Стек" },
  business_rule: { code: "BR", label: "Бизнес-правило" },
};

function InvariantLab() {
  const userId = useMemo(getUserId, []);
  const taskId = `${userId}:pulse-plan`;
  const [invariants, setInvariants] = useState<Invariant[]>([]);
  const [request, setRequest] = useState<string>(examples[0].text);
  const [exchange, setExchange] = useState<InvariantExchange | null>(null);
  const [isBooting, setIsBooting] = useState(true);
  const [isChecking, setIsChecking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const abortController = useRef<AbortController | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    loadInvariants(taskId, controller.signal)
      .then(setInvariants)
      .catch((caught) => {
        if (!(caught instanceof DOMException && caught.name === "AbortError")) setError(errorMessage(caught));
      })
      .finally(() => setIsBooting(false));
    return () => controller.abort();
  }, [taskId]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalized = request.trim();
    if (!normalized || isChecking) return;

    abortController.current?.abort();
    const controller = new AbortController();
    abortController.current = controller;
    setIsChecking(true);
    setError(null);
    setExchange(null);
    try {
      const result = await checkInvariants(taskId, userId, "engineer", normalized, controller.signal);
      setExchange(result);
      setInvariants(result.invariants);
    } catch (caught) {
      if (!(caught instanceof DOMException && caught.name === "AbortError")) setError(errorMessage(caught));
    } finally {
      if (abortController.current === controller) setIsChecking(false);
    }
  }

  return (
    <main className="page-shell invariant-page">
      <nav className="topbar" aria-label="Навигация">
        <a className="brand" href="#day-14"><span className="brand-mark">AI</span><span>Challenge</span></a>
        <div className="day-switcher" aria-label="Выбор задания">
          <a href="#day-02">День 2</a><a href="#day-11">День 11</a><a href="#day-12">День 12</a><a href="#day-13">День 13</a><a className="selected" href="#day-14">День 14</a>
        </div>
        <a className="github-link" href="https://github.com/eugeneappledev-source/AI-Challenge" target="_blank" rel="noreferrer">GitHub <span>↗</span></a>
      </nav>

      <section className="invariant-hero">
        <div>
          <p className="eyebrow">День 14 · Invariant Guard</p>
          <h1>Свобода решений.<br /><em>Нерушимые границы.</em></h1>
          <p className="hero-description">Compass хранит обязательные правила отдельно от диалога, проверяет каждый запрос и не запускает основной ответ, если решение ломает архитектуру, стек или бизнес-ограничения.</p>
        </div>
        <aside className="guard-flow" aria-label="Поток проверки инвариантов">
          <div><span>01</span><strong>Hard check</strong><small>явные запрещённые решения</small></div>
          <i>→</i>
          <div><span>02</span><strong>Semantic guard</strong><small>смысловые противоречия</small></div>
          <i>→</i>
          <div><span>03</span><strong>Main agent</strong><small>только разрешённый запрос</small></div>
        </aside>
      </section>

      <section className="invariant-workspace">
        <div className="section-heading">
          <div><p className="eyebrow">Отдельно от диалога</p><h2>Реестр обязательных правил</h2></div>
          <p>Эти ограничения загружаются из SQLite и добавляются к каждому запросу независимо от истории сообщений.</p>
        </div>

        {isBooting ? <GuardLoading text="Загружаю реестр инвариантов из SQLite…" /> : (
          <div className="invariant-registry">
            {invariants.map((invariant) => <InvariantCard key={invariant.id} invariant={invariant} />)}
          </div>
        )}

        <section className="guard-experiment">
          <div className="guard-experiment-heading">
            <div><p className="eyebrow">Проверка конфликта</p><h2>Попробуйте нарушить правило</h2></div>
            <p>Для наглядности есть три сценария: разрешённый, смысловой конфликт и нарушение, которое можно поймать без LLM.</p>
          </div>

          <div className="guard-examples">
            {examples.map((example) => (
              <button key={example.kind} className={example.kind} type="button" onClick={() => setRequest(example.text)} disabled={isChecking}>
                <span>{example.kind === "allow" ? "✓" : example.kind === "semantic" ? "≈" : "!"}</span>
                <div><strong>{example.label}</strong><small>{example.kind === "allow" ? "дойдёт до агента" : example.kind === "semantic" ? "проверит отдельная LLM" : "остановит код"}</small></div>
              </button>
            ))}
          </div>

          <form className="guard-composer" onSubmit={submit}>
            <label htmlFor="invariant-request">Запрос пользователя</label>
            <textarea id="invariant-request" value={request} onChange={(event) => setRequest(event.target.value)} rows={5} maxLength={4000} disabled={isChecking} />
            <footer>
              <span>{request.length} / 4000</span>
              <button type="submit" disabled={!request.trim() || isChecking}>
                {isChecking ? <><span className="spinner" /> Guard проверяет правила…</> : <>Проверить и отправить <span>→</span></>}
              </button>
            </footer>
          </form>
        </section>

        {isChecking && <GuardLoading text="Сначала hard check, затем независимый смысловой Guard…" />}
        {error && <div className="error-card" role="alert"><span>!</span><div><strong>Проверка не выполнена</strong><p>{error}</p></div></div>}
        {exchange && <GuardResult exchange={exchange} />}
      </section>

      <footer><p>AI Advent Challenge #9 · День 14</p><p>SQLite · Hybrid Guard · DeepSeek</p></footer>
    </main>
  );
}

function InvariantCard({ invariant }: { invariant: Invariant }) {
  const category = categories[invariant.category];
  return (
    <article className="invariant-card">
      <header><span>{category.code}</span><div><small>{category.label}</small><h3>{invariant.title}</h3></div><b className={invariant.protection}>{invariant.protection === "hard_check" ? "CODE + LLM" : "LLM GUARD"}</b></header>
      <p>{invariant.rule}</p>
      <footer><span>Зачем</span><p>{invariant.rationale}</p></footer>
    </article>
  );
}

function GuardResult({ exchange }: { exchange: InvariantExchange }) {
  const allowed = exchange.verdict === "allowed";
  const assessmentByID = new Map(exchange.assessments.map((item) => [item.invariantId, item]));
  return (
    <section className={`guard-result ${exchange.verdict}`} aria-live="polite">
      <header>
        <div className="guard-verdict-icon">{allowed ? "✓" : "×"}</div>
        <div><p>{allowed ? "Запрос разрешён" : "Запрос отклонён"}</p><h2>{exchange.explanation}</h2></div>
        <span>{allowed ? "MAIN AGENT CALLED" : "MAIN AGENT BLOCKED"}</span>
      </header>

      <div className="guard-result-grid">
        <article className="guard-audit">
          <h3>Проверка каждого инварианта</h3>
          <ol>{exchange.invariants.map((invariant) => <AssessmentRow key={invariant.id} invariant={invariant} assessment={assessmentByID.get(invariant.id)} />)}</ol>
        </article>
        <aside className="guard-meta">
          <h3>Маршрут запроса</h3>
          <ol>{exchange.trace.map((step, index) => <li key={`${index}-${step}`}><span>{String(index + 1).padStart(2, "0")}</span><p>{step}</p></li>)}</ol>
          <dl><div><dt>Модель</dt><dd>{exchange.model || "не вызвана"}</dd></div><div><dt>Завершение</dt><dd>{exchange.finishReason || (allowed ? "—" : "blocked")}</dd></div><div><dt>Всего токенов</dt><dd>{exchange.usage.totalTokens}</dd></div></dl>
        </aside>
      </div>

      {exchange.conflicts.length > 0 && <div className="guard-conflicts">
        {exchange.conflicts.map((conflict) => <article key={`${conflict.invariantId}-${conflict.detectedBy}`}><span>{conflict.detectedBy === "hard_check" ? "CODE" : "LLM"}</span><div><strong>{conflict.title}</strong><p>{conflict.reason}</p></div></article>)}
      </div>}

      {!allowed && exchange.safeAlternative && <article className="safe-alternative"><span>Безопасная альтернатива</span><p>{exchange.safeAlternative}</p></article>}

      {allowed && exchange.answer && <article className="guard-answer">
        <header><div><span>C</span><div><strong>Ответ Compass</strong><small>сформирован только после успешной проверки</small></div></div>{exchange.taskState && <b>{exchange.taskState.phase} · revision {exchange.taskState.revision}</b>}</header>
        <div className="markdown-answer"><ReactMarkdown>{exchange.answer}</ReactMarkdown></div>
      </article>}
    </section>
  );
}

function AssessmentRow({ invariant, assessment }: { invariant: Invariant; assessment?: InvariantAssessment }) {
  const verdict = assessment?.verdict ?? "compliant";
  const conflict = verdict === "conflict";
  const pending = verdict === "pending_semantic_check";
  return <li className={conflict ? "conflict" : pending ? "pending" : "compliant"}><span>{conflict ? "!" : pending ? "—" : "✓"}</span><div><strong>{invariant.title}</strong><p>{assessment?.note ?? "Конфликт не обнаружен."}</p></div><b>{conflict ? "КОНФЛИКТ" : pending ? "НЕ ПРОВЕРЕН" : "СОБЛЮДЁН"}</b></li>;
}

function GuardLoading({ text }: { text: string }) {
  return <div className="guard-loading" aria-live="polite"><span className="spinner" /><div><strong>{text}</strong><p>Основной агент получит запрос только после разрешения Guard.</p></div></div>;
}

function getUserId(): string {
  const key = "ai-challenge-memory-user";
  let value = localStorage.getItem(key);
  if (!value) { value = crypto.randomUUID(); localStorage.setItem(key, value); }
  return value;
}

function errorMessage(error: unknown): string {
  return error instanceof APIError || error instanceof Error ? error.message : "Произошла неизвестная ошибка.";
}

export default InvariantLab;
