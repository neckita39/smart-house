import { useState } from 'react'
import { api } from './api'
import { label, modeLabel, unit } from './labels'
import { Icon } from './icons'
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
        out.push({ type: cap.type, instance: 'temperature_k', kind: 'number', params: { range: { min, max, precision: 100 }, unit: 'unit.temperature.kelvin' } })
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
    <form className="tile editor" onSubmit={save}>
      <div className="top">
        <div>
          <div className="display title">{macro.id ? 'Изменить макрос' : 'Новый макрос'}</div>
          <div className="state">Выполняется одним запросом к Яндексу</div>
        </div>
        <span className="badge busy">{busy ? 'Сохраняем…' : 'не сохранён'}</span>
      </div>

      <div className="field">
        <span className="sub">Название</span>
        <label className="input inset">
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Например, Кино" autoFocus aria-label="Название макроса" />
        </label>
      </div>

      <div className="field">
        <span className="sub">Действия</span>
        {actions.map((a, i) => (
          <ActionRow key={i} action={a} devices={devices} onChange={(patch) => update(i, patch)} onRemove={() => remove(i)} />
        ))}
        {!actions.length && <span className="state">Пока ни одного действия — добавьте первое.</span>}
      </div>

      <div className="row between">
        <button type="button" className="btn inset" onClick={addAction} disabled={!devices.length}>
          <Icon name="PLUS" size={16} /> Действие
        </button>
        <div className="row">
          <button type="button" className="btn ghost" onClick={onCancel}>
            Отмена
          </button>
          <button type="submit" className="btn primary" disabled={busy || !name.trim() || !actions.length}>
            Сохранить
          </button>
        </div>
      </div>

      {!devices.length && <div className="banner info">Нет устройств, которыми можно управлять.</div>}
      {error && <div className="banner">{error}</div>}
    </form>
  )
}

// SelectPill — пилюля .sel с нативным select и иконкой CHEV справа.
function SelectPill({ value, onChange, children, title }) {
  return (
    <label className="sel inset wide">
      <select value={value} onChange={onChange} aria-label={title}>
        {children}
      </select>
      <Icon name="CHEV" size={14} stroke={2} />
    </label>
  )
}

function ActionRow({ action, devices, onChange, onRemove }) {
  const device = devices.find((d) => d.id === action.device_id)
  const options = device ? capabilityOptions(device) : []
  const opt = options.find((o) => o.type === action.type && o.instance === action.instance)

  // Устройство или умение могло исчезнуть из дома — показываем это и даём убрать строку.
  if (!device || !opt) {
    return (
      <div className="action missing">
        <div className="banner">
          {!device
            ? `Устройство недоступно (id ${shortId(action.device_id)}) — убрано из дома в приложении Яндекса`
            : `Умение недоступно (${label(action.instance)}) — устройство его больше не поддерживает`}
        </div>
        <button type="button" className="icon-btn inset" onClick={onRemove} title="Убрать действие" aria-label="Убрать действие">
          <Icon name="X" size={16} />
        </button>
      </div>
    )
  }

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
      <SelectPill value={device.id} title="Устройство" onChange={(e) => pickDevice(e.target.value)}>
        {devices.map((d) => (
          <option key={d.id} value={d.id}>
            {d.name}
          </option>
        ))}
      </SelectPill>
      <SelectPill value={opt.key} title="Умение" onChange={(e) => pickCapability(e.target.value)}>
        {options.map((o) => (
          <option key={o.key} value={o.key}>
            {label(o.instance)}
          </option>
        ))}
      </SelectPill>
      <ValueInput opt={opt} value={action.value} onChange={(value) => onChange({ value })} />
      <button type="button" className="icon-btn inset" onClick={onRemove} title="Убрать действие" aria-label="Убрать действие">
        <Icon name="X" size={16} />
      </button>
    </div>
  )
}

const shortId = (id = '') => (id.length > 8 ? id.slice(0, 8) + '…' : id)

function ValueInput({ opt, value, onChange }) {
  switch (opt.kind) {
    case 'bool':
      return (
        <SelectPill value={value ? '1' : '0'} title="Значение" onChange={(e) => onChange(e.target.value === '1')}>
          <option value="1">включить</option>
          <option value="0">выключить</option>
        </SelectPill>
      )
    case 'number': {
      const r = opt.params.range || {}
      const u = unit(opt.params.unit)
      return (
        <label className="input inset num">
          <input
            type="number"
            min={r.min}
            max={r.max}
            step={r.precision || 1}
            value={value ?? ''}
            aria-label="Значение"
            onChange={(e) => onChange(Number(e.target.value))}
          />
          {u && <span className="muted">{u}</span>}
        </label>
      )
    }
    case 'mode':
      return (
        <SelectPill value={value ?? ''} title="Значение" onChange={(e) => onChange(e.target.value)}>
          {(opt.params.modes || []).map((m) => (
            <option key={m.value} value={m.value}>
              {modeLabel(m.value)}
            </option>
          ))}
        </SelectPill>
      )
    case 'color': {
      const hex = opt.instance === 'rgb' ? rgbIntToHex(value ?? 0xffffff) : hsvToHex(value || { h: 0, s: 0, v: 100 })
      return (
        <label className="row inset color-val">
          <input
            type="color"
            className="swatch sel"
            value={hex}
            aria-label="Цвет"
            onChange={(e) => onChange(opt.instance === 'rgb' ? hexToRgbInt(e.target.value) : hexToHsv(e.target.value))}
          />
          <span className="muted">{hex}</span>
        </label>
      )
    }
    default:
      return null
  }
}
