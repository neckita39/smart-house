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
}

const UNIT = {
  'unit.percent': '%', 'unit.temperature.celsius': '°C', 'unit.temperature.kelvin': 'K',
  'unit.ppm': 'ppm', 'unit.watt': 'Вт', 'unit.volt': 'В', 'unit.ampere': 'А',
  'unit.kilowatt_hour': 'кВт·ч', 'unit.pressure.mmhg': 'мм рт. ст.', 'unit.pressure.pascal': 'Па',
  'unit.pressure.atm': 'атм', 'unit.pressure.bar': 'бар', 'unit.lux': 'лк',
  'unit.density.mcg_m3': 'мкг/м³', 'unit.cubic_meter': 'м³', 'unit.gigacalorie': 'Гкал',
}

const EVENT = {
  opened: 'открыто', closed: 'закрыто', detected: 'обнаружено', not_detected: 'нет',
  high: 'высокий', low: 'низкий', normal: 'норма', click: 'нажатие', double_click: 'двойное нажатие',
  long_press: 'долгое нажатие', dry: 'сухо', leak: 'протечка', tilt: 'наклон', fall: 'падение',
  vibration: 'вибрация', empty: 'пусто', full: 'полно',
}

const MODE = {
  auto: 'авто', heat: 'обогрев', cool: 'охлаждение', dry: 'осушение', fan_only: 'вентилятор',
  eco: 'эко', low: 'низкая', medium: 'средняя', high: 'высокая', turbo: 'турбо', quiet: 'тихий',
  normal: 'обычный', max: 'макс', min: 'мин', fast: 'быстро', slow: 'медленно', express: 'экспресс',
  horizontal: 'горизонтально', vertical: 'вертикально', stationary: 'без вращения',
}

const TYPE = {
  light: 'Лампа', socket: 'Розетка', switch: 'Выключатель', thermostat: 'Термостат',
  'thermostat.ac': 'Кондиционер', 'media_device.tv': 'Телевизор', 'media_device.tv_box': 'ТВ-приставка',
  'media_device.receiver': 'Ресивер', humidifier: 'Увлажнитель', purifier: 'Очиститель',
  vacuum_cleaner: 'Пылесос', 'cooking.kettle': 'Чайник', 'cooking.coffee_maker': 'Кофеварка',
  washing_machine: 'Стиральная машина', dishwasher: 'Посудомойка', 'openable.curtain': 'Шторы',
  openable: 'Дверь/окно', sensor: 'Датчик', 'sensor.climate': 'Климат', 'sensor.motion': 'Движение',
  'sensor.open': 'Открытие', 'sensor.button': 'Кнопка', 'sensor.water_leak': 'Протечка',
  'sensor.smoke': 'Дым', 'sensor.gas': 'Газ', 'sensor.vibration': 'Вибрация', 'sensor.illumination': 'Освещённость',
  camera: 'Камера', fan: 'Вентилятор', iron: 'Утюг', 'pet_feeder': 'Кормушка', other: 'Устройство',
}

export const label = (instance) => INSTANCE[instance] || instance || ''
export const unit = (u) => (u && UNIT[u]) || ''
export const eventLabel = (e) => EVENT[e] || e
export const modeLabel = (m) => MODE[m] || m

// typeLabel: 'devices.types.sensor.climate' → 'Климат'; неизвестное — последний сегмент.
export function typeLabel(type = '') {
  const short = type.replace(/^devices\.types\./, '')
  return TYPE[short] || short.split('.').pop()
}

// actionErrors собирает ошибки из ответа devices/actions в человекочитаемый список.
export function actionErrors(resp) {
  const out = []
  for (const d of resp?.devices || []) {
    for (const c of d.capabilities || []) {
      const r = c.state?.action_result
      if (r && r.status !== 'DONE') {
        out.push(`${label(c.state?.instance)}: ${r.error_message || errorText(r.error_code)}`)
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
