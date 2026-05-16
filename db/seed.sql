-- ============================================================
-- Seed Data: pintour-travel
-- ============================================================

-- ============================================================
-- Destinations
-- ============================================================
INSERT INTO destinations (id, name, country, description, image_url) VALUES
  ('a1000000-0000-0000-0000-000000000001', 'Jepang', 'Japan',
   'Negeri matahari terbit dengan perpaduan budaya tradisional dan teknologi modern. Terkenal dengan Gunung Fuji, kuil-kuil kuno, dan kuliner khas ramen & sushi.',
   'https://images.unsplash.com/photo-1493976040374-85c8e12f0c0e?w=800'),
  ('a1000000-0000-0000-0000-000000000002', 'Bali', 'Indonesia',
   'Pulau Dewata yang terkenal dengan pantai eksotis, pura megah, seni budaya yang kaya, dan keindahan alam yang memukau.',
   'https://images.unsplash.com/photo-1537996194471-e657df975ab4?w=800'),
  ('a1000000-0000-0000-0000-000000000003', 'Korea Selatan', 'South Korea',
   'Negeri ginseng dengan budaya K-Pop, kuliner lezat, istana bersejarah, dan kota modern Seoul yang tak pernah tidur.',
   'https://images.unsplash.com/photo-1517154421773-0529f29ea451?w=800'),
  ('a1000000-0000-0000-0000-000000000004', 'Turki', 'Turkey',
   'Persimpangan budaya Timur dan Barat. Hagia Sophia, Cappadocia dengan balon udara, dan Pamukkale yang memukau.',
   'https://images.unsplash.com/photo-1524231757912-21f4fe3a7200?w=800'),
  ('a1000000-0000-0000-0000-000000000005', 'Swiss', 'Switzerland',
   'Surga pegunungan Alpen dengan pemandangan salju abadi, kota-kota bersih, coklat legendaris, dan jam tangan mewah.',
   'https://images.unsplash.com/photo-1527668752968-14dc70a27c95?w=800'),
  ('a1000000-0000-0000-0000-000000000006', 'Dubai', 'UAE',
   'Kota futuristik di padang pasir dengan gedung pencakar langit tertinggi di dunia, belanja mewah, dan pengalaman desert safari.',
   'https://images.unsplash.com/photo-1512453979798-5ea266f8880c?w=800');

-- Update existing package to reference destination
UPDATE tour_packages SET destination_id = 'a1000000-0000-0000-0000-000000000001' WHERE slug = 'jepang';

-- ============================================================
-- Tour Packages
-- ============================================================
INSERT INTO tour_packages (id, destination_id, title, slug, description, price, price_label, duration_days, max_participants, min_participants, package_type, cover_image_url, is_active) VALUES
  ('b1000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000001',
   'Jepang Classic: Tokyo & Kyoto', 'jepang-classic',
   'Paket wisata Jepang paling populer! Jelajahi Tokyo yang modern dan Kyoto yang klasik. Nikmati sakura di Ueno Park, kunjungi Kuil Senso-ji, dan rasakan pengalaman memakai kimono di Gion Kyoto.',
   18500000, 'Mulai dari Rp 18.500.000/orang', 8, 20, 2, 'regular',
   'https://images.unsplash.com/photo-1536098561742-ca998e48cbcc?w=800', true),

  ('b1000000-0000-0000-0000-000000000002', 'a1000000-0000-0000-0000-000000000001',
   'Japan Premium: Tokyo, Osaka & Mount Fuji', 'japan-premium',
   'Paket premium Jepang 12 hari yang mencakup kota-kota utama plus pengalaman mendaki Gunung Fuji. Hotel bintang 5, guide berbahasa Indonesia, dan kunjungan ke Universal Studios Japan.',
   32000000, 'Mulai dari Rp 32.000.000/orang', 12, 15, 2, 'premium',
   'https://images.unsplash.com/photo-1540959733332-eab4deabeeaf?w=800', true),

  ('b1000000-0000-0000-0000-000000000003', 'a1000000-0000-0000-0000-000000000002',
   'Bali Honeymoon Escape', 'bali-honeymoon',
   'Paket romantis khusus pasangan baru menikah. Akomodasi villa private pool di Ubud, sunset dinner di Jimbaran Bay, spa couple, dan banyak lagi kenangan indah bersama pasangan.',
   12500000, 'Mulai dari Rp 12.500.000/pasang', 5, 1, 1, 'premium',
   'https://images.unsplash.com/photo-1558618666-fcd25c85cd64?w=800', true),

  ('b1000000-0000-0000-0000-000000000004', 'a1000000-0000-0000-0000-000000000002',
   'Bali Adventure & Culture', 'bali-adventure',
   'Eksplorasi Bali dari sisi petualangan dan budaya! Rafting di Sungai Ayung, trekking Gunung Batur, kunjungi Pura Besakih, dan saksikan tari Kecak di Uluwatu.',
   8900000, 'Mulai dari Rp 8.900.000/orang', 6, 25, 4, 'regular',
   'https://images.unsplash.com/photo-1555400038-63f5ba517a47?w=800', true),

  ('b1000000-0000-0000-0000-000000000005', 'a1000000-0000-0000-0000-000000000003',
   'Korea Selatan: Seoul & Jeju Island', 'korea-seoul-jeju',
   'Paket Korea 9 hari yang menggabungkan kesibukan Seoul dan keindahan alam Pulau Jeju. Kunjungi Istana Gyeongbokgung, Nami Island, Lotte World, dan nikmati kuliner street food Korea.',
   21000000, 'Mulai dari Rp 21.000.000/orang', 9, 20, 2, 'regular',
   'https://images.unsplash.com/photo-1578637387939-43c525550085?w=800', true),

  ('b1000000-0000-0000-0000-000000000006', 'a1000000-0000-0000-0000-000000000004',
   'Turki Magical: Istanbul & Cappadocia', 'turki-magical',
   'Rasakan keajaiban Turki! Terbang dengan balon udara di Cappadocia saat matahari terbit, kunjungi Hagia Sophia yang megah, jelajahi Grand Bazaar, dan berendam di Pamukkale.',
   24500000, 'Mulai dari Rp 24.500.000/orang', 10, 20, 4, 'regular',
   'https://images.unsplash.com/photo-1541432901042-2d8bd64b4a9b?w=800', true),

  ('b1000000-0000-0000-0000-000000000007', 'a1000000-0000-0000-0000-000000000005',
   'Swiss Alps Luxury Tour', 'swiss-alps',
   'Paket mewah Swiss 7 hari menikmati keindahan pegunungan Alpen. Naik Jungfraujoch si puncak Eropa, city tour Zurich dan Lucerne, cruise di Danau Genewa, dan belanja jam tangan asli Swiss.',
   45000000, 'Mulai dari Rp 45.000.000/orang', 7, 12, 2, 'premium',
   'https://images.unsplash.com/photo-1506905925346-21bda4d32df4?w=800', true),

  ('b1000000-0000-0000-0000-000000000008', 'a1000000-0000-0000-0000-000000000006',
   'Dubai Gold & Glamour', 'dubai-glamour',
   'Paket Dubai 5 hari penuh kemewahan. Kunjungi Burj Khalifa, berbelanja di Dubai Mall, desert safari dengan BBQ malam, naik dhow cruise di Dubai Creek, dan nikmati brunch mewah.',
   17800000, 'Mulai dari Rp 17.800.000/orang', 5, 20, 2, 'regular',
   'https://images.unsplash.com/photo-1551801691-f0bce83d4f68?w=800', true),

  ('b1000000-0000-0000-0000-000000000009', 'a1000000-0000-0000-0000-000000000003',
   'Korea K-Pop & Food Tour', 'korea-kpop',
   'Paket khusus pecinta K-Pop dan kuliner Korea! Kunjungi SM Town, beli merchandise K-Pop di Myeongdong, ikuti K-Pop dance class, dan kulineran dari Bibimbap sampai Korean BBQ.',
   19500000, 'Mulai dari Rp 19.500.000/orang', 7, 20, 2, 'regular',
   'https://images.unsplash.com/photo-1601621915196-2621bfb0cd6e?w=800', true),

  ('b1000000-0000-0000-0000-000000000010', 'a1000000-0000-0000-0000-000000000002',
   'Bali Family Fun Package', 'bali-family',
   'Paket Bali ramah keluarga untuk semua usia! Water Boom Bali, Bali Zoo & Safari, Tanah Lot sunset, kuliner khas Bali, dan banyak aktivitas seru untuk anak-anak.',
   7500000, 'Mulai dari Rp 7.500.000/orang', 4, 30, 4, 'regular',
   'https://images.unsplash.com/photo-1602002418082-a4443e081dd1?w=800', true);

-- ============================================================
-- Itinerary Items
-- ============================================================

-- Jepang Classic (b1000000-...-001)
INSERT INTO itinerary_items (tour_package_id, day_number, title, description, location, start_time, end_time, activity_type, sort_order) VALUES
  ('b1000000-0000-0000-0000-000000000001', 1, 'Keberangkatan dari Jakarta', 'Berkumpul di Terminal 3 Bandara Soekarno-Hatta. Check-in dan boarding penerbangan menuju Tokyo Narita.', 'Jakarta - Bandara Soeta', '06:00', '23:00', 'transport', 1),
  ('b1000000-0000-0000-0000-000000000001', 2, 'Tiba di Tokyo – Asakusa & Akihabara', 'Tiba pagi di Tokyo. Setelah check-in hotel, kunjungi Kuil Senso-ji di Asakusa dan nikmati suasana Nakamise Shopping Street. Sore hari eksplorasi Akihabara.', 'Tokyo', '10:00', '21:00', 'sightseeing', 2),
  ('b1000000-0000-0000-0000-000000000001', 3, 'Tokyo: Shibuya, Harajuku & Shinjuku', 'Mulai dari Shibuya Crossing yang ikonik, belanja di Takeshita Street Harajuku, picnic di Yoyogi Park, dan malam hari menikmati gemerlap Shinjuku.', 'Tokyo', '09:00', '22:00', 'sightseeing', 3),
  ('b1000000-0000-0000-0000-000000000001', 4, 'Day Trip ke Nikko', 'Perjalanan ke Nikko untuk mengunjungi Tosho-gu Shrine yang megah dan indah, Air Terjun Kegon, dan Danau Chuzenji.', 'Nikko', '07:00', '20:00', 'sightseeing', 4),
  ('b1000000-0000-0000-0000-000000000001', 5, 'Tokyo ke Kyoto via Shinkansen', 'Naik Shinkansen (bullet train) dari Tokyo ke Kyoto. Sore hari kunjungi Fushimi Inari dengan ribuan gerbang torii merah yang ikonik.', 'Kyoto', '09:00', '19:00', 'transport', 5),
  ('b1000000-0000-0000-0000-000000000001', 6, 'Kyoto: Kuil & Budaya', 'Kunjungi Kinkaku-ji (Kuil Emas), Arashiyama Bamboo Grove, Tenryu-ji, dan pengalaman memakai kimono di kawasan Gion.', 'Kyoto', '08:00', '21:00', 'culture', 6),
  ('b1000000-0000-0000-0000-000000000001', 7, 'Nara & Osaka', 'Pagi hari ke Nara untuk memberi makan rusa dan kunjungi Todai-ji Temple. Siang pindah ke Osaka, malam kulineran di Dotonbori.', 'Nara - Osaka', '08:00', '22:00', 'sightseeing', 7),
  ('b1000000-0000-0000-0000-000000000001', 8, 'Kembali ke Jakarta', 'Check-out hotel, belanja oleh-oleh di bandara, dan penerbangan kembali ke Jakarta.', 'Osaka - Jakarta', '07:00', '23:00', 'transport', 8);

-- Bali Adventure (b1000000-...-004)
INSERT INTO itinerary_items (tour_package_id, day_number, title, description, location, start_time, end_time, activity_type, sort_order) VALUES
  ('b1000000-0000-0000-0000-000000000004', 1, 'Tiba di Bali – Ubud', 'Dijemput di Ngurah Rai Airport. Check-in villa di Ubud. Sore hari jalan-jalan di Monkey Forest dan Pasar Ubud.', 'Ubud, Bali', '13:00', '20:00', 'sightseeing', 1),
  ('b1000000-0000-0000-0000-000000000004', 2, 'White Water Rafting Ayung River', 'Pagi hari rafting seru di Sungai Ayung sepanjang 9 km. Siang istirahat di resort. Sore kunjungi Tegalalang Rice Terraces yang indah.', 'Ubud, Bali', '08:00', '18:00', 'adventure', 2),
  ('b1000000-0000-0000-0000-000000000004', 3, 'Trekking Gunung Batur', 'Berangkat dini hari untuk trekking ke puncak Gunung Batur (1717m). Saksikan matahari terbit yang spektakuler dari puncak. Siang berendam di hot spring.', 'Kintamani, Bali', '02:00', '16:00', 'adventure', 3),
  ('b1000000-0000-0000-0000-000000000004', 4, 'Pura Besakih & Kintamani', 'Kunjungi Pura Besakih "Ibu dari segala pura" di lereng Gunung Agung. Menikmati pemandangan Danau Batur dari Kintamani sambil makan siang.', 'Karangasem, Bali', '08:00', '18:00', 'culture', 4),
  ('b1000000-0000-0000-0000-000000000004', 5, 'Uluwatu & Kecak Dance', 'Kunjungi Pura Uluwatu di tebing 70m di atas samudra Hindia. Sore hari saksikan pertunjukan Tari Kecak yang memukau saat sunset.', 'Uluwatu, Bali', '14:00', '21:00', 'culture', 5),
  ('b1000000-0000-0000-0000-000000000004', 6, 'Belanja & Kembali', 'Pagi bebas untuk belanja oleh-oleh di Seminyak dan Kuta. Siang check-out dan transfer ke bandara.', 'Kuta, Bali', '08:00', '16:00', 'leisure', 6);

-- Korea Seoul Jeju (b1000000-...-005)
INSERT INTO itinerary_items (tour_package_id, day_number, title, description, location, start_time, end_time, activity_type, sort_order) VALUES
  ('b1000000-0000-0000-0000-000000000005', 1, 'Jakarta ke Seoul', 'Keberangkatan dari Bandara Soeta menuju Incheon International Airport Seoul.', 'Jakarta - Seoul', '08:00', '23:00', 'transport', 1),
  ('b1000000-0000-0000-0000-000000000005', 2, 'Seoul: Istana & Hanok Village', 'Kunjungi Istana Gyeongbokgung, Bukchon Hanok Village, dan Insadong untuk belanja kerajinan tangan Korea.', 'Seoul', '09:00', '21:00', 'culture', 2),
  ('b1000000-0000-0000-0000-000000000005', 3, 'Nami Island & Petite France', 'Day trip ke Nami Island yang terkenal berkat serial drama Winter Sonata. Kunjungi juga Petite France dan Garden of Morning Calm.', 'Gapyeong', '07:00', '20:00', 'sightseeing', 3),
  ('b1000000-0000-0000-0000-000000000005', 4, 'Seoul: Shopping & K-Pop', 'Kulineran di Gwangjang Market, belanja di Myeongdong, dan malam hari hiburan di Hongdae.', 'Seoul', '10:00', '23:00', 'leisure', 4),
  ('b1000000-0000-0000-0000-000000000005', 5, 'Seoul ke Jeju Island', 'Penerbangan domestik ke Pulau Jeju. Kunjungi Hallasan Mountain, Seongsan Ilchulbong (Sunrise Peak), dan Manjanggul Cave.', 'Jeju', '08:00', '20:00', 'sightseeing', 5),
  ('b1000000-0000-0000-0000-000000000005', 6, 'Jeju Exploration', 'Kunjungi Cheonjiyeon Waterfall, Jeongbang Waterfall, Teddy Bear Museum, dan pantai eksotis Hamdeok Beach.', 'Jeju', '09:00', '19:00', 'sightseeing', 6),
  ('b1000000-0000-0000-0000-000000000005', 7, 'Jeju: Olle Trail & Dongmun Market', 'Hiking santai di Jeju Olle Trail, belanja di Dongmun Traditional Market, dan menikmati kuliner khas Jeju seperti black pork BBQ.', 'Jeju', '08:00', '21:00', 'adventure', 7),
  ('b1000000-0000-0000-0000-000000000005', 8, 'Kembali ke Seoul', 'Penerbangan ke Seoul. Sore hari bebas untuk belanja di Lotte Duty Free atau COEX Mall.', 'Seoul', '10:00', '21:00', 'leisure', 8),
  ('b1000000-0000-0000-0000-000000000005', 9, 'Kembali ke Jakarta', 'Check-out hotel, menuju Incheon Airport, dan penerbangan kembali ke Jakarta.', 'Seoul - Jakarta', '07:00', '22:00', 'transport', 9);

-- Dubai (b1000000-...-008)
INSERT INTO itinerary_items (tour_package_id, day_number, title, description, location, start_time, end_time, activity_type, sort_order) VALUES
  ('b1000000-0000-0000-0000-000000000008', 1, 'Tiba di Dubai', 'Tiba di Dubai International Airport. Check-in hotel di kawasan Downtown Dubai. Malam hari menikmati Dubai Fountain Show.', 'Dubai', '14:00', '22:00', 'sightseeing', 1),
  ('b1000000-0000-0000-0000-000000000008', 2, 'Burj Khalifa & Dubai Mall', 'Naik ke dek observasi Burj Khalifa di lantai 124 untuk pemandangan 360 derajat. Belanja dan kuliner di Dubai Mall — mall terbesar di dunia.', 'Downtown Dubai', '10:00', '22:00', 'sightseeing', 2),
  ('b1000000-0000-0000-0000-000000000008', 3, 'Desert Safari', 'Pagi bebas. Sore hari desert safari: dune bashing, camel riding, sandboarding, dan malam BBQ dinner dengan pertunjukan belly dance di bawah bintang.', 'Dubai Desert', '14:00', '23:00', 'adventure', 3),
  ('b1000000-0000-0000-0000-000000000008', 4, 'Palm Jumeirah & Atlantis', 'Kunjungi Palm Jumeirah, foto-foto di depan Atlantis Hotel, bersenang-senang di Aquaventure Waterpark, dan sunset dari Boardwalk.', 'Palm Jumeirah', '10:00', '20:00', 'leisure', 4),
  ('b1000000-0000-0000-0000-000000000008', 5, 'Old Dubai & Kembali', 'Pagi hari kunjungi Old Dubai: Dubai Creek Dhow Cruise, Gold Souk, Spice Souk, dan Al Fahidi Historical Neighbourhood. Siang check-out dan ke bandara.', 'Old Dubai', '08:00', '16:00', 'culture', 5);

-- ============================================================
-- Package Galleries
-- ============================================================
INSERT INTO package_galleries (tour_package_id, image_url, caption, sort_order) VALUES
  -- Jepang Classic
  ('b1000000-0000-0000-0000-000000000001', 'https://images.unsplash.com/photo-1493976040374-85c8e12f0c0e?w=800', 'Kuil Senso-ji di Asakusa Tokyo', 1),
  ('b1000000-0000-0000-0000-000000000001', 'https://images.unsplash.com/photo-1528360983277-13d401cdc186?w=800', 'Fushimi Inari Shrine Kyoto', 2),
  ('b1000000-0000-0000-0000-000000000001', 'https://images.unsplash.com/photo-1540959733332-eab4deabeeaf?w=800', 'Gunung Fuji dengan Sakura', 3),
  ('b1000000-0000-0000-0000-000000000001', 'https://images.unsplash.com/photo-1503899036084-c55cdd92da26?w=800', 'Shibuya Crossing Tokyo', 4),
  -- Japan Premium
  ('b1000000-0000-0000-0000-000000000002', 'https://images.unsplash.com/photo-1536098561742-ca998e48cbcc?w=800', 'Tokyo Skyline by Night', 1),
  ('b1000000-0000-0000-0000-000000000002', 'https://images.unsplash.com/photo-1524413840807-0c3cb6fa808d?w=800', 'Universal Studios Japan Osaka', 2),
  -- Bali Honeymoon
  ('b1000000-0000-0000-0000-000000000003', 'https://images.unsplash.com/photo-1558618666-fcd25c85cd64?w=800', 'Villa Private Pool Ubud', 1),
  ('b1000000-0000-0000-0000-000000000003', 'https://images.unsplash.com/photo-1537996194471-e657df975ab4?w=800', 'Sunset di Jimbaran Bay', 2),
  ('b1000000-0000-0000-0000-000000000003', 'https://images.unsplash.com/photo-1555400038-63f5ba517a47?w=800', 'Pura Tanah Lot Bali', 3),
  -- Bali Adventure
  ('b1000000-0000-0000-0000-000000000004', 'https://images.unsplash.com/photo-1573790387438-4da905039392?w=800', 'Rafting Sungai Ayung', 1),
  ('b1000000-0000-0000-0000-000000000004', 'https://images.unsplash.com/photo-1592364395653-83e648b20cc2?w=800', 'Trekking Gunung Batur', 2),
  ('b1000000-0000-0000-0000-000000000004', 'https://images.unsplash.com/photo-1602002418082-a4443e081dd1?w=800', 'Tari Kecak Uluwatu', 3),
  -- Korea
  ('b1000000-0000-0000-0000-000000000005', 'https://images.unsplash.com/photo-1517154421773-0529f29ea451?w=800', 'Istana Gyeongbokgung Seoul', 1),
  ('b1000000-0000-0000-0000-000000000005', 'https://images.unsplash.com/photo-1578637387939-43c525550085?w=800', 'Nami Island Musim Gugur', 2),
  ('b1000000-0000-0000-0000-000000000005', 'https://images.unsplash.com/photo-1601621915196-2621bfb0cd6e?w=800', 'Seongsan Ilchulbong Jeju', 3),
  -- Dubai
  ('b1000000-0000-0000-0000-000000000008', 'https://images.unsplash.com/photo-1512453979798-5ea266f8880c?w=800', 'Burj Khalifa di Siang Hari', 1),
  ('b1000000-0000-0000-0000-000000000008', 'https://images.unsplash.com/photo-1551801691-f0bce83d4f68?w=800', 'Desert Safari Dubai', 2),
  ('b1000000-0000-0000-0000-000000000008', 'https://images.unsplash.com/photo-1582672060674-bc2bd808a8b5?w=800', 'Palm Jumeirah dari Atas', 3);

-- Gallery untuk paket Jepang okinawa yang sudah ada
INSERT INTO package_galleries (tour_package_id, image_url, caption, sort_order)
SELECT id, 'https://images.unsplash.com/photo-1590559899731-a382839e5549?w=800', 'Pantai Okinawa yang Indah', 1
FROM tour_packages WHERE slug = 'jepang'
ON CONFLICT DO NOTHING;

INSERT INTO package_galleries (tour_package_id, image_url, caption, sort_order)
SELECT id, 'https://images.unsplash.com/photo-1601823984263-b87b59798b70?w=800', 'Kastil Shuri Okinawa', 2
FROM tour_packages WHERE slug = 'jepang'
ON CONFLICT DO NOTHING;

-- ============================================================
-- Testimonials
-- ============================================================
INSERT INTO testimonials (tour_package_id, customer_name, content, rating, photo_url, is_published) VALUES
  ('b1000000-0000-0000-0000-000000000001', 'Budi Hartono',
   'Perjalanan ke Jepang bersama Pintour benar-benar tak terlupakan! Guide-nya sangat profesional dan ramah, semua jadwal berjalan tepat waktu. Wajib coba paket ini!',
   5, 'https://i.pravatar.cc/150?img=1', true),
  ('b1000000-0000-0000-0000-000000000001', 'Siti Rahayu',
   'Ini pertama kalinya saya ke Jepang dan saya sangat puas dengan pelayanan Pintour. Hotelnya bagus-bagus, makanan halal tersedia, dan guide selalu siap membantu.',
   5, 'https://i.pravatar.cc/150?img=5', true),
  ('b1000000-0000-0000-0000-000000000004', 'Reza Pratama',
   'Rafting di Bali seru banget! Paket Bali Adventure ini worth every rupiah. Guide berpengalaman, aman, dan banyak foto bagus. Pasti balik lagi bawa keluarga!',
   5, 'https://i.pravatar.cc/150?img=3', true),
  ('b1000000-0000-0000-0000-000000000003', 'Dewi & Andi Susanto',
   'Honeymoon kami di Bali via Pintour sungguh sempurna. Villa private pool-nya romantis banget, dinner di Jimbaran kayak di surga. Terima kasih Pintour sudah buat momen kami istimewa!',
   5, 'https://i.pravatar.cc/150?img=9', true),
  ('b1000000-0000-0000-0000-000000000005', 'Mega Wulandari',
   'Korea trip bareng Pintour asik banget! Bisa ke Nami Island, shopping di Myeongdong, dan nyobain semua street food Korea. Guide-nya juga ngerti banget soal budaya Korea.',
   4, 'https://i.pravatar.cc/150?img=7', true),
  ('b1000000-0000-0000-0000-000000000008', 'Hendra Kurniawan',
   'Dubai trip 5 hari bareng Pintour sangat memuaskan. Desert safari-nya seru, Burj Khalifa views-nya luar biasa, dan pengaturan hotelnya top. Highly recommended!',
   5, 'https://i.pravatar.cc/150?img=11', true),
  (NULL, 'Nurul Hidayah',
   'Sudah 3x pakai Pintour untuk wisata luar negeri dan tidak pernah kecewa. Tim Pintour selalu responsif sejak konsultasi sampai kembali ke Indonesia. Terpercaya!',
   5, 'https://i.pravatar.cc/150?img=13', true),
  ('b1000000-0000-0000-0000-000000000007', 'Agus Setiawan',
   'Swiss Alps tour adalah pengalaman terbaik hidup saya. Salju di Jungfraujoch, pemandangan Lucerne, semua lebih indah dari foto. Worth the premium price!',
   5, 'https://i.pravatar.cc/150?img=15', true);

-- ============================================================
-- Inquiries (Leads)
-- ============================================================
INSERT INTO inquiries (full_name, email, phone, destination, tour_package_id, num_people, budget, duration_days, departure_date, notes, status, wa_link) VALUES
  ('Ahmad Fauzi', 'ahmad.fauzi@email.com', '08112345678', 'Jepang', 'b1000000-0000-0000-0000-000000000001', 4, 80000000, 8, '2026-07-15', 'Minta info ketersediaan Juli', 'new', 'https://wa.me/628001234567'),
  ('Lisa Permata', 'lisa.p@gmail.com', '08287654321', 'Bali', 'b1000000-0000-0000-0000-000000000003', 2, 30000000, 5, '2026-06-20', 'Untuk honeymoon, ada paket upgrade?', 'contacted', 'https://wa.me/628001234567'),
  ('Dian Kusuma', 'diankusuma@work.com', '081398765432', 'Korea', NULL, 3, 70000000, 9, '2026-08-10', 'Minta custom paket untuk keluarga kecil', 'new', 'https://wa.me/628001234567'),
  ('Bambang Santoso', 'bambang.s@email.com', '085612345678', 'Dubai', 'b1000000-0000-0000-0000-000000000008', 6, 120000000, 5, '2026-09-05', 'Untuk acara gathering kantor, ada harga group?', 'proposal', 'https://wa.me/628001234567'),
  ('Yuni Astuti', 'yuni.astuti@gmail.com', '087812345678', 'Swiss', 'b1000000-0000-0000-0000-000000000007', 2, 100000000, 7, '2026-12-20', 'Ingin merayakan anniversary ke-10', 'new', 'https://wa.me/628001234567'),
  ('Rizki Maulana', 'rizki.m@email.com', '089612345678', 'Turki', 'b1000000-0000-0000-0000-000000000006', 5, 130000000, 10, '2026-10-01', 'Sudah tanya harga, menunggu konfirmasi', 'booked', 'https://wa.me/628001234567');

-- ============================================================
-- Quotations
-- ============================================================
INSERT INTO quotations (id, title, customer_name, customer_email, customer_phone, valid_until, total_price, notes, status) VALUES
  ('c1000000-0000-0000-0000-000000000001', 'Penawaran Jepang Classic - Ahmad Fauzi', 'Ahmad Fauzi', 'ahmad.fauzi@email.com', '08112345678', '2026-06-30', 74000000, 'Harga sudah termasuk tiket pesawat PP, hotel bintang 4, dan guide berbahasa Indonesia.', 'sent'),
  ('c1000000-0000-0000-0000-000000000002', 'Penawaran Dubai Group - Bambang Santoso', 'Bambang Santoso', 'bambang.s@email.com', '085612345678', '2026-07-31', 95400000, 'Harga group 6 orang dengan diskon 10%. Termasuk city tour private bus.', 'draft'),
  ('c1000000-0000-0000-0000-000000000003', 'Penawaran Swiss Honeymoon - Yuni Astuti', 'Yuni Astuti', 'yuni.astuti@gmail.com', '087812345678', '2026-08-31', 92000000, 'Paket premium termasuk romantic dinner di Zurich dan upgrade kamar.', 'draft');

INSERT INTO quotation_items (quotation_id, description, category, quantity, unit_price) VALUES
  -- Ahmad Fauzi Jepang
  ('c1000000-0000-0000-0000-000000000001', 'Tiket Pesawat PP Jakarta-Tokyo (Economy)', 'Transportasi', 4, 7500000),
  ('c1000000-0000-0000-0000-000000000001', 'Hotel Bintang 4 Tokyo (4 malam)', 'Akomodasi', 4, 2500000),
  ('c1000000-0000-0000-0000-000000000001', 'Hotel Bintang 4 Kyoto (2 malam)', 'Akomodasi', 4, 2000000),
  ('c1000000-0000-0000-0000-000000000001', 'Tour Package & Guide Fee', 'Paket', 4, 3000000),
  -- Bambang Dubai Group
  ('c1000000-0000-0000-0000-000000000002', 'Tiket Pesawat PP Jakarta-Dubai (Economy)', 'Transportasi', 6, 6500000),
  ('c1000000-0000-0000-0000-000000000002', 'Hotel Bintang 4 Dubai (4 malam)', 'Akomodasi', 6, 2200000),
  ('c1000000-0000-0000-0000-000000000002', 'Desert Safari + BBQ Dinner', 'Aktivitas', 6, 850000),
  ('c1000000-0000-0000-0000-000000000002', 'Burj Khalifa Admission (At the Top)', 'Aktivitas', 6, 600000),
  -- Yuni Swiss
  ('c1000000-0000-0000-0000-000000000003', 'Tiket Pesawat PP Jakarta-Zurich (Economy)', 'Transportasi', 2, 14000000),
  ('c1000000-0000-0000-0000-000000000003', 'Hotel Bintang 5 Zurich & Lucerne (6 malam)', 'Akomodasi', 2, 7500000),
  ('c1000000-0000-0000-0000-000000000003', 'Jungfraujoch Pass + Swiss Travel Pass', 'Transportasi', 2, 5500000),
  ('c1000000-0000-0000-0000-000000000003', 'Romantic Anniversary Dinner', 'Aktivitas', 2, 1500000);

-- ============================================================
-- Bookings
-- ============================================================
INSERT INTO bookings (id, tour_package_id, booking_code, customer_name, customer_email, customer_phone, departure_date, num_people, total_price, payment_status, booking_status, notes) VALUES
  ('d1000000-0000-0000-0000-000000000001', 'b1000000-0000-0000-0000-000000000004',
   'BK-20260515-0001', 'Reza Pratama', 'reza.p@email.com', '08123456789',
   '2026-07-10', 4, 35600000, 'lunas', 'confirmed', 'Group 4 orang, sudah DP dan lunas'),
  ('d1000000-0000-0000-0000-000000000002', 'b1000000-0000-0000-0000-000000000005',
   'BK-20260520-0002', 'Mega Wulandari', 'mega.w@gmail.com', '08234567890',
   '2026-08-01', 2, 42000000, 'dp', 'confirmed', 'Sudah DP 50%, sisa pelunasan H-7 keberangkatan'),
  ('d1000000-0000-0000-0000-000000000003', 'b1000000-0000-0000-0000-000000000008',
   'BK-20260601-0003', 'Hendra Kurniawan', 'hendra.k@email.com', '08345678901',
   '2026-09-15', 2, 35600000, 'lunas', 'completed', 'Perjalanan selesai, customer sangat puas'),
  ('d1000000-0000-0000-0000-000000000004', 'b1000000-0000-0000-0000-000000000001',
   'BK-20260610-0004', 'Siti Rahayu', 'siti.r@email.com', '08456789012',
   '2026-10-05', 3, 55500000, 'pending', 'confirmed', 'Menunggu konfirmasi pembayaran DP'),
  ('d1000000-0000-0000-0000-000000000005', 'b1000000-0000-0000-0000-000000000003',
   'BK-20260615-0005', 'Dewi Susanto', 'dewi.s@email.com', '08567890123',
   '2026-11-14', 2, 25000000, 'lunas', 'confirmed', 'Honeymoon package, sudah lunas');

INSERT INTO booking_participants (booking_id, full_name, id_type, id_number, date_of_birth, phone) VALUES
  -- Reza group
  ('d1000000-0000-0000-0000-000000000001', 'Reza Pratama', 'ktp', '3201011990123456', '1990-01-15', '08123456789'),
  ('d1000000-0000-0000-0000-000000000001', 'Rina Pratama', 'ktp', '3201014992987654', '1992-03-20', '08123456780'),
  ('d1000000-0000-0000-0000-000000000001', 'Bima Pratama', 'ktp', '3201011988111222', '1988-07-08', '08123456781'),
  ('d1000000-0000-0000-0000-000000000001', 'Citra Pratama', 'ktp', '3201014995333444', '1995-11-25', '08123456782'),
  -- Mega
  ('d1000000-0000-0000-0000-000000000002', 'Mega Wulandari', 'ktp', '3302011995555666', '1995-05-12', '08234567890'),
  ('d1000000-0000-0000-0000-000000000002', 'Tara Wulandari', 'ktp', '3302014998777888', '1998-09-30', '08234567891'),
  -- Hendra
  ('d1000000-0000-0000-0000-000000000003', 'Hendra Kurniawan', 'ktp', '3171011985999000', '1985-04-22', '08345678901'),
  ('d1000000-0000-0000-0000-000000000003', 'Sari Kurniawan', 'ktp', '3171014987001002', '1987-12-03', '08345678902'),
  -- Siti group
  ('d1000000-0000-0000-0000-000000000004', 'Siti Rahayu', 'ktp', '3578011991003004', '1991-06-18', '08456789012'),
  ('d1000000-0000-0000-0000-000000000004', 'Ahmad Rahayu', 'ktp', '3578011989005006', '1989-02-14', '08456789013'),
  ('d1000000-0000-0000-0000-000000000004', 'Fatimah Rahayu', 'ktp', '3578014993007008', '1993-08-27', '08456789014'),
  -- Dewi honeymoon
  ('d1000000-0000-0000-0000-000000000005', 'Dewi Susanto', 'ktp', '3273014994009010', '1994-10-01', '08567890123'),
  ('d1000000-0000-0000-0000-000000000005', 'Andi Susanto', 'ktp', '3273011992011012', '1992-03-15', '08567890124');
