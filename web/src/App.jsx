import { useCallback, useEffect, useState } from 'react'
import { api } from './api'
import Login from './Login'
import Home from './Home'

export default function App() {
  const [auth, setAuth] = useState(null) // null — ещё не проверяли

  // refreshAuth перепроверяет статус на сервере. forceLogout=true — вызвано после
  // 401 от API: даже если сервер (из-за гонки) всё ещё ответит authorized:true,
  // всё равно уходим на форму входа.
  const refreshAuth = useCallback((forceLogout) => {
    api
      .authStatus()
      .then((a) => setAuth(forceLogout ? { ...a, authorized: false } : a))
      .catch((e) => setAuth({ authorized: false, login_url: '', error: e.message }))
  }, [])

  const onUnauthorized = useCallback(() => refreshAuth(true), [refreshAuth])

  useEffect(() => {
    refreshAuth(false)
  }, [refreshAuth])

  if (auth === null) return <div className="loader">Загрузка…</div>
  if (!auth.authorized) return <Login loginUrl={auth.login_url} error={auth.error} onLoggedIn={refreshAuth} />
  return <Home onUnauthorized={onUnauthorized} />
}
