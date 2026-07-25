import { nextTick, type Directive, type DirectiveBinding } from "vue"

export type ModalDialogOptions = {
  close?: () => void
  initialFocus?: string
}

type ModalDialogBinding = (() => void) | ModalDialogOptions | undefined

type DialogEntry = {
  element: HTMLElement
  close?: () => void
  initialFocus: string
  returnFocus: HTMLElement | null
  originalAriaHidden: string | null
  originalInert: boolean
  keydown: (event: KeyboardEvent) => void
}

const focusableSelector = [
  "a[href]",
  "button:not([disabled])",
  "iframe",
  "input:not([disabled])",
  "select:not([disabled])",
  "summary",
  "textarea:not([disabled])",
  '[contenteditable="true"]',
  '[tabindex]:not([tabindex="-1"])',
].join(",")

const dialogEntries = new WeakMap<HTMLElement, DialogEntry>()
const dialogStack: DialogEntry[] = []
let generatedTitleId = 0
let pageScrollSnapshot: {
  rootOverflow: string
  bodyOverflow: string
  bodyPaddingRight: string
} | null = null

function bindingOptions(binding: DirectiveBinding<ModalDialogBinding>): ModalDialogOptions {
  if (typeof binding.value === "function") return { close: binding.value }
  return binding.value || {}
}

function visibleFocusableElements(dialog: HTMLElement) {
  return Array.from(dialog.querySelectorAll<HTMLElement>(focusableSelector))
    .filter((element) => element.tabIndex >= 0 && element.getClientRects().length > 0 && !element.closest("[inert]"))
}

function focusDialog(entry: DialogEntry) {
  const requestedTarget = entry.element.querySelector<HTMLElement>(entry.initialFocus)
  const canFocusRequestedTarget = requestedTarget
    && !requestedTarget.matches(":disabled")
    && requestedTarget.tabIndex >= 0
    && requestedTarget.getClientRects().length > 0
  const focusTarget = canFocusRequestedTarget ? requestedTarget : entry.element
  focusTarget.focus({ preventScroll: true })
}

function trapDialogFocus(event: KeyboardEvent, entry: DialogEntry) {
  if (event.key !== "Tab" || dialogStack.at(-1) !== entry) return

  const focusable = visibleFocusableElements(entry.element)
  if (!focusable.length) {
    event.preventDefault()
    entry.element.focus({ preventScroll: true })
    return
  }

  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  const active = document.activeElement
  if (!(active instanceof HTMLElement) || !focusable.includes(active)) {
    event.preventDefault()
    const fallbackTarget = event.shiftKey ? last : first
    fallbackTarget.focus()
  } else if (event.shiftKey && active === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && active === last) {
    event.preventDefault()
    first.focus()
  }
}

function handleDialogKeydown(event: KeyboardEvent, entry: DialogEntry) {
  if (dialogStack.at(-1) !== entry) return
  if (event.key === "Escape" && entry.close) {
    event.preventDefault()
    event.stopPropagation()
    entry.close()
    return
  }
  trapDialogFocus(event, entry)
}

function lockPageScroll() {
  if (pageScrollSnapshot) return

  const root = document.documentElement
  const body = document.body
  const scrollbarWidth = Math.max(0, window.innerWidth - root.clientWidth)
  const bodyPaddingRight = Number.parseFloat(window.getComputedStyle(body).paddingRight) || 0
  pageScrollSnapshot = {
    rootOverflow: root.style.overflow,
    bodyOverflow: body.style.overflow,
    bodyPaddingRight: body.style.paddingRight,
  }

  root.style.overflow = "hidden"
  body.style.overflow = "hidden"
  if (scrollbarWidth > 0) body.style.paddingRight = `${bodyPaddingRight + scrollbarWidth}px`
}

function unlockPageScroll() {
  if (!pageScrollSnapshot) return

  const root = document.documentElement
  const body = document.body
  root.style.overflow = pageScrollSnapshot.rootOverflow
  body.style.overflow = pageScrollSnapshot.bodyOverflow
  body.style.paddingRight = pageScrollSnapshot.bodyPaddingRight
  pageScrollSnapshot = null
}

function updateDialogLayers() {
  const topIndex = dialogStack.length - 1
  dialogStack.forEach((entry, index) => {
    if (index < topIndex) {
      entry.element.setAttribute("aria-hidden", "true")
      entry.element.inert = true
      return
    }

    if (entry.originalAriaHidden === null) entry.element.removeAttribute("aria-hidden")
    else entry.element.setAttribute("aria-hidden", entry.originalAriaHidden)
    entry.element.inert = entry.originalInert
  })
}

function ensureDialogSemantics(element: HTMLElement) {
  if (!element.hasAttribute("role")) element.setAttribute("role", "dialog")
  if (!element.hasAttribute("aria-modal")) element.setAttribute("aria-modal", "true")
  if (!element.hasAttribute("tabindex")) element.setAttribute("tabindex", "-1")
  element.style.outline = "none"

  if (element.hasAttribute("aria-label") || element.hasAttribute("aria-labelledby")) return
  const title = element.querySelector<HTMLElement>("[data-dialog-title], h1, h2, h3")
  if (!title) return
  if (!title.id) {
    generatedTitleId += 1
    title.id = `admin-modal-title-${generatedTitleId}`
  }
  element.setAttribute("aria-labelledby", title.id)
}

function mountDialog(element: HTMLElement, binding: DirectiveBinding<ModalDialogBinding>) {
  ensureDialogSemantics(element)
  const options = bindingOptions(binding)
  const entry: DialogEntry = {
    element,
    close: options.close,
    initialFocus: options.initialFocus || "[data-dialog-initial-focus]",
    returnFocus: document.activeElement instanceof HTMLElement ? document.activeElement : null,
    originalAriaHidden: element.getAttribute("aria-hidden"),
    originalInert: element.inert,
    keydown: () => undefined,
  }
  entry.keydown = (event) => handleDialogKeydown(event, entry)
  dialogEntries.set(element, entry)
  dialogStack.push(entry)
  element.addEventListener("keydown", entry.keydown)
  if (dialogStack.length === 1) lockPageScroll()
  updateDialogLayers()
  void nextTick(() => {
    if (dialogStack.at(-1) === entry) focusDialog(entry)
  })
}

function updateDialog(element: HTMLElement, binding: DirectiveBinding<ModalDialogBinding>) {
  const entry = dialogEntries.get(element)
  if (!entry) return
  const options = bindingOptions(binding)
  entry.close = options.close
  entry.initialFocus = options.initialFocus || "[data-dialog-initial-focus]"
}

function unmountDialog(element: HTMLElement) {
  const entry = dialogEntries.get(element)
  if (!entry) return

  const index = dialogStack.indexOf(entry)
  const wasTop = index === dialogStack.length - 1
  if (index >= 0) dialogStack.splice(index, 1)
  element.removeEventListener("keydown", entry.keydown)
  dialogEntries.delete(element)
  updateDialogLayers()
  if (!dialogStack.length) unlockPageScroll()

  if (!wasTop) return
  void nextTick(() => {
    if (entry.returnFocus?.isConnected) entry.returnFocus.focus({ preventScroll: true })
    else dialogStack.at(-1)?.element.focus({ preventScroll: true })
  })
}

export const modalDialog: Directive<HTMLElement, ModalDialogBinding> = {
  mounted: mountDialog,
  updated: updateDialog,
  beforeUnmount: unmountDialog,
}
