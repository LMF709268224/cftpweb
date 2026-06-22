import { ref } from "vue"

const SIDEBAR_COLLAPSED_STORAGE_KEY = "gfi_sidebar_collapsed"

const isSidebarCollapsed = ref(false)
let initialized = false
let handleSidebarStorage: ((event: StorageEvent) => void) | null = null

function applySidebarClass() {
  if (typeof document === "undefined") return
  document.body.classList.toggle("sidebar-collapsed", isSidebarCollapsed.value)
}

export function initializeSidebarCollapse() {
  if (initialized) {
    applySidebarClass()
    return
  }
  initialized = true

  if (typeof window !== "undefined") {
    isSidebarCollapsed.value = window.localStorage.getItem(SIDEBAR_COLLAPSED_STORAGE_KEY) === "1"
    handleSidebarStorage = (event: StorageEvent) => {
      if (event.key !== SIDEBAR_COLLAPSED_STORAGE_KEY) return
      isSidebarCollapsed.value = event.newValue === "1"
      applySidebarClass()
    }
    window.addEventListener("storage", handleSidebarStorage)
  }

  applySidebarClass()
}

export function disposeSidebarCollapse() {
  if (typeof window !== "undefined" && handleSidebarStorage) {
    window.removeEventListener("storage", handleSidebarStorage)
  }
  handleSidebarStorage = null
  initialized = false
}

export function useSidebarCollapse() {
  function setSidebarCollapsed(value: boolean) {
    isSidebarCollapsed.value = value
    if (typeof window !== "undefined") {
      window.localStorage.setItem(SIDEBAR_COLLAPSED_STORAGE_KEY, value ? "1" : "0")
    }
    applySidebarClass()
  }

  function toggleSidebarCollapsed() {
    setSidebarCollapsed(!isSidebarCollapsed.value)
  }

  return {
    isSidebarCollapsed,
    setSidebarCollapsed,
    toggleSidebarCollapsed,
  }
}
