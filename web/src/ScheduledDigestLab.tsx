import { useEffect, useRef, useState } from "react";
import { APIError, loadDigestDashboard, runBusinessDigest } from "./api";
import type { DigestDashboard } from "./types";
import { RadarNav } from "./BusinessResearchLab";

function ScheduledDigestLab() {
  const [dashboard, setDashboard] = useState<DigestDashboard | null>(null);
  const [booting, setBooting] = useState(true);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    loadDigestDashboard(controller.signal).then(setDashboard).catch((caught) => {
      if (!(caught instanceof DOMException && caught.name === "AbortError")) setError(message(caught));
    }).finally(() => setBooting(false));
    return () => controller.abort();
  }, []);

  async function runNow() {
    if (running) return;
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setRunning(true);
    setError(null);
    try {
      await runBusinessDigest(controller.signal);
      setDashboard(await loadDigestDashboard(controller.signal));
    } catch (caught) {
      if (!(caught instanceof DOMException && caught.name === "AbortError")) setError(message(caught));
    } finally {
      if (abortRef.current === controller) setRunning(false);
    }
  }

  const latest = dashboard?.digests[0];
  return <main className="radar-page schedule-page">
    <RadarNav active={18} />
    <header className="radar-hero schedule-hero">
      <div><p className="radar-kicker">AI Advent Challenge · День 18</p><h1>Фоновый<br /><em>дайджест</em></h1><p>Отдельный cron-контейнер будит агента каждый день. Агент вызывает MCP, собирает бизнес-сигналы и сохраняет готовую сводку в SQLite.</p><div className="radar-tags"><span>cron 24/7</span><span>Agent + MCP</span><span>SQLite</span></div></div>
      <aside className="schedule-clock"><div><span className={dashboard?.schedule.enabled ? "online" : ""} />SCHEDULER ACTIVE</div><strong>08:00</strong><p>каждый день · {dashboard?.schedule.timezone || "UTC"}</p><small>{dashboard ? `Следующий запуск: ${formatDate(dashboard.schedule.nextRunAt)}` : "Загружаем расписание…"}</small></aside>
    </header>
    <section className="radar-workspace">
      <div className="radar-heading"><div><span>01 / SCHEDULE</span><h2>Агент работает без пользователя</h2></div><button type="button" onClick={() => void runNow()} disabled={running || booting}>{running ? <><i className="radar-spinner" /> Собираем дайджест…</> : "Запустить сейчас"}</button></div>
      {error && <div className="radar-error"><b>Фоновый запуск не выполнен</b><p>{error}</p></div>}
      {dashboard && <div className="schedule-grid"><article><span>CRON EXPRESSION</span><strong>{dashboard.schedule.cron}</strong><p>Конфигурация контейнера `radar-scheduler`</p></article><article><span>LAST RUN</span><strong>{dashboard.schedule.lastRunAt ? formatDate(dashboard.schedule.lastRunAt) : "Ещё не запускался"}</strong><p>Status: {dashboard.schedule.lastStatus}</p></article><article><span>PERSISTENCE</span><strong>{dashboard.digests.length} сохранено</strong><p>SQLite переживает перезапуск контейнеров</p></article></div>}
      {running && <div className="schedule-progress"><div className="schedule-pulse" /><div><b>Cron-сценарий запущен вручную для демонстрации</b><p>Research Agent → MCP news tool → analysis → SQLite</p></div></div>}

      <div className="research-section-heading"><span>02 / LATEST DIGEST</span><h2>Последняя сводка</h2><p>Открывается из хранилища — повторный LLM-вызов для просмотра не нужен.</p></div>
      {!latest && !booting && <div className="empty-digest"><b>История пока пуста</b><p>Нажмите «Запустить сейчас». После деплоя следующие записи будет автоматически создавать cron.</p></div>}
      {latest && <article className="latest-digest"><header><div><span>{latest.trigger === "cron" ? "AUTOMATIC CRON" : "MANUAL DEMO"}</span><h3>{formatDate(latest.createdAt)}</h3></div><b>{latest.result.stories.length} источников</b></header><p>{latest.result.advice.summary}</p><div>{latest.result.advice.opportunities.map((item, index) => <section key={`${item.title}-${index}`}><b>{String(index + 1).padStart(2,"0")}</b><h4>{item.title}</h4><p>{item.firstStep}</p><small>Риск: {item.risk}</small></section>)}</div></article>}

      <div className="research-section-heading"><span>03 / HISTORY</span><h2>История запусков</h2><p>Все результаты сохраняются отдельно вместе с причиной запуска.</p></div>
      <div className="digest-history">{dashboard?.digests.map((digest) => <article key={digest.id}><span>{digest.trigger}</span><strong>{formatDate(digest.createdAt)}</strong><p>{digest.result.advice.summary}</p><small>{digest.result.usage.totalTokens} токенов · {digest.result.model}</small></article>)}</div>
    </section>
    <footer className="radar-footer"><span>Startup & Business Radar</span><span>cron · Agent · MCP · SQLite</span></footer>
  </main>;
}

function message(error: unknown) { return error instanceof APIError || error instanceof Error ? error.message : "Неизвестная ошибка."; }
function formatDate(value: string) { return new Date(value).toLocaleString("ru-RU", { day:"2-digit", month:"short", hour:"2-digit", minute:"2-digit" }); }

export default ScheduledDigestLab;
