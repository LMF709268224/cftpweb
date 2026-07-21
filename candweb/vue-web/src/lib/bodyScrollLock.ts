import { onBeforeUnmount, watch, type WatchSource } from "vue"

let activeLockCount = 0
let previousBodyOverflow = ""
let previousBodyPaddingRight = ""
let previousHtmlOverflow = ""

function acquireBodyScrollLock() {
  if (typeof window === "undefined" || typeof document === "undefined") return false

  if (activeLockCount === 0) {
    const body = document.body
    const html = document.documentElement
    const scrollbarWidth = Math.max(0, window.innerWidth - html.clientWidth)
    const bodyPaddingRight = Number.parseFloat(window.getComputedStyle(body).paddingRight) || 0

    previousBodyOverflow = body.style.overflow
    previousBodyPaddingRight = body.style.paddingRight
    previousHtmlOverflow = html.style.overflow

    html.style.overflow = "hidden"
    body.style.overflow = "hidden"
    if (scrollbarWidth > 0) body.style.paddingRight = `${bodyPaddingRight + scrollbarWidth}px`
  }

  activeLockCount += 1
  return true
}

function releaseBodyScrollLock() {
  if (activeLockCount === 0 || typeof document === "undefined") return

  activeLockCount -= 1
  if (activeLockCount > 0) return

  document.documentElement.style.overflow = previousHtmlOverflow
  document.body.style.overflow = previousBodyOverflow
  document.body.style.paddingRight = previousBodyPaddingRight
}

export function useBodyScrollLock(locked: WatchSource<boolean>) {
  let ownsLock = false

  function releaseOwnedLock() {
    if (!ownsLock) return
    ownsLock = false
    releaseBodyScrollLock()
  }

  watch(locked, (shouldLock) => {
    if (shouldLock && !ownsLock) {
      ownsLock = acquireBodyScrollLock()
      return
    }
    if (!shouldLock) releaseOwnedLock()
  }, { immediate: true })

  onBeforeUnmount(releaseOwnedLock)
}
