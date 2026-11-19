import type { ApiError } from '@/api/http'

export type ErrorKind =
  | 'network'
  | 'auth'
  | 'forbidden'
  | 'notfound'
  | 'outdated'
  | 'ratelimit'
  | 'server'
  | 'validation'
  | 'claude'
  | 'unknown'

/** Error yang sudah diterjemahkan agar mudah dipahami pengguna. */
export type AppError = {
  kind: ErrorKind
  title: string
  message: string
  /** Saran langkah berikutnya. */
  hint?: string
  /** Detail teknis (kode status, endpoint, pesan asli). */
  detail?: string
}

/** Pesan dari CLI `claude` yang sering muncul, beserta penjelasan dan saran. */
function describeClaudeMessage(raw: string): AppError | null {
  if (/not logged in|please run \/login/i.test(raw)) {
    return {
      kind: 'claude',
      title: 'Claude CLI belum login',
      message: 'CLI claude di server/container belum masuk ke akun Claude Anda, jadi balasan tidak bisa dibuat.',
      hint: 'Buka Profil & keamanan → Claude CLI, tempel token dari `claude setup-token` (di Mac).',
      detail: raw,
    }
  }
  if (/does not support this model|or newer is required/i.test(raw)) {
    return {
      kind: 'claude',
      title: 'Model tidak didukung CLI ini',
      message: 'Versi CLI claude yang dipakai terlalu lama untuk model yang dipilih.',
      hint: 'Pilih model lain di menu pengaturan, atau perbarui Claude (claude update / aplikasi Claude Desktop).',
      detail: raw,
    }
  }
  if (/rate.?limit|usage limit|429|overloaded|529/i.test(raw)) {
    return {
      kind: 'ratelimit',
      title: 'Batas pemakaian atau server sibuk',
      message: 'Claude sedang menolak permintaan karena batas pemakaian atau beban server.',
      hint: 'Tunggu beberapa saat lalu kirim ulang, atau pilih model yang lebih ringan.',
      detail: raw,
    }
  }
  if (/folder (tidak diizinkan|session tidak valid)/i.test(raw)) {
    return {
      kind: 'validation',
      title: 'Folder tidak bisa dipakai',
      message: 'Folder kerja yang dipilih tidak ada atau berada di luar folder yang diizinkan.',
      hint: 'Pilih folder lain lewat "Buat session baru". Folder yang diizinkan diatur lewat AYUMI_ROOTS.',
      detail: raw,
    }
  }
  return null
}

/** Pengenalan berbasis bentuk (bukan instanceof) agar tetap benar walau modul termuat dua kali (HMR). */
function isApiError(e: unknown): e is ApiError {
  return (
    typeof e === 'object' &&
    e !== null &&
    (e as { name?: unknown }).name === 'ApiError' &&
    typeof (e as { status?: unknown }).status === 'number'
  )
}

/** Menerjemahkan error apa pun menjadi pesan yang jelas. `action` = apa yang sedang dilakukan. */
export function describeError(e: unknown, action = 'Permintaan'): AppError {
  if (isApiError(e)) {
    const where = e.endpoint ? ` (${e.endpoint})` : ''
    const detail = `Status ${e.status}${where}\n${e.message}`
    switch (true) {
      case e.code === 'empty_response' && e.status >= 500:
      case e.status === 502 || e.status === 503 || e.status === 504:
        return {
          kind: 'network',
          title: 'Backend tidak terjangkau',
          message: `${action} gagal karena server Ayumi Link tidak menjawab.`,
          hint: 'Pastikan backend berjalan (./ayumi-link di terminal), lalu coba lagi.',
          detail,
        }
      case e.status === 401:
        return {
          kind: 'auth',
          title: 'Sesi login berakhir',
          message: 'Anda perlu masuk lagi untuk melanjutkan.',
          hint: 'Muat ulang halaman lalu login dengan password dan kode 2FA.',
          detail,
        }
      case e.status === 403:
        return {
          kind: 'forbidden',
          title: 'Akses ditolak',
          message: e.message || 'Server menolak permintaan ini.',
          hint: 'Pastikan Anda membuka aplikasi dari alamat yang diizinkan.',
          detail,
        }
      case e.status === 404 && /tidak ditemukan/i.test(e.message) && !/session/i.test(e.message):
        return {
          kind: 'outdated',
          title: 'Fitur belum tersedia di server',
          message: `${action} gagal karena server tidak mengenal perintah ini. Server kemungkinan masih versi lama.`,
          hint: 'Restart backend (hentikan lalu jalankan ulang ./ayumi-link) agar memuat versi terbaru.',
          detail,
        }
      case e.status === 404:
        return {
          kind: 'notfound',
          title: 'Data tidak ditemukan',
          message: e.message || 'Data yang diminta sudah tidak ada.',
          hint: 'Mungkin sudah dihapus atau dipindahkan. Muat ulang daftar session.',
          detail,
        }
      case e.status === 413 || e.status === 415 || e.status === 400:
        return {
          kind: 'validation',
          title: `${action} ditolak`,
          message: e.message,
          hint: 'Periksa isian Anda lalu coba lagi.',
          detail,
        }
      case e.status === 429:
        return {
          kind: 'ratelimit',
          title: 'Terlalu banyak permintaan',
          message: e.message || 'Anda terlalu sering mengirim permintaan.',
          hint: e.retryAfter ? `Coba lagi dalam ${e.retryAfter} detik.` : 'Tunggu sebentar lalu coba lagi.',
          detail,
        }
      case e.status >= 500:
        return {
          kind: 'server',
          title: 'Terjadi kesalahan di server',
          message: `${action} gagal karena masalah di sisi server.`,
          hint: 'Lihat log di terminal backend untuk penyebabnya, lalu coba lagi.',
          detail,
        }
    }
    return { kind: 'unknown', title: `${action} gagal`, message: e.message, detail }
  }

  // fetch() gagal total: server mati, jaringan putus, atau diblokir.
  if (e instanceof TypeError) {
    return {
      kind: 'network',
      title: 'Tidak bisa terhubung ke server',
      message: 'Aplikasi tidak mendapat jawaban dari backend Ayumi Link.',
      hint: 'Pastikan backend berjalan (./ayumi-link) dan koneksi jaringan Anda aktif.',
      detail: e.message,
    }
  }

  const raw = e instanceof Error ? e.message : String(e)
  return (
    describeClaudeMessage(raw) ?? {
      kind: 'unknown',
      title: `${action} gagal`,
      message: raw || 'Terjadi kesalahan yang tidak dikenal.',
      detail: raw || undefined,
    }
  )
}
