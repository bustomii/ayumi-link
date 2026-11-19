# Ayumi Link

**Antarmuka web self-hosted untuk session [Claude Code](https://docs.claude.com/en/docs/claude-code) lokal Anda.**
Baca, lanjutkan, dan mulai percakapan Claude Code dari browser mana pun, termasuk HP, dengan login
milik Anda sendiri (password + 2FA). Claude tetap berjalan di mesin *Anda*; tidak ada data yang dikirim
ke pihak ketiga.

> **Proyek tidak resmi.** Ayumi Link tidak berafiliasi, didukung, atau disponsori oleh Anthropic.
> "Claude" adalah merek dagang Anthropic. Aplikasi ini menjalankan CLI `claude` resmi yang Anda pasang
> dan login sendiri.

🇬🇧 [Read in English](README.md)

---

## Kenapa

Claude Code hidup di terminal atau aplikasi desktop. Ayumi Link memberi antarmuka chat di depannya
supaya Anda bisa bekerja dari HP atau komputer lain **tanpa** memberi perangkat itu login Claude dan
tanpa klien VPN: Anda login ke *Ayumi Link* (password + kode autentikator), sedangkan CLI `claude` di
mesin Anda yang bekerja.

## Fitur

- **Telusuri semua session** dari `~/.claude`, dikelompokkan seperti folder Claude Desktop, dengan badge live.
- **Lanjutkan chat** dengan balasan streaming; balasan tetap berjalan di background saat Anda pindah session.
- **Mulai session baru** di folder yang dipilih lewat pemilih folder yang dibatasi.
- **Lampirkan file ke pesan** (tombol, seret-dan-lepas, atau tempel): gambar/foto, PDF, Excel (`.xlsx`, `.csv`) dan Word (`.docx`); Claude membacanya. Jenis file lain ditolak.
- **Menu model / effort / izin tool** per pesan (Fable, Opus, Sonnet, Haiku, dan model lama).
- **Ubah nama, pindah, sembunyikan, atau hapus** session dan folder (lokal di Ayumi Link; tidak mengubah data Claude).
- **Autentikasi sungguhan:** password argon2id, 2FA TOTP, kode pemulihan sekali pakai, penguncian akun,
  daftar perangkat dengan sign-out jarak jauh, verifikasi ulang untuk aksi sensitif.
- **Kontrol akses lewat `.env`:** **allowlist IP** klien, dukungan proxy tepercaya, pengerasan cookie.
- **Tautan langsung:** `/s/<id-session>` membuka session yang sama setelah reload; skeleton loading;
  popup error yang jelas; tampilan responsif.
- **Satu image, satu perintah:** `docker compose up -d --build`.

## Cara kerja

```
 Browser ──HTTPS──▶  Ayumi Link (API Go + Fiber, menyajikan aplikasi React)
 (HP/laptop)               │  menjalankan  `claude -p --resume <id> --output-format stream-json`
                           ▼
                     CLI claude  ──▶  API Anthropic
                           │
                    ~/.claude  (session berupa transkrip JSONL, dibaca langsung)
```

## Mulai cepat — Docker (satu perintah)

Butuh Docker dengan Compose v2.24+.

```bash
git clone https://github.com/bustomii/ayumi-link.git && cd ayumi-link
docker compose up -d --build          # atau: make up
```

Build pertama mengunduh image dasar dan CLI Claude Code (sekitar 1 GB), jadi bisa memakan beberapa
menit di koneksi lambat; build berikutnya memakai cache.

1. Buka <http://localhost:8081>.
2. Buat akun. **Token setup** sekali pakai dicetak di log:
   `docker compose logs ayumi-link | grep -A1 SETUP` (atau `make token`). Pindai QR dengan aplikasi
   autentikator dan **simpan kode pemulihan**.
3. Login CLI Claude **di dalam container** (kredensial disimpan di volume Docker):
   `docker compose exec -it ayumi-link claude auth login` (atau `make claude-login`).
   Alternatif: isi `CLAUDE_CODE_OAUTH_TOKEN` (dari `claude setup-token`) atau `ANTHROPIC_API_KEY` di `.env`.
4. Taruh proyek yang boleh dikerjakan Claude di `./workspace` (atau atur `WORKSPACE_DIR` di `.env`).

Secara bawaan port hanya dipublikasikan di **127.0.0.1**. Baca [Akses jarak jauh](#akses-jarak-jauh-hp)
sebelum mengubahnya.

> **Docker vs. dijalankan langsung.** Container Linux tidak bisa membaca Keychain macOS, grup folder
> Claude Desktop, atau path host seperti `/Users/anda/...`. Docker cocok untuk server/NAS/VPS dan untuk
> mencoba. Untuk meniru **setup desktop yang sudah ada** (semua session dan grup Claude Desktop),
> jalankan langsung di mesin tersebut.

## Mulai cepat — jalankan langsung (macOS / Linux)

Syarat: Go ≥ 1.24, Node ≥ 22, dan CLI `claude` yang sudah terpasang dan login.

```bash
cp .env.example .env                       # opsional
# pengembangan (dua terminal)
cd backend  && go run .
cd frontend && npm install && npm run dev  # http://localhost:5173

# atau satu proses produksi
cd frontend && npm install && npm run build
cd ../backend && AYUMI_STATIC_DIR=../frontend/dist go run .   # http://localhost:8081
```

## Konfigurasi

Semua opsional, lewat environment variable atau file `.env` (lihat [`.env.example`](.env.example);
tiap opsi dijelaskan di sana). Environment variable sungguhan selalu mengalahkan `.env`.

| Variabel | Bawaan | Fungsi |
|---|---|---|
| `AYUMI_ALLOWED_IPS` | *(kosong = semua)* | **Allowlist IP**: IP / CIDR (dipisah koma) yang boleh mengakses aplikasi sama sekali |
| `AYUMI_ALLOWED_HOSTS` | *(kosong = semua)* | **Allowlist host**: hanya nama domain ini yang boleh dipakai mengakses aplikasi (wildcard `*.contoh.id` didukung) |
| `AYUMI_TRUST_PROXY` | `false` | Percaya header IP klien dan `X-Forwarded-Proto` dari proxy |
| `AYUMI_PROXY_HEADER` | `X-Forwarded-For` | Header berisi IP klien asli (`CF-Connecting-IP`, `X-Real-IP`, …) |
| `AYUMI_TRUSTED_PROXIES` | – | Hanya proxy ini yang dipercaya (IP/CIDR) |
| `AYUMI_COOKIE_SECURE` | `false` | Tandai cookie sesi `Secure` (**set `true` di HTTPS**) |
| `ALLOWED_ORIGINS` | – | Origin tambahan yang diizinkan (domain sendiri) |
| `HOST` / `PORT` | `127.0.0.1` / `8081` | Alamat listen |
| `AYUMI_ROOTS` | folder home | Folder yang bisa dipilih untuk session baru (dipisah `:`) |
| `AYUMI_PERMISSION_MODE` | `auto` | Mode izin tool bawaan: `auto`, `acceptEdits`, `manual`, `plan` |
| `AYUMI_AUTH_RATE_LIMIT` | `10` | Permintaan auth / menit / IP |
| `AYUMI_MAX_UPLOAD_MB` / `AYUMI_UPLOAD_MAX_TOTAL_MB` / `AYUMI_UPLOAD_TTL_DAYS` | `20` / `2048` / `30` | Ukuran lampiran per file, total penyimpanan, dan masa simpan |
| `AYUMI_DATA_DIR` | folder config OS | Lokasi akun, 2FA, dan state (mode `0600`) |
| `CLAUDE_BIN`, `CLAUDE_HOME`, `CLAUDE_DESKTOP_DIR` | otomatis | Lokasi CLI dan data Claude |
| `AYUMI_STATIC_DIR` | – | Sajikan frontend hasil build dari folder ini (otomatis di Docker) |
| `AYUMI_BIND`, `AYUMI_PORT`, `WORKSPACE_DIR`, `CLAUDE_CODE_VERSION` | lihat file | Opsi khusus Docker |

### Contoh allowlist IP

```env
AYUMI_ALLOWED_IPS=127.0.0.1, ::1, 192.168.1.0/24, 100.64.0.0/10
```

Permintaan dari alamat lain mendapat `403` sebelum diproses apa pun (termasuk halaman login).
Sertakan alamat Anda sendiri, kalau tidak browser Anda terkunci (aplikasinya tetap jalan; ubah `.env`
lalu restart).

## Akses jarak jauh (HP)

Ayumi Link bisa menjalankan perintah di mesin Anda lewat Claude, jadi perlakukan seperti membuka SSH.
**Jangan pernah publikasikan lewat HTTP biasa.** Pilihan yang baik:

- **Tailscale** (atau VPN mesh privat lain): set `AYUMI_BIND` ke IP Tailscale (Docker) atau `HOST`
  (jalan langsung), dan tambahkan rentang tailnet ke `AYUMI_ALLOWED_IPS`.
- **Cloudflare Tunnel / reverse proxy (HTTPS):** biarkan aplikasi di loopback, akhiri TLS di proxy,
  lalu set `AYUMI_COOKIE_SECURE=true`, `AYUMI_TRUST_PROXY=true`, `AYUMI_TRUSTED_PROXIES=<ip proxy>`, dan
  header yang *di-set* proxy Anda (`CF-Connecting-IP`, `X-Real-IP`). `X-Forwarded-For` bisa dipalsukan
  klien bila proxy hanya menambahkan ke dalamnya.

Daftar periksa sebelum akses jarak jauh: HTTPS ✔ · 2FA aktif ✔ · kode pemulihan tersimpan ✔ ·
allowlist atau jaringan privat ✔ · `AYUMI_PERMISSION_MODE` yang Anda nyaman ✔.

## Model keamanan

- **Autentikasi:** satu akun pemilik; argon2id; 2FA TOTP wajib dan tidak bisa dimatikan; kode TOTP
  yang sama tidak bisa dipakai ulang; 8 kode pemulihan sekali pakai; 5 kegagalan mengunci akun 15 menit;
  pembatasan laju per IP; sesi habis setelah 7 hari tidak aktif (maksimal 30 hari) dan bisa dicabut.
- **Aksi sensitif** (ganti password, buat ulang kode pemulihan, ganti autentikator) menuntut password +
  kode 2FA lagi, dan mengeluarkan semua perangkat lain.
- **Pengerasan web:** pemeriksaan origin, mutasi hanya JSON, cookie `SameSite=Strict` HttpOnly, CSP ketat,
  respons API `no-store`, tanpa permintaan eksternal (font di-bundle), tanpa kredensial di log.
- **Filesystem:** pemilih folder hanya menampilkan folder di dalam `AYUMI_ROOTS`, menguraikan symlink,
  menyembunyikan folder berawalan titik, dan server memvalidasi ulang saat session dibuat.
- **Lampiran:** hanya gambar (PNG/JPG/GIF/WEBP), PDF, `.xlsx`, `.csv`, dan `.docx`. Jenis diverifikasi dari **isi** file (magic bytes dan struktur dokumen), bukan ekstensi, jadi `.exe` yang diganti nama ditolak; SVG/HTML/skrip/arsip/`.doc`/`.xls` lama dan file Office bermakro ditolak. Nama file dibersihkan, id acak, file disimpan di folder data dengan mode `0600`, disajikan dengan `nosniff` + CSP sandbox (hanya gambar yang inline), dihitung ke kuota penyimpanan, dan kedaluwarsa otomatis. Word/Excel diubah menjadi teks di server (dibatasi ukuran, dicek zip-bomb) sebelum dibaca Claude; Claude hanya bisa membaca folder lampiran yang Anda kirim lewat `--add-dir`. `bypassPermissions`. Tool yang butuh persetujuan ditolak pada mode
  `manual` karena CLI non-interaktif tidak bisa bertanya.
- **Batas yang diketahui:** hanya satu pengguna; rahasia TOTP tersimpan tanpa enkripsi di `auth.json`
  (mode `0600`, lindungi folder datanya); chat dari Ayumi Link tidak muncul di Claude Desktop.

Menemukan kerentanan? Lihat [SECURITY.md](SECURITY.md).

## Keterbatasan

- Session yang **dibuat di Ayumi Link tidak muncul di Claude Desktop**. Session dari Desktop bisa dilanjutkan di Ayumi Link.
- Ubah nama/pindah/sembunyikan hanya berlaku di Ayumi Link.
- Lampiran: `.doc`/`.xls` lama tidak didukung (simpan sebagai `.docx`/`.xlsx`). Tanggal Excel tampil sebagai angka serial dan lembar yang sangat besar dipotong (3.000 baris per lembar) pada teks yang diterima Claude.
- Antarmuka saat ini hanya berbahasa Indonesia.
- Mode Docker tidak bisa meniru grup Claude Desktop (lihat catatan di atas).

## Pengembangan

```bash
make test                              # go vet + go test + type check frontend
cd backend  && go test ./...
cd frontend && npx tsc -b && npm run lint
```

Kontribusi disambut baik, lihat [CONTRIBUTING.md](CONTRIBUTING.md).

## Lisensi

[MIT](LICENSE). Font bawaan: IBM Plex Sans (SIL Open Font License 1.1).

## Sanggahan

Disediakan "apa adanya", tanpa jaminan. Anda bertanggung jawab atas cara Anda mengeksposnya dan atas
kepatuhan pada ketentuan layanan dan kebijakan penggunaan Anthropic untuk paket atau API key yang Anda pakai.
