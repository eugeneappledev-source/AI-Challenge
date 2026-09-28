import { useEffect, useState } from "react";
import { APIError, inspectMCPConnection } from "./api";
import type { MCPConnectionResult } from "./types";
import BusinessResearchLab from "./BusinessResearchLab";

type RadarDay = "day-16" | "day-17" | "day-18" | "day-19" | "day-20";

function StartupRadarLab({ day }: { day: RadarDay }) {
  if (day === "day-17") return <BusinessResearchLab />;
  const [result, setResult] = useState<MCPConnectionResult | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void inspect();
  }, []);

  async function inspect() {
    if (isLoading) return;
    const controller = new AbortController();
    setIsLoading(true);
    setError(null);
    try {
      setResult(await inspectMCPConnection(controller.signal));
    } catch (caught) {
      setError(caught instanceof APIError || caught instanceof Error ? caught.message : "Не удалось проверить MCP.");
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <main className="radar-page">
      <nav className="radar-nav" aria-label="Навигация Startup Radar">
        <a className="radar-brand" href="#day-16"><span>SR</span> Startup Radar</a>
        <div className="radar-days">
          {[16, 17, 18, 19, 20].map((number) => (
            <a key={number} className={day === `day-${number}` ? "active" : ""} href={`#day-${number}`}>День {number}</a>
          ))}
        </div>
        <a className="radar-github" href="https://github.com/eugeneappledev-source/AI-Challenge" target="_blank" rel="noreferrer">GitHub ↗</a>
      </nav>

      <header className="radar-hero">
        <div>
          <p className="radar-kicker">AI Advent Challenge · День 16</p>
          <h1>Подключение<br /><em>MCP</em></h1>
          <p>Клиент устанавливает настоящее Streamable HTTP-соединение, согласует версию протокола и запрашивает у сервера доступные инструменты.</p>
          <div className="radar-tags"><span>Official Go SDK</span><span>Streamable HTTP</span><span>tools/list</span></div>
        </div>
        <aside className="radar-status-card">
          <span className={result?.connected ? "online" : ""} />
          <p>Состояние соединения</p>
          <strong>{isLoading ? "Проверяем…" : result?.connected ? "MCP подключён" : "Ожидает проверки"}</strong>
          <small>{result ? new Date(result.checkedAt).toLocaleString("ru-RU") : "Автоматическая проверка при открытии"}</small>
        </aside>
      </header>

      <section className="radar-workspace">
        <div className="radar-heading">
          <div><span>01 / CONNECTION</span><h2>Что вернул MCP-сервер</h2></div>
          <button type="button" onClick={() => void inspect()} disabled={isLoading}>
            {isLoading ? <><i className="radar-spinner" /> Подключаемся…</> : "Повторить tools/list"}
          </button>
        </div>

        {error && <div className="radar-error"><b>Соединение не установлено</b><p>{error}</p></div>}

        {result && (
          <>
            <div className="radar-protocol-grid">
              <article><span>SERVER</span><strong>{result.serverName}</strong><small>{result.serverVersion}</small></article>
              <article><span>PROTOCOL</span><strong>{result.protocolVersion}</strong><small>{result.transport}</small></article>
              <article><span>CLIENT</span><strong>{result.clientName}</strong><small>{result.tools.length} tool зарегистрирован</small></article>
            </div>

            <div className="radar-flow" aria-label="Последовательность MCP-вызова">
              {result.trace.map((step, index) => <div key={step}><b>{String(index + 1).padStart(2, "0")}</b><span>{step}</span></div>)}
            </div>

            <section className="radar-tools">
              <header><div><span>02 / TOOLS</span><h2>Доступные инструменты</h2></div><b>{result.tools.length}</b></header>
              {result.tools.map((tool) => (
                <article key={tool.name}>
                  <div className="radar-tool-icon">⌘</div>
                  <div><span>{tool.title || tool.name}</span><h3>{tool.name}</h3><p>{tool.description}</p></div>
                  <details><summary>Input schema</summary><pre>{JSON.stringify(tool.inputSchema, null, 2)}</pre></details>
                </article>
              ))}
            </section>
          </>
        )}
      </section>

      <footer className="radar-footer"><span>Startup & Business Radar</span><span>MCP · Go · React · DeepSeek</span></footer>
    </main>
  );
}

export default StartupRadarLab;
