import { useState } from 'react'
import { api } from './api'

export default function Login({ loginUrl, error: initialError, onLoggedIn }) {
  const [code, setCode] = useState('')
  const [error, setError] = useState(initialError || '')
  const [busy, setBusy] = useState(false)

  async function submit(e) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await api.login(code.trim())
      onLoggedIn()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  const empty = !code.trim()

  return (
    <div className="login">
      <div className="tile">
        <div>
          <h1>Умный дом</h1>
          <div className="sub">Вход через Яндекс — один раз, дальше помним</div>
        </div>

        <div className="steps">
          <div className="step">
            <div className="num">1</div>
            <div className="txt">
              <a href={loginUrl} target="_blank" rel="noreferrer">
                Открыть страницу Яндекса
              </a>{' '}
              и разрешить доступ
            </div>
          </div>
          <div className="step">
            <div className="num">2</div>
            <div className="txt">Скопировать код подтверждения и вставить сюда</div>
          </div>
        </div>

        <form onSubmit={submit}>
          <label className="input">
            <input
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="Код подтверждения"
              autoFocus
              aria-label="Код подтверждения"
            />
          </label>
          <button type="submit" className={`btn primary${busy || empty ? ' disabled' : ''}`} disabled={busy || empty}>
            {busy ? 'Проверяем…' : 'Войти'}
          </button>
        </form>

        {error && <div className="banner">{error}</div>}
      </div>
    </div>
  )
}
