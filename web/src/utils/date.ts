// Calendar-date rendering.
//
// The API sends due dates and departure dates as plain "YYYY-MM-DD" — they are
// days, not moments. `new Date("2026-08-15")` reads that as midnight UTC, and
// `toLocaleDateString` then renders it in the viewer's zone: west of Greenwich
// that prints 14 August. Splitting the string instead keeps the day the server
// meant, wherever the page is open.

const DAY_ONLY = /^(\d{4})-(\d{2})-(\d{2})$/

// parseCalendarDate turns an API date into a Date pinned to local midnight, so
// no formatting of it can move the day. Timestamps and anything unparseable
// fall through to the built-in parser.
export function parseCalendarDate(value: string | null | undefined): Date | null {
  if (!value) return null
  const parts = DAY_ONLY.exec(value)
  if (parts) {
    return new Date(Number(parts[1]), Number(parts[2]) - 1, Number(parts[3]))
  }
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

const DEFAULT_OPTIONS: Intl.DateTimeFormatOptions = {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
}

// formatDate renders an API date in Indonesian. An absent or unreadable value
// renders as an em dash rather than "Invalid Date".
export function formatDate(
  value: string | null | undefined,
  options: Intl.DateTimeFormatOptions = DEFAULT_OPTIONS,
): string {
  const date = parseCalendarDate(value)
  return date ? date.toLocaleDateString('id-ID', options) : '—'
}

// formatDateLong spells the month out, for the headings that have room for it.
export function formatDateLong(value: string | null | undefined): string {
  return formatDate(value, { day: 'numeric', month: 'long', year: 'numeric' })
}

// WITH_WEEKDAY names the day of the week too — for a departure, which people
// plan around rather than merely note.
export const WITH_WEEKDAY: Intl.DateTimeFormatOptions = {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  year: 'numeric',
}
