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
  NOT_FOUND: {
    zh: "请求的资源不存在",
    en: "The requested resource was not found.",
  },
  NOT_IMPLEMENTED: {
    zh: "该功能暂未开放",
    en: "This feature is not available yet.",
  },
  PRECONDITION_FAILED: {
    zh: "当前操作条件不满足，请先补齐必要信息。",
    en: "The operation requirements are not met. Please complete the required information first.",
  },
  AUTH_FAILED: {
    zh: "认证失败，请重新登录",
    en: "Authentication failed. Please log in again.",
  },
  TOKEN_EXPIRED: {
    zh: "登录已过期，请重新登录",
    en: "Session expired, please log in again.",
  },
  INVALID_TOKEN: {
    zh: "登录状态无效，请重新登录",
    en: "Invalid session. Please log in again.",
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
  PIPELINE_NOT_FOUND: {
    zh: "未找到对应管线",
    en: "Pipeline not found.",
  },
  ALREADY_PURCHASED: {
    zh: "您已购买该项目",
    en: "You have already purchased this item.",
  },
  INVALID_PIPELINE: {
    zh: "管线信息无效",
    en: "Invalid pipeline.",
  },
  EXAM_NOT_FOUND: {
    zh: "未找到对应考试",
    en: "Exam not found.",
  },
  NOT_ELIGIBLE: {
    zh: "当前不满足操作条件",
    en: "You are not eligible for this action.",
  },
  SIGNUP_FAILED: {
    zh: "报名失败，请稍后再试",
    en: "Signup failed. Please try again later.",
  },
  RETAKE_DENIED: {
    zh: "暂不满足补考条件",
    en: "Retake request was denied.",
  },
  PAYMENT_FAILED: {
    zh: "支付失败，请稍后再试",
    en: "Payment failed. Please try again later.",
  },
  ORDER_NOT_FOUND: {
    zh: "未找到对应订单",
    en: "Order not found.",
  },
  INVALID_AMOUNT: {
    zh: "金额信息无效",
    en: "Invalid amount.",
  },
  MEMBERSHIP_EXPIRED: {
    zh: "会员资格已过期",
    en: "Membership has expired.",
  },
  RECORD_REJECTED: {
    zh: "档案已被驳回",
    en: "Record was rejected.",
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
