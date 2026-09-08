import { useMemo, useState } from 'react'
import { api } from './api'
import { useHome } from './useHome'
import { actionErrors } from './labels'
import { Icon } from './icons'
import Dashboard, { groupByRoom } from './Dashboard'
import Scenarios from './Scenarios'
import Automations from './Automations'

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

// filterRooms — чистая функция поиска: пустой/пробельный запрос возвращает вход как есть,
// иначе оставляет в каждой комнате устройства с совпадением по имени (регистр не важен)
// и выкидывает комнаты без совпадений.
export function filterRooms(rooms, query) {
  const q = query.trim().toLocaleLowerCase('ru')
  if (!q) return rooms
  return rooms
    .map((r) => ({ ...r, devices: r.devices.filter((d) => (d.name || '').toLocaleLowerCase('ru').includes(q)) }))
    .filter((r) => r.devices.length)
}

export default function Home({ onUnauthorized }) {
  const [tab, setTab] = useState('devices')
  const { home, error, reload } = useHome(onUnauthorized)
  const [room, setRoom] = useState(readRoom)
  const [query, setQuery] = useState('')
  const [actionError, setActionError] = useState('')
  const [busy, setBusy] = useState(false)

  // rooms уже отфильтрован (без пустых комнат) и отсортирован стабильно (groupByRoom);
  // чипы переиспользуют этот же порядок, чтобы не «прыгать» между опросами.
  const rooms = useMemo(() => (home ? groupByRoom(home) : []), [home])
  const chips = useMemo(() => rooms.map((r) => ({ id: r.id, name: r.name, count: r.devices.length })), [rooms])

  // По умолчанию — первая комната дома; забытая или исчезнувшая комната сбрасывается.
  const known = room === ALL || chips.some((c) => c.id === room)
  const selected = known ? room : chips[0]?.id ?? ALL
  const visible = selected === ALL ? rooms : rooms.filter((r) => r.id === selected)
  const filtered = useMemo(() => filterRooms(visible, query), [visible, query])

  function pickRoom(id) {
    setRoom(id)
    saveRoom(id)
    setQuery('')
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
          <button className={`tab${tab === 'scenarios' ? ' active' : ''}`} onClick={() => setTab('scenarios')}>
            Сценарии
          </button>
          <button className={`tab${tab === 'automations' ? ' active' : ''}`} onClick={() => setTab('automations')}>
            Автоматизации
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
        <div className="chips-row">
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
          <label className="input search">
            <input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Поиск по названию"
              aria-label="Поиск по названию"
            />
            {query && (
              <button
                type="button"
                className="icon-btn sm inset"
                onClick={() => setQuery('')}
                title="Очистить"
                aria-label="Очистить поиск"
              >
                <Icon name="X" size={14} />
              </button>
            )}
          </label>
        </div>
      )}

      {error && <div className="banner">{error}</div>}
      {actionError && <div className="banner">{actionError}</div>}

      {!home ? (
        <div className="placeholder">Загружаем устройства…</div>
      ) : devicesTab ? (
        query.trim() && !filtered.length ? (
          <div className="banner info">Ничего не найдено по «{query.trim()}»</div>
        ) : (
          <Dashboard rooms={filtered} reload={reload} onUnauthorized={onUnauthorized} />
        )
      ) : tab === 'scenarios' ? (
        <Scenarios home={home} onUnauthorized={onUnauthorized} />
      ) : (
        <Automations home={home} onUnauthorized={onUnauthorized} />
      )}
    </div>
  )
}
