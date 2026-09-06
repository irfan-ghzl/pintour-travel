import { useQuery } from '@tanstack/react-query'
import axios from 'axios'

import { portalApi } from '../utils/api'
import type { PortalMeResponse } from '../types'

// The portal opens the moment a participant exists; what stays shut until the
// payment is confirmed is the travel content — itinerary, briefing, and the tour
// leader's number. Several screens need to know which side of that line the
// participant is on, so they all read it here rather than each deciding for
// themselves.
//
// The query key is shared with the pages that already fetch /portal/me, so this
// costs no extra request: react-query serves them all from one cache entry, and
// invalidating it after a payment reopens the menus without a re-login.
export const portalMeKey = ['portal-me']

// The sentence the whole locked state is written in, in one place. It appears in
// the sidebar, on the bottom nav, on two dashboard tiles, and on each locked
// page; spelling it at every site meant rewording it was five edits, four of
// which someone would eventually miss.
export const PAYMENT_LOCKED_TITLE = 'Terbuka setelah pembayaran dikonfirmasi'
export const PAYMENT_LOCKED_DETAIL =
  'Halaman ini menunggu pembayaran Anda dikonfirmasi. Sementara itu, Anda tetap bisa ' +
  'membayar, mengunggah bukti transfer, dan melengkapi dokumen perjalanan.'

export function usePortalMe() {
  return useQuery({
    queryKey: portalMeKey,
    queryFn: () =>
      portalApi.get<{ success: boolean; data: PortalMeResponse }>('/portal/me')
        .then(r => r.data.data),
  })
}

// usePaymentSettled reports whether the payment has been confirmed.
//
// Until there is an answer it says yes, so a menu never flickers from open to
// locked, and one failed request does not grey out a portal the participant may
// well be entitled to. Being wrong in this direction costs a 403 on a click;
// being wrong in the other tells someone who has paid that they have not. The
// API is what actually enforces the rule, and it never guesses.
export function usePaymentSettled(): boolean {
  const { data } = usePortalMe()
  if (!data) return true
  return Boolean(data.participant?.is_active)
}

// PAYMENT_REQUIRED is the code the API answers with on a locked endpoint. Pages
// read the code rather than inferring the reason from a status, so what the page
// says and what the server decided cannot drift apart.
export function paymentRequiredMessage(error: unknown): string | null {
  if (!axios.isAxiosError(error)) return null
  const body = error.response?.data as { error?: string; message?: string } | undefined
  if (body?.error !== 'PAYMENT_REQUIRED') return null
  return body.message || PAYMENT_LOCKED_DETAIL
}
