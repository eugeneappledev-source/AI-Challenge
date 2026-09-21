import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import {
  APIError,
  clearLayeredMemory,
  loadLayeredMemoryState,
  sendLayeredMemoryMessage,
} from "./api";
import type {
  AgentMessage,
  LayeredMemoryExchange,
  LayeredMemoryState,
  MemoryItem,
  MemoryLayer,
  MemoryScope,
} from "./types";

const layers: Array<{ id: MemoryLayer; title: string; hint: string }> = [
  { id: "auto", title: "Авто", hint: "Агент решает сам" },
  { id: "short_term", title: "Краткосрочная", hint: "Текущая беседа" },
  { id: "working", title: "Рабочая", hint: "Текущая задача" },
  { id: "long_term", title: "Долговременная", hint: "Профиль пользователя" },
];

const demoSteps: Array<{ layer: MemoryLayer; label: string; message: string }> = [
  {
    layer: "auto",
    label: "1 · Пользователь",
    message: "Я iOS-разработчик и предпочитаю SwiftUI, async/await и лаконичные технические ответы.",
  },
  {
    layer: "auto",
    label: "2 · Задача",
    message: "В проекте PulsePlan делаем экран недельной статистики для iOS 17. Он должен работать offline-first и поддерживать VoiceOver.",
  },
  {
    layer: "short_term",
    label: "3 · Реплика",
    message: "Сейчас сравни два варианта архитектуры, но пока не пиши код.",
  },
  {
    layer: "auto",
    label: "4 · Проверка",
    message: "Что ты обо мне и задаче помнишь? Предложи подход с учётом всех сохранённых данных.",
  },
];

function MemoryLab() {
  const scope = useMemo(getMemoryScope, []);
  const [message, setMessage] = useState("");
  const [layer, setLayer] = useState<MemoryLayer>("auto");
  const [state, setState] = useState<LayeredMemoryState>(() => emptyState(scope));
  const [exchange, setExchange] = useState<LayeredMemoryExchange | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isResetting, setIsResetting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const abortController = useRef<AbortController | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    loadLayeredMemoryState(scope, controller.signal)
      .then(setState)
      .catch((caughtError: unknown) => {
        if (!(caughtError instanceof DOMException && caughtError.name === "AbortError")) {
          setError(errorMessage(caughtError));
        }
      });
    return () => controller.abort();
  }, [scope]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalized = message.trim();
    if (!normalized || isLoading) return;

    abortController.current?.abort();
    const controller = new AbortController();
    abortController.current = controller;
    setIsLoading(true);
    setError(null);
    try {
      const result = await sendLayeredMemoryMessage(scope, normalized, layer, controller.signal);
      setExchange(result);
      setState(result.state);
      setMessage("");
    } catch (caughtError) {
      if (!(caughtError instanceof DOMException && caughtError.name === "AbortError")) {
        setError(errorMessage(caughtError));
      }
    } finally {
      if (abortController.current === controller) setIsLoading(false);
    }
  }

  async function resetMemory() {
    if (isResetting || isLoading) return;
    setIsResetting(true);
    setError(null);
    try {
      await clearLayeredMemory(scope);
      setState(emptyState(scope));
      setExchange(null);
      setMessage("");
      setLayer("auto");
    } catch (caughtError) {
      setError(errorMessage(caughtError));
    } finally {
      setIsResetting(false);
    }
  }

  function useDemo(step: (typeof demoSteps)[number]) {
    setLayer(step.layer);
    setMessage(step.message);
    document.getElementById("memory-message")?.focus();
  }

  return (
    <main className="page-shell memory-page">
      <nav className="topbar" aria-label="Навигация">
        <a className="brand" href="#day-11" aria-label="AI Challenge — День 11">
          <span className="brand-mark" aria-hidden="true">AI</span>
          <span>Challenge</span>
        </a>
        <div className="day-switcher" aria-label="Выбор задания">
          <a href="#day-02">День 2</a>
          <a className="selected" href="#day-11">День 11</a>
          <a href="#day-12">День 12</a>
          <a href="#day-13">День 13</a>
          <a href="#day-14">День 14</a>
        </div>
        <a className="github-link" href="https://github.com/eugeneappledev-source/AI-Challenge" target="_blank" rel="noreferrer">
          GitHub <span aria-hidden="true">↗</span>
        </a>
      </nav>

      <section className="memory-hero">
        <div>
          <p className="eyebrow">День 11 · Модель памяти</p>
          <h1>Три срока жизни.<br /><em>Один контекст.</em></h1>
          <p className="hero-description">
            Compass определяет, сколько должна жить информация, сохраняет её в нужном слое
            и собирает контекст для следующего ответа без RAG и векторной базы.
          </p>
        </div>
        <aside className="memory-diagram" aria-label="Поток памяти">
          <div><span>01</span><strong>Сообщение</strong><small>ввод пользователя</small></div>
          <i>→</i>
          <div><span>02</span><strong>Router</strong><small>выбор слоя</small></div>
          <i>→</i>
          <div><span>03</span><strong>Context</strong><small>сборка ответа</small></div>
        </aside>
      </section>

      <section className="memory-workspace">
        <div className="section-heading memory-heading">
          <div><p className="eyebrow">Интерактивный эксперимент</p><h2>Научите агента помнить</h2></div>
          <button className="reset-button" type="button" onClick={resetMemory} disabled={isResetting || isLoading}>
            {isResetting ? "Очищаю…" : "Очистить все слои"}
          </button>
        </div>

        <div className="demo-strip">
          <div><strong>Сценарий для видео</strong><span>Нажимайте примеры по порядку и отправляйте</span></div>
          <div className="demo-actions">
            {demoSteps.map((step) => <button key={step.label} type="button" onClick={() => useDemo(step)} disabled={isLoading}>{step.label}</button>)}
          </div>
        </div>

        <form className="memory-composer" onSubmit={submit}>
          <fieldset disabled={isLoading}>
            <legend>Куда сохранить сообщение</legend>
            <div className="layer-selector">
              {layers.map((option) => (
                <label className={layer === option.id ? "selected" : ""} key={option.id}>
                  <input type="radio" name="memory-layer" value={option.id} checked={layer === option.id} onChange={() => setLayer(option.id)} />
                  <span><strong>{option.title}</strong><small>{option.hint}</small></span>
                </label>
              ))}
            </div>
          </fieldset>
          <label className="memory-input-label" htmlFor="memory-message">Сообщение агенту</label>
          <textarea
            id="memory-message"
            value={message}
            onChange={(event) => setMessage(event.target.value)}
            placeholder="Расскажите о себе, добавьте условие задачи или задайте текущий вопрос…"
            rows={4}
            maxLength={4000}
            disabled={isLoading}
          />
          <div className="memory-composer-footer">
            <span>{layer === "auto" ? "Router сам выберет срок жизни" : `Слой зафиксирован: ${layerTitle(layer)}`}</span>
            <button type="submit" disabled={!message.trim() || isLoading}>
              {isLoading ? <><span className="spinner" /> Маршрутизирую и отвечаю…</> : <>Отправить агенту <span>→</span></>}
            </button>
          </div>
        </form>

        {error && <div className="error-card" role="alert"><span>!</span><div><strong>Запрос не выполнен</strong><p>{error}</p></div></div>}
        {isLoading && <div className="memory-loading"><span className="spinner" /><div><strong>Compass анализирует сообщение</strong><p>Выбирает слой, обновляет память и собирает контекст.</p></div></div>}

        {exchange && !isLoading && <ExchangeResult exchange={exchange} />}

        <section className="memory-layers" aria-labelledby="layers-title">
          <div className="memory-layers-heading">
            <div><p className="eyebrow">Состояние SQLite</p><h2 id="layers-title">Что агент помнит сейчас</h2></div>
            <span>{state.shortTerm.length + state.working.length + state.longTerm.length} записей</span>
          </div>
          <div className="memory-grid">
            <MemoryCard index="01" title="Краткосрочная" subtitle="Session · последние 8 сообщений" tone="orange">
              <ShortTermItems messages={state.shortTerm} />
            </MemoryCard>
            <MemoryCard index="02" title="Рабочая" subtitle="Task · только PulsePlan" tone="green">
              <FactItems items={state.working} empty="Здесь появятся цели, требования и решения текущей задачи." />
            </MemoryCard>
            <MemoryCard index="03" title="Долговременная" subtitle="User · между задачами" tone="blue">
              <FactItems items={state.longTerm} empty="Здесь появятся устойчивые сведения и предпочтения пользователя." />
            </MemoryCard>
          </div>
        </section>
      </section>

      <footer><p>AI Advent Challenge #9 · День 11</p><p>React · Go · SQLite · DeepSeek</p></footer>
    </main>
  );
}

function ExchangeResult({ exchange }: { exchange: LayeredMemoryExchange }) {
  return (
    <section className="memory-result" aria-live="polite">
      <div className="route-card">
        <p className="eyebrow">Решение маршрутизатора</p>
        <div className={`route-layer ${exchange.route.selectedLayer}`}><span />{layerTitle(exchange.route.selectedLayer)}</div>
        <h3>{exchange.route.key}</h3>
        <p>{exchange.route.value}</p>
        <blockquote>{exchange.route.reason}</blockquote>
        <small>{exchange.route.automatic ? "Слой выбран автоматически" : "Слой задан пользователем"}</small>
      </div>
      <article className="memory-answer">
        <div className="memory-answer-heading">
          <div><span className="agent-avatar">C</span><div><strong>Compass</strong><small>{exchange.model}</small></div></div>
          <span>{exchange.usage.totalTokens} токенов · {exchange.finishReason}</span>
        </div>
        <div className="markdown-answer"><ReactMarkdown>{exchange.answer}</ReactMarkdown></div>
        <details className="context-preview">
          <summary>Как был собран контекст</summary>
          <ol>{exchange.contextPreview.map((item, index) => <li key={`${index}-${item}`}>{item}</li>)}</ol>
        </details>
      </article>
    </section>
  );
}

function MemoryCard({ index, title, subtitle, tone, children }: { index: string; title: string; subtitle: string; tone: string; children: React.ReactNode }) {
  return <article className={`memory-card ${tone}`}><header><span>{index}</span><div><h3>{title}</h3><p>{subtitle}</p></div></header><div className="memory-card-body">{children}</div></article>;
}

function ShortTermItems({ messages }: { messages: AgentMessage[] }) {
  if (messages.length === 0) return <EmptyMemory>Диалог пока пуст. Реплики живут только в текущей сессии.</EmptyMemory>;
  return <div className="short-messages">{messages.slice(-6).map((message) => <div className={message.role} key={message.id}><span>{message.role === "user" ? "Вы" : "AI"}</span><p>{message.content}</p></div>)}</div>;
}

function FactItems({ items, empty }: { items: MemoryItem[]; empty: string }) {
  if (items.length === 0) return <EmptyMemory>{empty}</EmptyMemory>;
  return <dl className="fact-list">{items.map((item) => <div key={item.key}><dt>{humanizeKey(item.key)}</dt><dd>{item.value}</dd></div>)}</dl>;
}

function EmptyMemory({ children }: { children: React.ReactNode }) {
  return <div className="empty-memory"><span>＋</span><p>{children}</p></div>;
}

function getMemoryScope(): MemoryScope {
  const key = "ai-challenge-memory-user";
  let userId = localStorage.getItem(key);
  if (!userId) {
    userId = crypto.randomUUID();
    localStorage.setItem(key, userId);
  }
  return { sessionId: `${userId}:day11`, taskId: `${userId}:pulse-plan`, userId };
}

function emptyState(scope: MemoryScope): LayeredMemoryState {
  return { ...scope, shortTerm: [], working: [], longTerm: [], updatedAt: new Date().toISOString() };
}

function layerTitle(layer: MemoryLayer): string {
  return layers.find((item) => item.id === layer)?.title ?? layer;
}

function humanizeKey(value: string): string {
  return value.replaceAll("_", " ");
}

function errorMessage(error: unknown): string {
  return error instanceof APIError || error instanceof Error ? error.message : "Произошла неизвестная ошибка.";
}

export default MemoryLab;
