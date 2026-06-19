import { computed, ref } from "vue"
import { en } from "./locales/en"
import { zh } from "./locales/zh"

export type Lang = "zh" | "en"

const initial = typeof window !== "undefined" ? localStorage.getItem("app_lang") : null
const lang = ref<Lang>(initial === "en" ? "en" : "zh")

if (typeof window !== "undefined") {
  window.addEventListener("storage", () => {
    const stored = localStorage.getItem("app_lang")
    if (stored === "en" || stored === "zh") lang.value = stored
  })
  window.addEventListener("lang_change", () => {
    const stored = localStorage.getItem("app_lang")
    if (stored === "en" || stored === "zh") lang.value = stored
  })
}

export function changeLanguage(newLang: Lang) {
  localStorage.setItem("app_lang", newLang)
  lang.value = newLang
  window.dispatchEvent(new Event("lang_change"))
}

export function useTranslation() {
  const t = computed(() => (lang.value === "zh" ? zh : en))
  return { t, lang, changeLanguage }
}
