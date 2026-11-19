# FINSEC-TERM // Macro & Threat Intelligence Terminal & Web Dashboard

A production-ready, cross-platform (Windows, Linux, macOS) intelligence suite built with Go. `finsec-term` delivers high-density, Bloomberg-style real-time market and cyber threat intelligence right to your terminal or modern web browser.

![Language](https://img.shields.io/badge/Language-Go%201.24+-00ADD8?style=flat-square&logo=go)
![TUI](https://img.shields.io/badge/TUI-Bubble%20Tea-FFB000?style=flat-square)
![Web](https://img.shields.io/badge/Web-Single--Binary%20Embedded-00E5FF?style=flat-square)
![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-brightgreen?style=flat-square)

---

## ⚡ Dua Mode Tampilan: Web Dashboard & Terminal (TUI)

Aplikasi ini hadir dalam 2 mode yang saling melengkapi menggunakan core intelligence engine yang sama:

| Fitur | 🌐 Web Dashboard (`finsec-web`) | 📟 Terminal TUI (`finsec-term`) |
|:---|:---|:---|
| **Keterbacaan Berita** | ⭐ **100% Lega & Utuh.** Teks wrap otomatis, drawer detail lega, tidak ada teks terpotong. | High-density 4-pane grid di terminal. |
| **Penyajian Data** | 3 Kolom Responsif + Slide-Over Inspector Drawer + Live SSE push. | 4-Pane Grid interaktif (Bubble Tea + Lipgloss). |
| **Navigasi** | Mouse klik + Shortcut Keyboard (`/`, `r`, `m`, `o`, `Esc`). | Keyboard murni (`Tab`, `1-4`, `j/k`, `Enter`, `/`). |
| **Distribusi** | Single Binary (Frontend HTML/CSS/JS di-embed langsung via `embed.FS`). | Single Binary CLI/TUI tanpa CGO. |

---

## 🏛️ Arsitektur Data & Kategori Feed

`finsec-term` mengagregasi, mendeduplikasi, memberi tag, dan menampilkan data secara real-time:

1. **[1] CVE & TECH STACK RADAR**
   - Active CISA Known Exploited Vulnerabilities (KEV) catalog (JSON).
   - Real-time advisories dari BleepingComputer, The Hacker News, SecurityWeek, dan Google Cloud Security.
   - Deteksi khusus stack backend: `PHP`, `Node.js`, `Nginx`, `Go`, `PostgreSQL`, `Redis`, `RabbitMQ`, `Google Cloud`, `Brotli`, `gRPC`, `RCE`, `Zero-Day`.

2. **[2] GLOBAL MACRO & GEOPOLITICS**
   - Berita makroekonomi global dari Investing.com, FXStreet, CoinDesk, dan Google News.
   - Pemantauan The Fed, FOMC, Jerome Powell, CPI, PCE, US Treasury Yields, Crude Oil, Geopolitik, dan Bitcoin.

3. **[3] INDONESIA MARKET, POLITICS & CYBER**
   - Antara News Economy, Google News ID Market & Google News ID Cyber.
   - Pemantauan Bank Indonesia (BI Rate), Rupiah (USD/IDR), IHSG, SBN, OJK, APBN, Danantara, dan kebijakan fiskal.
   - Pemantauan insiden siber lokal: BSSN, CSIRT, kebocoran data, ransomware perbankan/fintech.

4. **[4] INSPECTOR / DETAIL DRAWER**
   - Detail artikel lengkap: Judul utuh tanpa truncate, sumber, timestamp WIB & relatif, severity score, CWE & CVE mapping (link langsung ke NVD NIST), vendor terdampak, deskripsi yang sudah dibersihkan dari tag HTML, dan tombol buka link asli.

---

## 🏷️ Smart Tagging & Severity Badges

- **`[STACK-CRIT]`** (Merah Menyala): Terdeteksi keyword stack backend (PHP, Nginx, Go, Redis, RCE, 0-day) atau katalog CISA KEV.
- **`[ID-MACRO]`** (Amber/Emas): Keyword makro Indonesia (Rupiah, IDR, BI Rate, IHSG, SBN, OJK, APBN, Cadangan Devisa, Inflasi).
- **`[ID-POL]`** (Magenta): Kebijakan fiskal & politik RI (Prabowo, Purbaya, Kemenkeu, Danantara, dsb).
- **`[GLOBAL-MKT]`** (Cyan): Sentimen pasar global (The Fed, FOMC, CPI, Treasury, Crude Oil, Bitcoin).
- **`[CVE]`** (Merah): Deteksi ID Common Vulnerabilities and Exposures (`CVE-\d{4}-\d{4,7}`).

---

## 🚀 Cara Menjalankan

### Opsi 1: Menjalankan Web Dashboard (Rekomendasi ⭐)

Frontend web sudah ter-embed di dalam binary `finsec-web.exe`. Tinggal jalankan:

```powershell
# Jalankan binary:
.\finsec-web.exe

# Atau jalankan via go run:
go run ./cmd/finsec-web
```

> Browser default kamu akan otomatis terbuka ke `http://localhost:8080`.

**Opsi Flag CLI:**
```text
  -p, --port int       Port HTTP server (default 8080)
  -c, --config string  Path ke file konfigurasi YAML (default "config.yaml")
  --open               Otomatis buka browser saat startup (default true)
  -v, --version        Tampilkan informasi versi
```

### Opsi 2: Menjalankan Versi Terminal (TUI)

```powershell
# Jalankan binary terminal:
.\finsec-term.exe

# Atau jalankan via go run:
go run ./cmd/finsec-term
```

---

## ⌨️ Shortcut Keyboard

| Tombol | Web Dashboard (`finsec-web`) | Terminal TUI (`finsec-term`) |
|:---|:---|:---|
| `/` | Fokus ke live search bar | Buka search/filter bar |
| `Esc` | Tutup inspector drawer / bersihkan filter | Batalkan / bersihkan filter |
| `r` | Sync & refresh seluruh feeds secara concurrent | Sync & refresh manual |
| `m` | Tandai seluruh feed sebagai sudah dibaca | Tandai feed di pane aktif sebagai read |
| `o` | Buka artikel asli di tab baru | Buka artikel asli di default browser |
| `Tab` / `1-4` | - | Berpindah fokus antar Pane |
| `j` / `k` | - | Navigasi atas / bawah di list item |
| `Enter` | - | Buka inspector dan tandai read |
| `q` | - | Keluar dari terminal |

---

## 📁 Struktur Proyek

```text
finsec-term/
├── cmd/
│   ├── finsec-web/
│   │   └── main.go              # Entrypoint Web Dashboard server & auto-browser launcher
│   └── finsec-term/
│       └── main.go              # Entrypoint Terminal TUI (Bubble Tea)
├── internal/
│   ├── config/
│   │   ├── config.go            # Model YAML config, default endpoints & keyword rules
│   │   └── config_test.go       # Unit test config
│   ├── model/
│   │   ├── item.go              # Model FeedItem, Severity, Badge enums & filter matcher
│   │   └── item_test.go         # Unit test model
│   ├── fetcher/
│   │   ├── manager.go           # Concurrent fetch orchestrator (errgroup + timeouts)
│   │   ├── cisa.go              # CISA KEV JSON catalog fetcher
│   │   ├── rss.go               # Generic RSS/Atom fetcher (gofeed + custom UA)
│   │   ├── tagger.go            # Regex CVE/CWE extractor, scoring engine & HTML sanitizer
│   │   └── tagger_test.go       # Unit test extractor & sanitizer
│   ├── storage/
│   │   ├── cache.go             # Thread-safe JSON cache untuk deduplikasi & status read
│   │   └── cache_test.go        # Unit test cache
│   ├── ui/                      # Modul UI Terminal (Bubble Tea, Lipgloss, Viewport)
│   │   ├── app.go
│   │   ├── layout.go
│   │   ├── styles.go
│   │   └── browser.go
│   └── web/                     # Modul Web Server & Embedded SPA
│       ├── server.go            # REST API, SSE live push event & static handler
│       └── static/              # Embedded Frontend Assets
│           ├── index.html       # Bloomberg terminal layout
│           ├── style.css        # Tema gelap Bloomberg (#FFB000, #00E5FF, obsidian)
│           └── app.js           # Client-side reactivity, live clocks, filter & SSE
├── config.example.yaml          # Konfigurasi feed siap pakai
├── go.mod                       # Modul Go 1.24+
├── go.sum                       # Checksum dependensi
├── README.md                    # Dokumentasi lengkap
├── finsec-term.exe              # Binary Terminal (TUI)
└── finsec-web.exe               # Binary Web Dashboard (Standalone executable)
```

---

## 📄 License
MIT License. Dibuat untuk security engineers, financial market analysts, dan terminal power users.
