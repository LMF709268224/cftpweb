export const ErrorMessages: Record<string, { zh: string; en: string }> = {
  // 通用错误
  INTERNAL_ERROR: {
    zh: "服务器开小差了，请稍后再试",
    en: "Internal server error. Please try again later.",
  },
  INVALID_REQUEST: {
    zh: "请求参数错误",
    en: "Invalid request parameters.",
  },
  UNAUTHORIZED: {
    zh: "登录已过期，请重新登录",
    en: "Session expired, please login again.",
  },
  FORBIDDEN: {
    zh: "您没有权限执行此操作",
    en: "You do not have permission to perform this action.",
  },

  // 用户与设置相关
  PASSWORD_INCORRECT: {
    zh: "原密码不正确或不符合要求",
    en: "Incorrect old password or password does not meet requirements.",
  },
  PROFILE_UPDATE_FAILED: {
    zh: "个人资料更新失败，请检查输入",
    en: "Failed to update profile. Please check your input.",
  },

  // 兜底未知错误
  UNKNOWN_ERROR: {
    zh: "发生未知错误，请联系客服",
    en: "An unknown error occurred. Please contact support.",
  },
}

export function getErrorMessage(errorCode: string | undefined | null, lang: "zh" | "en" = "zh"): string {
  if (!errorCode) return ErrorMessages["UNKNOWN_ERROR"][lang]
  const tip = ErrorMessages[errorCode]
  if (tip) {
    return tip[lang]
  }
  return ErrorMessages["UNKNOWN_ERROR"][lang]
}
