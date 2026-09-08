import { useCallback, useEffect, useState } from 'react'
import { api } from './api'
import Login from './Login'
import Home from './Home'

export default function App() {
  const [auth, setAuth] = useState(null) // null — ещё не проверяли

  const refreshAuth = useCallback(() => {
    api
      .authStatus()
      .then(setAuth)
      .catch((e) => setAuth({ authorized: false, login_url: '', error: e.message }))
  }, [])

  useEffect(() => {
    refreshAuth()
  }, [refreshAuth])

  if (auth === null) return <p className="muted center">Загрузка…</p>
  if (!auth.authorized) return <Login loginUrl={auth.login_url} error={auth.error} onLoggedIn={refreshAuth} />
  return <Home onUnauthorized={refreshAuth} />
}
