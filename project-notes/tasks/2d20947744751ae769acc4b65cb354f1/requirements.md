# ada issue critical

Task: 2d20947744751ae769acc4b65cb354f1
Status: done
Updated: 2026-10-22T16:15:50Z

Reference material; confirm applicability before reusing.

# PRD: Screenshot setelah login memakai sesi agent sendiri

Status: **usulan untuk persetujuan; implementasi belum dimulai.**

## Overview

Agent perlu mengambil screenshot halaman setelah login, lalu memakai sesi yang sama untuk halaman lain atau ukuran layar berbeda. Saat ini setiap capture membuka Chromium baru tanpa sesi aplikasi.

Perbaikan memakai akun uji berizin minimum dengan sesi khusus agent. Keberhasilan pada aplikasi uji berbasis cookie harus dibuktikan terpisah dari keberhasilan Pukat.

## Goals dan Non-goals

**Scope: medium.** Perubahan mencakup penyimpanan sesi, viewer screenshot, tindakan login terbatas, serta pembersihan lintas siklus tugas.

Hasil terukur: satu login yang didukung dapat dipakai untuk screenshot dua halaman terlindungi pada ukuran desktop dan ponsel, kemudian dibersihkan.

**Di luar scope:** UI atau stylesheet baru, landing/dashboard/login Workspace baru, sesi manusia, browser persisten, JavaScript atau request bebas, otomasi browser umum, MFA, CAPTCHA, SSO, magic link, popup, serta dukungan `localStorage` atau `document.cookie`.

## Scope: masuk dan keluar

Masuk:

- Perluasan tool screenshot existing dengan sesi cookie agent dalam memori server.
- Tool `screenshot_session` terbatas untuk login, logout, reset melalui form email/password dengan hidden CSRF.
- Kredensial dari referensi nama rahasia Project Env.
- Dukungan hanya untuk aplikasi berjalan di project aktif dengan mekanisme autentikasi yang kompatibel.
- Pesan kegagalan yang jelas serta pembaruan dokumentasi penggunaan.

Pukat belum dijanjikan berhasil. Sebelum perubahan luas, tim wajib memastikan mekanisme login, kebutuhan storage/origin, path aplikasi, serta indikator keberhasilannya. Jika tidak kompatibel, hentikan perluasan dan kembalikan desain untuk ditinjau.

## User stories dan acceptance criteria

| Kebutuhan pengguna | Kriteria penerimaan |
|---|---|
| Agent masuk memakai akun uji sendiri. | Form login yang ditentukan berhasil memakai referensi Project Env; indikator halaman terlindungi yang ditentukan muncul. Cookie Workspace dan sesi manusia tidak dipakai. |
| Agent melihat beberapa halaman tanpa login berulang. | Dua route terlindungi dapat difoto pada desktop dan ponsel dengan sesi sama, walaupun Chromium dibuat baru setiap capture. |
| Pengguna mengetahui hasil yang sebenarnya. | Laporan membedakan login gagal, indikator belum muncul, sesi kedaluwarsa, aplikasi berhenti, serta mekanisme tidak didukung. Tidak ada deteksi login otomatis atau klaim sukses hanya karena gambar tersedia. |
| Sesi berakhir dengan aman. | Logout memanggil alur logout aplikasi; reset menghapus sesi lokal agent. Capture berikutnya kembali anonim. Sesi tidak tersedia bagi pengguna, project, atau tugas lain. |

## Requirements

1. **Gate kompatibilitas Pukat.** QA/DevOps memeriksa project yang benar. Catat form, CSRF, cookie/storage, redirect, kebutuhan origin, dua route target, indikator login, serta indikator halaman siap. Researcher melengkapi rujukan implementasi sebelum coding.

2. **Sesi terisolasi.** Identitas sesi ditentukan server dari `UserID + ProjectID + TaskID + containerID`, bukan argumen model. Satu operasi per sesi; timeout dapat dibatalkan, maksimum 90 detik.

3. **Umur sesi terbatas.** Simpan hanya dalam memori dengan kapasitas terbatas. Kedaluwarsa setelah idle 30 menit atau umur total 4 jam. Bersihkan saat reset, tugas selesai/dibatalkan/diarsipkan/dihapus, aplikasi berhenti/restart, atau Workspace restart. Menunggu jawaban pengguna bukan akhir tugas.

4. **Capture memakai sesi tersebut.** Pertahankan Chromium sekali pakai dan socket viewer sementara per capture. Gunakan prefix internal stabil `/run/<slug>/` tanpa `~preview`, dengan pengujian aset serta hydration Next.js. Proxy manusia tetap memakai mekanismenya sendiri.

5. **Rahasia terlindungi.** Tool menerima nama Project Env, bukan nilai kredensial. Nilai disalurkan lewat stdin; tidak masuk argumen proses, transcript, log, atau keluaran tool. Screenshot memakai akun serta data uji yang aman ditampilkan.

6. **Batas keamanan tetap.** Jangan melonggarkan CSP/CSRF, memalsukan origin, mengubah visibilitas situs, atau mengikuti redirect keluar dari aplikasi. Kebutuhan storage browser atau penolakan `Origin: null` dilaporkan sebagai ketidakcocokan.

7. **Arsitektur existing.** Application memiliki kebijakan dan ports; adapter menangani cookie jar, socket, CDP; bootstrap menghubungkan cleanup tugas. Dokumentasi menjelaskan dukungan, masa berlaku, reset, dan batas Pukat.

## Done when

1. **Uji cookie + CSRF:** login akun uji menghasilkan screenshot dua route terlindungi pada desktop dan ponsel; logout/reset mengembalikan keadaan anonim.
2. **Uji batas:** kredensial salah, expiry, lintas scope, operasi bersamaan, cleanup, redirect eksternal, 404, serta timeout memberikan hasil yang sesuai tanpa membocorkan rahasia.
3. **Bukti Pukat terpisah:** laporan mencatat hasil live pada project yang tepat. Jika kompatibel, sertakan screenshot target beserta indikatornya; jika tidak, nyatakan belum selesai dan hentikan perluasan untuk tinjauan desain.

## Assumptions, risiko, dan open questions

- Pengguna sudah memilih **sesi agent sendiri**. Pilihan ini tidak ditanyakan ulang.
- Stack: Go/Fiber dan Chromium/CDP dengan TypeScript capture existing; frontend Svelte tidak perlu berubah.
- [Riset tugas]( /workspace/project-notes/tasks/2d20947744751ae769acc4b65cb354f1/research.md) membuktikan capture belum membawa sesi, tetapi belum mereproduksi Pukat. Rujukan resmi yang dicatat: [CDP](https://chromedevtools.github.io/devtools-protocol/), [Go cookie jar](https://pkg.go.dev/net/http/cookiejar), dan [CSP sandbox](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Content-Security-Policy/sandbox).
- Risiko utama: Pukat membutuhkan storage/origin yang tidak didukung; Next.js masih gagal pada basePath; screenshot menampilkan data akun uji.
- **Penghambat perluasan:** mekanisme autentikasi Pukat, indikator target, serta ketersediaan akun uji berizin minimum belum terkonfirmasi. Needs QA/DevOps untuk reproduksi; Needs Researcher untuk rujukan build; Needs Architect bila pendekatan tidak cocok.
- Persetujuan PRD ini belum diberikan. Orchestrator mengajukan dokumen ini sebelum implementasi.

## Todo

1. **QA, dibantu DevOps/Researcher — laporan kompatibilitas Pukat.** Verifikasi mekanisme auth, project aktif, indikator eksplisit, dan akun uji. Jika tidak cocok, laporkan kebutuhan tinjauan Architect sebelum perubahan luas.
2. **Developer — sesi agent terisolasi.** Verifikasi scope, penguncian, timeout, TTL, kapasitas, serta cleanup; screenshot anonim existing tetap berfungsi.
3. **Developer — screenshot memakai cookie jar agent.** Verifikasi fixture cookie, prefix stabil, dua route terlindungi, aset serta hydration Next.js.
4. **Developer — tindakan sesi terbatas tersedia.** Verifikasi login/logout/reset, hidden CSRF, referensi rahasia, serta pesan gagal. Perbarui dokumentasi dukungan.
5. **QA, Security, Architect, Reviewer — bukti penerimaan final.** Periksa perubahan sesuai peran sekali; pisahkan bukti fixture dari Pukat live. Verifikasi seluruh kriteria dan batas keamanan.
6. **DevOps — perubahan terkirim sesuai repo.** Bukti: `make verify` lulus → commit/push `dev` → Fast hijau pada commit tersebut → `make redeploy-dev` → `/api/health` port 8081 sehat. Jangan redeploy sumber belum dipush atau CI merah.

Pemeriksaan Planner hanya membaca aturan, indeks, riset, dan dokumentasi. Tidak ada file diubah, aplikasi dijalankan, atau tes dieksekusi.