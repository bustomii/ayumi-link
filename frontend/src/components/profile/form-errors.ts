import { ApiError } from '@/api/http'

/** Pesan error untuk formulir (inline): kunci akun diberi estimasi waktu, selebihnya pesan server. */
export function formErrorText(e: unknown): string {
  if (e instanceof ApiError || (typeof e === 'object' && e !== null && (e as { name?: string }).name === 'ApiError')) {
    const err = e as ApiError
    if (err.code === 'locked') {
      const min = Math.max(1, Math.ceil((err.retryAfter ?? 900) / 60))
      return `Terlalu banyak percobaan. Coba lagi dalam sekitar ${min} menit.`
    }
    if (err.status === 429) return 'Terlalu banyak permintaan. Tunggu sebentar lalu coba lagi.'
    if (err.status >= 500 || err.code === 'empty_response') {
      return 'Server tidak merespons dengan benar. Pastikan backend berjalan lalu coba lagi.'
    }
    return err.message
  }
  if (e instanceof TypeError) return 'Tidak bisa terhubung ke server.'
  return e instanceof Error ? e.message : 'Terjadi kesalahan.'
}
