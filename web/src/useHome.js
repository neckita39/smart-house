import { useCallback, useEffect, useState } from 'react'
import { api } from './api'

const POLL_MS = 10_000

// useHome держит актуальное состояние дома: загружает сразу и опрашивает раз в 10 с
// (пока вкладка видима). 401 отдаёт наверх — там покажут форму входа.
export function useHome(onUnauthorized) {
  const [home, setHome] = useState(null)
  const [error, setError] = useState('')

  const reload = useCallback(async () => {
    try {
      setHome(await api.home())
      setError('')
    } catch (e) {
      if (e.status === 401) onUnauthorized()
      else setError(e.message)
    }
  }, [onUnauthorized])

  useEffect(() => {
    reload()
    const timer = setInterval(() => {
      if (!document.hidden) reload()
    }, POLL_MS)
    return () => clearInterval(timer)
  }, [reload])

  return { home, error, reload }
}
