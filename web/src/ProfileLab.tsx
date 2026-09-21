import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import { APIError, loadProfiles, saveProfile, sendPersonalizedMessage } from "./api";
import type { MemoryScope, PersonalizedExchange, UserProfile } from "./types";

const defaultPrompt = "Предложи архитектуру и план MVP для экрана недельной статистики PulsePlan с учётом уже известных требований.";

function ProfileLab() {
  const userId = useMemo(getUserId, []);
  const [profiles, setProfiles] = useState<UserProfile[]>([]);
  const [selectedID, setSelectedID] = useState("engineer");
  const [draft, setDraft] = useState<UserProfile | null>(null);
  const [prompt, setPrompt] = useState(defaultPrompt);
  const [results, setResults] = useState<PersonalizedExchange[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const abortController = useRef<AbortController | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    loadProfiles(userId, controller.signal)
      .then((loaded) => {
        setProfiles(loaded);
        const selected = loaded.find((profile) => profile.id === selectedID) ?? loaded[0] ?? null;
        setSelectedID(selected?.id ?? "");
        setDraft(selected ? cloneProfile(selected) : null);
      })
      .catch((caughtError: unknown) => setError(errorMessage(caughtError)))
      .finally(() => setIsLoading(false));
    return () => controller.abort();
  }, [userId]);

  function selectProfile(profile: UserProfile) {
    setSelectedID(profile.id);
    setDraft(cloneProfile(profile));
  }

  async function persistProfile() {
    if (!draft || isSaving) return;
    setIsSaving(true);
    setError(null);
    try {
      const saved = await saveProfile(draft);
      setProfiles((current) => current.map((profile) => profile.id === saved.id ? saved : profile));
      setDraft(cloneProfile(saved));
    } catch (caughtError) {
      setError(errorMessage(caughtError));
    } finally {
      setIsSaving(false);
    }
  }

  async function compare(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalized = prompt.trim();
    if (!normalized || profiles.length < 2 || isLoading) return;
    abortController.current?.abort();
    const controller = new AbortController();
    abortController.current = controller;
    setIsLoading(true);
    setError(null);
    setResults([]);
    try {
      const exchanges = await Promise.all(profiles.slice(0, 2).map((profile) =>
        sendPersonalizedMessage(profileScope(userId, profile.id), profile.id, normalized, controller.signal),
      ));
      setResults(exchanges);
    } catch (caughtError) {
      if (!(caughtError instanceof DOMException && caughtError.name === "AbortError")) setError(errorMessage(caughtError));
    } finally {
      if (abortController.current === controller) setIsLoading(false);
    }
  }

  return (
    <main className="page-shell profile-page">
      <nav className="topbar" aria-label="Навигация">
        <a className="brand" href="#day-12"><span className="brand-mark">AI</span><span>Challenge</span></a>
        <div className="day-switcher" aria-label="Выбор задания">
          <a href="#day-02">День 2</a><a href="#day-11">День 11</a><a className="selected" href="#day-12">День 12</a><a href="#day-13">День 13</a>
        </div>
        <a className="github-link" href="https://github.com/eugeneappledev-source/AI-Challenge" target="_blank" rel="noreferrer">GitHub <span>↗</span></a>
      </nav>

      <section className="profile-hero">
        <div>
          <p className="eyebrow">День 12 · Персонализация</p>
          <h1>Память знает факты.<br /><em>Профиль задаёт работу.</em></h1>
          <p className="hero-description">Один запрос и одна память проходят через разные конфигурации: стиль, формат, ограничения и собственный pipeline навыков.</p>
        </div>
        <aside className="profile-formula">
          <span>Ответ</span><strong>=</strong><div><b>Профиль</b><i>оркестрация</i></div><strong>+</strong><div><b>Память</b><i>накопленные факты</i></div><strong>+</strong><div><b>Запрос</b><i>текущая задача</i></div>
        </aside>
      </section>

      <section className="profile-workspace">
        <div className="section-heading"><div><p className="eyebrow">Конфигурация пользователя</p><h2>Выберите и настройте профиль</h2></div><p>Профиль сохраняется отдельно от памяти и автоматически подключается к каждому запросу.</p></div>

        {isLoading && profiles.length === 0 ? <LoadingProfile text="Загружаю профили…" /> : (
          <div className="profile-layout">
            <div className="profile-selector">
              {profiles.map((profile) => <button type="button" className={profile.id === selectedID ? "selected" : ""} onClick={() => selectProfile(profile)} key={profile.id}>
                <span>{profile.id === "engineer" ? "⌘" : "◉"}</span><div><strong>{profile.name}</strong><small>{pipelineTitle(profile.pipelineId)}</small></div><i>→</i>
              </button>)}
              <div className="profile-separation-note"><strong>Профиль ≠ память</strong><p>Здесь хранится способ работы агента. Факты по-прежнему живут в слоях Дня 11.</p></div>
            </div>
            {draft && <ProfileEditor profile={draft} onChange={setDraft} onSave={persistProfile} isSaving={isSaving} />}
          </div>
        )}

        {error && <div className="error-card" role="alert"><span>!</span><div><strong>Операция не выполнена</strong><p>{error}</p></div></div>}

        <form className="profile-experiment" onSubmit={compare}>
          <div className="profile-experiment-heading"><div><p className="eyebrow">A/B-проверка</p><h2>Один запрос — два профиля</h2></div><span>Одинаковая модель · одинаковая память</span></div>
          <textarea value={prompt} onChange={(event) => setPrompt(event.target.value)} rows={4} maxLength={4000} disabled={isLoading} />
          <div className="profile-experiment-footer"><p>Compass применит pipeline каждого профиля и покажет промежуточные навыки.</p><button type="submit" disabled={!prompt.trim() || isLoading || profiles.length < 2}>{isLoading && profiles.length > 0 ? <><span className="spinner" /> Выполняю два pipeline…</> : <>Сравнить профили <span>→</span></>}</button></div>
        </form>

        {isLoading && profiles.length > 0 && <LoadingProfile text="Два профиля независимо обрабатывают одинаковый запрос…" />}
        {results.length > 0 && !isLoading && <ComparisonResults results={results} />}
      </section>
      <footer><p>AI Advent Challenge #9 · День 12</p><p>Profiles · Skills · Memory · DeepSeek</p></footer>
    </main>
  );
}

function ProfileEditor({ profile, onChange, onSave, isSaving }: { profile: UserProfile; onChange: (profile: UserProfile) => void; onSave: () => void; isSaving: boolean }) {
  const update = (patch: Partial<UserProfile>) => onChange({ ...profile, ...patch });
  return <section className="profile-editor">
    <header><div><p>Редактирование</p><h3>{profile.name}</h3></div><span>{profile.id}</span></header>
    <div className="profile-fields">
      <label><span>Название</span><input value={profile.name} onChange={(event) => update({ name: event.target.value })} /></label>
      <label><span>Как обращаться</span><input value={profile.address} onChange={(event) => update({ address: event.target.value })} /></label>
      <label className="wide"><span>Стиль</span><textarea rows={2} value={profile.style} onChange={(event) => update({ style: event.target.value })} /></label>
      <label className="wide"><span>Формат</span><input value={profile.responseFormat} onChange={(event) => update({ responseFormat: event.target.value })} /></label>
      <label className="wide"><span>Ограничения · по одному на строку</span><textarea rows={4} value={profile.constraints.join("\n")} onChange={(event) => update({ constraints: event.target.value.split("\n") })} /></label>
      <label className="wide"><span>Pipeline навыков</span><select value={profile.pipelineId} onChange={(event) => update({ pipelineId: event.target.value as UserProfile["pipelineId"] })}><option value="engineering_review">Requirements → Architecture → Answer</option><option value="product_discovery">User Value → Priorities → Answer</option></select></label>
    </div>
    <button className="save-profile" type="button" onClick={onSave} disabled={isSaving}>{isSaving ? "Сохраняю…" : "Сохранить профиль"}</button>
  </section>;
}

function ComparisonResults({ results }: { results: PersonalizedExchange[] }) {
  return <section className="profile-results"><div className="profile-results-heading"><p className="eyebrow">Результат</p><h2>Как профиль изменил работу агента</h2></div><div className="profile-results-grid">{results.map((result) => <article className={`profile-answer ${result.profile.id}`} key={result.profile.id}>
    <header><div><span>{result.profile.id === "engineer" ? "⌘" : "◉"}</span><div><strong>{result.profile.name}</strong><small>{result.profile.style}</small></div></div><b>{result.usage.totalTokens} токенов</b></header>
    <div className="pipeline-view"><p>Pipeline</p>{result.skills.map((run, index) => <details key={run.skill.id}><summary><span>{index + 1}</span><div><strong>{run.skill.name}</strong><small>{run.skill.description}</small></div><i>✓</i></summary><p>{run.output}</p></details>)}</div>
    <div className="personalized-output"><p>Финальный ответ</p><div className="markdown-answer"><ReactMarkdown>{result.answer}</ReactMarkdown></div></div>
    <div className="profile-answer-meta"><span>Профиль подключён</span><span>{result.memory.working.length} task facts</span><span>{result.memory.longTerm.length} user facts</span></div>
  </article>)}</div></section>;
}

function LoadingProfile({ text }: { text: string }) { return <div className="memory-loading"><span className="spinner" /><div><strong>{text}</strong><p>Профиль, память и навыки собираются на backend.</p></div></div>; }
function cloneProfile(profile: UserProfile): UserProfile { return { ...profile, constraints: [...profile.constraints] }; }
function pipelineTitle(id: UserProfile["pipelineId"]): string { return id === "engineering_review" ? "Requirements → Architecture" : "Value → Priorities"; }
function getUserId(): string { const key = "ai-challenge-memory-user"; let value = localStorage.getItem(key); if (!value) { value = crypto.randomUUID(); localStorage.setItem(key, value); } return value; }
function profileScope(userId: string, profileId: string): MemoryScope { return { userId, taskId: `${userId}:pulse-plan`, sessionId: `${userId}:day12:${profileId}` }; }
function errorMessage(error: unknown): string { return error instanceof APIError || error instanceof Error ? error.message : "Произошла неизвестная ошибка."; }

export default ProfileLab;
