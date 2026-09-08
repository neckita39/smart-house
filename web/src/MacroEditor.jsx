import { useState } from 'react'
import { api } from './api'
import { label, modeLabel } from './labels'
import { hexToHsv, hexToRgbInt, hsvToHex, rgbIntToHex } from './color'

// capabilityOptions — управляемые умения устройства в виде вариантов для редактора:
// [{key, type, instance, kind: 'bool'|'number'|'mode'|'color', params}].
export function capabilityOptions(device) {
  const out = []
  for (const cap of device.capabilities || []) {
    const kind = cap.type.split('.').pop()
    const p = cap.parameters || {}
    if (kind === 'on_off') out.push({ type: cap.type, instance: 'on', kind: 'bool', params: p })
    else if (kind === 'toggle') out.push({ type: cap.type, instance: p.instance, kind: 'bool', params: p })
    else if (kind === 'range') out.push({ type: cap.type, instance: p.instance, kind: 'number', params: p })
    else if (kind === 'mode') out.push({ type: cap.type, instance: p.instance, kind: 'mode', params: p })
    else if (kind === 'color_setting') {
      if (p.temperature_k) {
        const { min, max } = p.temperature_k
        out.push({ type: cap.type, instance: 'temperature_k', kind: 'number', params: { range: { min, max, precision: 100 } } })
      }
      if (p.color_model) out.push({ type: cap.type, instance: p.color_model, kind: 'color', params: p })
    }
  }
  return out.map((o) => ({ ...o, key: o.type + ':' + o.instance }))
}

function defaultValue(opt) {
  switch (opt.kind) {
    case 'bool':
      return true
    case 'number':
      return opt.params.range?.min ?? 0
    case 'mode':
      return opt.params.modes?.[0]?.value ?? ''
    case 'color':
      return opt.instance === 'rgb' ? 0xffffff : { h: 0, s: 0, v: 100 }
    default:
      return null
  }
}

export default function MacroEditor({ home, macro, onSaved, onCancel }) {
  const [name, setName] = useState(macro.name || '')
  const [actions, setActions] = useState(macro.actions || [])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const devices = (home.devices || []).filter((d) => capabilityOptions(d).length)

  function addAction() {
    const d = devices[0]
    if (!d) return
    const opt = capabilityOptions(d)[0]
    setActions((a) => [...a, { device_id: d.id, type: opt.type, instance: opt.instance, value: defaultValue(opt) }])
  }
  const update = (i, patch) => setActions((a) => a.map((x, j) => (j === i ? { ...x, ...patch } : x)))
  const remove = (i) => setActions((a) => a.filter((_, j) => j !== i))

  async function save(e) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      const body = { name: name.trim(), actions }
      if (macro.id) await api.updateMacro({ ...body, id: macro.id })
      else await api.createMacro(body)
      onSaved()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <form className="card editor" onSubmit={save}>
      <h3>{macro.id ? 'Изменить макрос' : 'Новый макрос'}</h3>
      <div className="control">
        <label>Название</label>
        <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Например, Кино" autoFocus />
      </div>
      {actions.map((a, i) => (
        <ActionRow key={i} action={a} devices={devices} onChange={(patch) => update(i, patch)} onRemove={() => remove(i)} />
      ))}
      <div className="row">
        <button type="button" onClick={addAction} disabled={!devices.length}>
          + Действие
        </button>
        <button type="submit" className="primary" disabled={busy || !name.trim() || !actions.length}>
          Сохранить
        </button>
        <button type="button" onClick={onCancel}>
          Отмена
        </button>
      </div>
      {!devices.length && <p className="muted">Нет устройств, которыми можно управлять.</p>}
      {error && <p className="error">{error}</p>}
    </form>
  )
}

function ActionRow({ action, devices, onChange, onRemove }) {
  const device = devices.find((d) => d.id === action.device_id) || devices[0]
  const options = capabilityOptions(device)
  const opt = options.find((o) => o.type === action.type && o.instance === action.instance) || options[0]

  function pickDevice(id) {
    const o = capabilityOptions(devices.find((d) => d.id === id))[0]
    onChange({ device_id: id, type: o.type, instance: o.instance, value: defaultValue(o) })
  }
  function pickCapability(key) {
    const o = options.find((x) => x.key === key)
    onChange({ type: o.type, instance: o.instance, value: defaultValue(o) })
  }

  return (
    <div className="action">
      <select value={device.id} onChange={(e) => pickDevice(e.target.value)}>
        {devices.map((d) => (
          <option key={d.id} value={d.id}>
            {d.name}
          </option>
        ))}
      </select>
      <select value={opt.key} onChange={(e) => pickCapability(e.target.value)}>
        {options.map((o) => (
          <option key={o.key} value={o.key}>
            {label(o.instance)}
          </option>
        ))}
      </select>
      <ValueInput opt={opt} value={action.value} onChange={(value) => onChange({ value })} />
      <button type="button" className="danger" onClick={onRemove} title="Убрать действие">
        ✕
      </button>
    </div>
  )
}

function ValueInput({ opt, value, onChange }) {
  switch (opt.kind) {
    case 'bool':
      return (
        <select value={value ? '1' : '0'} onChange={(e) => onChange(e.target.value === '1')}>
          <option value="1">включить</option>
          <option value="0">выключить</option>
        </select>
      )
    case 'number': {
      const r = opt.params.range || {}
      return (
        <input
          type="number"
          min={r.min}
          max={r.max}
          step={r.precision || 1}
          value={value ?? ''}
          onChange={(e) => onChange(Number(e.target.value))}
        />
      )
    }
    case 'mode':
      return (
        <select value={value ?? ''} onChange={(e) => onChange(e.target.value)}>
          {(opt.params.modes || []).map((m) => (
            <option key={m.value} value={m.value}>
              {modeLabel(m.value)}
            </option>
          ))}
        </select>
      )
    case 'color': {
      const hex = opt.instance === 'rgb' ? rgbIntToHex(value ?? 0xffffff) : hsvToHex(value || { h: 0, s: 0, v: 100 })
      return (
        <input
          type="color"
          value={hex}
          onChange={(e) => onChange(opt.instance === 'rgb' ? hexToRgbInt(e.target.value) : hexToHsv(e.target.value))}
        />
      )
    }
    default:
      return null
  }
}
