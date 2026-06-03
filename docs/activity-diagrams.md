# Activity Diagrams — Pintour Travel
**Perbandingan Sistem Manual vs Dengan Automasi**

Legend:
- 🟡 Kuning = langkah manual (rawan human error)
- 🔴 Merah = titik failure / data tidak akurat
- 🟢 Hijau = langkah otomatis oleh sistem

---

## 1. Inquiry Flow

### Sekarang (Manual)

```mermaid
flowchart TD
    S([Mulai]) --> A[Customer buka e-catalog]
    A --> B[Pilih paket wisata]
    B --> C[Klik tombol Chat WA]
    C --> D[WA terbuka - pesan paket pre-filled]
    D --> E[Admin terima WA dari customer]
    E --> F[Admin balas dan diskusi]
    F --> G{Customer mau booking?}
    G -->|Tidak| H([Selesai])
    G -->|Ya| I[Admin buat booking di CMS]
    I --> J([Booking Confirmed])

    style E fill:#fef3c7,stroke:#d97706
    style F fill:#fef3c7,stroke:#d97706
    style I fill:#fef3c7,stroke:#d97706
```

### Dengan Automasi

```mermaid
flowchart TD
    S([Mulai]) --> A[Customer buka e-catalog]
    A --> B[Pilih paket wisata]
    B --> C[Klik tombol Chat WA]
    C --> D[WA terbuka - pesan paket pre-filled]
    D --> E[Admin terima WA dari customer]
    E --> F[AUTO - Notif masuk di dashboard CMS]
    F --> G[Admin balas dan diskusi]
    G --> H{Customer mau booking?}
    H -->|Tidak| I([Selesai])
    H -->|Ya| J[Admin buat booking di CMS]
    J --> K[AUTO - WA konfirmasi ke customer]
    K --> L([Booking Confirmed])

    style F fill:#d1fae5,stroke:#059669
    style K fill:#d1fae5,stroke:#059669
```

---

## 2. Booking Lifecycle + Payment

### Sekarang (Manual)

```mermaid
flowchart TD
    A([Booking Confirmed]) --> B[Admin pantau booking tiap hari]
    B --> C[Admin ubah status booking]
    C --> D[Admin kirim WA ke customer]
    D --> E[Admin tunggu dokumen peserta]
    E --> F[Admin cek dokumen satu per satu]
    F --> G[Customer bayar]
    G --> H[Admin input payment di CMS]
    H --> I[Admin hitung total bayar manual]
    I --> J{Sudah lunas?}
    J -->|Ya| K[Admin ubah status = lunas]
    J -->|Belum| L[Admin ubah status = dp]
    K --> M[Admin ingat hari keberangkatan]
    L --> M
    M --> N{Hari H tiba?}
    N -->|Ingat| O[Admin ubah status = departed]
    N -->|Lupa| P[Status tidak terupdate]
    O --> Q([Trip berjalan])

    style B fill:#fef3c7,stroke:#d97706
    style C fill:#fef3c7,stroke:#d97706
    style D fill:#fef3c7,stroke:#d97706
    style I fill:#fef3c7,stroke:#d97706
    style K fill:#fef3c7,stroke:#d97706
    style L fill:#fef3c7,stroke:#d97706
    style O fill:#fef3c7,stroke:#d97706
    style P fill:#fee2e2,stroke:#dc2626
```

### Dengan Automasi

```mermaid
flowchart TD
    A([Booking Confirmed]) --> B[Admin ubah status booking di CMS]
    B --> C[AUTO - WA notif ke customer]
    C --> D[Customer upload dokumen]
    D --> E[Admin verifikasi dokumen]
    E --> F[AUTO - Cron cek kelengkapan dokumen]
    F --> G{Semua dokumen lengkap?}
    G -->|Belum| H[Admin follow up customer]
    G -->|Lengkap| I[AUTO - Notif admin dokumen siap]
    H --> D
    I --> J[Customer bayar]
    J --> K[Admin input dan verify payment]
    K --> L[AUTO - Hitung ulang payment status]
    L --> M{Total bayar >= tagihan?}
    M -->|Ya| N[AUTO - payment status = lunas]
    M -->|Belum| O[AUTO - payment status = dp]
    N --> P[AUTO - Reminder H-7 H-3 H-1 ke customer]
    O --> P
    P --> Q[AUTO - Status departed saat hari H]
    Q --> R[AUTO - Status completed setelah trip]
    R --> S([Booking Completed])

    style C fill:#d1fae5,stroke:#059669
    style F fill:#d1fae5,stroke:#059669
    style I fill:#d1fae5,stroke:#059669
    style L fill:#d1fae5,stroke:#059669
    style N fill:#d1fae5,stroke:#059669
    style O fill:#d1fae5,stroke:#059669
    style P fill:#d1fae5,stroke:#059669
    style Q fill:#d1fae5,stroke:#059669
    style R fill:#d1fae5,stroke:#059669
```

---

## 3. Quotation Expiry

### Sekarang (Manual)

```mermaid
flowchart TD
    A([Admin buat quotation]) --> B[Status: draft]
    B --> C[Admin isi detail dan item harga]
    C --> D[Admin kirim ke customer]
    D --> E[Status: sent]
    E --> F[Admin ingat tanggal valid_until]
    F --> G{Sudah lewat valid_until?}
    G -->|Ya - admin ingat| H[Admin ubah status = expired]
    G -->|Admin lupa| I[Status tetap sent]
    H --> J([Status: expired])
    I --> K([Data status tidak akurat])

    style F fill:#fef3c7,stroke:#d97706
    style H fill:#fef3c7,stroke:#d97706
    style I fill:#fee2e2,stroke:#dc2626
    style K fill:#fee2e2,stroke:#dc2626
```

### Dengan Automasi

```mermaid
flowchart TD
    A([Admin buat quotation]) --> B[Status: draft]
    B --> C[Admin isi detail dan item harga]
    C --> D[Admin kirim ke customer]
    D --> E[Status: sent]
    E --> F[AUTO - Badge peringatan 3 hari sebelum expired]
    F --> G[AUTO - Cron cek valid_until tiap malam]
    G --> H{valid_until sudah lewat?}
    H -->|Ya| I[AUTO - status = expired]
    H -->|Belum| G
    I --> J([Status expired - otomatis akurat])

    style F fill:#d1fae5,stroke:#059669
    style G fill:#d1fae5,stroke:#059669
    style I fill:#d1fae5,stroke:#059669
```
