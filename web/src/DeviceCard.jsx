import { useState } from 'react'
import { api } from './api'
import { actionErrors, typeLabel } from './labels'
import { Capability, PropertyReadout } from './Controls'

export default function DeviceCard({ device, reload, onUnauthorized }) {
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  // send отправляет одно действие и после ответа перечитывает состояние дома,
  // чтобы карточка показала фактическое состояние устройства. Возвращает true при
  // успехе и false при ошибке — контролы откатывают оптимистичное значение при false.
  async function send(type, instance, value) {
    setBusy(true)
    setError('')
    try {
      const resp = await api.deviceActions([{ id: device.id, actions: [{ type, state: { instance, value } }] }])
      const errors = actionErrors(resp)
      if (errors.length) {
        setError(errors.join('; '))
        return false
      }
      return true
    } catch (e) {
      if (e.status === 401) onUnauthorized()
      else setError(e.message)
      return false
    } finally {
      setBusy(false)
      reload()
    }
  }

  const caps = device.capabilities || []
  const props = device.properties || []

  return (
    <div className="card device">
      <div className="row between">
        <strong>{device.name}</strong>
        <span className="type">{typeLabel(device.type)}</span>
      </div>
      {caps.map((cap, i) => (
        <Capability key={i} cap={cap} busy={busy} onChange={send} />
      ))}
      {props.length > 0 && (
        <div className="props">
          {props.map((p, i) => (
            <PropertyReadout key={i} prop={p} />
          ))}
        </div>
      )}
      {error && <p className="error">{error}</p>}
    </div>
  )
}
