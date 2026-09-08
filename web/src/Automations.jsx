import { useCallback, useEffect, useState } from 'react'
import { api } from './api'
import { Icon } from './icons'
import { Switch } from './Controls'
import { describeRule } from './describeRule'

function fmtTime(iso) {
  if (!iso) return 'ещё не срабатывало'
  const d = new Date(iso)
  return d.toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
}

export default function Automations({ home, onUnauthorized }) {
  const [rules, setRules] = useState([])
  const [events, setEvents] = useState([])
  const [checks, setChecks] = useState({}) // id → результат /check
  const [error, setError] = useState('')

  const deviceName = useCallback(
    (id) => home?.devices?.find((d) => d.id === id)?.name || id,
    [home],
  )
  const unitOf = useCallback(
    (id, inst) => {
      const d = home?.devices?.find((x) => x.id === id)
      const p = d?.properties?.find((x) => x.parameters?.instance === inst)
      const c = d?.capabilities?.find((x) => (x.parameters?.instance || 'on') === inst)
      return p?.parameters?.unit || c?.parameters?.unit || ''
    },
    [home],
  )

  const load = useCallback(() => {
    Promise.all([api.rules(), api.events(50)])
      .then(([r, e]) => {
        setRules(r)
        setEvents(e)
        setError('')
      })
      .catch((e) => {
        if (e.status === 401) onUnauthorized()
        else setError(e.message)
      })
  }, [onUnauthorized])

  useEffect(() => {
    load()
  }, [load, home])

  // toggle возвращает true/false: Switch откатывает оптимистичное значение при false.
  const toggle = (rule) =>
    api
      .updateRule({ ...rule, enabled: !rule.enabled })
      .then(() => {
        load()
        return true
      })
      .catch((e) => {
        if (e.status === 401) onUnauthorized()
        else setError(e.message)
        return false
      })
  const remove = (rule) => api.deleteRule(rule.id).then(load).catch((e) => setError(e.message))
  const run = (rule) =>
    api.runRule(rule.id).then(load).catch((e) => setError(e.message))
  const check = (rule) =>
    api
      .checkRule(rule.id)
      .then((res) => setChecks((c) => ({ ...c, [rule.id]: res })))
      .catch((e) => setError(e.message))

  return (
    <>
      <section className="room">
        <div className="room-head">
          <h2 className="display">Автоматизации</h2>
          <span className="muted">правила проверяются каждые 10 секунд</span>
        </div>
        {error && <div className="banner">{error}</div>}
        {!rules.length && (
          <div className="tile hint">
            <div className="name">Правил пока нет</div>
            <div className="state">
              Опишите Claude в терминале: «когда в кабинете больше 25 °C днём — закрывай жалюзи до 40 %» — правило появится здесь.
            </div>
          </div>
        )}
        <div className="rules">
          {rules.map((rule) => {
            const d = describeRule(rule, deviceName, unitOf)
            const chk = checks[rule.id]
            return (
              <div className={`tile rule${rule.enabled ? '' : ' off'}`} key={rule.id}>
                <div className="top">
                  <div>
                    <div className="name">{rule.name}</div>
                    <div className="state">последний раз: {fmtTime(rule.last_fired)}</div>
                  </div>
                  <Switch text="Включено" checked={rule.enabled} onChange={() => toggle(rule)} />
                </div>
                <div className="rule-body">
                  <div className="rule-col">
                    <div className="muted small">Если</div>
                    {d.when.map((w, i) => (
                      <div className="rule-line" key={i}>
                        {chk && <span className={`dot${chk.conditions?.[i]?.ok ? ' ok' : ' no'}`}></span>}
                        <span>{w}</span>
                        {chk && chk.conditions?.[i]?.current !== undefined && chk.conditions[i].kind !== 'time' && (
                          <span className="muted"> · сейчас {String(chk.conditions[i].current)}</span>
                        )}
                      </div>
                    ))}
                    {d.extra.length > 0 && <div className="muted small">{d.extra.join(' · ')}</div>}
                  </div>
                  <div className="rule-col">
                    <div className="muted small">То</div>
                    {d.then.map((t, i) => (
                      <div className="rule-line" key={i}>{t}</div>
                    ))}
                  </div>
                </div>
                <div className="row rule-actions">
                  <button className="btn" onClick={() => check(rule)}>
                    <Icon name="CHECK" size={16} /> Проверить
                  </button>
                  <button className="btn" onClick={() => run(rule)}>
                    <Icon name="PLAY" size={16} /> Выполнить сейчас
                  </button>
                  <button className="icon-btn danger" onClick={() => remove(rule)} aria-label="Удалить правило">
                    <Icon name="TRASH" size={16} />
                  </button>
                  {chk && (
                    <span className={`badge ${chk.ok ? 'ok' : 'busy'}`}>{chk.ok ? 'условия выполнены' : 'условия не выполнены'}</span>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </section>

      <section className="room">
        <div className="room-head">
          <h2 className="display">Журнал</h2>
          <span className="muted">последние {events.length}</span>
        </div>
        {!events.length ? (
          <p className="muted">Срабатываний ещё не было.</p>
        ) : (
          <div className="events">
            {events.map((e, i) => (
              <div className="event" key={i}>
                <span className="muted mono">{fmtTime(e.time)}</span>
                <span className={`badge ${e.ok ? 'ok' : 'err'}`}>{e.ok ? 'ок' : 'ошибка'}</span>
                <span className="event-name">{e.rule_name}</span>
                <span className="muted">{e.details}</span>
              </div>
            ))}
          </div>
        )}
      </section>
    </>
  )
}
