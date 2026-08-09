// saveBlob hands a blob to the browser as a download.
//
// Two details decide whether the download actually happens, and three call
// sites each got a different subset of them right:
//
//   - The anchor has to be in the document before it is clicked. Firefox
//     ignores a click on a detached element, so the report export and the
//     invoice download did nothing there at all.
//   - The object URL has to outlive the click. Revoking it on the next line
//     races the browser's own fetch of it; deferring to the next task lets the
//     download start first.
export function saveBlob(blob: Blob, filename: string) {
  const href = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = href
  a.download = filename
  a.style.display = 'none'
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(href), 0)
}

// saveJSON writes structured data out as a .json file.
export function saveJSON(data: unknown, filename: string) {
  saveBlob(new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' }), filename)
}
