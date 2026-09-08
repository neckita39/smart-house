import { useMemo, useState } from 'react'
import { api } from './api'
import { useHome } from './useHome'
import { actionErrors } from './labels'
import { Icon } from './icons'
import Dashboard, { groupByRoom } from './Dashboard'
import Scenarios from './Scenarios'

const ROOM_KEY = 'smart-house.room'
const ALL = '_all'
const ON_OFF = 'devices.capabilities.on_off'
const LIGHT = 'devices.types.light'

// Выбранная комната переживает перезагрузку страницы; приватный режим Safari
// умеет бросать на localStorage, поэтому обе операции защищены.
function readRoom() {
  try {
    return localStorage.getItem(ROOM_KEY) || ''
  } catch {
    return ''
  }
}
function saveRoom(id) {
  try {
    localStorage.setItem(ROOM_KEY, id)
  } catch {
    /* не критично: комната просто не запомнится */
  }
}

export default function Home({ onUnauthorized }) {
  const [tab, setTab] = useState('devices')
  const { home, error, reload } = useHome(onUnauthorized)
  const [room, setRoom] = useState(readRoom)
  const [actionError, setActionError] = useState('')
  const [busy, setBusy] = useState(false)

  const rooms = useMemo(() => (home ? groupByRoom(home) : []), [home])
  // Чипы показывают все комнаты дома, даже пустые: в пустой видно пустое состояние.
  const chips = useMemo(() => {
    if (!home) return []
    const byId = new Map(rooms.map((r) => [r.id, r]))
    const list = (home.rooms || []).map((r) => ({
      id: r.id,
      name: r.name,
      count: byId.get(r.id)?.devices.length || 0,
    }))
    const none = byId.get('_none')
    if (none) list.push({ id: none.id, name: none.name, count: none.devices.length })
    return list
  }, [home, rooms])

  // По умолчанию — первая комната дома; забытая или исчезнувшая комната сбрасывается.
  const known = room === ALL || chips.some((c) => c.id === room)
  const selected = known ? room : chips[0]?.id ?? ALL
  const visible = selected === ALL ? rooms : rooms.filter((r) => r.id === selected)

  function pickRoom(id) {
    setRoom(id)
    saveRoom(id)
  }

  const lights = visible.flatMap((r) =>
    r.devices.filter(
      (d) => d.type?.startsWith(LIGHT) && (d.capabilities || []).some((c) => c.type === ON_OFF),
    ),
  )

  // lightsOff гасит один запросом только свет выбранной комнаты (или всех комнат).
  async function lightsOff() {
    if (!lights.length) return
    setBusy(true)
    setActionError('')
    try {
      const resp = await api.deviceActions(
        lights.map((d) => ({ id: d.id, actions: [{ type: ON_OFF, state: { instance: 'on', value: false } }] })),
      )
      const errors = actionErrors(resp, (id) => home.devices.find((d) => d.id === id)?.name)
      if (errors.length) setActionError(errors.join('; '))
    } catch (e) {
      if (e.status === 401) onUnauthorized()
      else setActionError(e.message)
    } finally {
      setBusy(false)
      reload()
    }
  }

  const devicesTab = tab === 'devices'

  return (
    <div className="app">
      <header className="app-head">
        <div className="brand display">Умный дом</div>
        <nav className="tabs">
          <button className={`tab${devicesTab ? ' active' : ''}`} onClick={() => setTab('devices')}>
            Устройства
          </button>
          <button className={`tab${devicesTab ? '' : ' active'}`} onClick={() => setTab('scenarios')}>
            Сценарии
          </button>
        </nav>
        <div className="head-actions">
          {devicesTab && (
            <button className={`btn${busy || !lights.length ? ' disabled' : ''}`} disabled={busy || !lights.length} onClick={lightsOff}>
              Выключить свет
            </button>
          )}
          <button className="icon-btn" onClick={reload} title="Обновить" aria-label="Обновить">
            <Icon name="REFRESH" size={18} />
          </button>
        </div>
      </header>

      {devicesTab && home && (
        <div className="chips">
          {chips.map((c) => (
            <button key={c.id} className={`chip${selected === c.id ? ' active' : ''}`} onClick={() => pickRoom(c.id)}>
              {c.name} <span className="count">{c.count}</span>
            </button>
          ))}
          <button className={`chip${selected === ALL ? ' active' : ''}`} onClick={() => pickRoom(ALL)}>
            Все <span className="count">{(home.devices || []).length}</span>
          </button>
        </div>
      )}

      {error && <div className="banner">{error}</div>}
      {actionError && <div className="banner">{actionError}</div>}

      {!home ? (
        <div className="placeholder">Загружаем устройства…</div>
      ) : devicesTab ? (
        <Dashboard rooms={visible} reload={reload} onUnauthorized={onUnauthorized} />
      ) : (
        <Scenarios home={home} onUnauthorized={onUnauthorized} />
      )}
    </div>
  )
}
