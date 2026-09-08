import { label, unit } from './labels'

const OPS = { '<': '<', '<=': '≤', '>': '>', '>=': '≥', '==': '=', '!=': '≠' }

function fmtValue(v, u) {
  if (typeof v === 'boolean') return v ? 'вкл' : 'выкл'
  if (typeof v === 'number') return `${Math.round(v * 10) / 10}${u ? ' ' + u : ''}`
  if (v && typeof v === 'object') return 'цвет'
  return String(v)
}

// describeRule превращает условия и действия в короткие русские фразы.
// deviceName(id) → имя устройства; unitOf(id, instance) → единица (может быть пустой).
export function describeRule(rule, deviceName, unitOf = () => '') {
  const when = (rule.when || []).map((c) => {
    if (c.time) return `время ${c.time.after}–${c.time.before}`
    const inst = c.property || c.capability
    const u = unit(unitOf(c.device_id, inst))
    return `${deviceName(c.device_id)}: ${label(inst).toLowerCase()} ${OPS[c.op] || c.op} ${fmtValue(c.value, u)}`
  })
  const then = (rule.then || []).map((a) => {
    if (a.macro_id) return `макрос ${a.macro_id}`
    if (a.scenario_id) return `сценарий Яндекса ${a.scenario_id}`
    const u = unit(unitOf(a.device_id, a.instance))
    return `${deviceName(a.device_id)} → ${label(a.instance).toLowerCase()} ${fmtValue(a.value, u)}`
  })
  const extra = []
  if (rule.for_minutes) extra.push(`держится ${rule.for_minutes} мин`)
  if (rule.cooldown_minutes) extra.push(`не чаще чем раз в ${rule.cooldown_minutes} мин`)
  return { when, then, extra }
}
