async function request(path, options = {}) {
  const res = await fetch(path, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
  })
  const text = await res.text()
  let data = null
  try {
    data = text ? JSON.parse(text) : null
  } catch {
    data = null
  }
  if (!res.ok) {
    const err = new Error(data?.message || `Ошибка HTTP ${res.status}`)
    err.status = res.status
    err.code = data?.error
    throw err
  }
  return data
}

const json = (method, body) => ({ method, body: JSON.stringify(body) })

export const api = {
  authStatus: () => request('/api/auth/status'),
  login: (code) => request('/api/auth/code', json('POST', { code })),
  home: () => request('/api/home'),
  deviceActions: (devices) => request('/api/devices/actions', json('POST', { devices })),
  runScenario: (id) => request(`/api/scenarios/${encodeURIComponent(id)}/run`, { method: 'POST' }),
  macros: () => request('/api/macros'),
  createMacro: (m) => request('/api/macros', json('POST', m)),
  updateMacro: (m) => request(`/api/macros/${encodeURIComponent(m.id)}`, json('PUT', m)),
  deleteMacro: (id) => request(`/api/macros/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  runMacro: (id) => request(`/api/macros/${encodeURIComponent(id)}/run`, { method: 'POST' }),
  rules: () => request('/api/rules'),
  createRule: (r) => request('/api/rules', json('POST', r)),
  updateRule: (r) => request(`/api/rules/${encodeURIComponent(r.id)}`, json('PUT', r)),
  deleteRule: (id) => request(`/api/rules/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  runRule: (id) => request(`/api/rules/${encodeURIComponent(id)}/run`, { method: 'POST' }),
  checkRule: (id) => request(`/api/rules/${encodeURIComponent(id)}/check`),
  events: (limit = 50) => request(`/api/events?limit=${limit}`),
}
