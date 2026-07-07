import { ApiError } from "./apiClient"

type JsonRecord = Record<string, unknown>

const fieldLabels: Record<string, string> = {
  application_id: "申请 ID",
  asset_file_hash: "资源文件 Hash",
  asset_object_key: "资源 Object Key",
  bundle_order_ulid: "订单编号",
  candidate_ulid: "候选人 ID",
  category: "分类",
  content_tpl: "内容模板",
  course_id: "课程 ID",
  cred_def_ulid: "资格定义 ID",
  description: "描述",
  file_hash: "文件 Hash",
  file_object_key: "文件 Object Key",
  file_type: "文件类型",
  html_body: "邮件内容",
  html_template: "HTML 模板",
  media_file_hash: "媒体文件 Hash",
  media_object_key: "媒体 Object Key",
  name: "名称",
  new_password: "新密码",
  object_key: "Object Key",
  old_password: "原密码",
  pack_id: "资源包 ID",
  path: "路径",
  payload: "Payload JSON",
  reason: "操作原因",
  reason_message: "操作原因",
  respath: "资源路径",
  subject_template: "标题模板",
  thumbnail: "封面",
  thumbnail_file_hash: "封面 File Hash",
  thumbnail_object_key: "封面 Object Key",
  title: "标题",
  title_tpl: "标题模板",
  user_ids: "用户",
  version: "版本号",
  webhook_msg_id: "Webhook 消息 ID",
}

function isRecord(value: unknown): value is JsonRecord {
  return !!value && typeof value === "object" && !Array.isArray(value)
}

function stringifyMessage(value: unknown): string {
  if (typeof value === "string") return value.trim()
  if (Array.isArray(value)) return value.map(stringifyMessage).filter(Boolean).join("；")
  if (isRecord(value)) {
    return stringifyMessage(value.message)
      || stringifyMessage(value.error)
      || stringifyMessage(value.detail)
      || stringifyMessage(value.reason)
      || stringifyMessage(value.error_message)
      || Object.entries(value)
        .map(([key, entry]): string => {
          const message: string = stringifyMessage(entry)
          return message ? `${fieldLabel(key)}：${message}` : ""
        })
        .filter(Boolean)
        .join("；")
  }
  return ""
}

function extractPayloadMessage(payload: unknown): string {
  if (!isRecord(payload)) return ""
  return stringifyMessage(payload.message)
    || stringifyMessage(payload.error)
    || stringifyMessage(payload.detail)
    || stringifyMessage(payload.reason)
    || stringifyMessage(payload.error_message)
    || stringifyMessage(payload.errors)
}

function rawErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    return extractPayloadMessage(err.payload) || err.message
  }
  if (err instanceof Error) return err.message
  return stringifyMessage(err)
}

function fieldLabel(field: string): string {
  const normalized = field
    .trim()
    .replace(/^[`'"]+|[`'".,，。:：]+$/g, "")
    .replace(/[A-Z]/g, (char) => `_${char.toLowerCase()}`)
    .replace(/^_+/, "")
  return fieldLabels[normalized] || normalized.replace(/_/g, " ")
}

function extractField(message: string): string {
  const patterns = [
    /(?:field|parameter|param)\s+[`'"]?([a-zA-Z0-9_.-]+)[`'"]?\s+(?:is\s+)?(?:required|missing)/i,
    /[`'"]?([a-zA-Z0-9_.-]+)[`'"]?\s+(?:is\s+)?(?:required|missing)/i,
    /missing\s+(?:required\s+)?(?:field|parameter|param)?\s*[`'"]?([a-zA-Z0-9_.-]+)[`'"]?/i,
    /缺少(?:必填)?(?:字段|参数)?[:：\s]*([a-zA-Z0-9_.\-\u4e00-\u9fa5]+)/,
  ]
  for (const pattern of patterns) {
    const match = message.match(pattern)
    if (match?.[1]) return fieldLabel(match[1])
  }
  return ""
}

function friendlyDetail(message: string, status?: number): string {
  const text = message.trim()
  if (!text) return ""
  const lower = text.toLowerCase()

  if (lower.includes("thumbnail") && (lower.includes("required") || lower.includes("missing"))) {
    return "缺少封面，请先配置封面 Object Key 或封面 File Hash 后再操作"
  }
  if (lower.includes("version") && (lower.includes("conflict") || lower.includes("mismatch") || lower.includes("stale"))) {
    return "数据版本已变化，请刷新后重试"
  }
  if (lower.includes("already exists") || lower.includes("duplicate")) {
    return "数据已存在，请检查是否重复创建"
  }
  if (status === 409 || lower.includes("conflict")) {
    return "当前数据状态已变化，请刷新后重试"
  }
  if (status === 404 || lower.includes("not found")) {
    return "相关数据不存在或已被删除，请刷新后重试"
  }
  if (status === 401 || lower.includes("unauthorized")) {
    return "登录状态已失效，请重新登录"
  }
  if (status === 403 || lower.includes("forbidden") || lower.includes("permission")) {
    return "当前账号没有权限执行此操作"
  }
  if (lower.includes("old password") || lower.includes("current password")) {
    return "原密码不正确，请重新输入"
  }
  if (lower.includes("precondition")) {
    return "当前数据还不满足操作条件，请先补齐必填信息后再操作"
  }
  if (lower.includes("invalid json") || lower.includes("json")) {
    return "JSON 格式不正确，请检查后重试"
  }
  if (lower.includes("required") || lower.includes("missing")) {
    const field = extractField(text)
    return field ? `缺少必填参数：${field}` : "缺少必填参数，请补齐后再操作"
  }
  if (lower.includes("invalid") || status === 400) {
    const field = extractField(text)
    return field ? `${field} 填写不正确，请检查后重试` : "提交内容不符合要求，请检查必填项和格式后再重试"
  }

  return text
}

export function apiErrorMessage(err: unknown, fallback: string): string {
  const status = err instanceof ApiError ? err.status : undefined
  const detail = friendlyDetail(rawErrorMessage(err), status)
  if (!detail) return fallback
  if (detail === fallback || fallback.includes(detail)) return fallback
  if (detail.includes(fallback)) return detail
  return `${fallback}：${detail}`
}
