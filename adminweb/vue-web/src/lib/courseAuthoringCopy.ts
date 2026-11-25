import { computed } from "vue"
import { useAdminLanguage } from "./language"

const en = {
  outline: "Course outline", outlineUnavailable: "The complete course outline is unavailable. Reload to try again.",
  reload: "Reload outline", empty: "No chapters", noLessons: "No lessons", unavailable: "Unavailable",
  minor: "Published course: title and body corrections are saved to the current version. Existing students see them when they reopen the content. Structure, media and quiz changes require a new draft version.",
  missing: "Required course data is unavailable. Reload before editing; no default values will be submitted.",
  reorder: "Drag to reorder, or use the arrow buttons. Changes are saved immediately.", up: "Move up", down: "Move down",
  reordered: "Order saved", reorderFailed: "Could not save order. Reload the outline before trying again.",
  discard: "Discard unsaved lesson changes?", edit: "Edit content", preview: "Body preview", source: "HTML source", visual: "Visual editor",
  bold: "Bold", italic: "Italic", underline: "Underline", heading: "Heading", paragraph: "Paragraph", bullet: "Bullet list", numbered: "Numbered list",
  undo: "Undo", redo: "Redo", link: "Insert link", unlink: "Remove link", image: "Insert image", url: "HTTPS URL", alt: "Image description",
  insert: "Insert", cancel: "Cancel", invalidUrl: "Enter a valid HTTPS URL.",
  previewEmpty: "The lesson body preview appears here.",
  markupNotice: "Visual editing uses standard HTML. Use HTML source to retain custom markup. The preview applies the candidate portal's content safety rules.",
  invalidMarkup: "This content cannot be edited visually without changing its structure. Use HTML source to edit it.",
  sourceLabel: "Lesson HTML source", editorLabel: "Lesson body editor", newVersion: "Create draft version",
  versionHint: "Creates a draft in the same course series. Publishing it replaces the current published version; it does not migrate existing enrolments or progress.",
  pdfPreview: "Saved PDF preview", pdfOpen: "Preview PDF", pdfLoading: "Loading PDF…", pdfRefresh: "Reload PDF",
  pdfSaved: "Previews the saved lesson file. Save a replacement file before previewing it here.",
  pdfFailed: "Could not load the PDF preview. Try again.", pdfExpired: "This preview has expired. Reload the PDF to continue.", pdfNewTab: "Open PDF in a new tab",
}
const zh: typeof en = {
  outline: "课程目录", outlineUnavailable: "暂时无法获取完整课程目录，请重新加载。",
  reload: "重新加载目录", empty: "暂无章节", noLessons: "暂无课时", unavailable: "不可获取",
  minor: "已发布课程：标题和正文修订直接保存到当前版本，已有学员重新打开内容时即可看到。结构、媒体和测验变更请创建新的草稿版本。",
  missing: "缺少必要的课程数据，请重新加载后再编辑，不会提交默认值。",
  reorder: "拖动或使用上下箭头调整顺序，调整后立即保存。", up: "上移", down: "下移",
  reordered: "排序已保存", reorderFailed: "排序保存失败，请重新加载目录后再试。",
  discard: "放弃尚未保存的课时修改？", edit: "编辑内容", preview: "正文预览", source: "HTML 源码", visual: "可视化编辑",
  bold: "加粗", italic: "斜体", underline: "下划线", heading: "标题", paragraph: "段落", bullet: "无序列表", numbered: "有序列表",
  undo: "撤销", redo: "重做", link: "插入链接", unlink: "移除链接", image: "插入图片", url: "HTTPS 地址", alt: "图片说明",
  insert: "插入", cancel: "取消", invalidUrl: "请输入有效的 HTTPS 地址。",
  previewEmpty: "课时正文预览将显示在这里。",
  markupNotice: "可视化编辑使用标准 HTML；自定义排版可通过 HTML 源码编辑保留。预览使用考生端的内容安全规则。",
  invalidMarkup: "此内容无法在保持原结构的情况下进行可视化编辑，请使用 HTML 源码编辑。",
  sourceLabel: "课时 HTML 源码", editorLabel: "课时正文编辑器", newVersion: "创建草稿版本",
  versionHint: "在同一课程系列下创建草稿。发布后替代当前已发布版本，不会迁移已有学员的报名或学习进度。",
  pdfPreview: "已保存 PDF 预览", pdfOpen: "预览 PDF", pdfLoading: "正在加载 PDF…", pdfRefresh: "重新加载 PDF",
  pdfSaved: "预览当前课时已保存的文件。替换文件后，请先保存再预览。",
  pdfFailed: "PDF 预览加载失败，请重试。", pdfExpired: "预览已过期，请重新加载 PDF。", pdfNewTab: "在新窗口打开 PDF",
}

export function useCourseAuthoringCopy() {
  const { lang } = useAdminLanguage()
  return computed(() => lang.value === "zh" ? zh : en)
}
