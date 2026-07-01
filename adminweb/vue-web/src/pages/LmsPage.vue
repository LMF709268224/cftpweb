<script setup lang="ts">
import { FileJson, Loader2, Plus, RefreshCw, Save, Trash2, UploadCloud } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

type CourseForm = {
  category_tips: string
  title: string
  description: string
  thumbnail_object_key: string
  thumbnail_file_hash: string
  duration_min: string
  certification_enabled: boolean
  certification_def_id: string
  respath: string
}

type ChapterForm = {
  title: string
  sort_order: string
}

type LessonForm = {
  chapter_id: string
  title: string
  sort_order: string
  lesson_type: string
  body: string
  asset_object_key: string
  asset_file_hash: string
}

type QuizScope = "course" | "chapter" | "lesson"

type QuizForm = {
  scope: QuizScope
  title: string
  description: string
  passing_score: string
  time_limit: string
  randomize_questions: boolean
}

type QuestionForm = {
  question_text: string
  question_type: string
  points: string
  sort_order: string
  is_required: boolean
  media_items_json: string
}

type OptionForm = {
  option_text: string
  is_correct: boolean
  sort_order: string
}

type MaterialForm = {
  title: string
  material_type: string
  description: string
  file_object_key: string
  file_hash: string
  file_size: string
  sort_order: string
}

type SupplementaryMaterialForm = {
  kind: string
  data_json: string
}

type SupplementaryMaterialItem = {
  key: string
  recordIndex: number
  chapter: string
  type: string
  title: string
  description: string
  url: string
  sourceKind: string
}

type SupplementaryItemForm = {
  chapter: string
  type: string
  title: string
  description: string
  url: string
  duration: string
}

type LessonListItem = {
  lesson: JsonRecord
  chapter: JsonRecord | null
}

type QuizListItem = {
  quiz: JsonRecord
  ownerType: number
  owner: JsonRecord | null
  chapter: JsonRecord | null
  lesson: JsonRecord | null
}

const pageSize = 20

const courses = ref<JsonRecord[]>([])
const selectedCourse = ref<JsonRecord | null>(null)
const courseDetail = ref<JsonRecord | null>(null)
const completeCourse = ref<JsonRecord | null>(null)
const chapters = ref<JsonRecord[]>([])
const selectedChapter = ref<JsonRecord | null>(null)
const lessons = ref<JsonRecord[]>([])
const materials = ref<JsonRecord[]>([])
const selectedMaterial = ref<JsonRecord | null>(null)
const supplementaryMaterial = ref<JsonRecord | null>(null)
const quizzes = ref<JsonRecord[]>([])
const selectedQuiz = ref<JsonRecord | null>(null)
const questions = ref<JsonRecord[]>([])
const selectedQuestion = ref<JsonRecord | null>(null)
const options = ref<JsonRecord[]>([])

const loading = ref(false)
const detailLoading = ref(false)
const completeLoading = ref(false)
const chaptersLoading = ref(false)
const lessonsLoading = ref(false)
const materialsLoading = ref(false)
const supplementaryMaterialLoading = ref(false)
const quizzesLoading = ref(false)
const questionsLoading = ref(false)
const optionsLoading = ref(false)
const savingCourse = ref(false)
const savingChapter = ref(false)
const savingLesson = ref(false)
const savingMaterial = ref(false)
const savingSupplementaryMaterial = ref(false)
const savingQuiz = ref(false)
const savingQuestion = ref(false)
const savingOption = ref(false)
const publishing = ref(false)
const importing = ref(false)
const courseView = ref<"list" | "detail">("list")

const categoryFilter = ref("")
const publishedOnly = ref(false)
const nextPageToken = ref("")
const courseForm = ref<CourseForm>(emptyCourseForm())
const chapterForm = ref<ChapterForm>(emptyChapterForm())
const lessonForm = ref<LessonForm>(emptyLessonForm())
const materialForm = ref<MaterialForm>(emptyMaterialForm())
const supplementaryMaterialForm = ref<SupplementaryMaterialForm>(emptySupplementaryMaterialForm())
const supplementaryItemForm = ref<SupplementaryItemForm>(emptySupplementaryItemForm())
const quizForm = ref<QuizForm>(emptyQuizForm())
const questionForm = ref<QuestionForm>(emptyQuestionForm())
const optionForm = ref<OptionForm>(emptyOptionForm())
const editingChapterId = ref("")
const editingLessonId = ref("")
const editingMaterialId = ref("")
const editingSupplementaryItemIndex = ref(-1)
const editingQuizId = ref("")
const editingQuestionId = ref("")
const editingOptionId = ref("")
const importOpen = ref(false)
const importScope = ref<"course" | "quiz">("course")
const importCategoryTips = ref("")
const importJson = ref("")

const selectedCourseId = computed(() => courseId(selectedCourse.value))
const selectedChapterId = computed(() => chapterId(selectedChapter.value))
const selectedMaterialId = computed(() => materialId(selectedMaterial.value))
const supplementaryMaterialItems = computed<SupplementaryMaterialItem[]>(() => parseSupplementaryMaterialItems(normalizeSupplementaryMaterials({
  ...(supplementaryMaterial.value || {}),
  kind: supplementaryMaterialForm.value.kind,
  data_json: supplementaryMaterialForm.value.data_json,
}), "Chapter"))
const selectedQuizId = computed(() => quizId(selectedQuiz.value))
const selectedQuestionId = computed(() => questionId(selectedQuestion.value))
const selectedCoursePublished = computed(() => Boolean(selectedCourse.value?.is_published))
const selectedCourseStatus = computed(() => selectedCourse.value?.status || (selectedCoursePublished.value ? "Published" : "Draft"))
const selectedLesson = computed(() => lessons.value.find((item) => lessonId(item) === editingLessonId.value) || null)
const selectedMaterialRecord = computed(() => materials.value.find((item) => materialId(item) === selectedMaterialId.value) || selectedMaterial.value)
const completeCourseRecord = computed(() => {
  const value = completeCourse.value?.complete_course
  return value && typeof value === "object" && !Array.isArray(value) ? value as JsonRecord : completeCourse.value
})
const allLessonItems = computed<LessonListItem[]>(() => {
  const chapterDetails = Array.isArray(completeCourseRecord.value?.chapters) ? completeCourseRecord.value.chapters : []
  const items: LessonListItem[] = []
  for (const detail of chapterDetails) {
    if (!detail || typeof detail !== "object" || Array.isArray(detail)) continue
    const record = detail as JsonRecord
    const chapter = record.chapter && typeof record.chapter === "object" && !Array.isArray(record.chapter) ? record.chapter as JsonRecord : record
    const lessonDetails = Array.isArray(record.lessons) ? record.lessons : []
    for (const lessonDetail of lessonDetails) {
      if (!lessonDetail || typeof lessonDetail !== "object" || Array.isArray(lessonDetail)) continue
      const lessonRecord = lessonDetail as JsonRecord
      const lesson = lessonRecord.lesson && typeof lessonRecord.lesson === "object" && !Array.isArray(lessonRecord.lesson) ? lessonRecord.lesson as JsonRecord : lessonRecord
      items.push({ lesson, chapter })
    }
  }
  const knownIds = new Set(items.map((item) => lessonId(item.lesson)).filter(Boolean))
  for (const lesson of lessons.value) {
    if (!knownIds.has(lessonId(lesson))) items.push({ lesson, chapter: selectedChapter.value })
  }
  return items
})
const selectedLessonOwnerChapter = computed(() => allLessonItems.value.find((item) => lessonId(item.lesson) === editingLessonId.value)?.chapter || selectedChapter.value)
const allQuizItems = computed<QuizListItem[]>(() => {
  const items: QuizListItem[] = []
  const complete = completeCourseRecord.value || {}
  const courseQuizzes = Array.isArray(complete.quizzes) ? complete.quizzes : []
  for (const quizDetail of courseQuizzes) {
    const quiz = extractQuizRecord(quizDetail)
    if (quiz) items.push({ quiz, ownerType: 3, owner: selectedCourse.value, chapter: null, lesson: null })
  }
  const chapterDetails = Array.isArray(complete.chapters) ? complete.chapters : []
  for (const detail of chapterDetails) {
    if (!detail || typeof detail !== "object" || Array.isArray(detail)) continue
    const record = detail as JsonRecord
    const chapter = record.chapter && typeof record.chapter === "object" && !Array.isArray(record.chapter) ? record.chapter as JsonRecord : record
    const chapterQuizzes = Array.isArray(record.quizzes) ? record.quizzes : []
    for (const quizDetail of chapterQuizzes) {
      const quiz = extractQuizRecord(quizDetail)
      if (quiz) items.push({ quiz, ownerType: 2, owner: chapter, chapter, lesson: null })
    }
    const lessonDetails = Array.isArray(record.lessons) ? record.lessons : []
    for (const lessonDetail of lessonDetails) {
      if (!lessonDetail || typeof lessonDetail !== "object" || Array.isArray(lessonDetail)) continue
      const lessonRecord = lessonDetail as JsonRecord
      const lesson = lessonRecord.lesson && typeof lessonRecord.lesson === "object" && !Array.isArray(lessonRecord.lesson) ? lessonRecord.lesson as JsonRecord : lessonRecord
      const lessonQuizzes = Array.isArray(lessonRecord.quizzes) ? lessonRecord.quizzes : []
      for (const quizDetail of lessonQuizzes) {
        const quiz = extractQuizRecord(quizDetail)
        if (quiz) items.push({ quiz, ownerType: 1, owner: lesson, chapter, lesson })
      }
    }
  }
  const knownIds = new Set(items.map((item) => quizId(item.quiz)).filter(Boolean))
  for (const quiz of quizzes.value) {
    if (knownIds.has(quizId(quiz))) continue
    const target = quizTarget(quizForm.value.scope)
    const ownerType = Number(quiz.quizzable_type || target.type)
    items.push({
      quiz,
      ownerType,
      owner: ownerType === 3 ? selectedCourse.value : ownerType === 2 ? selectedChapter.value : selectedLesson.value,
      chapter: ownerType === 2 ? selectedChapter.value : selectedLessonOwnerChapter.value,
      lesson: ownerType === 1 ? selectedLesson.value : null,
    })
  }
  return items
})
const selectedQuizItem = computed(() => allQuizItems.value.find((item) => quizId(item.quiz) === selectedQuizId.value) || null)

function emptyCourseForm(): CourseForm {
  return {
    category_tips: "",
    title: "",
    description: "",
    thumbnail_object_key: "",
    thumbnail_file_hash: "",
    duration_min: "0",
    certification_enabled: false,
    certification_def_id: "",
    respath: "",
  }
}

function emptyChapterForm(): ChapterForm {
  return { title: "", sort_order: "1" }
}

function emptyLessonForm(): LessonForm {
  return {
    chapter_id: "",
    title: "",
    sort_order: "1",
    lesson_type: "2",
    body: "",
    asset_object_key: "",
    asset_file_hash: "",
  }
}

function emptyMaterialForm(): MaterialForm {
  return {
    title: "",
    material_type: "1",
    description: "",
    file_object_key: "",
    file_hash: "",
    file_size: "0",
    sort_order: "1",
  }
}

function emptySupplementaryMaterialForm(): SupplementaryMaterialForm {
  return {
    kind: "supplementary_materials",
    data_json: "[]",
  }
}

function emptySupplementaryItemForm(): SupplementaryItemForm {
  return {
    chapter: "",
    type: "Article",
    title: "",
    description: "",
    url: "",
    duration: "",
  }
}

function emptyQuizForm(): QuizForm {
  return {
    scope: "chapter",
    title: "",
    description: "",
    passing_score: "70",
    time_limit: "0",
    randomize_questions: false,
  }
}

function emptyQuestionForm(): QuestionForm {
  return {
    question_text: "",
    question_type: "1",
    points: "10",
    sort_order: "1",
    is_required: true,
    media_items_json: "[]",
  }
}

function emptyOptionForm(): OptionForm {
  return {
    option_text: "",
    is_correct: false,
    sort_order: "1",
  }
}

function courseId(course: JsonRecord | null | undefined) {
  return String(pickFirst(course || {}, ["course_id", "course_ulid"]) || "")
}

function chapterId(chapter: JsonRecord | null | undefined) {
  return String(pickFirst(chapter || {}, ["chapter_id", "chapter_ulid"]) || "")
}

function lessonId(lesson: JsonRecord | null | undefined) {
  return String(pickFirst(lesson || {}, ["lesson_id", "lesson_ulid"]) || "")
}

function materialId(material: JsonRecord | null | undefined) {
  return String(pickFirst(material || {}, ["material_id", "material_ulid"]) || "")
}

function supplementaryMaterialId(material: JsonRecord | null | undefined) {
  return String(pickFirst(material || {}, ["material_id", "material_ulid", "materialId", "materialUlid"]) || "")
}

function quizId(quiz: JsonRecord | null | undefined) {
  return String(pickFirst(quiz || {}, ["quiz_id", "quiz_ulid"]) || "")
}

function questionId(question: JsonRecord | null | undefined) {
  return String(pickFirst(question || {}, ["question_id", "question_ulid"]) || "")
}

function optionId(option: JsonRecord | null | undefined) {
  return String(pickFirst(option || {}, ["option_id", "option_ulid"]) || "")
}

function versionOf(record: JsonRecord | null | undefined) {
  return Number(record?.version || 0)
}

function courseTitle(course: JsonRecord | null | undefined) {
  return String(pickFirst(course || {}, ["title", "name", "course_title"]) || courseId(course) || "课程")
}

function chapterTitle(chapter: JsonRecord | null | undefined) {
  return String(pickFirst(chapter || {}, ["title", "name"]) || chapterId(chapter) || "章节")
}

function chapterById(id: string) {
  return chapters.value.find((item) => chapterId(item) === id) || null
}

function lessonTitle(lesson: JsonRecord | null | undefined) {
  return String(pickFirst(lesson || {}, ["title", "name"]) || lessonId(lesson) || "课时")
}

function materialTitle(material: JsonRecord | null | undefined) {
  return String(pickFirst(material || {}, ["title", "name"]) || materialId(material) || "资料")
}

function quizTitle(quiz: JsonRecord | null | undefined) {
  return String(pickFirst(quiz || {}, ["title", "name"]) || quizId(quiz) || "测验")
}

function questionTitle(question: JsonRecord | null | undefined) {
  return String(pickFirst(question || {}, ["question_text", "title"]) || questionId(question) || "题目")
}

function optionTitle(option: JsonRecord | null | undefined) {
  return String(pickFirst(option || {}, ["option_text", "title"]) || optionId(option) || "选项")
}

function questionTypeLabel(value: unknown) {
  const type = Number(value || 0)
  if (type === 1) return "单选"
  if (type === 2) return "多选"
  if (type === 3) return "判断"
  return "未知"
}

function lessonTypeLabel(value: unknown) {
  const type = Number(value || 0)
  if (type === 1) return "视频"
  if (type === 2) return "文本"
  if (type === 3) return "PDF"
  if (type === 4) return "图片"
  if (type === 5) return "音频"
  if (type === 6) return "文件"
  if (type === 7) return "链接"
  return "未指定"
}

function materialTypeLabel(value: unknown) {
  const type = Number(value || 0)
  if (type === 1) return "教材/课本"
  if (type === 2) return "幻灯片/课件"
  if (type === 3) return "参考资料"
  if (type === 4) return "其他"
  return "未指定"
}

function supplementaryTypeLabel(type: string) {
  const normalized = type.trim().toLowerCase()
  if (normalized === "article") return "Article"
  if (normalized === "video") return "Video"
  if (normalized === "pdf") return "PDF"
  if (normalized === "link") return "Link"
  return type || "Material"
}

function supplementaryTypeClass(type: string) {
  const normalized = type.trim().toLowerCase()
  if (normalized === "video") return "border-violet-200 bg-violet-50 text-violet-700"
  if (normalized === "article") return "border-blue-200 bg-blue-50 text-blue-700"
  if (normalized === "pdf") return "border-red-200 bg-red-50 text-red-700"
  return "border-slate-200 bg-slate-50 text-slate-700"
}

function quizzableTypeLabel(value: unknown) {
  const type = Number(value || 0)
  if (type === 1) return "课时测验"
  if (type === 2) return "章节测验"
  if (type === 3) return "课程测验"
  return "未知归属"
}

function displayValue(value: unknown) {
  if (value === null || value === undefined || value === "") return "-"
  if (typeof value === "boolean") return value ? "是" : "否"
  if (typeof value === "object") return JSON.stringify(value, null, 2)
  return String(value)
}

function recordEntries(record: JsonRecord | null | undefined) {
  if (!record) return []
  return Object.entries(record).map(([key, value]) => ({ key, value: displayValue(value) }))
}

function normalizeSupplementaryMaterials(raw: unknown): JsonRecord[] {
  if (!raw) return []
  if (Array.isArray(raw)) return raw.filter(isJsonRecord)
  if (isJsonRecord(raw)) return [raw]
  return []
}

function parseSupplementaryMaterialItems(materialsToParse: JsonRecord[], fallbackChapter = "Chapter"): SupplementaryMaterialItem[] {
  return materialsToParse.flatMap((material, materialIndex) => {
    const data = parseSupplementaryJson(material.data_json ?? material.dataJson)
    const records = supplementaryRecordsFromData(data)

    return records.map((record, recordIndex) => {
      const title = stringFromRecord(record, ["title", "name", "label", "heading"])
      const description = stringFromRecord(record, ["description", "desc", "summary", "detail", "content"])
      const type = stringFromRecord(record, ["type", "material_type", "resource_type", "kind"]) || "Material"
      const chapter = stringFromRecord(record, ["chapter", "chapter_title", "chapterTitle", "section"]) || fallbackChapter
      const url = stringFromRecord(record, ["resource_link", "resourceLink", "url", "link", "href", "external_url", "externalUrl"])
      const sourceId = supplementaryMaterialId(material) || "supplementary"
      const fallbackKey = `${sourceId}-${materialIndex}-${recordIndex}`

      return {
        key: stringFromRecord(record, ["id", "material_id", "material_ulid", "materialId", "materialUlid", "resource_id", "resource_ulid", "resourceId", "resourceUlid", "key"]) || fallbackKey,
        recordIndex,
        chapter,
        type,
        title: title || "Untitled material",
        description,
        url,
        sourceKind: String(material.kind || ""),
      }
    })
  })
}

function parseSupplementaryJson(dataJson: unknown): unknown {
  if (!dataJson) return null
  if (typeof dataJson !== "string") return dataJson

  const trimmed = dataJson.trim()
  if (!trimmed) return null

  try {
    const parsed = JSON.parse(trimmed)
    if (typeof parsed === "string" && parsed.trim()) return parseSupplementaryJson(parsed)
    return parsed
  } catch {
    return null
  }
}

function supplementaryRecordsFromData(data: unknown): JsonRecord[] {
  if (Array.isArray(data)) return data.filter(isJsonRecord)
  if (!isJsonRecord(data)) return []

  for (const key of ["items", "resources", "materials", "data", "data_json", "dataJson", "list"]) {
    const value = data[key]
    if (Array.isArray(value)) return value.filter(isJsonRecord)
    const parsed = parseSupplementaryJson(value)
    if (Array.isArray(parsed)) return parsed.filter(isJsonRecord)
  }

  return [data]
}

function isJsonRecord(value: unknown): value is JsonRecord {
  return Boolean(value && typeof value === "object" && !Array.isArray(value))
}

function stringFromRecord(record: JsonRecord, keys: string[]) {
  for (const key of keys) {
    const value = record[key]
    if (typeof value === "string" && value.trim()) return value.trim()
    if (typeof value === "number" && Number.isFinite(value)) return String(value)
  }
  return ""
}

function formatSupplementaryDataJson(value: unknown) {
  if (typeof value === "string") {
    const trimmed = value.trim()
    if (!trimmed) return "[]"
    try {
      return JSON.stringify(JSON.parse(trimmed), null, 2)
    } catch {
      return trimmed
    }
  }
  if (value === null || value === undefined) return "[]"
  return JSON.stringify(value, null, 2)
}

function supplementaryEditableRecords() {
  const data = parseSupplementaryJson(supplementaryMaterialForm.value.data_json)
  return supplementaryRecordsFromData(data).map((item) => ({ ...item }))
}

function updateSupplementaryDataJson(records: JsonRecord[]) {
  const data = parseSupplementaryJson(supplementaryMaterialForm.value.data_json)
  if (Array.isArray(data)) {
    supplementaryMaterialForm.value.data_json = JSON.stringify(records, null, 2)
    return
  }
  if (isJsonRecord(data)) {
    for (const key of ["items", "resources", "materials", "data", "data_json", "dataJson", "list"]) {
      if (Array.isArray(data[key])) {
        supplementaryMaterialForm.value.data_json = JSON.stringify({ ...data, [key]: records }, null, 2)
        return
      }
    }
  }
  supplementaryMaterialForm.value.data_json = JSON.stringify(records, null, 2)
}

function supplementaryItemRecordFromForm(existing?: JsonRecord) {
  const record: JsonRecord = { ...(existing || {}) }
  record.title = supplementaryItemForm.value.title.trim()
  record.chapter = supplementaryItemForm.value.chapter.trim()
  record.material_type = supplementaryItemForm.value.type.trim()
  record.type = supplementaryItemForm.value.type.trim()
  record.description = supplementaryItemForm.value.description.trim() || null
  record.resource_link = supplementaryItemForm.value.url.trim()
  if (supplementaryItemForm.value.duration.trim()) record.duration = supplementaryItemForm.value.duration.trim()
  else delete record.duration
  return record
}

function extractQuizRecord(value: unknown) {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null
  const record = value as JsonRecord
  return record.quiz && typeof record.quiz === "object" && !Array.isArray(record.quiz) ? record.quiz as JsonRecord : record
}

function scopeFromQuizzableType(value: unknown): QuizScope {
  const type = Number(value || 0)
  if (type === 3) return "course"
  if (type === 1) return "lesson"
  return "chapter"
}

function quizItemOwnerTitle(item: QuizListItem | null | undefined) {
  if (!item) return "-"
  if (item.ownerType === 3) return courseTitle(item.owner)
  if (item.ownerType === 2) return chapterTitle(item.chapter || item.owner)
  return lessonTitle(item.lesson || item.owner)
}

function quizItemOwnerId(item: QuizListItem | null | undefined) {
  if (!item) return ""
  if (item.ownerType === 3) return courseId(item.owner)
  if (item.ownerType === 2) return chapterId(item.chapter || item.owner)
  return lessonId(item.lesson || item.owner)
}

function courseFormFrom(course: JsonRecord): CourseForm {
  return {
    category_tips: String(course.category_tips || ""),
    title: String(course.title || ""),
    description: String(course.description || ""),
    thumbnail_object_key: String(course.thumbnail_object_key || ""),
    thumbnail_file_hash: String(course.thumbnail_file_hash || ""),
    duration_min: String(course.duration_min || 0),
    certification_enabled: Boolean(course.certification_enabled),
    certification_def_id: String(course.certification_def_id || ""),
    respath: String(course.respath || course.course_gpath || ""),
  }
}

function coursePayload(version?: unknown) {
  const payload: JsonRecord = {
    category_tips: courseForm.value.category_tips.trim(),
    title: courseForm.value.title.trim(),
    description: courseForm.value.description.trim(),
    thumbnail_object_key: courseForm.value.thumbnail_object_key.trim(),
    thumbnail_file_hash: courseForm.value.thumbnail_file_hash.trim(),
    duration_min: Number(courseForm.value.duration_min || 0),
    certification_enabled: courseForm.value.certification_enabled,
    certification_def_id: courseForm.value.certification_def_id.trim(),
    respath: courseForm.value.respath.trim(),
  }
  if (version !== undefined) payload.version = Number(version || 0)
  return payload
}

function resetContent() {
  courseDetail.value = null
  completeCourse.value = null
  chapters.value = []
  selectedChapter.value = null
  lessons.value = []
  materials.value = []
  selectedMaterial.value = null
  supplementaryMaterial.value = null
  quizzes.value = []
  selectedQuiz.value = null
  questions.value = []
  selectedQuestion.value = null
  options.value = []
  editingChapterId.value = ""
  editingLessonId.value = ""
  editingMaterialId.value = ""
  editingSupplementaryItemIndex.value = -1
  editingQuizId.value = ""
  editingQuestionId.value = ""
  editingOptionId.value = ""
  chapterForm.value = emptyChapterForm()
  lessonForm.value = emptyLessonForm()
  materialForm.value = emptyMaterialForm()
  supplementaryMaterialForm.value = emptySupplementaryMaterialForm()
  supplementaryItemForm.value = emptySupplementaryItemForm()
  quizForm.value = emptyQuizForm()
  questionForm.value = emptyQuestionForm()
  optionForm.value = emptyOptionForm()
}

async function loadCourses(pageToken = "") {
  loading.value = true
  try {
    const params = new URLSearchParams({ page_size: String(pageSize) })
    if (categoryFilter.value.trim()) params.set("category_tips", categoryFilter.value.trim())
    if (publishedOnly.value) params.set("published_only", "true")
    if (pageToken) params.set("page_token", pageToken)
    const data = await apiClient<JsonRecord>(`/api/lms/courses?${params}`)
    const list = Array.isArray(data.courses) ? data.courses : []
    const next = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    courses.value = pageToken ? [...courses.value, ...next] : next
    nextPageToken.value = String(data.next_page_token || "")
  } catch (err) {
    console.error(err)
    toast.error("课程列表加载失败")
  } finally {
    loading.value = false
  }
}

async function selectCourse(course: JsonRecord) {
  selectedCourse.value = course
  courseForm.value = courseFormFrom(course)
  resetContent()
  courseView.value = "detail"
  await Promise.all([loadCourseDetail(), loadCompleteCourse(), loadChapters(), loadMaterials(), loadSupplementaryMaterial()])
}

function newCourse() {
  selectedCourse.value = null
  courseForm.value = emptyCourseForm()
  resetContent()
  courseView.value = "detail"
}

function backToCourseList() {
  courseView.value = "list"
}

async function loadCourseDetail() {
  if (!selectedCourseId.value) return
  detailLoading.value = true
  try {
    courseDetail.value = await apiClient<JsonRecord>(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/detail`)
  } catch (err) {
    console.error(err)
    courseDetail.value = null
  } finally {
    detailLoading.value = false
  }
}

async function loadCompleteCourse() {
  if (!selectedCourseId.value) return
  completeLoading.value = true
  try {
    completeCourse.value = await apiClient<JsonRecord>(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/complete`)
  } catch (err) {
    console.error(err)
    completeCourse.value = null
  } finally {
    completeLoading.value = false
  }
}

async function saveCourse() {
  if (!courseForm.value.title.trim()) {
    toast.error("请填写课程标题")
    return
  }
  if (!courseForm.value.respath.trim()) {
    toast.error("请填写 Respath")
    return
  }

  savingCourse.value = true
  try {
    if (selectedCourseId.value) {
      await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}`, {
        method: "PUT",
        body: JSON.stringify(coursePayload(selectedCourse.value?.version)),
      })
      toast.success("课程已更新")
    } else {
      const created = await apiClient<JsonRecord>("/api/lms/courses", {
        method: "POST",
        body: JSON.stringify(coursePayload()),
      })
      selectedCourse.value = created
      toast.success("课程已创建")
    }
    await loadCourses()
    if (selectedCourseId.value) {
      const refreshed = courses.value.find((item) => courseId(item) === selectedCourseId.value)
      if (refreshed) {
        selectedCourse.value = refreshed
        courseForm.value = courseFormFrom(refreshed)
      }
    }
    await Promise.all([loadCourseDetail(), loadCompleteCourse(), loadMaterials(), loadSupplementaryMaterial()])
  } catch (err) {
    console.error(err)
    toast.error("课程保存失败")
  } finally {
    savingCourse.value = false
  }
}

async function publishCourse() {
  if (!selectedCourseId.value) return
  publishing.value = true
  try {
    await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/publish`, {
      method: "POST",
      body: JSON.stringify({ version: versionOf(selectedCourse.value) }),
    })
    toast.success("课程已发布")
    await loadCourses()
    const refreshed = courses.value.find((item) => courseId(item) === selectedCourseId.value)
    if (refreshed) selectedCourse.value = refreshed
  } catch (err) {
    console.error(err)
    toast.error("课程发布失败，请确认章节、课时和测验配置完整")
  } finally {
    publishing.value = false
  }
}

async function deleteCourse() {
  if (!selectedCourseId.value || !window.confirm(`确认删除课程 ${courseTitle(selectedCourse.value)}？`)) return
  try {
    await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}?version=${versionOf(selectedCourse.value)}`, { method: "DELETE" })
    toast.success("课程已删除")
    newCourse()
    courseView.value = "list"
    await loadCourses()
  } catch (err) {
    console.error(err)
    toast.error("课程删除失败")
  }
}

async function loadChapters() {
  if (!selectedCourseId.value) return
  chaptersLoading.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/chapters`)
    const list = Array.isArray(data.chapters) ? data.chapters : []
    chapters.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
  } catch (err) {
    console.error(err)
    toast.error("章节加载失败")
  } finally {
    chaptersLoading.value = false
  }
}

function editChapter(chapter: JsonRecord) {
  selectedChapter.value = chapter
  editingChapterId.value = chapterId(chapter)
  chapterForm.value = {
    title: String(chapter.title || ""),
    sort_order: String(chapter.sort_order || 1),
  }
  void loadLessons()
  newQuiz("chapter")
  void loadQuizzes("chapter")
}

function newChapter() {
  selectedChapter.value = null
  editingChapterId.value = ""
  chapterForm.value = emptyChapterForm()
  lessons.value = []
  editingLessonId.value = ""
  lessonForm.value = emptyLessonForm()
  newQuiz("chapter")
  quizzes.value = []
}

async function saveChapter() {
  if (!selectedCourseId.value || !chapterForm.value.title.trim()) {
    toast.error("请先选择课程并填写章节标题")
    return
  }

  savingChapter.value = true
  try {
    const body = JSON.stringify({
      course_id: selectedCourseId.value,
      title: chapterForm.value.title.trim(),
      sort_order: Number(chapterForm.value.sort_order || 1),
      version: selectedChapter.value?.version || 0,
    })
    if (editingChapterId.value) {
      await apiClient(`/api/lms/chapters/${encodeURIComponent(editingChapterId.value)}`, { method: "PUT", body })
      toast.success("章节已更新")
    } else {
      await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/chapters`, { method: "POST", body })
      toast.success("章节已创建")
    }
    newChapter()
    await loadChapters()
  } catch (err) {
    console.error(err)
    toast.error("章节保存失败")
  } finally {
    savingChapter.value = false
  }
}

async function deleteChapter(chapter: JsonRecord) {
  const id = chapterId(chapter)
  if (!id || !window.confirm(`确认删除章节 ${chapterTitle(chapter)}？`)) return
  try {
    await apiClient(`/api/lms/chapters/${encodeURIComponent(id)}?version=${versionOf(chapter)}`, { method: "DELETE" })
    toast.success("章节已删除")
    newChapter()
    await loadChapters()
  } catch (err) {
    console.error(err)
    toast.error("章节删除失败")
  }
}

async function loadLessons() {
  if (!selectedChapterId.value) return
  lessonsLoading.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/lms/chapters/${encodeURIComponent(selectedChapterId.value)}/lessons`)
    const list = Array.isArray(data.lessons) ? data.lessons : []
    lessons.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
  } catch (err) {
    console.error(err)
    toast.error("课时加载失败")
  } finally {
    lessonsLoading.value = false
  }
}

function editLesson(lesson: JsonRecord) {
  editingLessonId.value = lessonId(lesson)
  const ownerChapter = allLessonItems.value.find((item) => lessonId(item.lesson) === editingLessonId.value)?.chapter
  if (ownerChapter && chapterId(ownerChapter) !== selectedChapterId.value) {
    selectedChapter.value = ownerChapter
  }
  lessonForm.value = {
    chapter_id: chapterId(ownerChapter) || String(lesson.chapter_id || lesson.chapter_ulid || selectedChapterId.value),
    title: String(lesson.title || ""),
    sort_order: String(lesson.sort_order || 1),
    lesson_type: String(lesson.lesson_type || 2),
    body: String(lesson.body || ""),
    asset_object_key: String(lesson.media_object_key || lesson.asset_object_key || lesson.file_object_key || ""),
    asset_file_hash: String(lesson.media_file_hash || lesson.asset_file_hash || lesson.file_hash || ""),
  }
  if (quizForm.value.scope === "lesson") {
    newQuiz("lesson")
    void loadQuizzes("lesson")
  }
}

function newLesson() {
  editingLessonId.value = ""
  lessonForm.value = emptyLessonForm()
  lessonForm.value.chapter_id = selectedChapterId.value
}

async function saveLesson() {
  const targetChapterId = lessonForm.value.chapter_id || selectedChapterId.value
  if (!targetChapterId || !lessonForm.value.title.trim()) {
    toast.error("请先选择章节并填写课时标题")
    return
  }

  savingLesson.value = true
  try {
    const type = Number(lessonForm.value.lesson_type || 2)
    const body = JSON.stringify({
      chapter_id: targetChapterId,
      title: lessonForm.value.title.trim(),
      sort_order: Number(lessonForm.value.sort_order || 1),
      lesson_type: type,
      body: lessonForm.value.body,
      media_object_key: type === 7 ? "" : lessonForm.value.asset_object_key.trim(),
      media_file_hash: type === 7 ? "" : lessonForm.value.asset_file_hash.trim(),
      external_url: type === 7 ? lessonForm.value.asset_object_key.trim() : "",
      version: lessons.value.find((item) => lessonId(item) === editingLessonId.value)?.version || 0,
    })
    if (editingLessonId.value) {
      await apiClient(`/api/lms/lessons/${encodeURIComponent(editingLessonId.value)}`, { method: "PUT", body })
      toast.success("课时已更新")
    } else {
      await apiClient(`/api/lms/chapters/${encodeURIComponent(targetChapterId)}/lessons`, { method: "POST", body })
      toast.success("课时已创建")
    }
    newLesson()
    await Promise.all([loadLessons(), loadCompleteCourse(), loadCourseDetail()])
  } catch (err) {
    console.error(err)
    toast.error("课时保存失败")
  } finally {
    savingLesson.value = false
  }
}

async function deleteLesson(lesson: JsonRecord) {
  const id = lessonId(lesson)
  if (!id || !window.confirm(`确认删除课时 ${lessonTitle(lesson)}？`)) return
  try {
    await apiClient(`/api/lms/lessons/${encodeURIComponent(id)}?version=${versionOf(lesson)}`, { method: "DELETE" })
    toast.success("课时已删除")
    newLesson()
    await Promise.all([loadLessons(), loadCompleteCourse(), loadCourseDetail()])
  } catch (err) {
    console.error(err)
    toast.error("课时删除失败")
  }
}

async function loadMaterials() {
  if (!selectedCourseId.value) return
  materialsLoading.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/materials`)
    const list = Array.isArray(data.materials) ? data.materials : []
    materials.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
  } catch (err) {
    console.error(err)
    toast.error("资料加载失败")
  } finally {
    materialsLoading.value = false
  }
}

async function loadSupplementaryMaterial() {
  if (!selectedCourseId.value) return
  supplementaryMaterialLoading.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/supplementary-material`)
    const material = isJsonRecord(data.material) ? data.material : null
    supplementaryMaterial.value = material
    supplementaryMaterialForm.value = material
      ? {
          kind: String(material.kind || "supplementary_materials"),
          data_json: formatSupplementaryDataJson(material.data_json ?? material.dataJson),
        }
      : emptySupplementaryMaterialForm()
    const firstItem = parseSupplementaryMaterialItems(normalizeSupplementaryMaterials({
      ...(material || {}),
      kind: supplementaryMaterialForm.value.kind,
      data_json: supplementaryMaterialForm.value.data_json,
    }), "Chapter")[0]
    if (firstItem) editSupplementaryItem(firstItem)
    else newSupplementaryItem()
  } catch (err) {
    console.error(err)
    supplementaryMaterial.value = null
    supplementaryMaterialForm.value = emptySupplementaryMaterialForm()
    newSupplementaryItem()
  } finally {
    supplementaryMaterialLoading.value = false
  }
}

function editMaterial(material: JsonRecord) {
  selectedMaterial.value = material
  editingMaterialId.value = materialId(material)
  materialForm.value = {
    title: String(material.title || ""),
    material_type: String(material.material_type || 1),
    description: String(material.description || ""),
    file_object_key: String(material.file_object_key || ""),
    file_hash: String(material.file_hash || ""),
    file_size: String(material.file_size || 0),
    sort_order: String(material.sort_order || 1),
  }
}

function newMaterial() {
  selectedMaterial.value = null
  editingMaterialId.value = ""
  materialForm.value = emptyMaterialForm()
}

function editSupplementaryItem(item: SupplementaryMaterialItem) {
  const records = supplementaryEditableRecords()
  const record = records[item.recordIndex] || {}
  editingSupplementaryItemIndex.value = item.recordIndex
  supplementaryItemForm.value = {
    chapter: stringFromRecord(record, ["chapter", "chapter_title", "chapterTitle", "section"]) || item.chapter,
    type: stringFromRecord(record, ["type", "material_type", "resource_type", "kind"]) || item.type || "Article",
    title: stringFromRecord(record, ["title", "name", "label", "heading"]) || item.title,
    description: stringFromRecord(record, ["description", "desc", "summary", "detail", "content"]) || item.description,
    url: stringFromRecord(record, ["resource_link", "resourceLink", "url", "link", "href", "external_url", "externalUrl"]) || item.url,
    duration: stringFromRecord(record, ["duration", "duration_min", "durationMin", "time", "length"]),
  }
}

function newSupplementaryItem() {
  editingSupplementaryItemIndex.value = -1
  supplementaryItemForm.value = emptySupplementaryItemForm()
}

async function saveSupplementaryItem() {
  if (!selectedCourseId.value) return
  if (!supplementaryItemForm.value.title.trim()) {
    toast.error("请填写资料标题")
    return
  }
  if (!supplementaryItemForm.value.url.trim()) {
    toast.error("请填写资料链接")
    return
  }
  const records = supplementaryEditableRecords()
  if (editingSupplementaryItemIndex.value >= 0) {
    records[editingSupplementaryItemIndex.value] = supplementaryItemRecordFromForm(records[editingSupplementaryItemIndex.value])
  } else {
    editingSupplementaryItemIndex.value = records.length
    records.push(supplementaryItemRecordFromForm())
  }
  updateSupplementaryDataJson(records)
  await saveSupplementaryMaterial()
}

async function deleteSupplementaryItem() {
  if (editingSupplementaryItemIndex.value < 0) return
  if (!window.confirm(`确认删除资料 ${supplementaryItemForm.value.title || ""}？`)) return
  const records = supplementaryEditableRecords()
  records.splice(editingSupplementaryItemIndex.value, 1)
  updateSupplementaryDataJson(records)
  newSupplementaryItem()
  await saveSupplementaryMaterial()
}

async function saveSupplementaryMaterial() {
  if (!selectedCourseId.value) return
  if (!supplementaryMaterialForm.value.kind.trim()) {
    toast.error("请填写辅助资料 kind")
    return
  }
  try {
    JSON.parse(supplementaryMaterialForm.value.data_json)
  } catch {
    toast.error("辅助资料 data_json 必须是合法 JSON")
    return
  }

  savingSupplementaryMaterial.value = true
  try {
    const body: JsonRecord = {
      kind: supplementaryMaterialForm.value.kind.trim(),
      data_json: supplementaryMaterialForm.value.data_json.trim(),
    }
    const id = supplementaryMaterialId(supplementaryMaterial.value)
    if (id) {
      body.version = versionOf(supplementaryMaterial.value)
      await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/supplementary-material/${encodeURIComponent(id)}`, { method: "PUT", body: JSON.stringify(body) })
      toast.success("辅助资料已更新")
    } else {
      await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/supplementary-material`, { method: "POST", body: JSON.stringify(body) })
      toast.success("辅助资料已创建")
    }
    await Promise.all([loadSupplementaryMaterial(), loadCompleteCourse(), loadCourseDetail()])
  } catch (err) {
    console.error(err)
    toast.error("辅助资料保存失败")
  } finally {
    savingSupplementaryMaterial.value = false
  }
}

async function deleteSupplementaryMaterial() {
  const id = supplementaryMaterialId(supplementaryMaterial.value)
  if (!selectedCourseId.value || !id || !window.confirm("确认删除这份候选端辅助资料配置？")) return
  try {
    await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/supplementary-material/${encodeURIComponent(id)}?version=${versionOf(supplementaryMaterial.value)}`, { method: "DELETE" })
    toast.success("辅助资料已删除")
    await Promise.all([loadSupplementaryMaterial(), loadCompleteCourse(), loadCourseDetail()])
  } catch (err) {
    console.error(err)
    toast.error("辅助资料删除失败")
  }
}

async function saveMaterial() {
  if (!selectedCourseId.value || !materialForm.value.title.trim() || !materialForm.value.file_object_key.trim()) {
    toast.error("请先选择课程，并填写资料标题和文件 Object Key")
    return
  }

  savingMaterial.value = true
  try {
    const body: JsonRecord = {
      title: materialForm.value.title.trim(),
      material_type: Number(materialForm.value.material_type || 0),
      description: materialForm.value.description.trim(),
      file_object_key: materialForm.value.file_object_key.trim(),
      file_hash: materialForm.value.file_hash.trim(),
      file_size: Number(materialForm.value.file_size || 0),
      sort_order: Number(materialForm.value.sort_order || 1),
    }
    if (editingMaterialId.value) {
      body.version = materials.value.find((item) => materialId(item) === editingMaterialId.value)?.version || 0
      await apiClient(`/api/lms/materials/${encodeURIComponent(editingMaterialId.value)}`, { method: "PUT", body: JSON.stringify(body) })
      toast.success("资料已更新")
    } else {
      await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/materials`, { method: "POST", body: JSON.stringify(body) })
      toast.success("资料已创建")
    }
    newMaterial()
    await Promise.all([loadMaterials(), loadCourseDetail(), loadCompleteCourse()])
  } catch (err) {
    console.error(err)
    toast.error("资料保存失败")
  } finally {
    savingMaterial.value = false
  }
}

async function deleteMaterial(material: JsonRecord) {
  const id = materialId(material)
  if (!id || !window.confirm(`确认删除资料 ${materialTitle(material)}？`)) return
  try {
    await apiClient(`/api/lms/materials/${encodeURIComponent(id)}?version=${versionOf(material)}`, { method: "DELETE" })
    toast.success("资料已删除")
    newMaterial()
    await Promise.all([loadMaterials(), loadCourseDetail(), loadCompleteCourse()])
  } catch (err) {
    console.error(err)
    toast.error("资料删除失败")
  }
}

function quizTarget(scope: QuizScope = quizForm.value.scope) {
  if (scope === "course") {
    return { type: 3, id: selectedCourseId.value, label: "课程", title: courseTitle(selectedCourse.value) }
  }
  if (scope === "chapter") {
    return { type: 2, id: selectedChapterId.value, label: "章节", title: chapterTitle(selectedChapter.value) }
  }
  return { type: 1, id: editingLessonId.value, label: "课时", title: lessonTitle(selectedLesson.value) }
}

function clearQuestionState() {
  selectedQuestion.value = null
  questions.value = []
  options.value = []
  editingQuestionId.value = ""
  editingOptionId.value = ""
  questionForm.value = emptyQuestionForm()
  optionForm.value = emptyOptionForm()
}

async function loadQuizzes(scope: QuizScope = quizForm.value.scope) {
  const target = quizTarget(scope)
  if (!target.id) {
    toast.error(`请先选择${target.label}`)
    return
  }

  quizzesLoading.value = true
  try {
    const params = new URLSearchParams({
      quizzable_type: String(target.type),
      quizzable_id: target.id,
    })
    const data = await apiClient<JsonRecord>(`/api/lms/quizzes?${params}`)
    const list = Array.isArray(data.quizzes) ? data.quizzes : []
    quizzes.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    selectedQuiz.value = null
    editingQuizId.value = ""
    clearQuestionState()
  } catch (err) {
    console.error(err)
    toast.error("测验加载失败")
  } finally {
    quizzesLoading.value = false
  }
}

function editQuiz(quiz: JsonRecord) {
  const item = allQuizItems.value.find((entry) => quizId(entry.quiz) === quizId(quiz))
  const scope = scopeFromQuizzableType(item?.ownerType || quiz.quizzable_type || quizForm.value.scope)
  if (item?.chapter) selectedChapter.value = item.chapter
  if (item?.lesson) editLesson(item.lesson)
  selectedQuiz.value = quiz
  editingQuizId.value = quizId(quiz)
  quizForm.value = {
    scope,
    title: String(quiz.title || ""),
    description: String(quiz.description || ""),
    passing_score: String(quiz.passing_score || 70),
    time_limit: String(quiz.time_limit || 0),
    randomize_questions: Boolean(quiz.randomize_questions),
  }
  void loadQuestions()
}

function newQuiz(scope: QuizScope = quizForm.value.scope) {
  selectedQuiz.value = null
  editingQuizId.value = ""
  quizForm.value = { ...emptyQuizForm(), scope }
  clearQuestionState()
}

async function saveQuiz() {
  if (!quizForm.value.title.trim()) {
    toast.error("请填写测验标题")
    return
  }
  const target = quizTarget()
  if (!target.id) {
    toast.error(`请先选择${target.label}`)
    return
  }

  savingQuiz.value = true
  try {
    const body: JsonRecord = {
      quizzable_type: target.type,
      quizzable_ulid: target.id,
      title: quizForm.value.title.trim(),
      description: quizForm.value.description.trim(),
      passing_score: Number(quizForm.value.passing_score || 0),
      time_limit: Number(quizForm.value.time_limit || 0),
      randomize_questions: quizForm.value.randomize_questions,
    }
    if (editingQuizId.value) {
      body.version = quizzes.value.find((item) => quizId(item) === editingQuizId.value)?.version || 0
      await apiClient(`/api/lms/quizzes/${encodeURIComponent(editingQuizId.value)}`, { method: "PUT", body: JSON.stringify(body) })
      toast.success("测验已更新")
    } else {
      await apiClient("/api/lms/quizzes", { method: "POST", body: JSON.stringify(body) })
      toast.success("测验已创建")
    }
    const scope = quizForm.value.scope
    newQuiz(scope)
    await Promise.all([loadQuizzes(scope), loadCourseDetail(), loadCompleteCourse()])
  } catch (err) {
    console.error(err)
    toast.error("测验保存失败")
  } finally {
    savingQuiz.value = false
  }
}

async function deleteQuiz(quiz: JsonRecord) {
  const id = quizId(quiz)
  if (!id || !window.confirm(`确认删除测验 ${quizTitle(quiz)}？`)) return
  try {
    await apiClient(`/api/lms/quizzes/${encodeURIComponent(id)}?version=${versionOf(quiz)}`, { method: "DELETE" })
    toast.success("测验已删除")
    const scope = quizForm.value.scope
    newQuiz(scope)
    await Promise.all([loadQuizzes(scope), loadCourseDetail(), loadCompleteCourse()])
  } catch (err) {
    console.error(err)
    toast.error("测验删除失败")
  }
}

async function loadQuestions(id = selectedQuizId.value) {
  if (!id) return
  questionsLoading.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/lms/quizzes/${encodeURIComponent(id)}/questions`)
    const list = Array.isArray(data.questions) ? data.questions : []
    questions.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    selectedQuestion.value = null
    options.value = []
    editingQuestionId.value = ""
    editingOptionId.value = ""
    questionForm.value = emptyQuestionForm()
    optionForm.value = emptyOptionForm()
  } catch (err) {
    console.error(err)
    toast.error("题目加载失败")
  } finally {
    questionsLoading.value = false
  }
}

function editQuestion(question: JsonRecord) {
  selectedQuestion.value = question
  editingQuestionId.value = questionId(question)
  questionForm.value = {
    question_text: String(question.question_text || ""),
    question_type: String(question.question_type || 1),
    points: String(question.points || 10),
    sort_order: String(question.sort_order || 1),
    is_required: question.is_required !== false,
    media_items_json: String(question.media_items_json || "[]"),
  }
  void loadOptions()
}

function newQuestion() {
  selectedQuestion.value = null
  editingQuestionId.value = ""
  questionForm.value = emptyQuestionForm()
  options.value = []
  editingOptionId.value = ""
  optionForm.value = emptyOptionForm()
}

async function saveQuestion() {
  if (!selectedQuizId.value || !questionForm.value.question_text.trim()) {
    toast.error("请先选择测验并填写题干")
    return
  }
  const mediaJson = questionForm.value.media_items_json.trim() || "[]"
  try {
    JSON.parse(mediaJson)
  } catch {
    toast.error("媒体 JSON 格式不正确")
    return
  }

  savingQuestion.value = true
  try {
    const body: JsonRecord = {
      question_text: questionForm.value.question_text.trim(),
      question_type: Number(questionForm.value.question_type || 1),
      points: Number(questionForm.value.points || 0),
      sort_order: Number(questionForm.value.sort_order || 1),
      is_required: questionForm.value.is_required,
      media_items_json: mediaJson,
    }
    if (editingQuestionId.value) {
      body.version = questions.value.find((item) => questionId(item) === editingQuestionId.value)?.version || 0
      await apiClient(`/api/lms/questions/${encodeURIComponent(editingQuestionId.value)}`, { method: "PUT", body: JSON.stringify(body) })
      toast.success("题目已更新")
    } else {
      await apiClient(`/api/lms/quizzes/${encodeURIComponent(selectedQuizId.value)}/questions`, { method: "POST", body: JSON.stringify(body) })
      toast.success("题目已创建")
    }
    newQuestion()
    await loadQuestions()
  } catch (err) {
    console.error(err)
    toast.error("题目保存失败")
  } finally {
    savingQuestion.value = false
  }
}

async function deleteQuestion(question: JsonRecord) {
  const id = questionId(question)
  if (!id || !window.confirm(`确认删除题目 ${questionTitle(question)}？`)) return
  try {
    await apiClient(`/api/lms/questions/${encodeURIComponent(id)}?version=${versionOf(question)}`, { method: "DELETE" })
    toast.success("题目已删除")
    newQuestion()
    await loadQuestions()
  } catch (err) {
    console.error(err)
    toast.error("题目删除失败")
  }
}

async function loadOptions(id = selectedQuestionId.value) {
  if (!id) return
  optionsLoading.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/lms/questions/${encodeURIComponent(id)}/options`)
    const list = Array.isArray(data.options) ? data.options : []
    options.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    editingOptionId.value = ""
    optionForm.value = emptyOptionForm()
  } catch (err) {
    console.error(err)
    toast.error("选项加载失败")
  } finally {
    optionsLoading.value = false
  }
}

function editOption(option: JsonRecord) {
  editingOptionId.value = optionId(option)
  optionForm.value = {
    option_text: String(option.option_text || ""),
    is_correct: Boolean(option.is_correct),
    sort_order: String(option.sort_order || 1),
  }
}

function newOption() {
  editingOptionId.value = ""
  optionForm.value = emptyOptionForm()
}

async function saveOption() {
  if (!selectedQuestionId.value || !optionForm.value.option_text.trim()) {
    toast.error("请先选择题目并填写选项")
    return
  }

  savingOption.value = true
  try {
    const body: JsonRecord = {
      option_text: optionForm.value.option_text.trim(),
      is_correct: optionForm.value.is_correct,
      sort_order: Number(optionForm.value.sort_order || 1),
    }
    if (editingOptionId.value) {
      body.version = options.value.find((item) => optionId(item) === editingOptionId.value)?.version || 0
      await apiClient(`/api/lms/options/${encodeURIComponent(editingOptionId.value)}`, { method: "PUT", body: JSON.stringify(body) })
      toast.success("选项已更新")
    } else {
      await apiClient(`/api/lms/questions/${encodeURIComponent(selectedQuestionId.value)}/options`, { method: "POST", body: JSON.stringify(body) })
      toast.success("选项已创建")
    }
    newOption()
    await loadOptions()
  } catch (err) {
    console.error(err)
    toast.error("选项保存失败")
  } finally {
    savingOption.value = false
  }
}

async function deleteOption(option: JsonRecord) {
  const id = optionId(option)
  if (!id || !window.confirm(`确认删除选项 ${optionTitle(option)}？`)) return
  try {
    await apiClient(`/api/lms/options/${encodeURIComponent(id)}?version=${versionOf(option)}`, { method: "DELETE" })
    toast.success("选项已删除")
    newOption()
    await loadOptions()
  } catch (err) {
    console.error(err)
    toast.error("选项删除失败")
  }
}

async function importLmsJson() {
  if (!importJson.value.trim()) {
    toast.error("请粘贴或上传 JSON")
    return
  }
  try {
    JSON.parse(importJson.value)
  } catch {
    toast.error("JSON 格式不正确")
    return
  }
  if (importScope.value === "quiz" && !selectedChapterId.value) {
    toast.error("导入测验前请先选择章节")
    return
  }

  importing.value = true
  try {
    const body = importScope.value === "course"
      ? {
          scope: "course",
          category_tips: importCategoryTips.value.trim(),
          course_json: importJson.value,
        }
      : {
          scope: "quiz",
          quizzable_type: 2,
          quizzable_id: selectedChapterId.value,
          quiz_json: importJson.value,
        }
    await apiClient("/api/lms/import", { method: "POST", body: JSON.stringify(body) })
    toast.success("导入完成")
    importOpen.value = false
    importJson.value = ""
    await loadCourses()
    if (selectedCourseId.value) await loadChapters()
  } catch (err) {
    console.error(err)
    toast.error("导入失败")
  } finally {
    importing.value = false
  }
}

async function loadImportFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  importJson.value = await file.text()
  input.value = ""
}

watch([categoryFilter, publishedOnly], () => {
  nextPageToken.value = ""
  void loadCourses()
})

onMounted(() => {
  void loadCourses()
})
</script>

<template>
  <div class="space-y-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">课程配置</h1>
        <p class="mt-2 text-slate-600">按 GLMS 层级维护课程、资料、章节、课时和测验题库。</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 font-bold shadow-sm disabled:opacity-60" type="button" :disabled="loading" @click="loadCourses()">
          <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
          刷新
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 font-bold shadow-sm" type="button" @click="importOpen = true">
          <FileJson class="h-4 w-4" />
          导入 JSON
        </button>
        <button v-if="courseView === 'detail'" class="rounded-xl border bg-white px-4 py-3 font-bold shadow-sm" type="button" @click="backToCourseList">
          返回列表
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-4 py-3 font-bold text-white shadow-lg shadow-sky-200" type="button" @click="newCourse">
          <Plus class="h-4 w-4" />
          新建课程
        </button>
      </div>
    </header>

    <section v-if="courseView === 'list'" class="rounded-3xl border border-slate-200 bg-white shadow-sm">
      <div class="grid gap-3 border-b border-slate-200 bg-slate-50/60 p-4 lg:grid-cols-[1fr_auto]">
        <input v-model="categoryFilter" class="h-10 rounded-xl border border-slate-200 bg-white px-4 text-sm shadow-sm outline-none transition focus:border-sky-300 focus:ring-2 focus:ring-sky-100" placeholder="分类筛选，例如 CFtP/CFtA" />
        <label class="inline-flex h-10 items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 text-sm font-bold text-slate-600 shadow-sm">
          <input v-model="publishedOnly" type="checkbox" />
          仅看已发布
        </label>
      </div>

      <div v-if="loading && !courses.length" class="p-12 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        正在加载...
      </div>
      <div v-else-if="!courses.length" class="p-12 text-center text-slate-500">暂无课程</div>
      <div v-else>
        <div class="hidden grid-cols-[minmax(0,1fr)_120px_260px_120px] gap-6 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-400 lg:grid">
          <span>课程</span>
          <span>版本</span>
          <span>更新时间</span>
          <span class="text-right">状态</span>
        </div>
        <button
          v-for="course in courses"
          :key="courseId(course)"
          class="block w-full border-b border-slate-100 px-5 py-3 text-left transition last:border-b-0 hover:bg-slate-50"
          :class="courseId(course) === selectedCourseId ? 'bg-sky-50/70' : ''"
          type="button"
          @click="selectCourse(course)"
        >
          <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_120px_260px_120px] lg:items-center lg:gap-6">
            <div class="min-w-0">
              <div class="truncate text-base font-black text-slate-950">{{ courseTitle(course) }}</div>
              <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500">
                <span>{{ course.category_tips || "未分类" }}</span>
                <span class="font-mono">ID: {{ courseId(course) || "-" }}</span>
              </div>
            </div>
            <div class="text-sm font-bold text-slate-700">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">版本</span>{{ course.version || 0 }}
            </div>
            <div class="text-sm text-slate-500">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">更新</span>{{ formatDate(String(course.updated_at || course.created_at || "")) }}
            </div>
            <span class="justify-self-start rounded-full border px-3 py-1 text-xs font-black lg:justify-self-end" :class="badgeClass(course.is_published ? 'COMPLETED' : 'PENDING')">
              {{ course.is_published ? "已发布" : "草稿" }}
            </span>
          </div>
        </button>
      </div>
      <div v-if="nextPageToken" class="border-t border-slate-200 p-4">
        <button class="w-full rounded-xl border px-4 py-3 font-bold transition hover:bg-slate-50 disabled:cursor-default disabled:border-slate-200 disabled:bg-slate-50 disabled:text-slate-400 disabled:opacity-100" type="button" :disabled="!nextPageToken || loading" @click="loadCourses(nextPageToken)">
          加载更多
        </button>
      </div>
    </section>

    <main v-else class="space-y-6">
      <section class="rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">{{ selectedCourseId ? "课程顶层数据" : "新建课程" }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ selectedCourseId || "填写课程基础信息后保存。" }}</p>
          </div>
          <span v-if="selectedCourseId" class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(selectedCourseStatus)">
            {{ selectedCoursePublished ? "已发布" : selectedCourseStatus }}
          </span>
        </div>

        <div class="grid gap-4 p-5 2xl:grid-cols-[minmax(0,1fr)_420px]">
          <form class="grid gap-3 lg:grid-cols-2" @submit.prevent="saveCourse">
            <label class="block">
              <span class="text-sm font-bold">课程标题</span>
              <input v-model="courseForm.title" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">分类提示</span>
              <input v-model="courseForm.category_tips" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <label class="block lg:col-span-2">
              <span class="text-sm font-bold">描述</span>
              <textarea v-model="courseForm.description" class="mt-2 min-h-20 w-full rounded-xl border border-slate-200 px-3 py-2" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">Respath</span>
              <input v-model="courseForm.respath" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="/gcc/pipeline/..." />
            </label>
            <label class="block">
              <span class="text-sm font-bold">时长分钟</span>
              <input v-model="courseForm.duration_min" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" type="number" min="0" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">封面 Object Key</span>
              <input v-model="courseForm.thumbnail_object_key" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">封面 File Hash</span>
              <input v-model="courseForm.thumbnail_file_hash" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <label class="inline-flex items-center gap-2 text-sm font-bold text-slate-600">
              <input v-model="courseForm.certification_enabled" type="checkbox" />
              启用证书
            </label>
            <label class="block">
              <span class="text-sm font-bold">证书定义 ID</span>
              <input v-model="courseForm.certification_def_id" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <div class="flex flex-wrap gap-3 lg:col-span-2">
              <button class="inline-flex h-10 items-center gap-2 rounded-xl bg-[#0b4ea2] px-4 font-bold text-white disabled:opacity-50" :disabled="savingCourse" type="submit">
                <Loader2 v-if="savingCourse" class="h-4 w-4 animate-spin" />
                <Save v-else class="h-4 w-4" />
                {{ savingCourse ? "保存中..." : "保存课程" }}
              </button>
              <button class="h-10 rounded-xl border px-4 font-bold disabled:opacity-40" :disabled="!selectedCourseId || publishing" type="button" @click="publishCourse">
                {{ publishing ? "发布中..." : "发布课程" }}
              </button>
              <button class="inline-flex h-10 items-center gap-2 rounded-xl border border-red-200 px-4 font-bold text-red-600 disabled:opacity-40" :disabled="!selectedCourseId" type="button" @click="deleteCourse">
                <Trash2 class="h-4 w-4" />
                删除课程
              </button>
            </div>
          </form>

          <aside class="space-y-3">
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="rounded-xl bg-slate-50 p-3">
                <div class="text-xs font-black uppercase text-slate-400">章节</div>
                <div class="mt-1 text-xl font-black">{{ courseDetail?.chapter_count ?? chapters.length }}</div>
              </div>
              <div class="rounded-xl bg-slate-50 p-3">
                <div class="text-xs font-black uppercase text-slate-400">课时</div>
                <div class="mt-1 text-xl font-black">{{ courseDetail?.lesson_count ?? lessons.length }}</div>
              </div>
              <div class="rounded-xl bg-slate-50 p-3">
                <div class="text-xs font-black uppercase text-slate-400">测验</div>
                <div class="mt-1 text-xl font-black">{{ courseDetail?.quiz_count ?? 0 }}</div>
              </div>
              <div class="rounded-xl bg-slate-50 p-3">
                <div class="text-xs font-black uppercase text-slate-400">资料</div>
                <div class="mt-1 text-xl font-black">{{ courseDetail?.material_count ?? materials.length }}</div>
              </div>
            </div>
            <div class="rounded-xl border border-slate-200 p-3">
              <h3 class="font-black">不可编辑字段</h3>
              <p class="mt-1 text-xs text-slate-500">这些字段来自 get/list 接口，仅展示不修改。</p>
              <div class="mt-3 max-h-80 space-y-2 overflow-y-auto pr-1">
                <label v-for="entry in recordEntries(selectedCourse)" :key="`course-${entry.key}`" class="block">
                  <span class="text-xs font-black text-slate-500">{{ entry.key }}</span>
                  <textarea class="mt-1 min-h-10 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" disabled />
                </label>
                <div v-if="detailLoading || completeLoading" class="text-sm text-slate-500">正在加载完整课程数据...</div>
                <label v-for="entry in recordEntries(courseDetail)" :key="`detail-${entry.key}`" class="block">
                  <span class="text-xs font-black text-slate-500">detail.{{ entry.key }}</span>
                  <textarea class="mt-1 min-h-10 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" disabled />
                </label>
                <label v-for="entry in recordEntries(completeCourse)" :key="`complete-${entry.key}`" class="block">
                  <span class="text-xs font-black text-slate-500">complete.{{ entry.key }}</span>
                  <textarea class="mt-1 min-h-10 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" disabled />
                </label>
              </div>
            </div>
          </aside>
        </div>
      </section>

      <section class="rounded-2xl border border-slate-200 bg-white shadow-sm" :class="!selectedCourseId ? 'opacity-50' : ''">
        <div class="border-b border-slate-200 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">课程资料</h2>
            <p class="mt-1 text-sm text-slate-500">候选端看到的 Supplementary Materials 来自辅助资料接口；普通文件资料由 materials 接口维护。</p>
          </div>
        </div>

        <div class="grid gap-4 p-5 xl:grid-cols-[minmax(0,1fr)_420px]">
          <div class="rounded-2xl border border-slate-200">
            <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 p-4">
              <div>
                <h3 class="font-black">候选端辅助资料</h3>
                <p class="mt-1 text-sm text-slate-500">这里对应用户端“课程资料 / Supplementary Materials”列表。</p>
              </div>
              <div class="flex items-center gap-2">
                <span class="rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-xs font-black text-slate-600">{{ supplementaryMaterialItems.length }} 条</span>
                <button class="rounded-xl border px-3 py-2 text-sm font-bold disabled:opacity-40" :disabled="!selectedCourseId || supplementaryMaterialLoading" type="button" @click="loadSupplementaryMaterial">
                  刷新
                </button>
              </div>
            </div>
            <div v-if="supplementaryMaterialLoading" class="px-6 py-10 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              正在加载...
            </div>
            <div v-else-if="!supplementaryMaterialItems.length" class="px-6 py-10 text-center text-slate-500">
              暂无候选端辅助资料
              <div class="mt-2 text-xs">如果用户端有资料，请确认 supplementary_material.data_json 是否已配置。</div>
            </div>
            <div v-else class="overflow-x-auto">
              <table class="min-w-full text-left text-sm">
                <thead class="bg-slate-50 text-xs font-black uppercase tracking-wide text-slate-500">
                  <tr>
                    <th class="px-4 py-3">Chapter</th>
                    <th class="px-4 py-3">Type</th>
                    <th class="px-4 py-3">Title & Description</th>
                    <th class="px-4 py-3">Resource Link</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-100">
                  <tr
                    v-for="item in supplementaryMaterialItems"
                    :key="item.key"
                    class="cursor-pointer transition hover:bg-sky-50"
                    :class="item.recordIndex === editingSupplementaryItemIndex ? 'bg-sky-50' : ''"
                    @click="editSupplementaryItem(item)"
                  >
                    <td class="whitespace-nowrap px-4 py-4 font-semibold text-slate-800">
                      <span class="mr-2 rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-500">#{{ item.recordIndex + 1 }}</span>
                      {{ item.chapter }}
                    </td>
                    <td class="px-4 py-4">
                      <span class="rounded-full border px-2 py-1 text-xs font-black" :class="supplementaryTypeClass(item.type)">{{ supplementaryTypeLabel(item.type) }}</span>
                    </td>
                    <td class="px-4 py-4">
                      <div class="font-black text-slate-950">{{ item.title }}</div>
                      <div v-if="item.description" class="mt-1 max-w-2xl text-sm text-slate-500">{{ item.description }}</div>
                    </td>
                    <td class="px-4 py-4">
                      <a v-if="item.url" class="inline-flex max-w-xs items-center rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-xs font-bold text-blue-700 hover:bg-blue-100" :href="item.url" target="_blank" rel="noreferrer">
                        <span class="truncate">{{ item.url }}</span>
                      </a>
                      <span v-else class="text-slate-400">-</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <form class="rounded-2xl border border-slate-200 p-4" @submit.prevent="saveSupplementaryItem">
            <div class="flex items-start justify-between gap-3">
              <div>
                <h3 class="font-black">{{ editingSupplementaryItemIndex >= 0 ? "编辑单条辅助资料" : "新增单条辅助资料" }}</h3>
                <p class="mt-1 text-xs text-slate-500">这里改的是 data_json 里的单条记录，保存后会整体提交辅助资料配置。</p>
              </div>
              <div class="flex gap-2">
                <button class="rounded-xl border px-3 py-2 text-sm font-bold disabled:opacity-40" :disabled="!selectedCourseId" type="button" @click="newSupplementaryItem">
                  新增
                </button>
                <button class="rounded-xl border border-red-200 px-3 py-2 text-sm font-bold text-red-600 disabled:opacity-40" :disabled="editingSupplementaryItemIndex < 0" type="button" @click="deleteSupplementaryItem">
                  删除本条
                </button>
              </div>
            </div>
            <div class="mt-4 grid gap-3">
              <label class="block">
                <span class="text-sm font-bold">所属章节</span>
                <input v-model="supplementaryItemForm.chapter" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="例如 Chapter 1: Overview of Fintech" />
              </label>
              <div class="grid gap-3 sm:grid-cols-2">
                <label class="block">
                  <span class="text-sm font-bold">资料类型</span>
                  <select v-model="supplementaryItemForm.type" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3">
                    <option value="Article">Article</option>
                    <option value="Video">Video</option>
                    <option value="PDF">PDF</option>
                    <option value="Link">Link</option>
                    <option value="Other">Other</option>
                  </select>
                </label>
                <label class="block">
                  <span class="text-sm font-bold">时长/备注</span>
                  <input v-model="supplementaryItemForm.duration" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="可选，例如 5 min" />
                </label>
              </div>
              <label class="block">
                <span class="text-sm font-bold">标题</span>
                <input v-model="supplementaryItemForm.title" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="资料标题" />
              </label>
              <label class="block">
                <span class="text-sm font-bold">描述</span>
                <textarea v-model="supplementaryItemForm.description" class="mt-2 min-h-20 w-full rounded-xl border border-slate-200 px-3 py-2" placeholder="可选，用户端会展示在标题下方" />
              </label>
              <label class="block">
                <span class="text-sm font-bold">资源链接</span>
                <input v-model="supplementaryItemForm.url" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="https://..." />
              </label>
              <button class="h-10 rounded-xl bg-[#0b4ea2] px-4 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingSupplementaryMaterial" type="submit">
                {{ savingSupplementaryMaterial ? "保存中..." : "保存本条资料" }}
              </button>
            </div>

            <details class="mt-5 border-t border-slate-200 pt-4">
              <summary class="cursor-pointer font-black">高级配置：整份辅助资料 JSON</summary>
              <div class="mt-4 grid gap-3">
                <label class="block">
                  <span class="text-sm font-bold">kind</span>
                  <input v-model="supplementaryMaterialForm.kind" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="supplementary_materials" />
                  <span class="mt-1 block text-xs text-slate-500">资料集合类型标识，一般保持默认即可。</span>
                </label>
                <label class="block">
                  <span class="text-sm font-bold">data_json</span>
                  <textarea v-model="supplementaryMaterialForm.data_json" class="mt-2 min-h-64 w-full rounded-xl border border-slate-200 p-4 font-mono text-xs leading-5" placeholder='例如 [{"chapter":"Chapter 1","type":"Article","title":"What is fintech?","resource_link":"https://..."}]' />
                  <span class="mt-1 block text-xs text-slate-500">高级模式会直接保存整份 JSON；常规增删改建议使用上面的单条表单。</span>
                </label>
                <div class="grid gap-2 sm:grid-cols-2">
                  <button class="rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingSupplementaryMaterial" type="button" @click="saveSupplementaryMaterial">
                    {{ savingSupplementaryMaterial ? "保存中..." : "保存整份 JSON" }}
                  </button>
                  <button class="rounded-xl border border-red-200 px-5 py-3 font-bold text-red-600 disabled:opacity-40" :disabled="!supplementaryMaterialId(supplementaryMaterial)" type="button" @click="deleteSupplementaryMaterial">
                    删除整份配置
                  </button>
                </div>
              </div>
            </details>

            <div v-if="supplementaryMaterial" class="mt-5 border-t border-slate-200 pt-4">
              <h4 class="font-black">辅助资料完整字段</h4>
              <div class="mt-3 max-h-48 space-y-3 overflow-y-auto pr-1">
                <label v-for="entry in recordEntries(supplementaryMaterial)" :key="`supplementary-${entry.key}`" class="block">
                  <span class="text-xs font-black text-slate-500">{{ entry.key }}</span>
                  <textarea class="mt-1 min-h-10 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" disabled />
                </label>
              </div>
            </div>
          </form>
        </div>

        <div class="border-t border-slate-200 px-5 pt-4">
          <div class="flex flex-wrap items-center justify-between gap-4">
            <div>
              <h3 class="font-black">普通文件资料</h3>
              <p class="mt-1 text-sm text-slate-500">这里对应 /materials 接口，通常是对象存储文件类资料；和上面的候选端辅助资料不是同一份数据。</p>
            </div>
            <button class="h-10 rounded-xl border px-4 font-bold disabled:opacity-40" :disabled="!selectedCourseId" type="button" @click="newMaterial">新增普通资料</button>
          </div>
        </div>
        <div class="grid gap-4 p-5 xl:grid-cols-[minmax(0,1fr)_400px]">
          <div class="rounded-2xl border border-slate-200">
            <div v-if="materialsLoading" class="px-6 py-10 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              正在加载...
            </div>
            <div v-else-if="!materials.length" class="px-6 py-10 text-center text-slate-500">暂无资料</div>
            <div v-else class="divide-y divide-slate-100">
              <div v-for="material in materials" :key="materialId(material)" class="grid gap-3 p-4 lg:grid-cols-[1fr_auto]" :class="materialId(material) === selectedMaterialId ? 'bg-sky-50' : ''">
                <button class="text-left" type="button" @click="editMaterial(material)">
                  <div class="font-black">{{ materialTitle(material) }}</div>
                  <div class="mt-1 text-sm text-slate-500">{{ materialTypeLabel(material.material_type) }} · 排序 {{ material.sort_order || 0 }}</div>
                  <div class="mt-1 break-all text-xs text-slate-400">{{ material.file_object_key || "-" }}</div>
                </button>
                <button class="rounded-xl border border-red-200 px-3 py-2 text-sm font-bold text-red-600" type="button" @click="deleteMaterial(material)">删除</button>
              </div>
            </div>
          </div>
          <form class="rounded-2xl border border-slate-200 p-4" @submit.prevent="saveMaterial">
            <h3 class="font-black">{{ editingMaterialId ? "编辑资料" : "创建资料" }}</h3>
            <div class="mt-4 grid gap-3">
              <input v-model="materialForm.title" class="h-10 rounded-xl border border-slate-200 px-3" placeholder="资料标题" />
              <select v-model="materialForm.material_type" class="h-10 rounded-xl border border-slate-200 px-3">
                <option value="1">教材/课本</option>
                <option value="2">幻灯片/课件</option>
                <option value="3">参考资料</option>
                <option value="4">其他</option>
              </select>
              <textarea v-model="materialForm.description" class="min-h-20 rounded-xl border border-slate-200 px-3 py-2" placeholder="资料描述" />
              <label class="block">
                <span class="text-sm font-bold">文件 Object Key</span>
                <input v-model="materialForm.file_object_key" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="例如 courses/xxx/materials/file.pdf" />
                <span class="mt-1 block text-xs text-slate-500">对象存储里的文件路径。一般先在资源/上传流程拿到，不建议手工猜。</span>
              </label>
              <label class="block">
                <span class="text-sm font-bold">文件 Hash</span>
                <input v-model="materialForm.file_hash" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="文件 SHA256 Hash" />
                <span class="mt-1 block text-xs text-slate-500">用于校验文件内容，上传接口或资源系统通常会返回。</span>
              </label>
              <div class="grid gap-3 sm:grid-cols-2">
                <label class="block">
                  <span class="text-sm font-bold">文件大小（字节）</span>
                  <input v-model="materialForm.file_size" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="例如 204800" type="number" min="0" />
                  <span class="mt-1 block text-xs text-slate-500">不知道时可先填 0。</span>
                </label>
                <label class="block">
                  <span class="text-sm font-bold">排序</span>
                  <input v-model="materialForm.sort_order" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="1" type="number" min="1" />
                  <span class="mt-1 block text-xs text-slate-500">数字越小越靠前。</span>
                </label>
              </div>
              <button class="h-10 rounded-xl bg-[#0b4ea2] px-4 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingMaterial" type="submit">
                {{ savingMaterial ? "保存中..." : "保存资料" }}
              </button>
            </div>
            <div v-if="selectedMaterialRecord" class="mt-5 border-t border-slate-200 pt-4">
              <h4 class="font-black">资料完整字段</h4>
              <div class="mt-3 max-h-72 space-y-3 overflow-y-auto pr-1">
                <label v-for="entry in recordEntries(selectedMaterialRecord)" :key="`material-${entry.key}`" class="block">
                  <span class="text-xs font-black text-slate-500">{{ entry.key }}</span>
                  <textarea class="mt-1 min-h-10 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" disabled />
                </label>
              </div>
            </div>
          </form>
        </div>
      </section>

      <section class="grid gap-6 2xl:grid-cols-[390px_minmax(0,1fr)]" :class="!selectedCourseId ? 'opacity-50' : ''">
        <aside class="rounded-3xl border border-slate-200 bg-white shadow-sm">
          <div class="flex items-center justify-between border-b border-slate-200 p-5">
            <div>
              <h2 class="text-xl font-black">章节列表</h2>
              <p class="mt-1 text-sm text-slate-500">左侧选择章节，右侧只处理章节本身。</p>
            </div>
            <button class="rounded-xl border px-3 py-2 font-bold" :disabled="!selectedCourseId" type="button" @click="newChapter">新章节</button>
          </div>
          <div v-if="chaptersLoading" class="p-8 text-center text-slate-500">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            正在加载...
          </div>
          <div v-else-if="!chapters.length" class="p-8 text-center text-slate-500">暂无章节</div>
          <div v-else class="max-h-[520px] divide-y divide-slate-100 overflow-y-auto">
            <div v-for="chapter in chapters" :key="chapterId(chapter)" class="flex items-center justify-between gap-3 p-4" :class="chapterId(chapter) === selectedChapterId ? 'bg-sky-50' : ''">
              <button class="flex-1 text-left" type="button" @click="editChapter(chapter)">
                <div class="font-black">{{ chapterTitle(chapter) }}</div>
                <div class="mt-1 text-sm text-slate-500">排序 {{ chapter.sort_order || 0 }} · 版本 {{ chapter.version || 0 }}</div>
              </button>
              <button class="rounded-xl border border-red-200 px-3 py-2 text-sm font-bold text-red-600" type="button" @click="deleteChapter(chapter)">删除</button>
            </div>
          </div>
          <form class="border-t border-slate-200 p-5" @submit.prevent="saveChapter">
            <h3 class="font-black">{{ editingChapterId ? "编辑章节" : "创建章节" }}</h3>
            <input v-model="chapterForm.title" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="章节标题" />
            <input v-model="chapterForm.sort_order" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="排序" type="number" min="1" />
            <button class="mt-3 rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingChapter" type="submit">
              {{ savingChapter ? "保存中..." : "保存章节" }}
            </button>
          </form>
        </aside>

        <section class="rounded-3xl border border-slate-200 bg-white shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
            <div>
              <h2 class="text-xl font-black">章节详情</h2>
              <p class="mt-1 text-sm text-slate-500">{{ selectedChapterId ? chapterTitle(selectedChapter) : "请选择左侧章节，或点击新章节创建。" }}</p>
            </div>
          </div>
          <div class="grid gap-6 p-5 xl:grid-cols-[minmax(0,1fr)_360px]">
            <div class="rounded-2xl border border-slate-200 p-4">
              <h3 class="font-black">章节完整字段</h3>
              <p class="mt-1 text-xs text-slate-500">接口返回但不直接修改的字段置灰展示。</p>
              <div v-if="!selectedChapterId" class="p-8 text-center text-slate-500">暂无选中的章节</div>
              <div v-else class="mt-3 grid gap-3 md:grid-cols-2">
                <label v-for="entry in recordEntries(selectedChapter)" :key="`chapter-${entry.key}`" class="block">
                  <span class="text-xs font-black text-slate-500">{{ entry.key }}</span>
                  <textarea class="mt-1 min-h-10 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" disabled />
                </label>
              </div>
            </div>
            <form class="rounded-2xl border border-slate-200 p-4" @submit.prevent="saveChapter">
              <h3 class="font-black">{{ editingChapterId ? "编辑章节" : "创建章节" }}</h3>
              <input v-model="chapterForm.title" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="章节标题" />
              <input v-model="chapterForm.sort_order" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="排序" type="number" min="1" />
              <button class="mt-3 w-full rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingChapter" type="submit">
                {{ savingChapter ? "保存中..." : "保存章节" }}
              </button>
            </form>
          </div>
        </section>
      </section>

      <section class="grid gap-6 2xl:grid-cols-[390px_minmax(0,1fr)]" :class="!selectedCourseId ? 'opacity-50' : ''">
        <aside class="rounded-3xl border border-slate-200 bg-white shadow-sm">
          <div class="flex items-center justify-between border-b border-slate-200 p-5">
            <div>
              <h2 class="text-xl font-black">课时列表</h2>
              <p class="mt-1 text-sm text-slate-500">课时独立维护，详情里显示所属章节。</p>
            </div>
            <button class="rounded-xl border px-3 py-2 font-bold disabled:opacity-40" :disabled="!selectedChapterId" type="button" @click="newLesson">新课时</button>
          </div>
          <div v-if="completeLoading || lessonsLoading" class="p-8 text-center text-slate-500">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            正在加载...
          </div>
          <div v-else-if="!allLessonItems.length" class="p-8 text-center text-slate-500">暂无课时</div>
          <div v-else class="max-h-[520px] divide-y divide-slate-100 overflow-y-auto">
            <div v-for="item in allLessonItems" :key="lessonId(item.lesson)" class="grid gap-3 p-4 lg:grid-cols-[1fr_auto]" :class="lessonId(item.lesson) === editingLessonId ? 'bg-sky-50' : ''">
              <button class="text-left" type="button" @click="editLesson(item.lesson)">
                <div class="font-black">{{ lessonTitle(item.lesson) }}</div>
                <div class="mt-1 text-sm text-slate-500">属于章节：{{ chapterTitle(item.chapter) }}</div>
                <div class="mt-1 text-xs text-slate-500">排序 {{ item.lesson.sort_order || 0 }} · {{ lessonTypeLabel(item.lesson.lesson_type) }}</div>
              </button>
              <button class="rounded-xl border border-red-200 px-3 py-2 text-sm font-bold text-red-600" type="button" @click="deleteLesson(item.lesson)">删除</button>
            </div>
          </div>
        </aside>

        <section class="rounded-3xl border border-slate-200 bg-white shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
            <div>
              <h2 class="text-xl font-black">课时详情</h2>
              <p class="mt-1 text-sm text-slate-500">{{ editingLessonId ? lessonTitle(selectedLesson) : "选择课时编辑，或先选择章节后新建课时。" }}</p>
            </div>
          </div>
          <div class="grid gap-6 p-5 xl:grid-cols-[minmax(0,1fr)_420px]">
            <div class="space-y-6">
              <div class="grid gap-3">
                <div class="rounded-2xl bg-blue-50 p-4">
                  <div class="text-xs font-black text-blue-600">所属章节</div>
                  <div class="mt-1 text-lg font-black text-slate-900">{{ selectedLessonOwnerChapter ? chapterTitle(selectedLessonOwnerChapter) : (selectedChapterId ? chapterTitle(selectedChapter) : "未选择章节") }}</div>
                  <div class="mt-2 break-all font-mono text-sm font-bold text-blue-900">ID: {{ selectedLessonOwnerChapter ? chapterId(selectedLessonOwnerChapter) : (selectedChapterId || "-") }}</div>
                </div>
              </div>
              <div class="rounded-2xl border border-slate-200 p-4">
                <h3 class="font-black">课时完整字段</h3>
                <p class="mt-1 text-xs text-slate-500">接口返回但不直接修改的字段置灰展示。</p>
                <div v-if="!selectedLesson" class="p-8 text-center text-slate-500">暂无选中的课时</div>
                <div v-else class="mt-3 grid gap-3 md:grid-cols-2">
                  <label v-for="entry in recordEntries(selectedLesson)" :key="`lesson-${entry.key}`" class="block">
                    <span class="text-xs font-black text-slate-500">{{ entry.key }}</span>
                    <textarea class="mt-1 min-h-10 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" disabled />
                  </label>
                </div>
              </div>
            </div>
            <form class="rounded-2xl border border-slate-200 p-4" @submit.prevent="saveLesson">
              <h3 class="font-black">{{ editingLessonId ? "编辑课时" : "创建课时" }}</h3>
              <div class="mt-3 rounded-2xl border border-blue-100 bg-blue-50 p-3 text-sm text-blue-900">
                保存后属于章节：{{ lessonForm.chapter_id ? chapterTitle(chapterById(lessonForm.chapter_id)) : "请选择章节" }}
              </div>
              <label class="mt-3 block">
                <span class="text-sm font-bold">所属章节</span>
                <select v-model="lessonForm.chapter_id" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
                  <option value="">请选择章节</option>
                  <option v-for="chapter in chapters" :key="chapterId(chapter)" :value="chapterId(chapter)">{{ chapterTitle(chapter) }}</option>
                </select>
              </label>
              <input v-model="lessonForm.title" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="课时标题" />
              <div class="mt-3 grid gap-3 sm:grid-cols-2">
                <label class="block">
                  <span class="text-sm font-bold">排序</span>
                  <input v-model="lessonForm.sort_order" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="例如 1" type="number" min="1" />
                  <span class="mt-1 block text-xs text-slate-500">当前章节内的展示顺序，数字越小越靠前。</span>
                </label>
                <select v-model="lessonForm.lesson_type" class="rounded-xl border border-slate-200 px-4 py-3">
                  <option value="1">视频</option>
                  <option value="2">文本</option>
                  <option value="3">PDF</option>
                  <option value="4">图片</option>
                  <option value="5">音频</option>
                  <option value="6">文件</option>
                  <option value="7">链接</option>
                </select>
              </div>
              <textarea v-model="lessonForm.body" class="mt-3 min-h-24 w-full rounded-xl border border-slate-200 p-4" placeholder="正文 / 链接说明" />
              <input v-model="lessonForm.asset_object_key" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="资产 Object Key 或外部链接" />
              <label class="mt-3 block">
                <span class="text-sm font-bold">资产 File Hash</span>
                <span class="ml-2 cursor-help rounded-full border border-slate-300 px-2 py-0.5 text-xs text-slate-500" title="文件内容的 SHA256 Hash，一般由上传接口或资源管理返回；文本/外链课时通常可以留空。">?</span>
                <input v-model="lessonForm.asset_file_hash" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="上传接口返回的文件 Hash，可为空" />
              </label>
              <button class="mt-3 w-full rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedChapterId || savingLesson" type="submit">
                {{ savingLesson ? "保存中..." : "保存课时" }}
              </button>
            </form>
          </div>
        </section>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white shadow-sm" :class="!selectedCourseId ? 'opacity-50' : ''">
        <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
          <div>
            <h2 class="text-xl font-black">测验题库</h2>
            <p class="mt-1 text-sm text-slate-500">测验可挂在课程、章节或课时下；左侧列表选择测验，右侧详情展示归属和题目选项。</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <select v-model="quizForm.scope" class="rounded-xl border border-slate-200 px-4 py-2 font-bold" @change="newQuiz(quizForm.scope)">
              <option value="course">课程测验</option>
              <option value="chapter">章节测验</option>
              <option value="lesson">课时测验</option>
            </select>
            <button
              class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40"
              :disabled="!selectedCourseId || (quizForm.scope === 'chapter' && !selectedChapterId) || (quizForm.scope === 'lesson' && !editingLessonId)"
              type="button"
              @click="loadQuizzes()"
            >
              加载测验
            </button>
            <button
              class="rounded-xl bg-[#0b4ea2] px-4 py-2 font-bold text-white disabled:opacity-40"
              :disabled="!selectedCourseId || (quizForm.scope === 'chapter' && !selectedChapterId) || (quizForm.scope === 'lesson' && !editingLessonId)"
              type="button"
              @click="newQuiz(quizForm.scope)"
            >
              新测验
            </button>
          </div>
        </div>

        <div class="grid gap-6 p-5 2xl:grid-cols-[390px_minmax(0,1fr)]">
          <div class="rounded-2xl border border-slate-200">
            <div class="border-b border-slate-200 p-4">
              <h3 class="font-black">测验列表</h3>
              <p class="mt-1 text-xs text-slate-500">左侧只负责选择测验，右侧展示详情。</p>
            </div>
            <div v-if="quizzesLoading" class="p-6 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-5 w-5 animate-spin" />
              正在加载...
            </div>
            <div v-else-if="!allQuizItems.length" class="p-6 text-center text-slate-500">暂无测验</div>
            <div v-else class="max-h-96 divide-y divide-slate-100 overflow-y-auto">
              <div v-for="item in allQuizItems" :key="quizId(item.quiz)" class="flex items-center justify-between gap-3 p-4" :class="quizId(item.quiz) === selectedQuizId ? 'bg-sky-50' : ''">
                <button class="flex-1 text-left" type="button" @click="editQuiz(item.quiz)">
                  <div class="font-black">{{ quizTitle(item.quiz) }}</div>
                  <div class="mt-1 text-xs text-slate-500">通过分 {{ item.quiz.passing_score || 0 }} · 题目 {{ item.quiz.question_count || 0 }}</div>
                  <div class="mt-2 inline-flex max-w-full items-center gap-2 rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-700">
                    <span>{{ quizzableTypeLabel(item.ownerType) }}</span>
                    <span class="truncate">属于：{{ quizItemOwnerTitle(item) }}</span>
                  </div>
                  <div class="mt-1 break-all font-mono text-[11px] text-slate-400">归属 ID: {{ quizItemOwnerId(item) || "-" }}</div>
                </button>
                <button class="rounded-xl border border-red-200 px-3 py-2 text-xs font-bold text-red-600" type="button" @click="deleteQuiz(item.quiz)">删除</button>
              </div>
            </div>
          </div>

          <div class="space-y-6">
            <section class="rounded-2xl border border-slate-200">
              <div class="border-b border-slate-200 p-4">
                <h3 class="font-black">测验详情</h3>
                <p class="mt-1 text-xs text-slate-500">{{ selectedQuizId ? quizTitle(selectedQuiz) : "选择测验查看详情，或按当前归属创建测验。" }}</p>
              </div>
              <div class="grid gap-4 p-4 lg:grid-cols-3">
                <div class="rounded-2xl bg-blue-50 p-4">
                  <div class="text-xs font-black text-blue-600">属于</div>
                  <div class="mt-1 text-lg font-black text-slate-900">{{ selectedQuizItem ? quizzableTypeLabel(selectedQuizItem.ownerType) : `${quizTarget().label}级测验` }}</div>
                </div>
                <div class="rounded-2xl bg-slate-50 p-4">
                  <div class="text-xs font-black text-slate-500">所属章节</div>
                  <div class="mt-1 text-lg font-black text-slate-900">{{ selectedQuizItem?.chapter ? chapterTitle(selectedQuizItem.chapter) : (quizForm.scope === "chapter" ? chapterTitle(selectedChapter) : "-") }}</div>
                </div>
                <div class="rounded-2xl bg-slate-50 p-4">
                  <div class="text-xs font-black text-slate-500">所属课时</div>
                  <div class="mt-1 text-lg font-black text-slate-900">{{ selectedQuizItem?.lesson ? lessonTitle(selectedQuizItem.lesson) : (quizForm.scope === "lesson" ? lessonTitle(selectedLesson) : "-") }}</div>
                </div>
              </div>
            <form class="border-t border-slate-200 p-4" @submit.prevent="saveQuiz">
              <h4 class="font-black">{{ editingQuizId ? "编辑测验" : "创建测验" }}</h4>
              <div class="mt-3 rounded-2xl border border-blue-100 bg-blue-50 p-3 text-sm text-blue-900">
                保存后归属：{{ quizTarget().label }} · {{ quizTarget().id ? quizTarget().title : `未选择${quizTarget().label}` }}
              </div>
              <input v-model="quizForm.title" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="测验标题" />
              <textarea v-model="quizForm.description" class="mt-3 min-h-20 w-full rounded-xl border border-slate-200 p-4" placeholder="描述" />
              <div class="mt-3 grid gap-3 sm:grid-cols-2">
                <input v-model="quizForm.passing_score" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="通过分" type="number" />
                <input v-model="quizForm.time_limit" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="限时分钟，0 不限制" type="number" />
              </div>
              <label class="mt-3 inline-flex items-center gap-2 text-sm font-bold text-slate-600">
                <input v-model="quizForm.randomize_questions" type="checkbox" />
                随机题目
              </label>
              <button class="mt-3 w-full rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="savingQuiz" type="submit">
                {{ savingQuiz ? "保存中..." : "保存测验" }}
              </button>
            </form>
            </section>

          <section class="grid gap-6 2xl:grid-cols-[360px_minmax(0,1fr)]">
            <aside class="rounded-2xl border border-slate-200">
              <div class="flex items-center justify-between gap-3 border-b border-slate-200 p-4">
                <div>
                  <h3 class="font-black">题目列表</h3>
                  <p class="mt-1 text-xs text-slate-500">题目属于当前选中的测验。</p>
                </div>
                <button class="rounded-xl border px-3 py-2 text-xs font-bold" :disabled="!selectedQuizId" type="button" @click="newQuestion">新题目</button>
              </div>
              <div v-if="questionsLoading" class="p-6 text-center text-slate-500">
                <Loader2 class="mx-auto mb-2 h-5 w-5 animate-spin" />
                正在加载...
              </div>
              <div v-else-if="!questions.length" class="p-6 text-center text-slate-500">暂无题目</div>
              <div v-else class="max-h-96 divide-y divide-slate-100 overflow-y-auto">
                <div v-for="question in questions" :key="questionId(question)" class="flex items-center justify-between gap-3 p-4" :class="questionId(question) === selectedQuestionId ? 'bg-sky-50' : ''">
                  <button class="flex-1 text-left" type="button" @click="editQuestion(question)">
                    <div class="line-clamp-2 font-black">{{ questionTitle(question) }}</div>
                    <div class="mt-1 text-xs text-slate-500">{{ questionTypeLabel(question.question_type) }} · {{ question.points || 0 }} 分</div>
                  </button>
                  <button class="rounded-xl border border-red-200 px-3 py-2 text-xs font-bold text-red-600" type="button" @click="deleteQuestion(question)">删除</button>
                </div>
              </div>
            </aside>

            <div class="space-y-6">
              <section class="rounded-2xl border border-slate-200">
                <div class="border-b border-slate-200 p-4">
                  <h3 class="font-black">题目详情</h3>
                  <p class="mt-1 text-xs text-slate-500">{{ selectedQuestionId ? questionTitle(selectedQuestion) : "选择题目编辑，或点击新题目创建。" }}</p>
                </div>
                <div class="grid gap-3 p-4 sm:grid-cols-2">
                  <div class="rounded-2xl bg-blue-50 p-4">
                    <div class="text-xs font-black text-blue-600">所属测验</div>
                    <div class="mt-1 text-lg font-black text-slate-900">{{ selectedQuizId ? quizTitle(selectedQuiz) : "未选择测验" }}</div>
                    <div class="mt-2 break-all font-mono text-sm font-bold text-blue-900">ID: {{ selectedQuizId || "-" }}</div>
                  </div>
                  <div class="rounded-2xl bg-slate-50 p-4">
                    <div class="text-xs font-black text-slate-500">题目类型/分值</div>
                    <div class="mt-1 text-lg font-black text-slate-900">{{ selectedQuestion ? questionTypeLabel(selectedQuestion.question_type) : questionTypeLabel(questionForm.question_type) }} · {{ selectedQuestion?.points || questionForm.points || 0 }} 分</div>
                  </div>
                </div>
                <form class="border-t border-slate-200 p-4" @submit.prevent="saveQuestion">
                  <h4 class="font-black">{{ editingQuestionId ? "编辑题目" : "创建题目" }}</h4>
                  <textarea v-model="questionForm.question_text" class="mt-3 min-h-24 w-full rounded-xl border border-slate-200 p-4" placeholder="题干" />
                  <div class="mt-3 grid gap-3 sm:grid-cols-3">
                    <select v-model="questionForm.question_type" class="rounded-xl border border-slate-200 px-4 py-3">
                      <option value="1">单选</option>
                      <option value="2">多选</option>
                      <option value="3">判断</option>
                    </select>
                    <input v-model="questionForm.points" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="分值" type="number" />
                    <input v-model="questionForm.sort_order" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="排序" type="number" />
                  </div>
                  <label class="mt-3 inline-flex items-center gap-2 text-sm font-bold text-slate-600">
                    <input v-model="questionForm.is_required" type="checkbox" />
                    必答
                  </label>
                  <textarea v-model="questionForm.media_items_json" class="mt-3 min-h-20 w-full rounded-xl border border-slate-200 p-4 font-mono text-xs" placeholder="媒体 JSON，默认 []" />
                  <button class="mt-3 w-full rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedQuizId || savingQuestion" type="submit">
                    {{ savingQuestion ? "保存中..." : "保存题目" }}
                  </button>
                </form>
              </section>

              <section class="rounded-2xl border border-slate-200">
                <div class="border-b border-slate-200 p-4">
                  <h3 class="font-black">选项</h3>
                  <p class="mt-1 text-xs text-slate-500">{{ selectedQuestionId ? "选项属于当前题目。" : "请先选择题目。" }}</p>
                </div>
                <div v-if="optionsLoading" class="p-6 text-center text-slate-500">
                  <Loader2 class="mx-auto mb-2 h-5 w-5 animate-spin" />
                  正在加载...
                </div>
                <div v-else-if="!options.length" class="p-6 text-center text-slate-500">暂无选项</div>
                <div v-else class="max-h-72 divide-y divide-slate-100 overflow-y-auto">
                  <div v-for="option in options" :key="optionId(option)" class="flex items-center justify-between gap-3 p-4" :class="optionId(option) === editingOptionId ? 'bg-sky-50' : ''">
                    <button class="flex-1 text-left" type="button" @click="editOption(option)">
                      <div class="font-black">{{ optionTitle(option) }}</div>
                      <div class="mt-1 text-xs" :class="option.is_correct ? 'text-emerald-600' : 'text-slate-500'">
                        {{ option.is_correct ? "正确答案" : "普通选项" }} · 排序 {{ option.sort_order || 0 }}
                      </div>
                    </button>
                    <button class="rounded-xl border border-red-200 px-3 py-2 text-xs font-bold text-red-600" type="button" @click="deleteOption(option)">删除</button>
                  </div>
                </div>
                <form class="border-t border-slate-200 p-4" @submit.prevent="saveOption">
                  <div class="flex items-center justify-between gap-3">
                    <h4 class="font-black">{{ editingOptionId ? "编辑选项" : "创建选项" }}</h4>
                    <button class="rounded-xl border px-3 py-2 text-xs font-bold" :disabled="!selectedQuestionId" type="button" @click="newOption">新选项</button>
                  </div>
                  <input v-model="optionForm.option_text" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="选项内容" />
                  <div class="mt-3 grid gap-3 sm:grid-cols-2">
                    <input v-model="optionForm.sort_order" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="排序" type="number" />
                    <label class="inline-flex items-center gap-2 rounded-xl border border-slate-200 px-4 py-3 text-sm font-bold text-slate-600">
                      <input v-model="optionForm.is_correct" type="checkbox" />
                      正确答案
                    </label>
                  </div>
                  <button class="mt-3 w-full rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedQuestionId || savingOption" type="submit">
                    {{ savingOption ? "保存中..." : "保存选项" }}
                  </button>
                </form>
                <div v-if="selectedQuiz || selectedQuestion" class="border-t border-slate-200 p-4">
                  <h4 class="font-black">测验/题目只读字段</h4>
                  <div class="mt-3 max-h-72 space-y-3 overflow-y-auto pr-1">
                    <label v-for="entry in recordEntries(selectedQuiz)" :key="`quiz-${entry.key}`" class="block">
                      <span class="text-xs font-black text-slate-500">quiz.{{ entry.key }}</span>
                      <textarea class="mt-1 min-h-10 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" disabled />
                    </label>
                    <label v-for="entry in recordEntries(selectedQuestion)" :key="`question-${entry.key}`" class="block">
                      <span class="text-xs font-black text-slate-500">question.{{ entry.key }}</span>
                      <textarea class="mt-1 min-h-10 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" disabled />
                    </label>
                  </div>
                </div>
              </section>
            </div>
          </section>
          </div>
        </div>
      </section>
    </main>

    <div v-if="importOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-6" @click.self="importOpen = false">
      <div class="w-full max-w-3xl rounded-3xl bg-white p-6 shadow-2xl">
        <div class="mb-5 flex items-center justify-between">
          <h2 class="text-2xl font-black">导入 LMS JSON</h2>
          <button class="rounded-xl border px-3 py-2 font-bold" type="button" @click="importOpen = false">关闭</button>
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <label>
            <span class="text-sm font-bold">导入类型</span>
            <select v-model="importScope" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
              <option value="course">课程</option>
              <option value="quiz">章节测验</option>
            </select>
          </label>
          <label>
            <span class="text-sm font-bold">分类提示</span>
            <input v-model="importCategoryTips" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="导入课程时使用" />
          </label>
        </div>
        <label class="mt-4 block">
          <span class="text-sm font-bold">JSON 文件</span>
          <input class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" type="file" accept=".json,application/json" @change="loadImportFile" />
        </label>
        <textarea v-model="importJson" class="mt-4 min-h-80 w-full rounded-xl border border-slate-200 p-4 font-mono text-sm" placeholder="也可以直接粘贴 JSON" />
        <button class="mt-5 inline-flex items-center gap-2 rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="importing" type="button" @click="importLmsJson">
          <Loader2 v-if="importing" class="h-4 w-4 animate-spin" />
          <UploadCloud v-else class="h-4 w-4" />
          {{ importing ? "导入中..." : "开始导入" }}
        </button>
      </div>
    </div>
  </div>
</template>
