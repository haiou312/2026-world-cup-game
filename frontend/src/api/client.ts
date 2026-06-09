const BASE = '/api'

// Held in memory only (not persisted) so the Settings/Wheel password must be
// re-entered each browser session.
let settingsPw = ''
export function setSettingsPw(pw: string) {
  settingsPw = pw
}

async function request<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((opts.headers as Record<string, string>) || {}),
  }
  if (settingsPw) headers['X-Settings-Password'] = settingsPw

  const res = await fetch(BASE + path, { ...opts, headers })
  const text = await res.text()
  const data = text ? JSON.parse(text) : {}
  if (!res.ok) {
    throw new Error((data as { error?: string }).error || `HTTP ${res.status}`)
  }
  return data as T
}

export const api = {
  get: <T>(p: string) => request<T>(p),
  post: <T>(p: string, body?: unknown) =>
    request<T>(p, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
  put: <T>(p: string, body?: unknown) =>
    request<T>(p, { method: 'PUT', body: body ? JSON.stringify(body) : undefined }),
  del: <T>(p: string) => request<T>(p, { method: 'DELETE' }),
}
