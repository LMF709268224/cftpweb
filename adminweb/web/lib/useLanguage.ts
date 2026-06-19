import { useEffect, useState } from "react"
import { zh } from "./locales/zh"
import { en } from "./locales/en"

export type Lang = "zh" | "en"

export function useLanguage() {
  const [lang, setLang] = useState<Lang>("zh")

  useEffect(() => {
    // Load the persisted language from localStorage on first render.
    const storedLang = localStorage.getItem("app_lang") as Lang
    if (storedLang === "en" || storedLang === "zh") {
      setLang(storedLang)
    }

    // Keep this tab in sync when another part of the app changes language.
    const handleStorage = () => {
      const updatedLang = localStorage.getItem("app_lang") as Lang
      if (updatedLang === "en" || updatedLang === "zh") {
        setLang(updatedLang)
      }
    }

    window.addEventListener("storage", handleStorage)
    // Custom event for state changes within the current window.
    window.addEventListener("lang_change", handleStorage)

    return () => {
      window.removeEventListener("storage", handleStorage)
      window.removeEventListener("lang_change", handleStorage)
    }
  }, [])

  const changeLanguage = (newLang: Lang) => {
    localStorage.setItem("app_lang", newLang)
    setLang(newLang)
    window.dispatchEvent(new Event("lang_change"))
  }

  return { lang, changeLanguage }
}

export function useTranslation() {
  const { lang, changeLanguage } = useLanguage()
  const t = lang === "zh" ? zh : en
  return { t, lang, changeLanguage }
}
