import { computed, ref } from "vue"
import { apiClient } from "./apiClient"
import { isAuthenticated } from "./authStorage"
import { en } from "./locales/en"
import { zh } from "./locales/zh"

export type Lang = "zh" | "en"
let languageSave = Promise.resolve()
let languageChangeVersion = 0

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
  if (newLang !== "zh" && newLang !== "en") return
  const previous = lang.value
  const version = ++languageChangeVersion
  localStorage.setItem("app_lang", newLang)
  lang.value = newLang
  window.dispatchEvent(new Event("lang_change"))
  if (isAuthenticated()) {
    languageSave = languageSave.then(async () => {
      await apiClient("/api/user/language", {
        method: "PUT",
        body: JSON.stringify({ language: newLang }),
      })
    }).catch((error: unknown) => {
      console.error("Failed to save language preference", error)
      if (languageChangeVersion === version) applyAccountLanguage(previous)
    })
  }
}

export function applyAccountLanguage(value: unknown) {
  if (value !== "zh" && value !== "en") return
  localStorage.setItem("app_lang", value)
  lang.value = value
  window.dispatchEvent(new Event("lang_change"))
}

export function useTranslation() {
  const t = computed(() => (lang.value === "zh" ? zh : en))
  return { t, lang, changeLanguage }
}
