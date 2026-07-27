# GuestFlow Platform Remediation TODO

Dokumen ini menurunkan hasil audit end-to-end menjadi pekerjaan implementasi. Urutan mengikuti dampak terhadap operasi acara, bukan urutan menu UI.

## Prinsip Flow Bisnis

`Member -> Tenant -> Acara -> Roster Tamu Acara -> Undangan -> Delivery -> RSVP -> Check-in -> Seating/Gift -> Reporting`

Data kontak tamu adalah master tenant. Data operasional tamu harus selalu memakai roster acara (`event_guests`) agar satu tamu dapat hadir pada banyak acara tanpa mencampur status.

## P0/P1 - Wajib Stabil Sebelum Operasional

- [x] Kirim email manual dan email undangan benar-benar memanggil SMTP, menyimpan `sent` atau `failed`, dan memvalidasi alamat penerima.
- [x] Implementasikan retry pesan dari riwayat dengan endpoint yang membuat attempt baru tanpa mengubah record lama.
- [x] Tambahkan polling/stream riwayat pesan saat konteks acara tetap terbuka.
- [x] Jadikan RSVP list berbasis roster acara dengan virtual row `no_response`, sehingga filter dan update manual mencakup semua tamu aktif yang memiliki undangan.
- [x] Batasi status check-in detail tamu ke acara aktif; riwayat lintas acara tetap tersedia sebagai history.
- [x] Ambil status RSVP nyata pada pencarian check-in, bukan nilai hardcoded.
- [x] Integrasikan menu Kelompok Keluarga dengan API household nyata: list, create, update, delete, add/remove member.
- [x] Tambahkan aksi resend invitation yang membuat delivery attempt baru.
- [x] Bedakan generate invitation dan send invitation pada UI; channel Email harus benar-benar mengirim email.

## P2 - Wajib Dibereskan Sebelum Klaim Fitur Lengkap

- [x] Simpan pengaturan tenant dan preferensi notifikasi ke endpoint persisten.
- [ ] Implementasikan export log; retry pesan sudah membuat attempt baru dari menu Riwayat Pesan.
- [ ] Implementasikan ucapan/doa pada microsite atau sembunyikan form sampai endpoint tersedia.
- [ ] Putuskan worker: implementasikan job nyata atau keluarkan dari deployment sampai queue dipakai.
- [ ] Nonaktifkan endpoint campaign dengan feature flag jika UI memang masih disembunyikan untuk research.
- [ ] Ubah webhook pembayaran agar kegagalan validasi/persistensi tidak selalu di-acknowledge sebagai sukses.
- [ ] Hilangkan tombol 2FA, sesi aktif, upload avatar/logo, dan Hubungi Sales sampai backend tersedia atau implementasikan endpointnya.

## Verifikasi Untuk Setiap Item

- [ ] Unit test service/repository untuk aturan bisnis.
- [ ] Contract test endpoint untuk status sukses dan gagal.
- [ ] Browser E2E pada flow member, acara, roster, undangan, RSVP, check-in, seating, gift, dan delivery.
- [ ] Smoke test Docker lokal dan live tanpa mengirim pesan eksternal kecuali eksplisit diminta.

## Status Batch Saat Ini

- Batch 1: email manual, RSVP roster, check-in event scope, RSVP check-in search, dan message refresh.
- Batch 2: household API/UI, resend invitation, email batch UI, dan pengaturan tenant.
- Batch 3: public wishes, worker/queue, billing webhook, campaign flag, dan cleanup placeholder.
