import { useCallback, useEffect, useState } from 'react'
import { api } from './api'
import { actionErrors } from './labels'
import MacroEditor from './MacroEditor'

export default function Scenarios({ home, onUnauthorized }) {
  const [macros, setMacros] = useState([])
  const [editing, setEditing] = useState(null) // null | {} для нового | существующий макрос
  const [status, setStatus] = useState({}) // id → текст результата запуска
  const [error, setError] = useState('')

  const loadMacros = useCallback(
    () =>
      api
        .macros()
        .then(setMacros)
        .catch((e) => setError(e.message)),
    [],
  )
  useEffect(() => {
    loadMacros()
  }, [loadMacros])

  // run запускает сценарий/макрос и пишет результат рядом с кнопкой.
  function run(id, promise) {
    setStatus((s) => ({ ...s, [id]: 'Запускаем…' }))
    promise
      .then((resp) => {
        const errs = actionErrors(resp)
        setStatus((s) => ({ ...s, [id]: errs.length ? 'Ошибка: ' + errs.join('; ') : 'Выполнено ✓' }))
      })
      .catch((e) => {
        if (e.status === 401) onUnauthorized()
        setStatus((s) => ({ ...s, [id]: 'Ошибка: ' + e.message }))
      })
  }

  function remove(id) {
    api
      .deleteMacro(id)
      .then(loadMacros)
      .catch((e) => setError(e.message))
  }

  const scenarios = home.scenarios || []

  return (
    <>
      <section className="room">
        <h2>Сценарии Яндекса</h2>
        {!scenarios.length ? (
          <p className="muted">Сценариев нет — их создают в приложении «Дом с Алисой».</p>
        ) : (
          <div className="list">
            {scenarios.map((s) => (
              <div className="card item" key={s.id}>
                <span>{s.name}</span>
                <span className="row">
                  <span className="muted">{status[s.id]}</span>
                  <button className="primary" onClick={() => run(s.id, api.runScenario(s.id))}>
                    Запустить
                  </button>
                </span>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="room">
        <div className="row between">
          <h2>Мои макросы</h2>
          {!editing && <button onClick={() => setEditing({})}>+ Новый макрос</button>}
        </div>
        {error && <p className="error">{error}</p>}
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
        {!macros.length && !editing && <p className="muted">Макросов пока нет.</p>}
        <div className="list">
          {macros.map((m) => (
            <div className="card item" key={m.id}>
              <span>
                <strong>{m.name}</strong> <span className="muted">· действий: {m.actions.length}</span>
              </span>
              <span className="row">
                <span className="muted">{status[m.id]}</span>
                <button className="primary" onClick={() => run(m.id, api.runMacro(m.id))}>
                  Запустить
                </button>
                <button onClick={() => setEditing(m)}>Изменить</button>
                <button className="danger" onClick={() => remove(m.id)}>
                  Удалить
                </button>
              </span>
            </div>
          ))}
        </div>
      </section>
    </>
  )
}
