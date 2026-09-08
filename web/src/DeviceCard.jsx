import { useState } from 'react'
import { api } from './api'
import { actionErrors, climateProps, deviceState, shortType } from './labels'
import { deviceIcon, deviceTint, Icon } from './icons'
import { Capability, PropertyReadout, Switch } from './Controls'

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
  const onOff = caps.find((c) => c.type.split('.').pop() === 'on_off')
  const controls = caps.filter((c) => c !== onOff)
  const on = !!onOff?.state?.value
  const tint = on ? deviceTint(device.type) : null
  const state = deviceState(device)

  // Датчик климата — плитка без тумблера с тремя крупными показаниями.
  if (shortType(device.type || '').startsWith('sensor.climate')) {
    return (
      <div className="tile climate">
        <div className="top">
          <div>
            <div className="name">{device.name}</div>
            <div className="state">{state}</div>
          </div>
        </div>
        <div className="readouts">
          {climateProps(device).map((p, i) => (
            <PropertyReadout key={i} prop={p} />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className={`tile${tint ? ' ' + tint : ''}${busy ? ' busy' : ''}`}>
      <div className="top">
        <div className="ico">
          <Icon name={deviceIcon(device.type)} size={20} />
        </div>
        {onOff && <Switch text="Питание" checked={on} disabled={busy} onChange={(v) => send(onOff.type, 'on', v)} />}
      </div>
      <div className="body">
        <div>
          <div className="name">{device.name}</div>
          {busy ? <span className="badge busy">применяем…</span> : <div className="state">{state}</div>}
        </div>
        {controls.map((cap, i) => (
          <Capability key={i} cap={cap} busy={busy} onChange={send} />
        ))}
        {error && <div className="state err">{error}</div>}
      </div>
    </div>
  )
}
