import { useEffect, useRef, useState } from 'react'

export interface UseFileUploadOptions {
  accept?: string
  maxSizeMB?: number
}

export interface UseFileUpload {
  file: File | null
  preview: string | null
  error: string | null
  handleSelect: (f: File | null) => void
  clear: () => void
}

// useFileUpload validates a selected file (type + size) and exposes an image
// preview URL (§5.9). Client-side validation mirrors the portal upload rules.
//
// Every preview URL it hands out is released again: when another file replaces
// it, when the picker is dismissed with nothing chosen, when the selection is
// rejected, and when the component unmounts. A preview holds the whole image in
// memory until it is revoked, and a participant retaking a passport photo half
// a dozen times used to leak every attempt.
export function useFileUpload({
  accept = '.jpg,.jpeg,.png,.pdf',
  maxSizeMB = 5,
}: UseFileUploadOptions = {}): UseFileUpload {
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  // The live URL is mirrored in a ref so the unmount cleanup can release it
  // without re-subscribing on every preview change.
  const previewRef = useRef<string | null>(null)

  function releasePreview() {
    if (previewRef.current) URL.revokeObjectURL(previewRef.current)
    previewRef.current = null
  }

  function setPreviewURL(url: string | null) {
    releasePreview()
    previewRef.current = url
    setPreview(url)
  }

  useEffect(() => releasePreview, [])

  function handleSelect(f: File | null) {
    setError(null)
    // Dismissing the picker clears the selection rather than keeping the last
    // one alive behind a dialog the user just cancelled.
    if (!f) {
      setFile(null)
      setPreviewURL(null)
      return
    }

    const exts = accept.split(',').map((s) => s.trim().toLowerCase())
    const name = f.name.toLowerCase()
    const okExt = exts.some((e) => (e.startsWith('.') ? name.endsWith(e) : f.type.includes(e)))
    if (!okExt) {
      setError(`Format harus salah satu dari: ${accept}`)
      return
    }
    if (f.size > maxSizeMB * 1024 * 1024) {
      setError(`Ukuran file melebihi ${maxSizeMB}MB`)
      return
    }

    setFile(f)
    setPreviewURL(f.type.startsWith('image/') ? URL.createObjectURL(f) : null)
  }

  function clear() {
    setFile(null)
    setPreviewURL(null)
    setError(null)
  }

  return { file, preview, error, handleSelect, clear }
}
