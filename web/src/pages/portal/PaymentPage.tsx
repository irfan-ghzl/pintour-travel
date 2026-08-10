import { useEffect, useRef, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CreditCard, Loader2, CheckCircle, ArrowLeft } from 'lucide-react'
import toast from 'react-hot-toast'
import { portalApi, createPortalPayment } from '../../utils/api'
import { formatDate } from '../../utils/date'
import type { Invoice } from '../../types'
import StatusBadge from '../../components/StatusBadge'

// Snap.js is loaded dynamically; declare the global it attaches.
declare global {
  interface Window {
    snap?: {
      pay: (
        token: string,
        opts: {
          onSuccess?: (r: unknown) => void
          onPending?: (r: unknown) => void
          onError?: (r: unknown) => void
          onClose?: () => void
        },
      ) => void
    }
  }
}

const SNAP_URL =
  (import.meta.env.VITE_MIDTRANS_SNAP_URL as string | undefined) ??
  'https://app.sandbox.midtrans.com/snap/snap.js'

export default function PaymentPage() {
  const { invoiceId = '' } = useParams()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [snapReady, setSnapReady] = useState(false)

  const { data: invoices, isLoading: invLoading } = useQuery({
    queryKey: ['portal-invoices-dash'],
    queryFn: () =>
      portalApi.get<{ success: boolean; data: Invoice[] }>('/portal/invoices').then((r) => r.data.data ?? []),
  })
  const invoice = invoices?.find((i) => i.id === invoiceId)
  const isPaid = invoice?.status === 'lunas'

  // Opening a Snap session is a mutation, not a query: it creates a transaction
  // at the gateway. As a query, react-query refetched it whenever the window
  // regained focus, so every tab switch opened another order — each one leaving
  // the last unpayable while the invoice remembered only the newest.
  //
  // It fires exactly once per invoice, guarded by a ref rather than by the
  // mutation's own state, because React runs effects twice in development.
  const createPayment = useMutation({ mutationFn: () => createPortalPayment(invoiceId) })
  const requested = useRef('')
  useEffect(() => {
    if (!invoice || isPaid || requested.current === invoiceId) return
    requested.current = invoiceId
    createPayment.mutate()
  }, [invoice, isPaid, invoiceId, createPayment])

  const payment = createPayment.data
  const tokenLoading = createPayment.isPending
  const error = createPayment.error

  // Inject Snap.js once we have the client key.
  useEffect(() => {
    if (!payment?.client_key) return
    if (window.snap) {
      setSnapReady(true)
      return
    }
    const script = document.createElement('script')
    script.src = SNAP_URL
    script.setAttribute('data-client-key', payment.client_key)
    script.onload = () => setSnapReady(true)
    document.head.appendChild(script)
    return () => {
      script.remove()
    }
  }, [payment?.client_key])

  function handlePay() {
    if (!payment?.snap_token || !window.snap) return
    window.snap.pay(payment.snap_token, {
      onSuccess: () => {
        qc.invalidateQueries({ queryKey: ['portal-invoices-dash'] })
        navigate('/portal?payment=success')
      },
      onPending: () => {
        toast('Pembayaran sedang diproses…', { icon: '⏳' })
        navigate('/portal')
      },
      onError: () => toast.error('Pembayaran gagal. Silakan coba lagi.'),
      onClose: () => {
        /* user menutup popup tanpa membayar — tidak ada aksi */
      },
    })
  }

  if (invLoading) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="animate-spin text-emerald-600" size={28} />
      </div>
    )
  }

  if (!invoice) {
    return (
      <div className="text-center py-16 text-gray-500">
        <p>Invoice tidak ditemukan.</p>
        <button onClick={() => navigate('/portal')} className="mt-3 text-emerald-600 text-sm">← Kembali ke portal</button>
      </div>
    )
  }

  if (isPaid) {
    return (
      <div className="max-w-md mx-auto text-center py-16">
        <CheckCircle className="mx-auto text-emerald-500 mb-3" size={48} />
        <p className="font-semibold text-gray-800">Invoice ini sudah lunas</p>
        <p className="text-sm text-gray-500 mb-4">Tidak ada pembayaran yang perlu dilakukan.</p>
        <button onClick={() => navigate('/portal')} className="text-emerald-600 text-sm">← Kembali ke portal</button>
      </div>
    )
  }

  return (
    <div className="max-w-md mx-auto space-y-4">
      <button onClick={() => navigate(-1)} className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700">
        <ArrowLeft size={16} /> Kembali
      </button>

      <div className="bg-white rounded-xl border p-5 space-y-3">
        <div className="flex items-center gap-2 text-gray-800">
          <CreditCard className="text-emerald-600" size={20} />
          <h1 className="font-semibold">Pembayaran Invoice</h1>
        </div>

        <div className="text-sm space-y-1.5 text-gray-700">
          <div className="flex justify-between"><span className="text-gray-500">No. Invoice</span><span className="font-mono">{invoice.invoice_number}</span></div>
          <div className="flex justify-between"><span className="text-gray-500">Paket</span><span>{invoice.package_name}</span></div>
          <div className="flex justify-between"><span className="text-gray-500">Jatuh Tempo</span><span>{formatDate(invoice.due_date)}</span></div>
          <div className="flex justify-between items-center"><span className="text-gray-500">Status</span><StatusBadge status={invoice.status} /></div>
        </div>

        <div className="border-t pt-3 flex justify-between items-baseline">
          <span className="text-sm text-gray-500">Total Tagihan</span>
          <span className="text-2xl font-bold text-gray-800">Rp {invoice.amount.toLocaleString('id-ID')}</span>
        </div>

        {error ? (
          <p className="text-sm text-red-600 bg-red-50 rounded-lg p-3">
            Gagal menyiapkan pembayaran. Payment gateway mungkin belum dikonfigurasi.
          </p>
        ) : (
          <button
            onClick={handlePay}
            disabled={tokenLoading || !snapReady || !payment?.snap_token}
            className="w-full py-2.5 bg-emerald-600 text-white rounded-lg text-sm font-medium hover:bg-emerald-700 disabled:opacity-50 flex items-center justify-center gap-2"
          >
            {tokenLoading || !snapReady ? (
              <><Loader2 className="animate-spin" size={16} /> Menyiapkan…</>
            ) : (
              'Lanjutkan Pembayaran'
            )}
          </button>
        )}
        <p className="text-[11px] text-gray-400 text-center">Pembayaran diproses aman oleh Midtrans.</p>
      </div>
    </div>
  )
}
