# untuk Spaces & tools

Task: 709765c6edabf607142b270853b5e91b
Status: working
Updated: 2026-10-23T06:57:04Z

Reference material; confirm applicability before reusing.

Toggle Solo / Orchestrator belum ada. Saat ini task baru memakai tim secara default, sementara akses tool diatur terpisah lewat Spaces & tools.
Bukti menunjukkan mode per task sudah tersimpan; cakupan toggle baru dan batas tool yang dimaksud masih perlu diputuskan.

## Fakta yang ditemukan

- **Spaces & tools** menampilkan daftar tool per Space, dikelompokkan sebagai `read`, `write`, dan `exec`. Setiap tool punya checkbox daisyUI; perubahan disimpan lewat `PUT /api/spaces/:slug/tools/:tool`. Jika penyimpanan gagal, UI mengembalikan posisi toggle dan menampilkan pesan error. [SpaceTools.svelte](/workspace/web/src/features/workspace/settings/SpaceTools.svelte:17), [SpacesTab.svelte](/workspace/web/src/features/workspace/settings/SpacesTab.svelte:35), [connectors-api.ts](/workspace/web/src/features/workspace/logic/connectors-api.ts:40)
- Tool baca aktif secara default; tool tulis dan eksekusi tidak menetapkan `DefaultOn`, sehingga default-nya mati. Pilihan disimpan per pengguna, Space, dan tool di `tool_permissions`. [tools.go](/workspace/internal/modules/coding/domain/tools.go:32), [permissions.go](/workspace/internal/modules/space/application/permissions.go:23), [003_tool_permissions.sql](/workspace/internal/platform/database/migrations/003_tool_permissions.sql:1)
- **Mode task:** task baru memakai tim jika layanan tim tersedia dan API tidak mengirim pilihan. Endpoint pembuatan task menerima `team`, tetapi dokumentasi menyebut pengguna tidak memiliki switch untuk mematikannya. Nilai mode disimpan pada `tasks.team` dan dikembalikan dalam data task. [teamchoice.go](/workspace/internal/modules/task/application/teamchoice.go:5), [tasks.go](/workspace/internal/modules/task/adapter/http/tasks.go:101), [store.go](/workspace/internal/modules/task/adapter/sqlite/store.go:38), [029_agent_team.sql](/workspace/internal/platform/database/migrations/029_agent_team.sql:25), [using-the-app.md](/workspace/docs/using-the-app.md:277)
- Ada `agent_teams.use_by_default` dalam penyimpanan tim, tetapi catatan implementasi menyebut kolom itu tidak lagi dibaca. Karena itu, keberadaannya bukan bukti bahwa preferensi Solo/Orchestrator sudah bekerja. [store.go](/workspace/internal/modules/agentteam/adapter/sqlite/store.go:28), [implementation-notes.md](/workspace/agents/implementation-notes.md:65)
- **Penegakan akses tool sudah di server.** Loop task hanya menawarkan tool yang aktif dan memeriksa ulang izin sebelum panggilan. Untuk task tim, lead hanya mendapat tool baca yang aktif; tool berisiko ditolak saat ditawarkan maupun dipanggil. Izin peran teammate mempersempit izin pengguna. MCP juga memeriksa switch dan scope saat panggilan. [toolgate.go](/workspace/internal/modules/task/application/toolgate.go:13), [planlead.go](/workspace/internal/modules/task/application/planlead.go:60), [teamrunner.go](/workspace/internal/modules/task/application/teamrunner.go:65), [guard.go](/workspace/internal/modules/coding/adapter/mcp/guard.go:94)
- **Koordinasi tim bukan tool dalam daftar Spaces & tools.** Lead memakai `delegate`, `ask_user`, `update_plan`, dan `finish`; teammate menyelesaikan pekerjaannya dengan `report`, dan hanya peran yang berwenang yang mendapat `ask_user`. [teamtools.go](/workspace/internal/modules/task/application/teamtools.go:12), [agenttools.go](/workspace/internal/modules/task/application/agenttools.go:10), [planlead.go](/workspace/internal/modules/task/application/planlead.go:15), [teamtools.go](/workspace/internal/modules/task/application/teamtools.go:168)
- Switch Spaces & tools juga berlaku untuk AI apps yang terhubung. Mengubah switch di sana sebagai efek mode berpotensi mengubah akses aplikasi tersebut, bukan hanya perilaku task workspace. [using-the-app.md](/workspace/docs/using-the-app.md:393)

## Usulan dan keputusan terbuka

- **Label:** “Gunakan Orchestrator” dengan nama aksesibel yang tetap sama. Tampilkan status “Orchestrator” saat aktif dan “Solo” saat mati, disertai satu kalimat penjelas. Ini menjelaskan akibat pilihan tanpa mengubah label saat status berganti.
- **Perilaku yang disarankan:** saat aktif, koordinasi lead tetap tersedia; saat mati, task tidak memakai lead atau teammate. Pastikan keputusan tentang tool eksekusi menyebut apakah yang dimaksud hanya perintah (`run_task`, `run_command` dan tool turunannya) atau juga perubahan file (`write`).
- **Jangan mengubah izin Spaces & tools secara global saat mode berganti.** Izin itu dipakai task dan aplikasi terhubung. Terapkan pembatasan mode pada jalur task di server, dan biarkan izin tool pengguna tetap menjadi batas yang berlaku.
- Jika pilihan dibuat **per task**, `tasks.team` sudah menyediakan tempat penyimpanan, dan endpoint pembuatan task sudah menerima pilihan itu. Jika toggle di Settings mengatur **default task baru**, perlu preferensi pengguna yang benar-benar dibaca; jangan mengandalkan `agent_teams.use_by_default` tanpa memastikan alur tersebut diaktifkan.
- Perlu diputuskan apakah mode Orchestrator menonaktifkan eksekusi **lead saja** atau juga teammate. Saat ini lead memang tidak dapat mengubah file atau menjalankan tool berisiko, tetapi teammate memperoleh izin sesuai role dan switch pengguna. Mematikan seluruh tool eksekusi untuk teammate akan mengubah cara tim menyelesaikan pekerjaan.
- Bila toggle menggunakan switch visual, dokumentasi daisyUI menyatakannya sebagai checkbox native. WAI-ARIA menyarankan label switch tidak berubah saat status berubah; `Space` harus dapat mengganti status, dan label atau deskripsi harus bisa diakses. Lihat [daisyUI Toggle](https://daisyui.com/components/form/toggle/), [WAI-ARIA Switch Pattern](https://www.w3.org/WAI/ARIA/apg/patterns/switch/), dan [WCAG 2.2](https://www.w3.org/TR/WCAG22/) (Rekomendasi W3C, 12 Desember 2024). Desain sistem proyek juga meminta fokus terlihat dan kontras AA: [design-system.md](/workspace/docs/design-system.md:140).

## Tidak ditemukan

Belum ditemukan ketentuan yang menetapkan apakah toggle adalah preferensi akun atau pilihan tiap task, maupun apakah “tools eksekusi” mencakup perubahan file dan pekerjaan teammate.

## Saran untuk rencana

Tentukan cakupan mode dan arti “eksekusi” sebelum implementasi. Setelah itu, gunakan `tasks.team` untuk pilihan per task, atau tambahkan preferensi pengguna yang jelas untuk default task baru; pertahankan izin tool dan enforcement server sebagai batas terpisah.

Pemeriksaan hanya membaca berkas dan menelusuri dokumentasi resmi. Tidak ada berkas yang diubah dan kode proyek tidak dijalankan.

---

Ditemukan bahwa Space saat ini adalah endpoint MCP, sedangkan Orchestrator dan koordinasi tim berada di loop task.  
Task lama menyimpan pilihan timnya; risiko utama perubahan adalah task Orchestrator yang diam-diam lanjut sebagai Solo ketika layanan tim bermasalah.

## Temuan dari proyek

- **Registry hanya mendaftarkan Coding.** `newRegistry` berisi `codingSpace()`, dan `newSpaceMounts` membuat handler untuk Space Coding secara khusus. Bentuk Space mencakup slug, scope, dan daftar tools. [spaces.go](/workspace/internal/bootstrap/spaces.go:13), [space_mounts.go](/workspace/internal/bootstrap/space_mounts.go:16), [space.go](/workspace/internal/modules/space/domain/space.go:23)
- **Space terikat pada identitas dan izin tersendiri.** Token MCP divalidasi terhadap URI resource Space. Izin tool disimpan per pengguna, slug Space, dan tool; daftar Settings diambil dari `/api/spaces`. Menambah Orchestrator sebagai Space dapat memakai model izin ini, tanpa mengganti izin Coding. [mount.go](/workspace/internal/modules/space/adapter/http/mount.go:50), [permissions.go](/workspace/internal/modules/space/application/permissions.go:23), [003_tool_permissions.sql](/workspace/internal/platform/database/migrations/003_tool_permissions.sql:1), [SpacesTab.svelte](/workspace/web/src/features/workspace/settings/SpacesTab.svelte:35)
- **Coding memiliki scope terpisah untuk baca, tulis, dan eksekusi.** Tools Coding berasal dari satu katalog. Aplikasi terhubung memakai izin Coding dan project yang dipisahkan per pengguna serta aplikasi. Mengubah switch Coding sebagai efek mode Orchestrator berisiko mengubah akses aplikasi terhubung. [coding.go](/workspace/internal/bootstrap/coding.go:13), [tools.go](/workspace/internal/modules/coding/domain/tools.go:3), [connectedapps.go](/workspace/internal/bootstrap/connectedapps.go:20), [using-the-app.md](/workspace/docs/using-the-app.md:393)
- **Pilihan task belum tampak di UI.** Composer membuat task tanpa mengirim `team`; API menerima nilai itu. Backend memilih tim secara default bila layanan pendukung tersedia, dan menyimpan hasilnya ke `tasks.team`. [Composer.svelte](/workspace/web/src/features/workspace/home/Composer.svelte:26), [tasks-api.ts](/workspace/web/src/features/workspace/logic/tasks-api.ts:6), [tasks.go](/workspace/internal/modules/task/adapter/http/tasks.go:101), [teamchoice.go](/workspace/internal/modules/task/application/teamchoice.go:5), [store.go](/workspace/internal/modules/task/adapter/sqlite/store.go:19)
- **Mode tim mengatur prompt dan izin loop task.** Solo memakai prompt umum; task bertim menambahkan prompt lead, delegate, dan rencana. Lead hanya ditawari tools baca yang aktif, sedangkan akses teammate dipersempit oleh izin peran. Tools koordinasi seperti `delegate`, `ask_user`, dan `update_plan` adalah tools loop task, bukan tools di katalog Coding. [system.go](/workspace/internal/modules/task/application/system.go:12), [leadprompt.go](/workspace/internal/modules/task/application/leadprompt.go:11), [planlead.go](/workspace/internal/modules/task/application/planlead.go:50), [teamrunner.go](/workspace/internal/modules/task/application/teamrunner.go:39), [teamtools.go](/workspace/internal/modules/task/application/teamtools.go:20)
- **Task lama menyimpan modenya.** `Retry` mengantrekan task yang sama, dan pemilihan model saat run kembali memakai `task.Team`. Namun, jika roster gagal dimuat atau tidak ada teammate aktif, runner saat ini berjalan tanpa roster—sehingga task tersimpan sebagai tim tetapi perilakunya menjadi Solo. [actions.go](/workspace/internal/modules/task/application/actions.go:117), [runmodel.go](/workspace/internal/modules/task/application/runmodel.go:82), [teamrunner.go](/workspace/internal/modules/task/application/teamrunner.go:10)

## Rekomendasi cakupan minimal

- Jadikan Space Orchestrator terpisah dengan URI, scopes, katalog tools, dan izin sendiri. Biarkan URI, scopes, tools, dan izin Coding tetap seperti sekarang.
- Untuk **Space Orchestrator yang dipakai aplikasi terhubung**, rekomendasi saya: hanya expose koordinasi. Jangan salin tools baca Coding ke katalog Orchestrator. Untuk task UI, lead dapat terus membaca melalui jalur task yang sudah mematuhi switch baca Coding.
- Tambahkan pilihan eksplisit saat membuat task agar task baru Coding berjalan Solo dan task baru Orchestrator memakai tim—sesuai keputusan yang diberikan orchestrator. Simpan mode pada task, bukan sebagai perubahan global pada izin Spaces & tools.
- Saat layanan tim tidak tersedia, rekomendasi saya adalah hentikan atau tahan task Orchestrator dengan pesan yang dapat ditindaklanjuti. Jangan diam-diam mengubahnya menjadi Solo; pertahankan nilai task lama ketika Continue/Retry.

## Acceptance checks yang disarankan

- Daftar Space menampilkan Coding dan Orchestrator; resource, token audience, scope, dan switch izin masing-masing tetap terpisah.
- Task baru Coding tersimpan dan berjalan sebagai Solo; task baru Orchestrator tersimpan dan berjalan dengan lead/teammate.
- Continue, Retry, dan jawaban lanjutan mempertahankan mode yang tersimpan pada task lama.
- Kegagalan memuat roster atau tidak tersedianya layanan tim terlihat jelas dan tidak membuat task Orchestrator berjalan sebagai Solo tanpa pemberitahuan.
- Uji bahwa switch Coding dan izin connected app tidak berubah ketika Space Orchestrator ditambahkan.

## Referensi teknis resmi

Versi yang tercatat di repo adalah Go 1.27.2 dan Fiber 3.5.0 (`go.mod`), serta Svelte 5.57.1 dan daisyUI 5.7.47 (`web/package.json`). Dokumentasi Fiber saat ini adalah v3; catatan v3 menyebut Go minimum 1.26 dan beberapa perubahan API dari v2. [Fiber v3: What’s New](https://docs.gofiber.io/whats_new/), [Go 1.27 release notes](https://go.dev/doc/go1.27)

Untuk UI, dokumentasi Svelte menjelaskan binding nilai pada form, dan daisyUI menyediakan kelas `select` untuk elemen `<select>` native. Gunakan pola dan tema yang sudah ada; desain sistem proyek mewajibkan fokus terlihat dan kontras AA. [Svelte `bind:`](https://svelte.dev/docs/svelte/bind), [daisyUI Select](https://daisyui.com/components/select/), [design-system.md](/workspace/docs/design-system.md:140)

Satu advisory Fiber yang ditemukan membahas Fiber v2, bukan dependency Fiber v3.5.0 di repo. Advisory Svelte terkait versi sebelum 4.2.19; versi repo adalah Svelte 5.57.1. Ini bukan audit menyeluruh seluruh dependency. [Fiber CSRF advisory](https://github.com/gofiber/fiber/security/advisories/GHSA-94w9-97p3-p368), [Svelte XSS advisory](https://github.com/sveltejs/svelte/security/advisories/GHSA-8266-84wp-wv5c)

## Belum ditemukan — untuk Architect

- Apakah Space Orchestrator merupakan endpoint MCP bagi aplikasi terhubung, pilihan mode di task UI, atau keduanya? Arsitektur saat ini memakai istilah Space untuk endpoint MCP.
- Jika endpoint MCP: bagaimana tool koordinasi mengidentifikasi task aktif, project, dan owner? Tools koordinasi saat ini memerlukan state task yang sedang berjalan.
- Apakah task Orchestrator tanpa teammate aktif harus gagal, menunggu, atau memakai mode lain yang diberitahukan pengguna?

## Suggestions

Minta Business Analyst menetapkan apakah pilihan UI memilih **Space** atau hanya **mode task**, serta perilaku saat roster tidak tersedia. Architect perlu menentukan batas endpoint MCP dan task-loop sebelum PRD disetujui. Tidak ada berkas yang diubah atau kode proyek yang dijalankan; pemeriksaan berakhir sebagai riset baca-saja.

---

Space Orchestrator dapat memakai layanan task dan jalur MCP yang sudah ada; perubahan pentingnya adalah membuat pilihan mode eksplisit dan menutup fallback Solo saat tim tidak siap.  
Dokumentasi resmi mendukung pola resource dan scope terpisah, task queued, select native, serta pemeriksaan aksesibilitas yang sesuai dengan stack repo.

## Design references

**Fakta yang ditemukan**

- Composer belum mengirim pilihan `team`; `createTask` menerima properti itu dan backend menyimpannya pada task. Settings sudah punya daftar Space, alamat MCP, switch tools, dan Skip approvals per Space. [Composer.svelte](/workspace/web/src/features/workspace/home/Composer.svelte:26), [tasks-api.ts](/workspace/web/src/features/workspace/logic/tasks-api.ts:6), [SpacesTab.svelte](/workspace/web/src/features/workspace/settings/SpacesTab.svelte:35)
- Proyek sudah memiliki daisyUI `<select>` native dan `FormField` bersama. `FormField` menghubungkan label ke kontrol dengan `for`/`id`, dan menghubungkan petunjuk atau pesan error melalui `aria-describedby` serta `aria-invalid`. [FormField.svelte](/workspace/web/src/features/workspace/components/FormField.svelte:4), [DatabaseTab.svelte](/workspace/web/src/features/workspace/settings/DatabaseTab.svelte:104)
- Panduan desain menetapkan workspace ringkas, kontras AA, fokus terlihat, dan status yang tidak hanya dibedakan dengan warna. [design-system.md](/workspace/docs/design-system.md:123)

**Usulan untuk Designer/Developer**

- Tambahkan select mode ke Composer dengan pilihan **Coding — langsung** dan **Orchestrator — melalui tim**; nilai awal `false` dan request selalu mengirim `team`.
- Bungkus kontrol dengan `FormField`, beri label yang terlihat dan hint singkat tentang akibat pilihan. Gunakan kelas daisyUI `select` pada elemen `<select>` native; Svelte mendukung `bind:value`, dan dokumentasi daisyUI menunjukkan `select` sebagai styling untuk elemen native. [Svelte bind](https://svelte.dev/docs/svelte/bind), [daisyUI Select](https://daisyui.com/components/select/)
- Pertahankan susunan Settings dan komponen yang ada. Sertakan Orchestrator dalam daftar Space dengan alamat, scopes, tools, dan izin tersendiri; bawa capability `supportsApprovals` pada data Space agar Skip approvals hanya tampil untuk Coding.
- Untuk validasi/error, tampilkan penjelasan teks dan langkah berikutnya, tautkan ke kontrol, dan jangan mengandalkan warna saja. WCAG 2.2 SC 3.3.1 mewajibkan error yang terdeteksi dijelaskan dalam teks; SC 3.3.2 mensyaratkan label atau instruksi untuk input. [WAI error identification](https://www.w3.org/WAI/WCAG22/Understanding/error-identification), [WAI labels and instructions](https://www.w3.org/WAI/WCAG20/Understanding/labels-or-instructions.html)

## Technical references

**Versi dan kompatibilitas**

Versi yang dibaca dari `go.mod` dan `web/package.json`: Go `1.27.2`, Fiber `3.5.0`, MCP Go SDK `1.8.0`, SQLite `modernc.org/sqlite 1.59.0`, Svelte `5.57.1`, dan daisyUI `5.7.47`. Tidak ada kebutuhan PRD yang tampak memerlukan perubahan versi.

- Fiber v3 membutuhkan Go 1.26 atau lebih baru; release resmi Fiber mencantumkan v3.5.0. [Fiber v3 changes](https://docs.gofiber.io/next/whats_new/), [Fiber releases](https://github.com/gofiber/fiber/releases)
- MCP Go SDK v1.8.0 tercatat pada upstream; SDK versi 1.7+ mendukung MCP `2026-07-28` dan versi sebelumnya, dengan dukungan stateful lama tetap tersedia. [Go SDK v1.8.0 package docs](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk%40v1.8.0/mcp), [Go SDK compatibility](https://github.com/modelcontextprotocol/go-sdk)
- Go `1.27.2` bukan patch terbaru: Go `1.27.3`, dirilis 19 Oktober 2026, mencakup perbaikan keamanan `crypto/x509`. Repo mengambil versi Go dari `go.mod` di CI (`setup-go`). Ini temuan baseline di luar kebutuhan implementasi fitur; catat untuk pemilik dependensi/toolchain. [Go release history](https://go.dev/doc/devel/release)
- PRD tidak meminta MCP Tasks extension. Ekstensi tersebut masih berstatus draft pada revisi MCP saat ini. Karena `Create` yang ada sudah menyimpan task berstatus `queued` dan mengantrekannya sebelum kembali, pola yang disarankan adalah MCP mengembalikan status `queued` dan tautan UI sebagai hasil tool biasa. [MCP Tasks extension](https://tasks.extensions.modelcontextprotocol.io/specification/draft/tasks), [task Service]( /workspace/internal/modules/task/application/service.go:50)

**OAuth, scopes, kepemilikan, dan kontrak MCP**

- Proyek memiliki registry Space, resource URI berdasarkan URL Space, Protected Resource Metadata, middleware bearer token, dan pemeriksaan resource/audience saat validasi. Saat ini registry dan mount hanya mendaftarkan Coding. [spaces.go](/workspace/internal/bootstrap/spaces.go:13), [space_mounts.go](/workspace/internal/bootstrap/space_mounts.go:16), [mount.go](/workspace/internal/modules/space/adapter/http/mount.go:50)
- SDK Go menyediakan `RequireBearerToken`; middleware memeriksa token, masa berlaku, scope entry, dan metadata untuk challenge. Handler dapat membaca identitas terverifikasi dari token. [Go SDK authorization](https://go.sdk.modelcontextprotocol.io/protocol/)
- Rekomendasi: tambahkan Space dan katalog tools Orchestrator melalui registry/bootstrap yang sama. Terapkan scope `orchestrator:read` dan `orchestrator:run`, tetapi tetap periksa switch tool pada setiap panggilan. Ambil owner dari token terverifikasi; validasi ownership proyek/task, kecocokan project ID, `Team=true`, dan `Source=""` di kontrak application. Jangan mempercayai owner dari argumen tool.
- Service task sudah memiliki `Create`, `FollowUp`, `Retry`, serta pemeriksaan kepemilikan task untuk operasi terkait. `FollowUp` menolak task archived, running, atau menunggu jawaban; `Retry` hanya menerima task failed. [service.go](/workspace/internal/modules/task/application/service.go:50), [actions.go](/workspace/internal/modules/task/application/actions.go:117), [helpers.go](/workspace/internal/modules/task/application/helpers.go:46)
- Token Coding harus gagal pada resource Orchestrator. Resource URI dan verifikasi audience sudah dibedakan per Space; Activity log juga sudah terhubung di mount Space. [mount.go](/workspace/internal/modules/space/adapter/http/mount.go:78), [guard.go](/workspace/internal/modules/coding/adapter/mcp/guard.go:59)
- Temuan risiko langsung: `teamrunner.roster` mengembalikan `nil` bila dependency tim, roster, atau teammate tidak tersedia; komentar runner menyatakan run kemudian berlangsung Solo. Jalur CLI juga hanya menggunakan lead bila `task.Team` dan `leadOf` menghasilkan lead. Validasi PRD perlu menolak task Orchestrator sebelum create, lalu gagal dengan petunjuk pemulihan bila dependensi hilang saat run. [teamrunner.go](/workspace/internal/modules/task/application/teamrunner.go:10), [cli.go](/workspace/internal/modules/task/application/cli.go:38)
- Upstream MCP SDK memuat advisori lama: DNS rebinding ditambal di `1.4.0`, dan cross-site tool execution serta dua isu parsing ditambal di `1.4.1`; versi repo `1.8.0` berada di atas versi perbaikan tersebut. [DNS rebinding advisory](https://github.com/modelcontextprotocol/go-sdk/security/advisories/GHSA-xw59-hvm2-8pj6), [cross-site advisory](https://github.com/modelcontextprotocol/go-sdk/security/advisories/GHSA-89xv-2j6f-qhc8), [JSON parsing advisory](https://github.com/modelcontextprotocol/go-sdk/security/advisories/GHSA-q382-vc8q-7jhj)
- Fiber juga memiliki advisori upstream; cek rentang versi tiap advisori terhadap `3.5.0` saat review keamanan. Pemeriksaan ini bukan audit seluruh dependency dan tidak menyimpulkan semua dependency bebas advisori. [Fiber security advisories](https://github.com/gofiber/fiber/security)

**Rekomendasi per slice**

1. **Mode task:** beri API tanpa `team` default `false`; validasi `true` terhadap dependency, lead, dan minimal satu teammate. Ulangi validasi di batas run dan fail closed untuk model API maupun CLI. Jangan migrasi atau menghitung ulang mode task lama.
2. **Application Orchestrator:** sediakan operasi tipis di `task/application` untuk mulai, baca, kirim pesan, dan retry; gunakan Service serta validasi ownership/status yang sudah ada. Kembalikan task ID, project ID, status antre, dan URL UI.
3. **MCP adapter:** tempatkan di `internal/modules/task/adapter/mcp`, wire di bootstrap, dan pertahankan pola resource/scope/switch/activity yang ada. Ekspor hanya empat tool PRD; jangan mengekspor tools koordinasi internal.
4. **Settings/UI:** gunakan payload Space yang sama dengan capability `supportsApprovals`; isolasi switch Orchestrator dari izin Coding dan connected apps. Label task mode dari nilai tersimpan setelah reload.
5. **Dokumentasi:** perbarui connector, penggunaan aplikasi, desain sistem, dan memory backend/frontend sesuai batas final.

## Quality checks

**Checklist QA, Security, Architect, Reviewer**

- Create API: field hilang menjadi `team=false`; `team=true` tanpa dependency/lead/teammate ditolak. Coba kedua jalur eksekusi, model API dan CLI.
- Task lama: Continue, Retry, jawaban pertanyaan, pesan lanjutan, serta pending continuation mempertahankan `tasks.team`; tidak ada fallback Solo.
- MCP: discovery, tool list, resource URL, scope, switch, dan Activity diuji untuk keempat tools. Token Coding ke Orchestrator, token Orchestrator ke Coding, scope kurang, switch mati, owner lain, project-task mismatch, task Solo, dan `Source` bukan kosong harus ditolak.
- State: `send_message` menolak running, archived, needs-input/approval; `retry_task` hanya menerima status yang didukung Retry. `start_task` harus cepat kembali sebagai queued. Timeout client dapat terjadi setelah task dibuat; arahkan pengguna cek tautan/UI sebelum mencoba lagi karena PRD tidak mencakup idempotency.
- Approval: MCP tidak bisa menjawab pertanyaan atau melewati approval; Skip approvals hanya ditampilkan untuk Space yang mendukung capability-nya.
- UI: keyboard memilih mode, label/error terbaca screen reader, fokus tampak, pesan tidak hanya dibedakan warna, tampilan ponsel dan desktop sesuai tema.
- Periksa screenshots dan alur sesuai PRD. Lakukan Security review ownership/token/scope/approval serta review Architect/Reviewer setelah perbaikan terdampak.

**Fakta shipping repo**

- `make verify` menjalankan guardrails, pemeriksaan Svelte/TypeScript, lint, tes frontend, build frontend, `go mod tidy -diff`, tes Go, dan lint Go. [Makefile](/workspace/Makefile:78)
- Aturan AGENTS.md mensyaratkan: verify → commit → push `dev` → Fast hijau pada commit → `make redeploy-dev` → cek `/api/health` di port 8081. Untuk perubahan kode ini Fast relevan; jangan menarik sandbox image dan jangan redeploy dari build yang belum dipush/hijau. Bukti pipeline ada di [.github/workflows/fast.yml](/workspace/.github/workflows/fast.yml:24).

## Not found

Belum ada kontrak API final untuk bentuk error MCP `team_unavailable` dan tautan UI yang akan dikembalikan. Advisori yang diperiksa bukan audit lengkap seluruh modul `go.mod`/`package.json`.

## Suggestions

Gunakan operasi application task yang sudah ada dan satu mount Space tambahan; tidak perlu runtime atau server baru. Prioritaskan validasi roster di create dan run karena fallback Solo saat ini nyata di jalur runner. Catat toolchain Go 1.27.2 sebagai isu baseline terpisah karena versi rilis terbaru mencakup patch keamanan `crypto/x509`.

Riset membaca berkas dan mencari dokumentasi resmi. Tidak ada proyek yang diubah, dibangun, dijalankan, atau diuji.