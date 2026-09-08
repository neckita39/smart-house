const INSTANCE = {
  on: 'Питание', brightness: 'Яркость', temperature: 'Температура', humidity: 'Влажность',
  temperature_k: 'Цвет. температура', hsv: 'Цвет', rgb: 'Цвет', scene: 'Сцена',
  volume: 'Громкость', channel: 'Канал', open: 'Открытие', mute: 'Без звука', backlight: 'Подсветка',
  battery_level: 'Батарея', co2_level: 'CO₂', illumination: 'Освещённость', power: 'Мощность',
  voltage: 'Напряжение', amperage: 'Ток', pressure: 'Давление', water_level: 'Уровень воды',
  motion: 'Движение', water_leak: 'Протечка', button: 'Кнопка', vibration: 'Вибрация',
  smoke: 'Дым', gas: 'Газ', thermostat: 'Режим', fan_speed: 'Скорость', work_speed: 'Скорость',
  program: 'Программа', cleanup_mode: 'Режим уборки', swing: 'Поворот', heat: 'Обогрев',
  'pm2.5_density': 'PM2.5', pm10_density: 'PM10', tvoc: 'ЛОС', meter: 'Счётчик',
  food_level: 'Корм', controls_locked: 'Блокировка', ionization: 'Ионизация', keep_warm: 'Подогрев',
  oscillation: 'Вращение', pause: 'Пауза',
  camera_sw_mute: 'Микрофон', camera_hw_mute: 'Микрофон (кнопка)', camera_pan: 'Поворот',
  camera_tilt: 'Наклон', video_resolution: 'Разрешение', video_detection: 'В кадре',
  noise: 'Шум', voice_activity: 'Голос', signal_level: 'Сигнал', input_source: 'Источник',
  tea_mode: 'Режим чая',
}

const UNIT = {
  'unit.percent': '%', 'unit.temperature.celsius': '°C', 'unit.temperature.kelvin': 'K',
  'unit.ppm': 'ppm', 'unit.watt': 'Вт', 'unit.volt': 'В', 'unit.ampere': 'А',
  'unit.kilowatt_hour': 'кВт·ч', 'unit.pressure.mmhg': 'мм рт. ст.', 'unit.pressure.pascal': 'Па',
  'unit.pressure.atm': 'атм', 'unit.pressure.bar': 'бар',
  'unit.illumination.lux': 'лк', 'unit.lux': 'лк',
  'unit.density.mcg_m3': 'мкг/м³', 'unit.cubic_meter': 'м³', 'unit.gigacalorie': 'Гкал',
}

const EVENT = {
  opened: 'открыто', closed: 'закрыто', detected: 'обнаружено', not_detected: 'не обнаружено',
  high: 'высокий', low: 'низкий', normal: 'норма', click: 'нажатие', double_click: 'двойное нажатие',
  long_press: 'долгое нажатие', dry: 'сухо', leak: 'протечка', tilt: 'наклон', fall: 'падение',
  vibration: 'вибрация', empty: 'пусто', full: 'полно',
  hw_mute_enabled: 'выключен', hw_mute_disabled: 'включён',
  human_present: 'человек', human_not_present: 'нет людей', unidentified: 'не определён',
  silence: 'тишина', speech: 'речь', alarm: 'тревога',
}

const MODE = {
  auto: 'авто', heat: 'обогрев', cool: 'охлаждение', dry: 'осушение', fan_only: 'вентилятор',
  eco: 'эко', low: 'низкая', medium: 'средняя', high: 'высокая', turbo: 'турбо', quiet: 'тихий',
  normal: 'обычный', max: 'макс', min: 'мин', fast: 'быстро', slow: 'медленно', express: 'экспресс',
  horizontal: 'горизонтально', vertical: 'вертикально', stationary: 'без вращения',
  wet_cleaning: 'влажная', dry_cleaning: 'сухая', mixed_cleaning: 'комбинированная',
  black_tea: 'чёрный чай', green_tea: 'зелёный чай', red_tea: 'красный чай', white_tea: 'белый чай',
  flower_tea: 'цветочный чай', herbal_tea: 'травяной чай', oolong_tea: 'оолонг', puerh_tea: 'пуэр',
  one: 'первый', two: 'второй', three: 'третий', four: 'четвёртый',
  video_resolution_auto: 'авто', video_resolution_1440: '1440p',
  video_resolution_1080: '1080p', video_resolution_480: '480p',
}

const SCENE = {
  alice: 'Алиса', party: 'Вечеринка', romance: 'Романтика', candle: 'Свеча', night: 'Вечерний',
  reading: 'Чтение', christmas: 'Рождество', fairy: 'Сказочные огни', northern: 'Северное сияние',
  jungle: 'Джунгли', neon: 'Неон', ocean: 'Океан', siren: 'Сирена', alarm: 'Тревога',
  fantasy: 'Фантазия', gaming: 'Игра', miracle: 'Чудо', snake: 'Год змеи', sunrise: 'Рассвет',
  sunset: 'Закат', movie: 'Кино',
}

const TYPE = {
  light: 'Лампа', 'light.lamp': 'Лампа', 'light.ceiling': 'Люстра', 'light.strip': 'Лента',
  'light.torchere': 'Торшер', socket: 'Розетка', switch: 'Выключатель', thermostat: 'Термостат',
  'thermostat.ac': 'Кондиционер', 'media_device.tv': 'Телевизор', 'media_device.tv_box': 'ТВ-приставка',
  'media_device.receiver': 'Ресивер', media_device: 'Медиа', humidifier: 'Увлажнитель', purifier: 'Очиститель',
  vacuum_cleaner: 'Пылесос', 'cooking.kettle': 'Чайник', 'cooking.coffee_maker': 'Кофеварка',
  washing_machine: 'Стиральная машина', dishwasher: 'Посудомойка', 'openable.curtain': 'Шторы',
  openable: 'Дверь/окно', sensor: 'Датчик', 'sensor.climate': 'Климат', 'sensor.motion': 'Движение',
  'sensor.open': 'Открытие', 'sensor.button': 'Кнопка', 'sensor.water_leak': 'Протечка',
  'sensor.smoke': 'Дым', 'sensor.gas': 'Газ', 'sensor.vibration': 'Вибрация', 'sensor.illumination': 'Освещённость',
  camera: 'Камера', fan: 'Вентилятор', 'ventilation.fan': 'Вентилятор', ventilation: 'Вентиляция',
  iron: 'Утюг', pet_feeder: 'Кормушка', smart_speaker: 'Колонка', hub: 'Хаб', other: 'Устройство',
}

// Подписи слайдеров короче общих: в плитке под них 78 px.
const RANGE = {
  open: 'Открыть', temperature: 'Задать', temperature_k: 'Тепло света', brightness: 'Яркость',
  camera_pan: 'Поворот', camera_tilt: 'Наклон',
}

export const label = (instance) => INSTANCE[instance] || instance || ''
export const unit = (u) => (u && UNIT[u]) || ''
export const eventLabel = (e) => EVENT[e] || e
export const modeLabel = (m) => MODE[m] || m
export const sceneLabel = (s) => SCENE[s] || s
export const rangeLabel = (instance) => RANGE[instance] || label(instance)

// shortType: 'devices.types.light.lamp' → 'light.lamp'.
export const shortType = (type = '') => type.replace(/^devices\.types\./, '')

// typeLabel: 'devices.types.sensor.climate' → 'Климат'. Незнакомый подтип сводится
// к родительскому ('camera.yandex.mike' → 'camera' → 'Камера'), иначе последний сегмент.
export function typeLabel(type = '') {
  const parts = shortType(type).split('.')
  while (parts.length) {
    const known = TYPE[parts.join('.')]
    if (known) return known
    parts.pop()
  }
  return shortType(type).split('.').pop() || 'Устройство'
}

// plural выбирает форму слова по числу: plural(2, 'действие', 'действия', 'действий').
export function plural(n, one, few, many) {
  const rest100 = Math.abs(n) % 100
  const rest10 = rest100 % 10
  if (rest100 > 10 && rest100 < 20) return many
  if (rest10 > 1 && rest10 < 5) return few
  if (rest10 === 1) return one
  return many
}

export const devicesCount = (n) => `${n} ${plural(n, 'устройство', 'устройства', 'устройств')}`
export const actionsCount = (n) => `${n} ${plural(n, 'действие', 'действия', 'действий')}`

// formatNumber округляет показание по единице измерения: температуру до 0,1,
// остальное до целого (мелкие значения вроде 0,25 Вт сохраняют знаки), запятая — русская.
export function formatNumber(v, unitKey) {
  if (typeof v !== 'number' || !Number.isFinite(v)) return String(v ?? '—')
  let n
  if (unitKey === 'unit.temperature.celsius') n = Math.round(v * 10) / 10
  else if (Number.isInteger(v) || Math.abs(v) >= 10) n = Math.round(v)
  else if (Math.abs(v) >= 1) n = Math.round(v * 10) / 10
  else n = Math.round(v * 100) / 100
  return String(n).replace('.', ',')
}

// Приоритет показаний в строке состояния плитки: сначала воздух и климат,
// потом электрика, потом события датчиков; заряд и сигнал — в конце.
const PROP_ORDER = [
  'pm2.5_density', 'pm10_density', 'co2_level', 'tvoc', 'humidity', 'temperature', 'pressure',
  'voltage', 'power', 'amperage', 'meter', 'illumination', 'water_level', 'food_level',
  'water_leak', 'smoke', 'gas', 'motion', 'open', 'vibration', 'button', 'battery_level',
  'signal_level',
]

// Единицы, которые сами себя не объясняют: показание получает подпись
// («батарея 98 %»), с остальными подпись лишняя («230 В»).
const CAPTIONED = new Set(['', '%', '°C', 'мкг/м³', 'ppm'])

const MAX_PROPS = 2
const CLIMATE_TECH = /^(purifier|humidifier|ventilation|thermostat)/
const CLIMATE_BIG = new Set(['temperature', 'humidity', 'pressure'])

const lowerFirst = (s) => (/^[А-ЯЁ][а-яё]/.test(s) ? s[0].toLowerCase() + s.slice(1) : s)

const capOfType = (caps, kind) => caps.find((c) => c.type.split('.').pop() === kind)

// climateProps — три показания для плитки «Климат» в порядке макета.
export function climateProps(device) {
  const props = device.properties || []
  return ['temperature', 'humidity', 'pressure']
    .map((i) => props.find((p) => p.parameters?.instance === i))
    .filter(Boolean)
}

// deviceState — строка состояния плитки: нейтральные формы без склонения по
// названию устройства, детали через « · » («включено · 3400 K», «работает · 51 %»).
export function deviceState(device) {
  const type = shortType(device.type || '')
  const caps = device.capabilities || []
  const parts = []

  const onOff = capOfType(caps, 'on_off')
  if (onOff) parts.push(powerPart(type, !!onOff.state?.value, caps))
  else if (type.startsWith('sensor')) parts.push('датчик')

  parts.push(colorPart(caps))
  parts.push(...propParts(device, type))

  const text = parts.filter(Boolean).join(' · ')
  return text || lowerFirst(typeLabel(device.type))
}

function powerPart(type, on, caps) {
  if (type.startsWith('openable')) {
    if (!on) return 'закрыто'
    const open = caps.find((c) => c.parameters?.instance === 'open')
    const v = open?.state?.value
    return v == null ? 'открыто' : `открыто на ${formatNumber(v, 'unit.percent')} %`
  }
  if (CLIMATE_TECH.test(type)) return on ? 'работает' : 'выключено'
  return on ? 'включено' : 'выключено'
}

// colorPart — как настроен свет: «3400 K», «цвет» или «сцена «Вечеринка»».
function colorPart(caps) {
  const state = capOfType(caps, 'color_setting')?.state
  if (!state || state.value == null) return ''
  if (state.instance === 'temperature_k') return `${formatNumber(state.value, 'unit.temperature.kelvin')} K`
  if (state.instance === 'scene') return `сцена «${sceneLabel(state.value)}»`
  return 'цвет'
}

function propParts(device, type) {
  const skipBig = type.startsWith('sensor.climate')
  const props = (device.properties || []).filter((p) => {
    const i = p.parameters?.instance
    return p.state?.value != null && !(skipBig && CLIMATE_BIG.has(i))
  })
  const rank = (p) => {
    const i = PROP_ORDER.indexOf(p.parameters?.instance)
    return i < 0 ? PROP_ORDER.length : i
  }
  return props
    .sort((a, b) => rank(a) - rank(b))
    .slice(0, MAX_PROPS)
    .map((p) => propPart(p, type))
    .filter(Boolean)
}

function propPart(prop, type) {
  const p = prop.parameters || {}
  const v = prop.state?.value
  if (v == null) return ''
  const caption =
    p.instance === 'temperature' && type.startsWith('thermostat') ? 'сейчас' : lowerFirst(label(p.instance))
  if (prop.type.split('.').pop() === 'event') return `${caption}: ${eventLabel(v)}`
  const u = unit(p.unit)
  const value = typeof v === 'number' ? formatNumber(v, p.unit) : String(v)
  const text = u ? `${value} ${u}` : value
  return CAPTIONED.has(u) ? `${caption} ${text}` : text
}

// actionErrors собирает ошибки из ответа devices/actions в человекочитаемый список.
// deviceName — необязательная функция id → имя устройства; если передана,
// каждое сообщение получает префикс с именем («Торшер · Питание: устройство не отвечает»).
export function actionErrors(resp, deviceName) {
  const out = []
  for (const d of resp?.devices || []) {
    const name = typeof deviceName === 'function' ? deviceName(d.id) : null
    for (const c of d.capabilities || []) {
      const r = c.state?.action_result
      if (r && r.status !== 'DONE') {
        const msg = `${label(c.state?.instance)}: ${r.error_message || errorText(r.error_code)}`
        out.push(name ? `${name} · ${msg}` : msg)
      }
    }
  }
  return out
}

function errorText(code) {
  return {
    DEVICE_UNREACHABLE: 'устройство не отвечает',
    DEVICE_BUSY: 'устройство занято',
    DEVICE_NOT_FOUND: 'устройство не найдено',
    INTERNAL_ERROR: 'внутренняя ошибка Яндекса',
    INVALID_ACTION: 'недопустимое действие',
    INVALID_VALUE: 'недопустимое значение',
    NOT_SUPPORTED_IN_CURRENT_MODE: 'недоступно в текущем режиме',
  }[code] || code || 'неизвестная ошибка'
}
