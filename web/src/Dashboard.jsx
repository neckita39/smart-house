import DeviceCard from './DeviceCard'

// groupByRoom раскладывает устройства по комнатам; без комнаты — в «Без комнаты».
export function groupByRoom(home) {
  const byId = new Map((home.devices || []).map((d) => [d.id, d]))
  const rooms = (home.rooms || []).map((r) => ({
    id: r.id,
    name: r.name,
    devices: (r.devices || []).map((id) => byId.get(id)).filter(Boolean),
  }))
  const placed = new Set(rooms.flatMap((r) => r.devices.map((d) => d.id)))
  const rest = (home.devices || []).filter((d) => !placed.has(d.id))
  if (rest.length) rooms.push({ id: '_none', name: 'Без комнаты', devices: rest })
  return rooms.filter((r) => r.devices.length)
}

export default function Dashboard({ home, reload, onUnauthorized }) {
  const rooms = groupByRoom(home)
  if (!rooms.length) {
    return <p className="muted center">Устройств нет. Добавьте их в приложении «Дом с Алисой».</p>
  }
  return rooms.map((room) => (
    <section className="room" key={room.id}>
      <h2>{room.name}</h2>
      <div className="devices">
        {room.devices.map((d) => (
          <DeviceCard key={d.id} device={d} reload={reload} onUnauthorized={onUnauthorized} />
        ))}
      </div>
    </section>
  ))
}
