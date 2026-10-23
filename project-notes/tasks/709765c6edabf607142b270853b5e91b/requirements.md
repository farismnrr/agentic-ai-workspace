# untuk Spaces & tools

Task: 709765c6edabf607142b270853b5e91b
Status: working
Updated: 2026-10-23T06:57:04Z

Reference material; confirm applicability before reusing.

# PRD — Space Orchestrator terpisah, Coding Solo

## Overview

Pengguna dapat memilih bekerja langsung melalui Coding atau menyerahkan task kepada tim melalui Orchestrator. Orchestrator juga tersedia sebagai Space MCP tersendiri dengan izin terpisah.

**Scope: medium.** Perubahan memakai layanan task yang ada, tetapi menyentuh pemilihan mode, akses MCP, penanganan kegagalan tim, dan antarmuka Settings.

## Goals dan Non-goals

**Done when:**

1. Buat task Coding dan Orchestrator: keduanya menjalankan mode yang dipilih; task lama mempertahankan mode setelah Continue, Retry, atau pesan lanjutan.
2. Hubungkan klien MCP: empat tool Orchestrator berfungsi dengan izin yang benar; token Coding, akses pengguna lain, dan target task tidak sesuai ditolak.
3. Simulasikan tim tidak tersedia: task Orchestrator tidak berjalan Solo; pemeriksaan akhir, screenshots, `make verify`, Fast CI, dan health setelah pengiriman tercatat lulus.

**Out of scope:** membangun ulang landing, register, login, dashboard, profil, navigasi, atau 404. Tidak menambah loop agent, modul Orchestrator baru, pengaturan default akun, perubahan mode task lama, persetujuan lewat MCP, fitur idempotency, maupun runtime tambahan.

## Scope dan user stories

| Kebutuhan pengguna | Acceptance criteria |
| --- | --- |
| Saya memilih cara mengerjakan task baru. | Composer menawarkan **Coding — langsung** dan **Orchestrator — melalui tim**. Default Coding. Request selalu mengirim `team` eksplisit. Detail task menampilkan mode tersimpan setelah reload. |
| Saya melanjutkan pekerjaan lama tanpa mengubah caranya bekerja. | Continue, Retry, jawaban pertanyaan, dan pesan lanjutan mempertahankan `tasks.team`. Tidak ada migrasi yang menghitung ulang mode lama. |
| Saya mengatur akses kedua Space secara terpisah. | Settings menampilkan Coding dan Orchestrator dengan alamat, tool, serta izin masing-masing. Mengubah Orchestrator tidak mengubah Coding atau izin connected apps yang sudah ada. |
| Saya memulai pekerjaan tim dari aplikasi AI terhubung. | MCP dapat membuat task antre, membaca status, mengirim pesan lanjutan yang sah, dan mengulang task gagal. Respons menyediakan status serta tautan task di UI. |
| Saya mengetahui saat tim belum siap. | Pembuatan task ditolak dengan `team_unavailable` bila dependensi, lead, atau teammate aktif tidak tersedia. Kegagalan setelah antre menghasilkan `failed` dengan petunjuk pemulihan, tanpa menjalankan Solo. |
| Saya tetap mengendalikan persetujuan pekerjaan. | Pertanyaan, PRD, persetujuan, dan penerimaan hasil diselesaikan di UI. Pesan MCP tidak dapat menjawab atau melewati persetujuan yang sedang menunggu. |

## Requirements

### 1. Mode task dan kesiapan tim

- API create baru tanpa `team` memakai `false`; pilihan `true` wajib divalidasi.
- Task Orchestrator memerlukan dependensi tim lengkap, lead valid, dan minimal satu teammate aktif.
- Validasi berlaku sebelum create serta sebelum menjalankan task melalui model API maupun program CLI.
- Jika tim gagal dimuat setelah antre, pertahankan mode, transcript, delegations, dan pending continuation.
- Setelah konfigurasi dipulihkan, pengguna melanjutkan task yang sama melalui Retry/Continue sesuai status yang berlaku.

### 2. Kontrak MCP yang berfungsi

Resource baru: `<CONNECTOR_URL>/api/orchestrator`.

| Tool | Input wajib | Scope | Perilaku |
| --- | --- | --- | --- |
| `start_task` | `projectId`, `prompt` | `orchestrator:run` | Membuat task `Team=true`, `Source=""`, lalu mengantrekannya secara asinkron. |
| `get_task` | `projectId`, `taskId` | `orchestrator:read` | Mengembalikan status tersimpan dan tautan detail task. |
| `send_message` | `projectId`, `taskId`, `text` | `orchestrator:run` | Memakai aturan FollowUp existing untuk pesan lanjutan. |
| `retry_task` | `projectId`, `taskId` | `orchestrator:run` | Mengantrekan ulang task gagal yang sama melalui aturan Retry existing. |

Respons sukses memuat identitas task/proyek, status, dan tautan UI. Respons create menyatakan `queued`, bukan pekerjaan selesai; tool tidak menunggu seluruh pekerjaan tim.

`send_message` menolak task yang masih berjalan, diarsipkan, atau menunggu jawaban/persetujuan. `retry_task` hanya menerima status yang didukung Retry existing.

Tool koordinasi internal seperti `delegate`, `ask_user`, `update_plan`, `finish`, dan `report` tidak diekspor. Katalog Orchestrator tidak menyalin tool file atau eksekusi Coding.

### 3. Batas akses

- Gunakan discovery dan validasi resource OAuth existing untuk Space baru.
- Setiap panggilan memeriksa resource token, scope, switch tool, kepemilikan proyek/task, serta kecocokan proyek task.
- Target task wajib `Team=true` dan `Source=""`; task Solo serta rekaman percakapan connected app ditolak.
- Identitas pemilik berasal dari token terverifikasi, bukan input tool.
- Tool baca default aktif; tool pemicu run default mati. Status aktif tidak menggantikan pemeriksaan scope.
- Token Coding tidak memberi akses ke Orchestrator.
- Panggilan berhasil maupun ditolak dicatat melalui Activity existing.

### 4. Izin kerja dan persetujuan

Switch Orchestrator mengatur akses **MCP eksternal**. Task dari sesi UI tetap mengikuti izin Coding existing.

Lead hanya dapat memakai tool baca yang diizinkan. Teammate mengikuti irisan izin role dan izin Coding pengguna.

Tambahkan capability `supportsApprovals`: Coding `true`, Orchestrator `false`. Settings hanya menampilkan **Skip approvals** untuk Space yang mendukungnya. Persetujuan di dalam task tetap berlaku.

### 5. Arsitektur dan tampilan

Tambahkan `internal/modules/task/adapter/mcp` yang memanggil kontrak application/Service existing. Wiring Space berada di bootstrap; aturan kepemilikan dan status tetap di application.

Pertahankan satu binary dan satu port produksi. UI memakai komponen reusable, daisyUI, serta tema `masihawam-light`. Pilihan mode, status, error, dan petunjuk pemulihan harus terbaca melalui keyboard maupun layar ponsel.

### 6. Dokumentasi

Perbarui `docs/connector.md`, `docs/using-the-app.md`, dan `docs/system-design.md`. Ganti keterangan lama bahwa semua task selalu memakai tim.

Perbarui memory agent terkait perilaku mode, batas MCP, dan kegagalan tim. Jangan mengedit `project-notes/` secara manual.

## Assumptions, risiko, dan open questions

- **Audience:** pengguna existing yang bukan developer serta pemilik proyek yang menghubungkan aplikasi AI.
- **Stack:** Go/Fiber v3, Svelte TypeScript, daisyUI, dan penyimpanan existing; tidak mengganti stack.
- **Look:** mengikuti desain workspace `masihawam-light`; tidak memilih gaya baru.
- Keputusan BA dan Architect dalam brief menggantikan usulan toggle serta pertanyaan lama dalam riset.
- Dasar proyek: [riset task](/workspace/project-notes/tasks/709765c6edabf607142b270853b5e91b/research.md), [desain sistem](/workspace/docs/system-design.md), dan [panduan visual](/workspace/docs/design-system.md). Referensi UI dari tim: [Svelte binding](https://svelte.dev/docs/svelte/bind) dan [daisyUI Select](https://daisyui.com/components/select/).
- **Risiko duplikasi:** `start_task` yang timeout mungkin sudah membuat task. Jangan retry otomatis; dokumentasikan pemeriksaan melalui UI sebelum mencoba lagi.
- **Risiko regresi:** validasi harus mencakup jalur model API dan CLI, termasuk task lama serta kelanjutan yang sedang tertunda.
- **Open questions:** tidak ada keputusan wajib yang masih menghalangi PRD. Persetujuan mulai build tetap diminta oleh orchestrator.

## Todo

Setiap slice build harus meninggalkan proyek dalam keadaan berfungsi dan lulus pemeriksaan yang relevan.

| Urutan | Teammate dan hasil yang terlihat | Verifikasi |
| --- | --- | --- |
| 1 | **Designer:** rancangan perubahan Composer, detail task, dan Settings siap digunakan. | Review default Coding, penjelasan mode, keadaan gagal, akses keyboard, dan penggunaan shell existing. |
| 2 | **Developer:** mode task berjalan konsisten tanpa fallback Solo. | Test default API, pilihan eksplisit, task lama, dependensi tim gagal, serta jalur API/CLI. |
| 3 | **Developer:** operasi application Orchestrator tersedia. | Test ownership, proyek cocok, `Team`, `Source`, status FollowUp/Retry, dan kelanjutan tersimpan. |
| 4 | **Developer:** endpoint MCP Orchestrator dapat dipakai. | Test empat tool, antre asinkron, discovery, resource/scope/switch, Activity, dan penolakan token Coding. |
| 5 | **Developer:** alur pemilihan mode terhubung ke backend. | Uji Composer, label detail, dua Space Settings, persistensi izin, dan capability Skip approvals. |
| 6 | **Developer:** dokumentasi mencerminkan kontrak final. | Cocokkan docs/memory dengan implementasi; jalankan `make verify`. |
| 7 | **Security:** kandidat layak untuk pengujian tampilan. | Periksa isolasi pemilik, batas token, izin run, dan kemungkinan melewati approval. |
| 8 | **DevOps:** kandidat tersedia di lingkungan uji sementara. | Gunakan build → binary terisolasi setelah pemeriksaan Security; pertahankan app pengguna pada 8081. |
| 9 | **Designer, QA, Security, Architect, Reviewer:** bukti pemeriksaan final lengkap. | Designer memeriksa screenshots desktop/ponsel dan styling; QA menguji alur; Security/Architect/Reviewer memeriksa hasil akhir. Ulangi pemeriksaan terdampak setelah perbaikan. |
| 10 | **DevOps:** perubahan terkirim ke lingkungan dev. | `make verify` → commit → push `dev` → Fast pada commit tersebut hijau → `make redeploy-dev` → `/api/health` port 8081 sehat. Tidak menarik sandbox image. |

PRD siap diajukan untuk persetujuan. Pemeriksaan saat ini hanya membaca berkas; tidak ada perubahan, eksekusi kode proyek, pengujian aplikasi, atau deploy yang dilakukan.