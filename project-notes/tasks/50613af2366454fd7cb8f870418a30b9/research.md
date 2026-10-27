# Lanjutkan final activation gate dan persiapan Tahap 8 pada `…

Task: 50613af2366454fd7cb8f870418a30b9
Status: done
Updated: 2026-10-26T10:06:11Z

Reference material; confirm applicability before reusing.

**Ringkasan:** Kode dan skrip Stage 7 menjelaskan cara menjalankan gate Docker terisolasi, tetapi secara eksplisit menyisakan pengujian marimo-pair nyata sebagai gate terpisah.  
**Status:** Checkout saat ini `fff7a2b` (`dev`, tertinggal 3 commit dari `origin/dev`), bukan commit sasaran `37ac7bf`; catatan `.stage7/` adalah snapshot tidak terlacak yang saya gunakan untuk memeriksa gate.

## Fakta dari repo

- **Isolasi dan gate yang sudah ada:** [`.stage7/scripts/stage7-docker.sh`]( /workspace/.stage7/scripts/stage7-docker.sh ) membuat volume bernama unik `aw-stage7-*`, container bernama unik, dan menjalankan app pada port `18187`. Gate memeriksa volume `Subpath` untuk `/workspace` dan `/cache`, mount socket egress read-only, keberadaan Claude Code dan Codex, isolasi socket task termasuk runner vendor, uji race/revoke/fail-closed/migrasi, health app, serta persetujuan yang pulih setelah app restart. Skrip berakhir dengan pesan bahwa marimo-pair nyata masih gate eksternal. **Gate ini belum membuktikan** notebook marimo nyata dapat dibuka, WebSocket bertahan, atau revoke/expiry memutus sesi marimo nyata.

- **Cara menjalankan dan flag:** Tidak ada target Make untuk Stage 7. Skrip dijalankan langsung dari root repo dengan `bash scripts/stage7-docker.sh`; di snapshot `.stage7/`, path-nya `scripts/stage7-docker.sh`. Skrip mengambil image `ghcr.io/farismnrr/agentic-ai-workspace:sha-<HEAD pendek>` dan `ghcr.io/farismnrr/agentic-ai-workspace-sandbox:dev`. Ia mengaktifkan flag pada app sementara dengan `-e EGRESS_GRANTS=true`, tanpa `SANDBOX_EGRESS_HOSTS`. Compose meneruskan `EGRESS_GRANTS: ${EGRESS_GRANTS:-false}`. `SANDBOX_EGRESS_HOSTS` menggantikan seluruh default allowlist, bukan menambahkannya; nilai kustom harus memuat registry default yang ingin dipertahankan ([`config.go`]( /workspace/.stage7/internal/platform/config/config.go ), [`compose.dev.yaml`]( /workspace/.stage7/compose.dev.yaml )).

- **Alur grant:** `request_egress(host, reason)` hanya dapat meminta grant; approval, denial, dan revoke dilakukan owner/admin melalui UI. Grant terikat pada project dan task. Revoke/expiry menutup tunnel aktif; task selesai atau gagal mengakhiri grant dan socket task. Registry dibangun ulang untuk grant approved yang belum kedaluwarsa setelah restart; kegagalan database menolak koneksi. Detail dan lokasi kode dirangkum dalam [`agents/egress-grants.md`]( /workspace/.stage7/agents/egress-grants.md ), termasuk `bootstrap/agenttools_egress.go`, `coding/domain/egress_tool.go`, `platform/egressproxy`, dan `bootstrap/egresstasks.go`.

- **Runner untuk gate marimo nyata:** Catatan gate meminta task coding menjalankan klien yang benar-benar dipakai marimo-pair (contohnya Python `ssl`/`httpx`/`websockets`, `curl`, atau Node bila relevan), lalu meminta grant dan approval owner/admin. Claude Code dan Codex mendapat socket task melalui `Engine.Lease` saat diluncurkan sebagai bagian task. Stage 7 hanya menguji keberadaan binary dan isolasi socket; ia tidak membuat task marimo-pair.

- **Notebook dan autentikasi:** Saya tidak menemukan URL notebook atau kredensial/auth marimo di project notes tugas, snapshot, atau environment yang diperiksa. Tidak ada `.env` yang ditemukan pada kedalaman pemeriksaan. `agents/egress-grants.md` menyebut host marimo dalam catatan `.env` historis, tetapi itu bukan bukti URL notebook atau auth tersedia sekarang. Environment hanya menunjukkan `GH_TOKEN` terkait; nilainya tidak dibaca atau dilaporkan.

- **Compose, data, backup, dan rollback:** Compose memasang named volume tetap `agentic-data` ke `/data`; runner memakai subpath di dalam volume itu untuk workspace, cache, dan socket task. Dokumen deployment memberi prosedur backup saat app berhenti: `docker run --rm -v agentic-data:/data -v "$PWD":/backup alpine tar czf /backup/agentic-data.tar.gz -C /data .` ([`docs/deployment.md`]( /workspace/.stage7/docs/deployment.md )). Prosedur restore tidak ditemukan di dokumen. Migrasi `052_egress_grants.sql`, `053_egress_audit.sql`, dan `054_egress_actor_attribution.sql` bersifat append-only. Catatan grant menyatakan rollback setelah migrasi diterapkan memerlukan binary yang tetap menyertakan migrasi terkait; binary lama yang kehilangan migrasi dapat gagal start. Verifikasi skrip menguji journal 052–054, tetapi tidak menguji pemulihan backup.

- **Tujuh skip-worktree:** `git ls-files -v` menandai tujuh file `project-notes/` dengan huruf `S`: `project-notes/index.md` dan enam file di `project-notes/tasks/3b0f1c122f5e93a06b8c7040cd29e90f/` (plan, PRD, requirements, research, `research/57.md`, `research/72.md`). Aturan bahwa `project-notes/` adalah referensi yang dikelola server tercatat di `AGENTS.md` dan indeks proyek; saya tidak menemukan dokumentasi yang secara khusus menerangkan flag skip-worktree itu.

## Rujukan resmi: TLS dan WebSocket

- TLS 1.3 mengizinkan pesan handshake terfragmentasi di beberapa record; implementasi proxy ini hanya menerima satu ClientHello utuh dalam satu record, maksimum 16 KiB. Jadi klien yang fragmentasi atau ClientHello lebih besar dapat ditolak ([RFC 8446, §5.1](https://www.rfc-editor.org/rfc/rfc8446.html#section-5.1); [`clienthello.go`]( /workspace/.stage7/internal/platform/egressproxy/clienthello.go )).
- ALPN menegosiasikan protokol aplikasi dalam handshake TLS; dokumentasi gate menyebut pengujian `h2` dan `http/1.1` ([RFC 7301](https://www.rfc-editor.org/rfc/rfc7301.html)).
- ECH menyembunyikan ClientHello bagian dalam, termasuk SNI; proxy yang harus mencocokkan SNI tidak dapat memeriksa SNI tujuan tersembunyi dan akan menolak ekstensi ECH ([RFC 9849, terbit Maret 2026](https://www.rfc-editor.org/rfc/rfc9849.html); [`clienthello.go`]( /workspace/.stage7/internal/platform/egressproxy/clienthello.go )).
- Dokumentasi nginx menyatakan koneksi WebSocket proxied ditutup jika tidak ada data dari server selama 60 detik secara default; ping berkala atau pengaturan timeout dapat mempertahankannya. Proxy repo memiliki idle timeout 5 menit dan gate meminta pengujian idle lebih dari 5 menit, dengan dan tanpa ping ([dokumentasi nginx](https://nginx.org/en/docs/http/websocket.html); [`agents/egress-grants.md`]( /workspace/.stage7/agents/egress-grants.md )).

## Tidak ditemukan

URL notebook yang bisa dipakai, metode auth dan kredensial untuk marimo-pair, prosedur restore yang terdokumentasi, serta bukti bahwa Stage 7 Docker gate atau marimo E2E sudah dijalankan pada commit `37ac7bf`. Saya tidak menjalankan skrip atau gate.

## Saran

DevOps memerlukan URL notebook dan langkah auth yang aman untuk test instance, lalu perlu mencatat image digest serta hasil gate. Perlakukan gate Stage 7 sebagai bukti isolasi/runtime saja; hasilnya tidak menggantikan koneksi marimo nyata. Sebelum Stage 8, dokumentasikan dan uji restore dari arsip backup pada volume disposable, serta pastikan binary rollback tetap membawa migrasi 052–054. Checkout harus diselaraskan ke `37ac7bf` sebelum mengatribusikan hasil pemeriksaan ke commit itu.