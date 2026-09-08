import { useState } from 'react'
import { useHome } from './useHome'
import Dashboard from './Dashboard'
import Scenarios from './Scenarios'

export default function Home({ onUnauthorized }) {
  const [tab, setTab] = useState('devices')
  const { home, error, reload } = useHome(onUnauthorized)

  return (
    <div className="app">
      <header>
        <h1>Умный дом</h1>
        <nav>
          <button className={tab === 'devices' ? 'active' : ''} onClick={() => setTab('devices')}>
            Устройства
          </button>
          <button className={tab === 'scenarios' ? 'active' : ''} onClick={() => setTab('scenarios')}>
            Сценарии
          </button>
          <button onClick={reload} title="Обновить">
            ↻
          </button>
        </nav>
      </header>
      {error && <p className="error">{error}</p>}
      {!home ? (
        <p className="muted center">Загружаем устройства…</p>
      ) : tab === 'devices' ? (
        <Dashboard home={home} reload={reload} onUnauthorized={onUnauthorized} />
      ) : (
        <Scenarios home={home} onUnauthorized={onUnauthorized} />
      )}
    </div>
  )
}
