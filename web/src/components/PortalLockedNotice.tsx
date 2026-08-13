import { Lock, CreditCard } from 'lucide-react'
import { Link } from 'react-router-dom'

import { PAYMENT_LOCKED_TITLE, PAYMENT_LOCKED_DETAIL } from '../hooks/usePortalMe'

// What a participant sees where the travel content will be, once their payment
// is confirmed.
//
// It says what is waiting and why, and offers the one action that changes the
// answer. Rendering nothing — or a blank page reached by typing the URL — reads
// as a broken feature, which is exactly the conclusion this ticket exists to
// prevent; the locked state has to look deliberate.
export default function PortalLockedNotice({
  title,
  reason,
}: {
  title: string
  reason?: string
}) {
  return (
    <div className="space-y-4">
      <h2 className="font-semibold text-gray-800">{title}</h2>
      <div className="bg-white rounded-xl border p-8 text-center">
        <Lock className="mx-auto mb-3 text-gray-300" size={40} />
        <p className="text-gray-600 font-medium mb-1">{PAYMENT_LOCKED_TITLE}</p>
        {/* The server sends its own sentence with the 403; prefer it, so what the
            page says and what the API decided are literally the same words. */}
        <p className="text-sm text-gray-400 max-w-sm mx-auto">
          {reason ?? PAYMENT_LOCKED_DETAIL}
        </p>
        <Link
          to="/portal/invoices"
          className="mt-5 inline-flex items-center gap-1.5 px-4 py-2 bg-emerald-600 text-white text-sm rounded-lg hover:bg-emerald-700"
        >
          <CreditCard size={14} /> Lihat invoice & bayar
        </Link>
      </div>
    </div>
  )
}
