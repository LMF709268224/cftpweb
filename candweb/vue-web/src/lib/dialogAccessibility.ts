import { nextTick, onBeforeUnmount, watch, type Ref } from "vue"

const focusableSelector = [
  "a[href]",
  "button:not([disabled])",
  "input:not([disabled]):not([type='hidden'])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  "[contenteditable='true']",
  "[tabindex]:not([tabindex='-1'])",
].join(",")

function focusableElements(dialog: HTMLElement) {
  return Array.from(dialog.querySelectorAll<HTMLElement>(focusableSelector)).filter((element) => (
    element.getAttribute("aria-hidden") !== "true"
    && (element.offsetWidth > 0 || element.offsetHeight > 0 || element.getClientRects().length > 0)
  ))
}

export function useDialogAccessibility(
  isOpen: () => boolean,
  dialogRef: Ref<HTMLElement | null>,
  close: () => void,
) {
  let restoreFocusTo: HTMLElement | null = null

  function handleKeydown(event: KeyboardEvent) {
    if (!isOpen() || event.defaultPrevented) return

    if (event.key === "Escape") {
      event.preventDefault()
      close()
      return
    }

    if (event.key !== "Tab") return
    const dialog = dialogRef.value
    if (!dialog) return

    const focusable = focusableElements(dialog)
    if (!focusable.length) {
      event.preventDefault()
      dialog.focus()
      return
    }

    const first = focusable[0]
    const last = focusable[focusable.length - 1]
    const activeElement = document.activeElement
    if (!dialog.contains(activeElement)) {
      event.preventDefault()
      const focusTarget = event.shiftKey ? last : first
      focusTarget.focus()
    } else if (event.shiftKey && activeElement === first) {
      event.preventDefault()
      last.focus()
    } else if (!event.shiftKey && activeElement === last) {
      event.preventDefault()
      first.focus()
    }
  }

  watch(isOpen, (open) => {
    if (open) {
      restoreFocusTo = document.activeElement instanceof HTMLElement ? document.activeElement : null
      document.addEventListener("keydown", handleKeydown)
      void nextTick(() => {
        if (!isOpen()) return
        const dialog = dialogRef.value
        if (!dialog) return
        const firstFocusable = focusableElements(dialog)[0]
        const focusTarget = firstFocusable || dialog
        focusTarget.focus()
      })
      return
    }

    document.removeEventListener("keydown", handleKeydown)
    const focusTarget = restoreFocusTo
    restoreFocusTo = null
    void nextTick(() => {
      if (focusTarget?.isConnected) focusTarget.focus()
    })
  }, { immediate: true })

  onBeforeUnmount(() => {
    document.removeEventListener("keydown", handleKeydown)
    if (restoreFocusTo?.isConnected) restoreFocusTo.focus()
  })
}
