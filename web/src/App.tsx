import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import { APIError, compareAnswers, parseControlledAnswer } from "./api";
import type {
  ChatReply,
  Comparison,
  ControlledFoodAnswer,
  ResponseMode,
} from "./types";

const suggestions = [
  "Дай простой рецепт греческого салата.",
  "Что приготовить из курицы, риса и помидоров?",
  "Как сварить крем-суп из тыквы?",
];

function App() {
  const [message, setMessage] = useState("");
  const [comparison, setComparison] = useState<Comparison | null>(null);
  const [selectedMode, setSelectedMode] =
    useState<ResponseMode>("controlled");
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const abortController = useRef<AbortController | null>(null);

  useEffect(() => () => abortController.current?.abort(), []);

  const controlledAnswer = useMemo(() => {
    if (!comparison) return null;
    try {
      return parseControlledAnswer(comparison.controlled.answer);
    } catch {
      return null;
    }
  }, [comparison]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmedMessage = message.trim();
    if (!trimmedMessage || isLoading) return;

    abortController.current?.abort();
    const controller = new AbortController();
    abortController.current = controller;
    setIsLoading(true);
    setError(null);
    setComparison(null);

    try {
      const result = await compareAnswers(trimmedMessage, controller.signal);
      setComparison(result);
      setSelectedMode("controlled");
    } catch (caughtError) {
      if (caughtError instanceof DOMException && caughtError.name === "AbortError") {
        return;
      }
      setError(
        caughtError instanceof APIError || caughtError instanceof Error
          ? caughtError.message
          : "Произошла неизвестная ошибка.",
      );
    } finally {
      if (abortController.current === controller) {
        setIsLoading(false);
      }
    }
  }

  const selectedReply = comparison?.[selectedMode] ?? null;

  return (
    <main className="page-shell">
      <nav className="topbar" aria-label="Навигация">
        <a className="brand" href="#top" aria-label="AI Challenge — наверх">
          <span className="brand-mark" aria-hidden="true">AI</span>
          <span>Challenge</span>
        </a>
        <div className="day-badge"><span /> День 2 · Формат ответа</div>
        <a
          className="github-link"
          href="https://github.com/eugeneappledev-source/AI-Challenge"
          target="_blank"
          rel="noreferrer"
        >
          GitHub <span aria-hidden="true">↗</span>
        </a>
      </nav>

      <section className="hero" id="top">
        <div className="hero-copy">
          <p className="eyebrow">AI Advent Challenge · поток 9</p>
          <h1>Один вопрос.<br /><em>Два уровня контроля.</em></h1>
          <p className="hero-description">
            Задайте вопрос о еде. Сервис отправит его в DeepSeek дважды и покажет,
            как формат, длина и условия завершения меняют ответ модели.
          </p>
          <div className="hero-facts" aria-label="Особенности проекта">
            <span>SwiftUI</span><i />
            <span>Go API</span><i />
            <span>DeepSeek</span><i />
            <span>HTTPS</span>
          </div>
        </div>

        <aside className="brief-card">
          <p className="brief-number">02</p>
          <h2>Что сравниваем</h2>
          <ul>
            <li><span>01</span> Ответ без ограничений</li>
            <li><span>02</span> Строгая JSON-структура</li>
            <li><span>03</span> Лимит длины и завершение</li>
          </ul>
        </aside>
      </section>

      <section className="workspace" aria-labelledby="workspace-title">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Попробуйте сами</p>
            <h2 id="workspace-title">Спросите AI-повара</h2>
          </div>
          <p>Лучше всего работают рецепты, продукты и способы приготовления.</p>
        </div>

        <form className="prompt-card" onSubmit={submit}>
          <label htmlFor="message">Ваш вопрос</label>
          <textarea
            id="message"
            value={message}
            onChange={(event) => setMessage(event.target.value)}
            placeholder="Например: что приготовить из продуктов, которые есть дома?"
            maxLength={4000}
            rows={4}
            disabled={isLoading}
          />
          <div className="prompt-footer">
            <span>{message.length} / 4000</span>
            <button type="submit" disabled={!message.trim() || isLoading}>
              {isLoading ? (
                <><span className="spinner" aria-hidden="true" /> Сравниваю ответы…</>
              ) : (
                <>Сравнить ответы <span aria-hidden="true">→</span></>
              )}
            </button>
          </div>
        </form>

        <div className="suggestions" aria-label="Примеры вопросов">
          <span>Идеи:</span>
          {suggestions.map((suggestion) => (
            <button
              key={suggestion}
              type="button"
              onClick={() => setMessage(suggestion)}
              disabled={isLoading}
            >
              {suggestion}
            </button>
          ))}
        </div>

        <div className="scope-note">
          <span className="scope-icon" aria-hidden="true">i</span>
          <p>
            <strong>Область ассистента — еда.</strong> В controlled-режиме вопрос
            не по теме вернёт тот же JSON со статусом <code>out_of_scope</code>,
            пустыми массивами ингредиентов и шагов.
          </p>
        </div>

        {error && (
          <div className="error-card" role="alert">
            <span aria-hidden="true">!</span>
            <div><strong>Ответ не получен</strong><p>{error}</p></div>
          </div>
        )}

        {isLoading && <LoadingComparison />}

        {comparison && selectedReply && (
          <section className="results" aria-labelledby="results-title">
            <div className="results-header">
              <div>
                <p className="eyebrow">Результат сравнения</p>
                <h2 id="results-title">Как ответила модель</h2>
              </div>
              <div className="mode-switch" role="tablist" aria-label="Режим ответа">
                <ModeButton
                  mode="unrestricted"
                  selected={selectedMode === "unrestricted"}
                  onSelect={setSelectedMode}
                >
                  Без ограничений
                </ModeButton>
                <ModeButton
                  mode="controlled"
                  selected={selectedMode === "controlled"}
                  onSelect={setSelectedMode}
                >
                  С контролем
                </ModeButton>
              </div>
            </div>

            <div className="result-layout">
              <article className="answer-card">
                <div className="answer-card-heading">
                  <span className={`mode-dot ${selectedMode}`} />
                  <div>
                    <p>{selectedMode === "controlled" ? "Контролируемый ответ" : "Свободный ответ"}</p>
                    <span>{selectedMode === "controlled" ? "JSON · ≤ 80 слов" : "Естественный текст · без лимита"}</span>
                  </div>
                </div>

                {selectedMode === "controlled" ? (
                  <ControlledResult parsed={controlledAnswer} raw={selectedReply.answer} />
                ) : (
                  <div className="markdown-answer"><ReactMarkdown>{selectedReply.answer}</ReactMarkdown></div>
                )}
              </article>

              <ReplyMetadata reply={selectedReply} />
            </div>
          </section>
        )}
      </section>

      <footer>
        <p>Сделано для AI Advent Challenge #9</p>
        <p>SwiftUI · Go · DeepSeek · Caddy</p>
      </footer>
    </main>
  );
}

function ModeButton({
  mode,
  selected,
  onSelect,
  children,
}: {
  mode: ResponseMode;
  selected: boolean;
  onSelect: (mode: ResponseMode) => void;
  children: string;
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={selected}
      className={selected ? "selected" : ""}
      onClick={() => onSelect(mode)}
    >
      {children}
    </button>
  );
}

function ControlledResult({
  parsed,
  raw,
}: {
  parsed: ControlledFoodAnswer | null;
  raw: string;
}) {
  if (!parsed) {
    return (
      <div className="parse-fallback">
        <p>Не удалось разобрать структурированный ответ.</p>
        <RawJSON raw={raw} open />
      </div>
    );
  }

  const isOutOfScope = parsed.status === "out_of_scope";
  return (
    <div className="controlled-answer">
      <div className={`status-pill ${isOutOfScope ? "out" : "ok"}`}>
        <span /> {isOutOfScope ? "Вопрос вне темы" : "Готово"}
      </div>
      <p className="answer-summary">{parsed.answer}</p>

      {!isOutOfScope && parsed.ingredients.length > 0 && (
        <section className="food-section">
          <h3><span>01</span> Ингредиенты</h3>
          <div className="ingredient-list">
            {parsed.ingredients.map((ingredient) => (
              <span key={ingredient}>{ingredient}</span>
            ))}
          </div>
        </section>
      )}

      {!isOutOfScope && parsed.steps.length > 0 && (
        <section className="food-section">
          <h3><span>02</span> Приготовление</h3>
          <ol className="steps-list">
            {parsed.steps.map((step, index) => (
              <li key={`${index}-${step}`}><span>{index + 1}</span><p>{step}</p></li>
            ))}
          </ol>
        </section>
      )}

      <RawJSON raw={raw} />
    </div>
  );
}

function RawJSON({ raw, open = false }: { raw: string; open?: boolean }) {
  let formatted = raw;
  try {
    formatted = JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    // Preserve the original value so a malformed model response is still visible.
  }

  return (
    <details className="raw-json" open={open}>
      <summary><span>P.S.</span> Показать исходный JSON модели</summary>
      <pre>{formatted}</pre>
    </details>
  );
}

function ReplyMetadata({ reply }: { reply: ChatReply }) {
  return (
    <aside className="metadata-card">
      <p className="metadata-title">Параметры ответа</p>
      <dl>
        <div><dt>Модель</dt><dd>{reply.model}</dd></div>
        <div><dt>Режим</dt><dd>{reply.mode}</dd></div>
        <div><dt>Завершение</dt><dd><span className="finish-dot" /> {reply.finishReason}</dd></div>
        <div><dt>Prompt tokens</dt><dd>{reply.usage.promptTokens}</dd></div>
        <div><dt>Answer tokens</dt><dd>{reply.usage.completionTokens}</dd></div>
        <div className="total"><dt>Всего токенов</dt><dd>{reply.usage.totalTokens}</dd></div>
      </dl>
      <div className="metadata-note">
        {reply.mode === "controlled"
          ? "Формат и границы ответа заданы на сервере."
          : "Модель сама выбирает структуру и объём ответа."}
      </div>
    </aside>
  );
}

function LoadingComparison() {
  return (
    <div className="loading-comparison" aria-live="polite">
      <div className="loading-orbit"><span /><span /><span /></div>
      <div><strong>DeepSeek готовит два ответа</strong><p>Обычно это занимает несколько секунд.</p></div>
    </div>
  );
}

export default App;
