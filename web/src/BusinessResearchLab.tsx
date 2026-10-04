import { FormEvent, useRef, useState } from "react";
import { APIError, researchBusinessNews } from "./api";
import type { BusinessResearchResult } from "./types";

const examples = [
  "Найди свежие новости про AI-стартапы и предложи идеи небольших продуктов.",
  "Какие новые инструменты для разработчиков обсуждают и на чём можно построить микросервис?",
  "Собери бизнес-сигналы вокруг автоматизации малого бизнеса и предложи первый эксперимент.",
];

function BusinessResearchLab() {
  const [request, setRequest] = useState("");
  const [result, setResult] = useState<BusinessResearchResult | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!request.trim() || pending) return;
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setPending(true);
    setError(null);
    setResult(null);
    try {
      setResult(await researchBusinessNews(request.trim(), controller.signal));
    } catch (caught) {
      if (!(caught instanceof DOMException && caught.name === "AbortError")) {
        setError(caught instanceof APIError || caught instanceof Error ? caught.message : "Исследование не выполнено.");
      }
    } finally {
      if (abortRef.current === controller) setPending(false);
    }
  }

  return (
    <main className="radar-page research-page">
      <RadarNav active={17} />
      <header className="radar-hero research-hero">
        <div>
          <p className="radar-kicker">AI Advent Challenge · День 17</p>
          <h1>Первый<br /><em>MCP tool</em></h1>
          <p>Вы формулируете бизнес-задачу обычным языком. Агент читает зарегистрированные схемы, самостоятельно выбирает инструмент и использует его результат.</p>
          <div className="radar-tags"><span>Agent routing</span><span>Hacker News API</span><span>Structured result</span></div>
        </div>
        <aside className="research-formula">
          <span>USER</span><i>→</i><span>AGENT</span><i>→</i><span>MCP TOOL</span><i>→</i><span>ADVICE</span>
        </aside>
      </header>

      <section className="radar-workspace">
        <div className="radar-heading"><div><span>01 / ASK</span><h2>Что исследуем?</h2></div><p className="research-note">Источники загружаются в момент запроса. Ссылки не генерируются моделью.</p></div>
        <form className="research-composer" onSubmit={submit}>
          <textarea value={request} onChange={(event) => setRequest(event.target.value)} maxLength={4000} placeholder="Например: найди свежие новости про AI-стартапы и предложи идеи небольших продуктов…" disabled={pending} />
          <footer><span>{request.length} / 4000</span><button disabled={!request.trim() || pending}>{pending ? <><i className="radar-spinner" /> Агент выбирает tool…</> : "Запустить агента →"}</button></footer>
        </form>
        <div className="research-examples"><span>Попробовать:</span>{examples.map((example) => <button key={example} type="button" onClick={() => setRequest(example)} disabled={pending}>{example}</button>)}</div>
        {error && <div className="radar-error"><b>Запрос не выполнен</b><p>{error}</p></div>}
        {pending && <div className="research-loading"><i className="radar-spinner" /><div><b>Agent → tools/list → tool/call</b><p>Получаем свежие данные и готовим аккуратные гипотезы.</p></div></div>}

        {result && (
          <section className="research-result">
            <div className="tool-decision">
              <div><span>AGENT SELECTED</span><strong>{result.selection.tool}</strong><p>{result.selection.rationale}</p></div>
              <dl><div><dt>Server</dt><dd>{result.selection.server}</dd></div><div><dt>Arguments</dt><dd><code>{JSON.stringify(result.selection.arguments)}</code></dd></div></dl>
            </div>

            <div className="research-section-heading"><span>02 / SOURCES</span><h2>Свежие сигналы</h2><p>{result.stories.length} материалов · ссылки из MCP</p></div>
            <div className="story-grid">
              {result.stories.map((story, index) => (
                <article key={story.id}>
                  <span>{String(index + 1).padStart(2, "0")} · {story.source}</span>
                  <h3>{story.title}</h3>
                  <div><b>▲ {story.score}</b><time>{new Date(story.publishedAt).toLocaleDateString("ru-RU")}</time></div>
                  <footer><a href={story.url} target="_blank" rel="noreferrer">Источник ↗</a><a href={story.discussionUrl} target="_blank" rel="noreferrer">Обсуждение ↗</a></footer>
                </article>
              ))}
            </div>

            <div className="research-section-heading opportunity-heading"><span>03 / OPPORTUNITIES</span><h2>Что можно проверить</h2><p>{result.advice.summary}</p></div>
            <div className="opportunity-grid">
              {result.advice.opportunities.map((item, index) => <article key={`${item.title}-${index}`}><b>{String(index + 1).padStart(2, "0")}</b><h3>{item.title}</h3><dl><div><dt>Почему сейчас</dt><dd>{item.whyNow}</dd></div><div><dt>Первый шаг</dt><dd>{item.firstStep}</dd></div><div><dt>Риск</dt><dd>{item.risk}</dd></div></dl></article>)}
            </div>
            <div className="research-caveat"><b>Важно</b><p>{result.advice.caveat}</p></div>

            <details className="research-trace"><summary>Как агент выполнил запрос · {result.usage.totalTokens} токенов</summary><ol>{result.trace.map((step) => <li key={step}>{step}</li>)}</ol></details>
          </section>
        )}
      </section>
      <footer className="radar-footer"><span>Startup & Business Radar</span><span>Agent · MCP · Source-grounded advice</span></footer>
    </main>
  );
}

export function RadarNav({ active }: { active: number }) {
  return <nav className="radar-nav" aria-label="Навигация Startup Radar"><a className="radar-brand" href="#day-16"><span>SR</span> Startup Radar</a><div className="radar-days">{[16,17,18,19,20].map((number) => <a key={number} className={active === number ? "active" : ""} href={`#day-${number}`}>День {number}</a>)}<a className="radar-next-days" href="#day-21">21–25 →</a></div><a className="radar-github" href="https://github.com/eugeneappledev-source/AI-Challenge" target="_blank" rel="noreferrer">GitHub ↗</a></nav>;
}

export default BusinessResearchLab;
