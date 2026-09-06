-- Data tambahan untuk demo sidang. Aditif: tidak menghapus apa pun.
--
-- Nomor WA memakai awalan 62999, yang tidak dialokasikan ke operator Indonesia
-- mana pun. Ini bukan sekadar penanda supaya mudah dipisahkan — konversi lead
-- mengirim kredensial portal dan tagihan lewat WhatsApp sungguhan, jadi nomor
-- karangan yang kebetulan aktif akan menerimanya.
--
-- Versi pertama skrip ini memakai 62888 dan itu benar terjadi: pemilik salah
-- satu nomor membalas "Ini bukan nomor Maya Kusuma". JANGAN mengarang nomor di
-- blok operator (0812/0838/0857/62888/dst) untuk data uji.
--
-- Hapus ulang bila perlu:
--   DELETE FROM participants WHERE phone LIKE '62999%';
--   DELETE FROM leads        WHERE phone LIKE '62999%';

BEGIN;

-- ── Paket baru ───────────────────────────────────────────────────────────────
INSERT INTO packages (id, name, slug, destination, category, duration_days,
    description, base_price, is_active, facilities, requirements, itinerary)
VALUES
 ('aaaa1111-0000-4000-8000-000000000001','Umroh Plus Turki 12 Hari','umroh-plus-turki-12h',
  'Arab Saudi & Turki','umroh',12,'Umroh dilanjutkan ziarah Istanbul dan Bursa.',
  34500000,true,'["Hotel bintang 4","Bus AC","Makan 3x sehari","Muthawif"]',
  '["passport","ktp","rekening_koran"]','[]'),
 ('aaaa1111-0000-4000-8000-000000000002','Jepang Sakura 8 Hari','jepang-sakura-8h',
  'Jepang','reguler',8,'Tokyo, Kyoto, Osaka pada musim sakura.',
  27900000,true,'["Hotel bintang 4","JR Pass","Tour guide"]',
  '["passport","ktp","rekening_koran","visa_support"]','[]'),
 ('aaaa1111-0000-4000-8000-000000000003','Eropa Barat 11 Hari','eropa-barat-11h',
  'Belanda, Belgia, Prancis','reguler',11,'Amsterdam, Brussels, Paris.',
  42500000,true,'["Hotel bintang 4","Kereta cepat","Tour guide"]',
  '["passport","ktp","rekening_koran","visa_support"]','[]')
ON CONFLICT (id) DO NOTHING;

-- ── Keberangkatan ────────────────────────────────────────────────────────────
-- Tanggal relatif terhadap hari ini supaya selalu "mendatang" kapan pun demo
-- dijalankan; satu batch sengaja dibuat lampau untuk menguji filter `upcoming`.
INSERT INTO package_batches (id, package_id, departure_date, return_date, quota,
    price_single, price_double, price_triple, status)
VALUES
 -- H-10: sengaja di dalam jendela H-14 supaya briefing digital ikut terbuka
 -- saat demo, bukan hanya itinerary.
 ('bbbb1111-0000-4000-8000-000000000001','aaaa1111-0000-4000-8000-000000000001',
  CURRENT_DATE + 10, CURRENT_DATE + 22, 40, 38500000, 34500000, 32500000,'tersedia'),
 ('bbbb1111-0000-4000-8000-000000000002','aaaa1111-0000-4000-8000-000000000001',
  CURRENT_DATE + 75, CURRENT_DATE + 87, 40, 38500000, 34500000, 32500000,'tersedia'),
 ('bbbb1111-0000-4000-8000-000000000003','aaaa1111-0000-4000-8000-000000000002',
  CURRENT_DATE + 21, CURRENT_DATE + 29, 25, 31900000, 27900000, 26500000,'tersedia'),
 ('bbbb1111-0000-4000-8000-000000000004','aaaa1111-0000-4000-8000-000000000002',
  CURRENT_DATE + 60, CURRENT_DATE + 68, 25, 31900000, 27900000, 26500000,'penuh'),
 ('bbbb1111-0000-4000-8000-000000000005','aaaa1111-0000-4000-8000-000000000003',
  CURRENT_DATE + 45, CURRENT_DATE + 56, 30, 47500000, 42500000, 40500000,'tersedia'),
 ('bbbb1111-0000-4000-8000-000000000006','aaaa1111-0000-4000-8000-000000000003',
  CURRENT_DATE - 20, CURRENT_DATE - 9,  30, 47500000, 42500000, 40500000,'ditutup')
ON CONFLICT (id) DO NOTHING;

-- ── Leads di setiap status ───────────────────────────────────────────────────
-- Menutupi seluruh nilai leads_status_check kecuali 'peserta' (dibuat lewat
-- konversi di bawah), supaya papan Kanban terisi di semua kolom saat demo.
INSERT INTO leads (id, name, phone, email, package_id, batch_id, pax, message,
    source, status, assigned_to, consent_given)
VALUES
 ('cccc1111-0000-4000-8000-000000000001','Dewi Anggraini','629991000001','dewi.uji@contoh.test',
  'aaaa1111-0000-4000-8000-000000000001','bbbb1111-0000-4000-8000-000000000001',2,
  'Ingin tahu harga untuk berdua.','meta_ads','baru',NULL,true),
 ('cccc1111-0000-4000-8000-000000000002','Rangga Pratama','629991000002','rangga.uji@contoh.test',
  'aaaa1111-0000-4000-8000-000000000002','bbbb1111-0000-4000-8000-000000000003',1,
  'Apakah masih ada kuota sakura?','organic','dihubungi','e3be60ec-011b-4009-b379-bde24cec1744',true),
 ('cccc1111-0000-4000-8000-000000000003','Siti Nurhaliza','629991000003','siti.uji@contoh.test',
  'aaaa1111-0000-4000-8000-000000000003','bbbb1111-0000-4000-8000-000000000005',4,
  'Rombongan keluarga 4 orang.','referral','konsultasi','e3be60ec-011b-4009-b379-bde24cec1744',true),
 ('cccc1111-0000-4000-8000-000000000004','Bagus Setiawan','629991000004','bagus.uji@contoh.test',
  'aaaa1111-0000-4000-8000-000000000001','bbbb1111-0000-4000-8000-000000000001',2,
  'Sudah sepakat, siap proses.','meta_ads','deal','6c204bfe-16b0-493a-8c85-38364656170b',true),
 ('cccc1111-0000-4000-8000-000000000005','Maya Kusuma','629991000005','maya.uji@contoh.test',
  'aaaa1111-0000-4000-8000-000000000002','bbbb1111-0000-4000-8000-000000000003',2,
  'Deal untuk dua orang.','organic','deal','6c204bfe-16b0-493a-8c85-38364656170b',true),
 ('cccc1111-0000-4000-8000-000000000006','Hendra Wijaya','629991000006','hendra.uji@contoh.test',
  'aaaa1111-0000-4000-8000-000000000003','bbbb1111-0000-4000-8000-000000000005',1,
  'Budget belum cocok.','walk_in','tidak_deal','e3be60ec-011b-4009-b379-bde24cec1744',true),
 ('cccc1111-0000-4000-8000-000000000007','Lestari Ayu','629991000007','lestari.uji@contoh.test',
  'aaaa1111-0000-4000-8000-000000000002','bbbb1111-0000-4000-8000-000000000003',3,
  'Minta itinerary lengkap.','meta_ads','baru',NULL,true),
 ('cccc1111-0000-4000-8000-000000000008','Yoga Permana','629991000008','yoga.uji@contoh.test',
  'aaaa1111-0000-4000-8000-000000000001','bbbb1111-0000-4000-8000-000000000002',1,
  'Tanya jadwal keberangkatan berikutnya.','organic','deal','1abc8451-3552-4af2-b615-3e9f7084b86b',true)
ON CONFLICT (id) DO NOTHING;

COMMIT;
