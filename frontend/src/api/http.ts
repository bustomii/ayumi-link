/** Kesalahan dari API; `code` membedakan kasus (mis. locked, invalid, challenge_expired). */
export class ApiError extends Error {
  readonly status: number
  readonly code?: string
  readonly retryAfter?: number
  /** Mis. "PUT /api/groups/rename" — membantu menjelaskan error ke pengguna. */
  readonly endpoint?: string

  constructor(message: string, status: number, code?: string, retryAfter?: number, endpoint?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.retryAfter = retryAfter
    this.endpoint = endpoint
  }
}

/** Dipancarkan saat sebuah request terproteksi ditolak karena belum login. */
export const AUTH_REQUIRED_EVENT = 'ayumi:auth-required'

export async function toApiError(res: Response, method = 'GET'): Promise<ApiError> {
  let data: { error?: string; code?: string; retryAfter?: number } = {}
  try {
    data = await res.json()
  } catch {
    /* body bukan JSON */
  }
  if (res.status === 401 && (data.code === 'unauthenticated' || data.code === 'setup_required')) {
    window.dispatchEvent(new Event(AUTH_REQUIRED_EVENT))
  }
  let endpoint: string | undefined
  try {
    endpoint = `${method} ${new URL(res.url).pathname}`
  } catch {
    /* url tidak tersedia */
  }
  // Jawaban tanpa pesan JSON berarti bukan dari backend kita (mis. proxy/tunnel saat backend mati).
  const code = data.code ?? (data.error ? undefined : 'empty_response')
  return new ApiError(
    data.error ?? `Permintaan gagal (${res.status})`,
    res.status,
    code,
    data.retryAfter,
    endpoint,
  )
}

type Options = Omit<RequestInit, 'body'> & { json?: unknown }

/** fetch JSON same-origin (cookie sesi ikut terkirim). */
export async function api<T>(path: string, { json, headers, ...init }: Options = {}): Promise<T> {
  const res = await fetch(path, {
    credentials: 'same-origin',
    ...init,
    headers: { ...(json !== undefined ? { 'Content-Type': 'application/json' } : {}), ...headers },
    body: json !== undefined ? JSON.stringify(json) : undefined,
  })
  if (!res.ok) throw await toApiError(res, init.method ?? 'GET')
  return (await res.json()) as T
}
