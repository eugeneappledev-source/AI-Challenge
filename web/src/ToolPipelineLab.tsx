import { FormEvent, useRef, useState } from "react";
import { APIError, runBusinessPipeline } from "./api";
import type { BusinessPipelineResult } from "./types";
import { RadarNav } from "./BusinessResearchLab";

const example = "Исследуй свежие AI-инструменты для малого бизнеса, выдели ключевые сигналы и сохрани отчёт.";

export default function ToolPipelineLab() {
  const [request,setRequest]=useState("");
  const [result,setResult]=useState<BusinessPipelineResult|null>(null);
  const [pending,setPending]=useState(false);
  const [error,setError]=useState<string|null>(null);
  const abortRef=useRef<AbortController|null>(null);
  async function submit(event:FormEvent){
    event.preventDefault(); if(!request.trim()||pending)return;
    abortRef.current?.abort();const controller=new AbortController();abortRef.current=controller;
    setPending(true);setError(null);setResult(null);
    try{setResult(await runBusinessPipeline(request.trim(),controller.signal));}
    catch(caught){if(!(caught instanceof DOMException&&caught.name==="AbortError"))setError(caught instanceof APIError||caught instanceof Error?caught.message:"Pipeline не выполнен.");}
    finally{if(abortRef.current===controller)setPending(false);}
  }
  return <main className="radar-page pipeline-page">
    <RadarNav active={19}/>
    <header className="radar-hero pipeline-hero"><div><p className="radar-kicker">AI Advent Challenge · День 19</p><h1>Цепочка<br/><em>tools</em></h1><p>Агент видит несколько инструментов одного MCP-сервера, сам строит порядок вызовов и передаёт структурированный результат между этапами.</p><div className="radar-tags"><span>3 MCP tools</span><span>Automatic plan</span><span>Data handoff</span></div></div><aside className="pipeline-preview">{["SEARCH","SUMMARIZE","SAVE"].map((item,index)=><div key={item}><b>0{index+1}</b><span>{item}</span>{index<2&&<i>↓</i>}</div>)}</aside></header>
    <section className="radar-workspace">
      <div className="radar-heading"><div><span>01 / TASK</span><h2>Одна задача — три инструмента</h2></div><p className="research-note">Кнопка запускает агента, а не жёстко заданные browser-вызовы.</p></div>
      <form className="research-composer" onSubmit={submit}><textarea value={request} onChange={e=>setRequest(e.target.value)} placeholder="Опишите исследовательскую задачу…" disabled={pending}/><footer><button type="button" className="pipeline-example" onClick={()=>setRequest(example)} disabled={pending}>Подставить пример</button><button disabled={!request.trim()||pending}>{pending?<><i className="radar-spinner"/> Агент строит цепочку…</>:"Запустить pipeline →"}</button></footer></form>
      {error&&<div className="radar-error"><b>Цепочка не выполнена</b><p>{error}</p></div>}
      {pending&&<div className="research-loading"><i className="radar-spinner"/><div><b>tools/list → plan → tool/call × 3</b><p>Каждый следующий tool получает результат предыдущего.</p></div></div>}
      {result&&<section className="pipeline-result">
        <div className="pipeline-plan"><span>AGENT PLAN</span><h2>{result.plan.tools.join(" → ")}</h2><p>{result.plan.rationale}</p></div>
        <div className="research-section-heading"><span>02 / EXECUTION</span><h2>Живой handoff</h2><p>Вход и выход каждого шага видны отдельно.</p></div>
        <div className="pipeline-stages">{result.stages.map(stage=><article key={stage.order}><header><b>0{stage.order}</b><span>{stage.durationMs} ms</span></header><small>{stage.server}</small><h3>{stage.tool}</h3><dl><div><dt>INPUT</dt><dd>{stage.inputSummary}</dd></div><div><dt>OUTPUT</dt><dd>{stage.outputSummary}</dd></div></dl></article>)}</div>
        <div className="research-section-heading"><span>03 / RESULT</span><h2>Сводка сохранена</h2><p>{result.savedReport.id} · {result.savedReport.sourceCount} источников</p></div>
        <article className="pipeline-brief"><p>{result.brief.summary}</p><div><section><span>Сигналы</span><ul>{result.brief.keySignals.map(item=><li key={item}>{item}</li>)}</ul></section><section><span>Риски</span><ul>{result.brief.risks.map(item=><li key={item}>{item}</li>)}</ul></section></div><footer><b>Следующий вопрос</b><p>{result.brief.nextQuestion}</p><small>{result.usage.totalTokens} токенов · {result.brief.model}</small></footer></article>
      </section>}
    </section><footer className="radar-footer"><span>Startup & Business Radar</span><span>One MCP server · Three tools · Automatic chain</span></footer>
  </main>;
}
