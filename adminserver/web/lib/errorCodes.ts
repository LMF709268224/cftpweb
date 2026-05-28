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
  SERVICE_UNAVAILABLE: {
    zh: "依赖服务暂时不可用，请稍后再试。",
    en: "A dependent service is temporarily unavailable. Please try again later.",
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

const FieldLabels: Record<string, { zh: string; en: string }> = {
  body: { zh: "文本内容", en: "body" },
  category_id: { zh: "分类 ID", en: "category ID" },
  category_tips: { zh: "分类提示", en: "category tips" },
  candidate_id: { zh: "考生 ID", en: "candidate ID" },
  chapter_id: { zh: "章节 ID", en: "chapter ID" },
  code: { zh: "授权码", en: "authorization code" },
  course_id: { zh: "课程 ID", en: "course ID" },
  course_guid: { zh: "课程业务 ID", en: "course GUID" },
  cred_def_id: { zh: "资格定义 ID", en: "credential definition ID" },
  entity_id: { zh: "关联实体 ID", en: "entity ID" },
  entity_type: { zh: "关联实体类型", en: "entity type" },
  external_url: { zh: "外部链接", en: "external URL" },
  file_name: { zh: "文件名", en: "file name" },
  file_object_key: { zh: "文件 Object Key", en: "file object key" },
  items: { zh: "排序项", en: "items" },
  lesson_id: { zh: "课时 ID", en: "lesson ID" },
  lesson_type: { zh: "课时类型", en: "lesson type" },
  material_id: { zh: "资料 ID", en: "material ID" },
  material_type: { zh: "资料类型", en: "material type" },
  media_object_key: { zh: "媒体 Object Key", en: "media object key" },
  name: { zh: "名称", en: "name" },
  object_key: { zh: "Object Key", en: "object key" },
  option_id: { zh: "选项 ID", en: "option ID" },
  option_text: { zh: "选项内容", en: "option text" },
  pipeline_id: { zh: "管线 ID", en: "pipeline ID" },
  pipeline_guid: { zh: "管线业务 ID", en: "pipeline GUID" },
  question_id: { zh: "题目 ID", en: "question ID" },
  question_text: { zh: "题目内容", en: "question text" },
  question_type: { zh: "题目类型", en: "question type" },
  quiz_id: { zh: "测验 ID", en: "quiz ID" },
  quizzable_id: { zh: "测验对象 ID", en: "quizzable ID" },
  quizzable_type: { zh: "测验对象类型", en: "quizzable type" },
  required_entity_id: { zh: "前置实体 ID", en: "required entity ID" },
  required_entity_type: { zh: "前置实体类型", en: "required entity type" },
  required_result: { zh: "前置结果", en: "required result" },
  stages: { zh: "阶段", en: "stages" },
  state: { zh: "登录状态", en: "login state" },
  target_entity_id: { zh: "目标实体 ID", en: "target entity ID" },
  target_entity_type: { zh: "目标实体类型", en: "target entity type" },
  title: { zh: "标题", en: "title" },
  upload_type: { zh: "上传类型", en: "upload type" },
  version: { zh: "版本号", en: "version" },
  from_course_guid: { zh: "来源课程业务 ID", en: "source course GUID" },
  from_pipeline_guid: { zh: "来源管线业务 ID", en: "source pipeline GUID" },
  glms_course_id: { zh: "关联课程", en: "linked GLMS course" },
  stripe_product_id: { zh: "Stripe 产品 ID", en: "Stripe product ID" },
  stripe_price_id: { zh: "Stripe 价格 ID", en: "Stripe price ID" },
  unlock_stripe_product_id: { zh: "解锁 Stripe 产品 ID", en: "unlock Stripe product ID" },
  unlock_stripe_price_id: { zh: "解锁 Stripe 价格 ID", en: "unlock Stripe price ID" },
  package_stripe_product_id: { zh: "套餐 Stripe 产品 ID", en: "package Stripe product ID" },
  package_stripe_price_id: { zh: "套餐 Stripe 价格 ID", en: "package Stripe price ID" },
}

function getFieldLabel(field: string, lang: "zh" | "en"): string {
  const normalized = field.replace(/\[\d+\]/g, "")
  return FieldLabels[field]?.[lang] || FieldLabels[normalized]?.[lang] || field
}

export function localizeApiErrorMessage(
  errorCode: string | undefined | null,
  message: string | undefined | null,
  lang: "zh" | "en" = "zh"
): string {
  if (!message) return getErrorMessage(errorCode, lang)
  if (lang === "en") return message

  let match = message.match(/^(.+) is required$/)
  if (match) {
    return `请填写${getFieldLabel(match[1], lang)}`
  }

  match = message.match(/^(.+) are required$/)
  if (match) {
    return `请填写${match[1].split(/\s+and\s+/).map((field) => getFieldLabel(field, lang)).join("和")}`
  }

  match = message.match(/^(.+) must be greater than 0$/)
  if (match) {
    return `${getFieldLabel(match[1], lang)}必须大于 0`
  }

  match = message.match(/^(.+) is invalid$/)
  if (match) {
    return `${getFieldLabel(match[1], lang)}无效`
  }

  match = message.match(/^course "([^"]+)" must contain at least one chapter before publishing$/)
  if (match) {
    return `课程 ${match[1]} 发布前至少需要创建 1 个章节`
  }

  match = message.match(/^chapter "([^"]+)" must contain at least one lesson or quiz before publishing$/)
  if (match) {
    return `章节 ${match[1]} 发布前至少需要包含 1 个课时或测验`
  }

  match = message.match(/^published course "([^"]+)" cannot be modified$/)
  if (match) {
    return `已发布课程 ${match[1]} 不能直接修改，请先下架`
  }

  if (errorCode && ErrorMessages[errorCode]) {
    return getErrorMessage(errorCode, lang)
  }

  return message
}
