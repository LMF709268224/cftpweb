export type Lang = "zh" | "en"

export const UIMessages: Record<string, { zh: string; en: string }> = {
  PROFILE_UPDATE_SUCCESS: {
    zh: "个人资料修改成功",
    en: "Profile updated successfully",
  },
  PASSWORD_UPDATE_SUCCESS: {
    zh: "密码修改成功，请重新登录",
    en: "Password updated successfully. Please log in again.",
  },
  PASSWORD_MISMATCH: {
    zh: "两次输入的新密码不一致",
    en: "Passwords do not match",
  },
}

export function getMessage(key: string, lang: Lang = "zh"): string {
  return UIMessages[key]?.[lang] || key
}
