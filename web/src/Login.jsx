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

  return (
    <div className="login card">
      <h1>Вход через Яндекс</h1>
      <ol>
        <li>
          <a href={loginUrl} target="_blank" rel="noreferrer">
            Открыть страницу Яндекса
          </a>{' '}
          и разрешить доступ.
        </li>
        <li>Скопировать показанный код подтверждения и вставить его сюда.</li>
      </ol>
      <form onSubmit={submit} className="row">
        <input
          value={code}
          onChange={(e) => setCode(e.target.value)}
          placeholder="Код подтверждения"
          autoFocus
        />
        <button type="submit" disabled={busy || !code.trim()}>
          {busy ? 'Проверяем…' : 'Войти'}
        </button>
      </form>
      {error && <p className="error">{error}</p>}
    </div>
  )
}
