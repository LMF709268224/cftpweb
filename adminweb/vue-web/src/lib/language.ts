import { computed, ref } from "vue"

export type AdminLang = "zh" | "en"

function initialLang(): AdminLang {
  if (typeof window === "undefined") return "zh"
  const stored = window.localStorage.getItem("app_lang")
  if (stored === "zh" || stored === "en") return stored
  return window.navigator.language.toLowerCase().startsWith("zh") ? "zh" : "en"
}

const lang = ref<AdminLang>(initialLang())

export function setAdminLanguage(nextLang: AdminLang) {
  lang.value = nextLang
  if (typeof window !== "undefined") {
    window.localStorage.setItem("app_lang", nextLang)
    window.dispatchEvent(new Event("admin-language-change"))
  }
}

export function useAdminLanguage() {
  const isZh = computed(() => lang.value === "zh")
  return {
    lang,
    isZh,
    setAdminLanguage,
  }
}
