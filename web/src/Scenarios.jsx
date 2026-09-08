import { useCallback, useEffect, useState } from 'react'
import { api } from './api'
import { actionErrors, actionsCount } from './labels'
import { Icon } from './icons'
import MacroEditor from './MacroEditor'

// Сценариев Яндекса бывает много: сначала показываем первые VISIBLE.
const VISIBLE = 12

export default function Scenarios({ home, onUnauthorized }) {
  const [macros, setMacros] = useState([])
  const [editing, setEditing] = useState(null) // null | {} для нового | существующий макрос
  const [status, setStatus] = useState({}) // id → {kind, text} результата запуска
  const [error, setError] = useState('')
  const [expanded, setExpanded] = useState(false)

  const loadMacros = useCallback(
    () =>
      api
        .macros()
        .then(setMacros)
        .catch((e) => setError(e.message)),
    [],
  )
  // Перечитываем макросы при каждом изменении home (после опроса), чтобы макросы,
  // созданные Claude через API, появлялись без переключения вкладок.
  useEffect(() => {
    loadMacros()
  }, [loadMacros, home])

  // run запускает сценарий/макрос и пишет результат бейджем на плитке.
  function run(id, promise) {
    setStatus((s) => ({ ...s, [id]: { kind: 'busy', text: 'Запускаем…' } }))
    promise
      .then((resp) => {
        const errs = actionErrors(resp, (i) => home.devices?.find((d) => d.id === i)?.name)
        setStatus((s) => ({
          ...s,
          [id]: errs.length ? { kind: 'err', text: errs.join('; ') } : { kind: 'ok', text: 'Выполнено' },
        }))
      })
      .catch((e) => {
        if (e.status === 401) onUnauthorized()
        setStatus((s) => ({ ...s, [id]: { kind: 'err', text: e.message } }))
      })
  }

  function remove(id) {
    api
      .deleteMacro(id)
      .then(loadMacros)
      .catch((e) => setError(e.message))
  }

  const scenarios = home.scenarios || []
  const shown = expanded ? scenarios : scenarios.slice(0, VISIBLE)
  const hidden = scenarios.length - shown.length

  return (
    <>
      <section className="room">
        <div className="room-head">
          <h2 className="display">Сценарии Яндекса</h2>
          <span className="sub">{scenarios.length} · создаются в приложении «Дом с Алисой»</span>
          {hidden > 0 && (
            <button className="chip head-side" onClick={() => setExpanded(true)}>
              Показать ещё {hidden}
            </button>
          )}
        </div>
        {!scenarios.length ? (
          <div className="banner info">Сценариев нет — их создают в приложении «Дом с Алисой».</div>
        ) : (
          <div className="grid">
            {shown.map((s) => (
              <div className="tile scenario" key={s.id}>
                <div className="top">
                  <div className="name">{s.name}</div>
                  <button
                    className="icon-btn sm inset"
                    onClick={() => run(s.id, api.runScenario(s.id))}
                    title="Запустить"
                    aria-label={`Запустить сценарий «${s.name}»`}
                  >
                    <Icon name="PLAY" size={16} />
                  </button>
                </div>
                <div>
                  <StatusBadge status={status[s.id]} />
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="room">
        <div className="room-head">
          <h2 className="display">Мои макросы</h2>
          <span className="sub">хранятся локально</span>
          {!editing && (
            <button className="btn head-side" onClick={() => setEditing({})}>
              <Icon name="PLUS" size={16} /> Новый макрос
            </button>
          )}
        </div>

        {error && <div className="banner">{error}</div>}

        {editing && (
          <MacroEditor
            home={home}
            macro={editing}
            onSaved={() => {
              setEditing(null)
              loadMacros()
            }}
            onCancel={() => setEditing(null)}
          />
        )}

        <div className="grid macros">
          {macros.map((m) => (
            <div className="tile macro" key={m.id}>
              <div className="top">
                <div>
                  <div className="name">{m.name}</div>
                  <div className="state">{macroSummary(m, home)}</div>
                </div>
                <StatusBadge status={status[m.id]} />
              </div>
              <div className="row">
                <button className="btn primary" onClick={() => run(m.id, api.runMacro(m.id))}>
                  <Icon name="PLAY" size={16} /> Запустить
                </button>
                <button
                  className="icon-btn inset"
                  onClick={() => setEditing(m)}
                  title="Изменить"
                  aria-label={`Изменить макрос «${m.name}»`}
                >
                  <Icon name="PEN" size={16} />
                </button>
                <button
                  className="icon-btn inset danger"
                  onClick={() => remove(m.id)}
                  title="Удалить"
                  aria-label={`Удалить макрос «${m.name}»`}
                >
                  <Icon name="TRASH" size={16} />
                </button>
              </div>
            </div>
          ))}
          {!editing && (
            <div className="tile hint">
              <div className="name">Макросы собирает Claude</div>
              <div className="state">
                Напишите в терминале: «сделай сценарий кино — выключи большой свет, торшер на 30 %» — макрос появится
                здесь.
              </div>
            </div>
          )}
        </div>
      </section>
    </>
  )
}

function StatusBadge({ status }) {
  if (!status) return null
  if (status.kind === 'ok') {
    return (
      <span className="badge ok">
        {status.text} <Icon name="CHECK" size={12} stroke={2.4} />
      </span>
    )
  }
  return <span className={`badge ${status.kind === 'busy' ? 'busy' : 'err'}`}>{status.text}</span>
}

// macroSummary: «4 действия · Большой свет, Торшер, Телевизор» (до трёх имён).
function macroSummary(macro, home) {
  const names = []
  for (const a of macro.actions || []) {
    const name = home.devices?.find((d) => d.id === a.device_id)?.name
    if (name && !names.includes(name)) names.push(name)
  }
  const list = names.slice(0, 3).join(', ') + (names.length > 3 ? '…' : '')
  return [actionsCount((macro.actions || []).length), list].filter(Boolean).join(' · ')
}
