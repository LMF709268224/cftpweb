import { useState, useEffect } from "react"
import { zh } from "./locales/zh"
import { en } from "./locales/en"

export type Lang = "zh" | "en"

export function useLanguage() {
  const [lang, setLang] = useState<Lang>("zh")

  useEffect(() => {
    // 首次加载，从 localStorage 读取
    const storedLang = localStorage.getItem("app_lang") as Lang
    if (storedLang === "en" || storedLang === "zh") {
      setLang(storedLang)
    }

    // 监听其他组件切换语言
    const handleStorage = () => {
      const updatedLang = localStorage.getItem("app_lang") as Lang
      if (updatedLang === "en" || updatedLang === "zh") {
        setLang(updatedLang)
      }
    }
    
    window.addEventListener("storage", handleStorage)
    // 自定义事件，用于当前窗口内的状态同步
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
