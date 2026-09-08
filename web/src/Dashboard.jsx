import DeviceCard from './DeviceCard'
import { devicesCount } from './labels'

// groupByRoom раскладывает устройства по комнатам; без комнаты — в «Без комнаты».
// Пустые комнаты отбрасываются, а непустые сортируются по имени (стабильный порядок:
// Яндекс отдаёт комнаты в разном порядке на каждый опрос), «Без комнаты» — всегда последняя.
export function groupByRoom(home) {
  const byId = new Map((home.devices || []).map((d) => [d.id, d]))
  const rooms = (home.rooms || []).map((r) => ({
    id: r.id,
    name: r.name,
    devices: (r.devices || []).map((id) => byId.get(id)).filter(Boolean),
  }))
  const placed = new Set(rooms.flatMap((r) => r.devices.map((d) => d.id)))
  const rest = (home.devices || []).filter((d) => !placed.has(d.id))
  const result = rooms.filter((r) => r.devices.length)
  result.sort((a, b) => a.name.localeCompare(b.name, 'ru'))
  if (rest.length) result.push({ id: '_none', name: 'Без комнаты', devices: rest })
  return result
}

const isOn = (d) =>
  (d.capabilities || []).some((c) => c.type.split('.').pop() === 'on_off' && !!c.state?.value)

export default function Dashboard({ rooms, reload, onUnauthorized }) {
  if (!rooms.length) {
    return <div className="banner info">Здесь пока нет устройств. Добавьте их в приложении «Дом с Алисой».</div>
  }
  return rooms.map((room) => (
    <section className="room" key={room.id}>
      <div className="room-head">
        <h2 className="display">{room.name}</h2>
        <span className="sub">
          {devicesCount(room.devices.length)} · {room.devices.filter(isOn).length} включено
        </span>
      </div>
      <div className="grid">
        {room.devices.map((d) => (
          <DeviceCard key={d.id} device={d} reload={reload} onUnauthorized={onUnauthorized} />
        ))}
      </div>
    </section>
  ))
}
