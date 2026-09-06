import { useQuery } from '@tanstack/react-query'
import { BookOpen, Phone, Lock, Download, User, MessageCircle } from 'lucide-react'
import toast from 'react-hot-toast'
import { portalApi, downloadBriefingPDF } from '../../utils/api'
import { usePortalMe, usePaymentSettled, paymentRequiredMessage } from '../../hooks/usePortalMe'
import PortalLockedNotice from '../../components/PortalLockedNotice'
import type { TourLeader } from '../../types'

interface BatchLeaderResponse {
  tour_leader?: TourLeader | null
  wa_group_link?: string
}

export default function PortalBriefingPage() {
  const { data, isLoading } = usePortalMe()
  const settled = usePaymentSettled()

  // FR-BRIEF-02/03: fetch tour leader + WA group link assigned to this batch (§6.1).
  // Withheld until the payment is confirmed too — a tour leader should not be
  // called by someone whose departure is not yet certain (story 21) — so the
  // query only runs once it would be answered.
  const { data: batchData, error: batchError } = useQuery({
    queryKey: ['portal-batch-leader'],
    queryFn: () =>
      portalApi.get<{ success: boolean; data: BatchLeaderResponse | null }>('/portal/batch-leader')
        .then(r => r.data.data),
    enabled: settled,
    retry: (_count, err) => paymentRequiredMessage(err) === null,
  })
  const tlData = batchData?.tour_leader ?? null
  const waGroupLink = batchData?.wa_group_link

  const briefingActive = data?.briefing_active
  const countdown = data?.days_to_depart
  const participant = data?.participant

  if (isLoading) return (
    <div className="space-y-4 animate-pulse">
      <div className="h-32 bg-gray-200 rounded-xl" />
      <div className="h-48 bg-gray-200 rounded-xl" />
    </div>
  )

  // Two gates, and the participant is told which one they are behind. Payment
  // comes first, matching the order the API applies them in: being told "H-14"
  // when the real obstacle is an unpaid invoice would send them off to wait.
  if (!settled || paymentRequiredMessage(batchError)) {
    return (
      <PortalLockedNotice
        title="Briefing Digital"
        reason="Materi briefing, kontak tour leader, dan grup WhatsApp batch terbuka setelah pembayaran Anda dikonfirmasi. Briefing sendiri baru aktif 14 hari sebelum keberangkatan."
      />
    )
  }

  if (!briefingActive) {
    return (
      <div className="space-y-4">
        <h2 className="font-semibold text-gray-800">Briefing Digital</h2>
        <div className="bg-white rounded-xl border p-8 text-center">
          <Lock className="mx-auto mb-3 text-gray-300" size={40} />
          <p className="text-gray-600 font-medium mb-1">Briefing belum aktif</p>
          <p className="text-sm text-gray-400">
            Materi briefing akan tersedia 14 hari sebelum keberangkatan.
          </p>
          {countdown !== undefined && countdown > 14 && (
            <div className="mt-3 inline-block bg-gray-50 rounded-lg px-4 py-2">
              <p className="text-sm font-semibold text-gray-700">
                Tersedia dalam <span className="text-emerald-600">{countdown - 14} hari</span> lagi
              </p>
            </div>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <BookOpen className="text-emerald-600" size={20} />
          <h2 className="font-semibold text-gray-800">Briefing Digital</h2>
        </div>
        {/* Download PDF briefing */}
        <button
          onClick={() => downloadBriefingPDF().catch(() =>
            toast.error('Gagal mengunduh briefing, coba lagi.'),
          )}
          className="flex items-center gap-1.5 px-3 py-1.5 border border-emerald-600 text-emerald-700 text-xs rounded-lg hover:bg-emerald-50"
        >
          <Download size={13} /> Unduh PDF
        </button>
      </div>

      {/* Tour Leader Profile */}
      <div className="bg-emerald-50 border border-emerald-200 rounded-xl p-4">
        <h3 className="font-semibold text-sm text-emerald-800 mb-3 flex items-center gap-2">
          <User size={16} /> Tour Leader Anda
        </h3>
        {tlData ? (
          <div className="flex items-start gap-3">
            {tlData.photo_path && (
              <img src={tlData.photo_path} alt={tlData.name}
                className="w-12 h-12 rounded-full object-cover border-2 border-emerald-300" />
            )}
            <div>
              <p className="font-semibold text-emerald-800">{tlData.name}</p>
              {tlData.specialization && (
                <p className="text-xs text-emerald-600">Spesialisasi: {tlData.specialization}</p>
              )}
              {tlData.experience_years > 0 && (
                <p className="text-xs text-emerald-600">{tlData.experience_years} tahun pengalaman</p>
              )}
              {tlData.bio && <p className="text-xs text-gray-600 mt-1">{tlData.bio}</p>}
              {tlData.emergency_phone && (
                <a href={`tel:${tlData.emergency_phone}`}
                  className="flex items-center gap-1 text-xs text-emerald-700 mt-1">
                  <Phone size={11} /> {tlData.emergency_phone}
                </a>
              )}
            </div>
          </div>
        ) : (
          <p className="text-sm text-emerald-600">
            Tour leader akan diperkenalkan melalui grup WhatsApp dan diumumkan 2 hari sebelum keberangkatan.
          </p>
        )}
      </div>

      {/* WA Group Link (§6.1: grup WA briefing) */}
      {waGroupLink && (
        <div className="bg-green-50 border border-green-200 rounded-xl p-4">
          <h3 className="font-semibold text-sm text-green-800 mb-2 flex items-center gap-2">
            <MessageCircle size={16} /> Grup WhatsApp Batch
          </h3>
          <p className="text-xs text-green-700 mb-3">
            Bergabung dengan grup WA untuk koordinasi sebelum keberangkatan, briefing, dan update terkini dari tour leader.
          </p>
          <a
            href={waGroupLink}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 px-4 py-2 bg-green-600 text-white text-sm rounded-lg hover:bg-green-700"
          >
            <MessageCircle size={14} /> Gabung Grup WhatsApp
          </a>
        </div>
      )}

      {/* Tata tertib */}
      <div className="bg-white rounded-xl border p-4">
        <h3 className="font-semibold text-sm text-gray-800 mb-3">📋 Tata Tertib Perjalanan</h3>
        <ul className="space-y-2 text-sm text-gray-600">
          {[
            'Hadir di titik kumpul 3 jam sebelum jadwal penerbangan',
            'Bawa paspor asli dan semua dokumen perjalanan',
            'Batasan bagasi: 20kg bagasi check-in + 7kg kabin',
            'Patuhi jadwal dan arahan tour leader selama perjalanan',
            'Hormati budaya dan adat setempat negara tujuan',
            'Tidak meninggalkan grup tanpa izin tour leader',
            'Jaga kebersihan dan ketertiban di setiap fasilitas',
          ].map((item, i) => (
            <li key={i} className="flex items-start gap-2">
              <span className="text-emerald-500 mt-0.5 shrink-0">✓</span>
              {item}
            </li>
          ))}
        </ul>
      </div>

      {/* Barang bawaan */}
      <div className="bg-white rounded-xl border p-4">
        <h3 className="font-semibold text-sm text-gray-800 mb-3">🧳 Barang Bawaan Disarankan</h3>
        <div className="grid grid-cols-2 gap-x-4 gap-y-1.5 text-sm text-gray-600">
          {[
            'Pakaian sesuai cuaca', 'Obat-obatan pribadi',
            'Kartu debit/kredit', 'Adaptor universal',
            'Powerbank', 'Fotokopi dokumen penting',
            'Uang tunai secukupnya', 'Kamera/HP + charger',
            'Jaket/sweater', 'Sunscreen & toiletries',
          ].map((item, i) => (
            <span key={i} className="flex items-center gap-1.5">
              <span className="text-emerald-400">•</span> {item}
            </span>
          ))}
        </div>
      </div>

      {/* Panduan keimigrasian */}
      <div className="bg-white rounded-xl border p-4">
        <h3 className="font-semibold text-sm text-gray-800 mb-3">🛂 Panduan Keimigrasian</h3>
        <ul className="space-y-1.5 text-sm text-gray-600">
          <li>• Antri di jalur WNA/Foreigner saat tiba di luar negeri</li>
          <li>• Isi formulir imigrasi sebelum atau saat mendarat</li>
          <li>• Ikuti arahan tour leader untuk proses custom clearance</li>
          <li>• Jangan membawa barang terlarang (cek regulasi negara tujuan)</li>
          <li>• Simpan boarding pass hingga tiba di destinasi akhir</li>
        </ul>
      </div>

      {/* Kontak darurat */}
      <div className="bg-blue-50 border border-blue-200 rounded-xl p-4">
        <h3 className="font-semibold text-sm text-blue-800 mb-2">📞 Kontak Darurat</h3>
        <div className="space-y-1.5 text-sm">
          <p className="text-blue-700">
            <strong>Hotline Pintour:</strong> +62 811-xxxx-xxxx (24 jam)
          </p>
          <p className="text-blue-600">
            Tour leader akan berbagi kontak langsung via grup WhatsApp.
          </p>
        </div>
      </div>

      {/* Paket info */}
      {participant && (
        <div className="bg-gray-50 rounded-xl border p-4 text-xs text-gray-500 space-y-1">
          <p><strong>Paket:</strong> {participant.package_name}</p>
          {participant.batch_departure_date && (
            <p><strong>Keberangkatan:</strong>{' '}
              {new Date(participant.batch_departure_date).toLocaleDateString('id-ID', {
                weekday: 'long', day: 'numeric', month: 'long', year: 'numeric',
              })}
            </p>
          )}
        </div>
      )}
    </div>
  )
}
