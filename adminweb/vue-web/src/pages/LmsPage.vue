<script setup lang="ts">
import { FileJson, Loader2, Plus, RefreshCw, Save, Trash2, UploadCloud, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
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
const { t } = useAdminLanguage()
const copy = computed(() => t.value.lmsAdmin)

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
const supplementaryItemDialogOpen = ref(false)
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
const isCreatingCourse = computed(() => courseView.value === "detail" && !selectedCourseId.value)
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
  return String(pickFirst(course || {}, ["title", "name", "course_title"]) || courseId(course) || copy.value.fallbacks.course)
}

function chapterTitle(chapter: JsonRecord | null | undefined) {
  return String(pickFirst(chapter || {}, ["title", "name"]) || chapterId(chapter) || copy.value.fallbacks.chapter)
}

function chapterById(id: string) {
  return chapters.value.find((item) => chapterId(item) === id) || null
}

function lessonTitle(lesson: JsonRecord | null | undefined) {
  return String(pickFirst(lesson || {}, ["title", "name"]) || lessonId(lesson) || copy.value.fallbacks.lesson)
}

function materialTitle(material: JsonRecord | null | undefined) {
  return String(pickFirst(material || {}, ["title", "name"]) || materialId(material) || copy.value.fallbacks.material)
}

function quizTitle(quiz: JsonRecord | null | undefined) {
  return String(pickFirst(quiz || {}, ["title", "name"]) || quizId(quiz) || copy.value.fallbacks.quiz)
}

function questionTitle(question: JsonRecord | null | undefined) {
  return String(pickFirst(question || {}, ["question_text", "title"]) || questionId(question) || copy.value.fallbacks.question)
}

function optionTitle(option: JsonRecord | null | undefined) {
  return String(pickFirst(option || {}, ["option_text", "title"]) || optionId(option) || copy.value.fallbacks.option)
}

function questionTypeLabel(value: unknown) {
  const type = Number(value || 0)
  if (type === 1) return copy.value.questionTypes.single
  if (type === 2) return copy.value.questionTypes.multiple
  if (type === 3) return copy.value.questionTypes.judgement
  return copy.value.unknown
}

function lessonTypeLabel(value: unknown) {
  const type = Number(value || 0)
  if (type === 1) return copy.value.lessonTypes.video
  if (type === 2) return copy.value.lessonTypes.text
  if (type === 3) return copy.value.lessonTypes.pdf
  if (type === 4) return copy.value.lessonTypes.image
  if (type === 5) return copy.value.lessonTypes.audio
  if (type === 6) return copy.value.lessonTypes.file
  if (type === 7) return copy.value.lessonTypes.link
  return copy.value.unspecified
}

function materialTypeLabel(value: unknown) {
  const type = Number(value || 0)
  if (type === 1) return copy.value.materialTypes.textbook
  if (type === 2) return copy.value.materialTypes.slides
  if (type === 3) return copy.value.materialTypes.reference
  if (type === 4) return copy.value.materialTypes.other
  return copy.value.unspecified
}

function supplementaryTypeLabel(type: string) {
  const normalized = type.trim().toLowerCase()
  if (normalized === "article") return copy.value.supplementaryTypes.article
  if (normalized === "video") return copy.value.supplementaryTypes.video
  if (normalized === "pdf") return copy.value.supplementaryTypes.pdf
  if (normalized === "link") return copy.value.supplementaryTypes.link
  return type || copy.value.supplementaryTypes.material
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
  if (type === 1) return copy.value.quizScopes.lesson
  if (type === 2) return copy.value.quizScopes.chapter
  if (type === 3) return copy.value.quizScopes.course
  return copy.value.quizScopes.unknown
}

function displayValue(value: unknown) {
  if (value === null || value === undefined || value === "") return "-"
  if (typeof value === "boolean") return value ? copy.value.yes : copy.value.no
  if (typeof value === "object") return JSON.stringify(value, null, 2)
  return String(value)
}

function recordEntries(record: JsonRecord | null | undefined) {
  if (!record) return []
  return Object.entries(record).map(([key, value]) => ({ key, value: displayValue(value) }))
}

function courseReadonlyFieldLabel(key: string) {
  const labels: Record<string, string> = copy.value.readonlyCourseFieldLabels
  return labels[key] || key
}

function courseDetailReadonlyFieldLabel(key: string) {
  return `${copy.value.detailReadonlyPrefix}${courseReadonlyFieldLabel(key)}`
}

function completeCourseReadonlyFieldLabel(key: string) {
  return `${copy.value.completeReadonlyPrefix}${courseReadonlyFieldLabel(key)}`
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
    toast.error(copy.value.toasts.courseListLoadFailed)
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
    toast.error(copy.value.toasts.courseTitleRequired)
    return
  }
  if (!courseForm.value.respath.trim()) {
    toast.error(copy.value.toasts.respathRequired)
    return
  }

  savingCourse.value = true
  try {
    if (selectedCourseId.value) {
      await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}`, {
        method: "PUT",
        body: JSON.stringify(coursePayload(selectedCourse.value?.version)),
      })
      toast.success(copy.value.toasts.courseUpdated)
    } else {
      const created = await apiClient<JsonRecord>("/api/lms/courses", {
        method: "POST",
        body: JSON.stringify(coursePayload()),
      })
      selectedCourse.value = created
      toast.success(copy.value.toasts.courseCreated)
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
    toast.error(copy.value.toasts.courseSaveFailed)
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
    toast.success(copy.value.toasts.coursePublished)
    await loadCourses()
    const refreshed = courses.value.find((item) => courseId(item) === selectedCourseId.value)
    if (refreshed) selectedCourse.value = refreshed
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.coursePublishFailed)
  } finally {
    publishing.value = false
  }
}

async function deleteCourse() {
  if (!selectedCourseId.value || !window.confirm(copy.value.confirmDeleteCourse(courseTitle(selectedCourse.value)))) return
  try {
    await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}?version=${versionOf(selectedCourse.value)}`, { method: "DELETE" })
    toast.success(copy.value.toasts.courseDeleted)
    newCourse()
    courseView.value = "list"
    await loadCourses()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.courseDeleteFailed)
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
    toast.error(copy.value.toasts.chaptersLoadFailed)
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
    toast.error(copy.value.toasts.chapterRequired)
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
      toast.success(copy.value.toasts.chapterUpdated)
    } else {
      await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/chapters`, { method: "POST", body })
      toast.success(copy.value.toasts.chapterCreated)
    }
    newChapter()
    await loadChapters()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.chapterSaveFailed)
  } finally {
    savingChapter.value = false
  }
}

async function deleteChapter(chapter: JsonRecord) {
  const id = chapterId(chapter)
  if (!id || !window.confirm(copy.value.confirmDeleteChapter(chapterTitle(chapter)))) return
  try {
    await apiClient(`/api/lms/chapters/${encodeURIComponent(id)}?version=${versionOf(chapter)}`, { method: "DELETE" })
    toast.success(copy.value.toasts.chapterDeleted)
    newChapter()
    await loadChapters()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.chapterDeleteFailed)
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
    toast.error(copy.value.toasts.lessonsLoadFailed)
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
    toast.error(copy.value.toasts.lessonRequired)
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
      toast.success(copy.value.toasts.lessonUpdated)
    } else {
      await apiClient(`/api/lms/chapters/${encodeURIComponent(targetChapterId)}/lessons`, { method: "POST", body })
      toast.success(copy.value.toasts.lessonCreated)
    }
    newLesson()
    await Promise.all([loadLessons(), loadCompleteCourse(), loadCourseDetail()])
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.lessonSaveFailed)
  } finally {
    savingLesson.value = false
  }
}

async function deleteLesson(lesson: JsonRecord) {
  const id = lessonId(lesson)
  if (!id || !window.confirm(copy.value.confirmDeleteLesson(lessonTitle(lesson)))) return
  try {
    await apiClient(`/api/lms/lessons/${encodeURIComponent(id)}?version=${versionOf(lesson)}`, { method: "DELETE" })
    toast.success(copy.value.toasts.lessonDeleted)
    newLesson()
    await Promise.all([loadLessons(), loadCompleteCourse(), loadCourseDetail()])
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.lessonDeleteFailed)
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
    toast.error(copy.value.toasts.materialsLoadFailed)
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
    if (firstItem) editSupplementaryItem(firstItem, false)
    else newSupplementaryItem(false)
  } catch (err) {
    console.error(err)
    supplementaryMaterial.value = null
    supplementaryMaterialForm.value = emptySupplementaryMaterialForm()
    newSupplementaryItem(false)
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

function editSupplementaryItem(item: SupplementaryMaterialItem, openDialog = true) {
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
  supplementaryItemDialogOpen.value = openDialog
}

function newSupplementaryItem(openDialog = true) {
  editingSupplementaryItemIndex.value = -1
  supplementaryItemForm.value = emptySupplementaryItemForm()
  supplementaryItemDialogOpen.value = openDialog
}

function closeSupplementaryItemDialog() {
  supplementaryItemDialogOpen.value = false
}

async function saveSupplementaryItem() {
  if (!selectedCourseId.value) return
  if (!supplementaryItemForm.value.title.trim()) {
    toast.error(copy.value.toasts.materialTitleRequired)
    return
  }
  if (!supplementaryItemForm.value.url.trim()) {
    toast.error(copy.value.toasts.materialUrlRequired)
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
  supplementaryItemDialogOpen.value = false
}

async function deleteSupplementaryItem() {
  if (editingSupplementaryItemIndex.value < 0) return
  if (!window.confirm(copy.value.confirmDeleteMaterial(supplementaryItemForm.value.title || ""))) return
  const records = supplementaryEditableRecords()
  records.splice(editingSupplementaryItemIndex.value, 1)
  updateSupplementaryDataJson(records)
  newSupplementaryItem(false)
  await saveSupplementaryMaterial()
  supplementaryItemDialogOpen.value = false
}

async function saveSupplementaryMaterial() {
  if (!selectedCourseId.value) return
  if (!supplementaryMaterialForm.value.kind.trim()) {
    toast.error(copy.value.toasts.supplementaryKindRequired)
    return
  }
  try {
    JSON.parse(supplementaryMaterialForm.value.data_json)
  } catch {
    toast.error(copy.value.toasts.supplementaryInvalidJson)
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
      toast.success(copy.value.toasts.supplementaryUpdated)
    } else {
      await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/supplementary-material`, { method: "POST", body: JSON.stringify(body) })
      toast.success(copy.value.toasts.supplementaryCreated)
    }
    await Promise.all([loadSupplementaryMaterial(), loadCompleteCourse(), loadCourseDetail()])
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.supplementarySaveFailed)
  } finally {
    savingSupplementaryMaterial.value = false
  }
}

async function deleteSupplementaryMaterial() {
  const id = supplementaryMaterialId(supplementaryMaterial.value)
  if (!selectedCourseId.value || !id || !window.confirm(copy.value.confirmDeleteSupplementaryConfig)) return
  try {
    await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/supplementary-material/${encodeURIComponent(id)}?version=${versionOf(supplementaryMaterial.value)}`, { method: "DELETE" })
    toast.success(copy.value.toasts.supplementaryDeleted)
    await Promise.all([loadSupplementaryMaterial(), loadCompleteCourse(), loadCourseDetail()])
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.supplementaryDeleteFailed)
  }
}

async function saveMaterial() {
  if (!selectedCourseId.value || !materialForm.value.title.trim() || !materialForm.value.file_object_key.trim()) {
    toast.error(copy.value.toasts.materialFileRequired)
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
      toast.success(copy.value.toasts.materialUpdated)
    } else {
      await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/materials`, { method: "POST", body: JSON.stringify(body) })
      toast.success(copy.value.toasts.materialCreated)
    }
    newMaterial()
    await Promise.all([loadMaterials(), loadCourseDetail(), loadCompleteCourse()])
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.materialSaveFailed)
  } finally {
    savingMaterial.value = false
  }
}

async function deleteMaterial(material: JsonRecord) {
  const id = materialId(material)
  if (!id || !window.confirm(copy.value.confirmDeleteMaterial(materialTitle(material)))) return
  try {
    await apiClient(`/api/lms/materials/${encodeURIComponent(id)}?version=${versionOf(material)}`, { method: "DELETE" })
    toast.success(copy.value.toasts.materialDeleted)
    newMaterial()
    await Promise.all([loadMaterials(), loadCourseDetail(), loadCompleteCourse()])
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.materialDeleteFailed)
  }
}

function quizTarget(scope: QuizScope = quizForm.value.scope) {
  if (scope === "course") {
    return { type: 3, id: selectedCourseId.value, label: copy.value.fallbacks.course, title: courseTitle(selectedCourse.value) }
  }
  if (scope === "chapter") {
    return { type: 2, id: selectedChapterId.value, label: copy.value.fallbacks.chapter, title: chapterTitle(selectedChapter.value) }
  }
  return { type: 1, id: editingLessonId.value, label: copy.value.fallbacks.lesson, title: lessonTitle(selectedLesson.value) }
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
    toast.error(copy.value.toasts.selectTargetFirst(target.label))
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
    toast.error(copy.value.toasts.quizzesLoadFailed)
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
    toast.error(copy.value.toasts.quizTitleRequired)
    return
  }
  const target = quizTarget()
  if (!target.id) {
    toast.error(copy.value.toasts.selectTargetFirst(target.label))
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
      toast.success(copy.value.toasts.quizUpdated)
    } else {
      await apiClient("/api/lms/quizzes", { method: "POST", body: JSON.stringify(body) })
      toast.success(copy.value.toasts.quizCreated)
    }
    const scope = quizForm.value.scope
    newQuiz(scope)
    await Promise.all([loadQuizzes(scope), loadCourseDetail(), loadCompleteCourse()])
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.quizSaveFailed)
  } finally {
    savingQuiz.value = false
  }
}

async function deleteQuiz(quiz: JsonRecord) {
  const id = quizId(quiz)
  if (!id || !window.confirm(copy.value.confirmDeleteQuiz(quizTitle(quiz)))) return
  try {
    await apiClient(`/api/lms/quizzes/${encodeURIComponent(id)}?version=${versionOf(quiz)}`, { method: "DELETE" })
    toast.success(copy.value.toasts.quizDeleted)
    const scope = quizForm.value.scope
    newQuiz(scope)
    await Promise.all([loadQuizzes(scope), loadCourseDetail(), loadCompleteCourse()])
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.quizDeleteFailed)
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
    toast.error(copy.value.toasts.questionsLoadFailed)
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
    toast.error(copy.value.toasts.questionRequired)
    return
  }
  const mediaJson = questionForm.value.media_items_json.trim() || "[]"
  try {
    JSON.parse(mediaJson)
  } catch {
    toast.error(copy.value.toasts.mediaInvalidJson)
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
      toast.success(copy.value.toasts.questionUpdated)
    } else {
      await apiClient(`/api/lms/quizzes/${encodeURIComponent(selectedQuizId.value)}/questions`, { method: "POST", body: JSON.stringify(body) })
      toast.success(copy.value.toasts.questionCreated)
    }
    newQuestion()
    await loadQuestions()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.questionSaveFailed)
  } finally {
    savingQuestion.value = false
  }
}

async function deleteQuestion(question: JsonRecord) {
  const id = questionId(question)
  if (!id || !window.confirm(copy.value.confirmDeleteQuestion(questionTitle(question)))) return
  try {
    await apiClient(`/api/lms/questions/${encodeURIComponent(id)}?version=${versionOf(question)}`, { method: "DELETE" })
    toast.success(copy.value.toasts.questionDeleted)
    newQuestion()
    await loadQuestions()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.questionDeleteFailed)
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
    toast.error(copy.value.toasts.optionsLoadFailed)
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
    toast.error(copy.value.toasts.optionRequired)
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
      toast.success(copy.value.toasts.optionUpdated)
    } else {
      await apiClient(`/api/lms/questions/${encodeURIComponent(selectedQuestionId.value)}/options`, { method: "POST", body: JSON.stringify(body) })
      toast.success(copy.value.toasts.optionCreated)
    }
    newOption()
    await loadOptions()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.optionSaveFailed)
  } finally {
    savingOption.value = false
  }
}

async function deleteOption(option: JsonRecord) {
  const id = optionId(option)
  if (!id || !window.confirm(copy.value.confirmDeleteOption(optionTitle(option)))) return
  try {
    await apiClient(`/api/lms/options/${encodeURIComponent(id)}?version=${versionOf(option)}`, { method: "DELETE" })
    toast.success(copy.value.toasts.optionDeleted)
    newOption()
    await loadOptions()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.optionDeleteFailed)
  }
}

async function importLmsJson() {
  if (!importJson.value.trim()) {
    toast.error(copy.value.toasts.importJsonRequired)
    return
  }
  try {
    JSON.parse(importJson.value)
  } catch {
    toast.error(copy.value.toasts.importInvalidJson)
    return
  }
  if (importScope.value === "quiz" && !selectedChapterId.value) {
    toast.error(copy.value.toasts.importQuizNeedsChapter)
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
    toast.success(copy.value.toasts.imported)
    importOpen.value = false
    importJson.value = ""
    await loadCourses()
    if (selectedCourseId.value) await loadChapters()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.importFailed)
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
        <h1 class="text-4xl font-black tracking-tight">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button v-if="!isCreatingCourse" class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 font-bold shadow-sm disabled:opacity-60" type="button" :disabled="loading" @click="loadCourses()">
          <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
        <button v-if="!isCreatingCourse" class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 font-bold shadow-sm" type="button" @click="importOpen = true">
          <FileJson class="h-4 w-4" />
          {{ copy.importJson }}
        </button>
        <button v-if="courseView === 'detail'" class="rounded-xl border bg-white px-4 py-3 font-bold shadow-sm" type="button" @click="backToCourseList">
          {{ copy.backToList }}
        </button>
        <button v-if="!isCreatingCourse" class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-4 py-3 font-bold text-white shadow-lg shadow-sky-200" type="button" @click="newCourse">
          <Plus class="h-4 w-4" />
          {{ copy.newCourse }}
        </button>
      </div>
    </header>

    <section v-if="courseView === 'list'" class="rounded-3xl border border-slate-200 bg-white shadow-sm">
      <div class="grid gap-3 border-b border-slate-200 bg-slate-50/60 p-4 lg:grid-cols-[1fr_auto]">
        <input v-model="categoryFilter" class="h-10 rounded-xl border border-slate-200 bg-white px-4 text-sm shadow-sm outline-none transition focus:border-sky-300 focus:ring-2 focus:ring-sky-100" :placeholder="copy.categoryPlaceholder" />
        <label class="inline-flex h-10 items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 text-sm font-bold text-slate-600 shadow-sm">
          <input v-model="publishedOnly" type="checkbox" />
          {{ copy.publishedOnly }}
        </label>
      </div>

      <div v-if="loading && !courses.length" class="p-12 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        {{ copy.loading }}
      </div>
      <div v-else-if="!courses.length" class="p-12 text-center text-slate-500">{{ copy.emptyCourses }}</div>
      <div v-else>
        <div class="hidden grid-cols-[minmax(0,1fr)_120px_260px_120px] gap-6 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-400 lg:grid">
          <span>{{ copy.columns.course }}</span>
          <span>{{ copy.columns.version }}</span>
          <span>{{ copy.columns.updatedAt }}</span>
          <span class="text-right">{{ copy.columns.status }}</span>
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
                <span>{{ course.category_tips || copy.uncategorized }}</span>
                <span class="font-mono">ID: {{ courseId(course) || "-" }}</span>
              </div>
            </div>
            <div class="text-sm font-bold text-slate-700">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.columns.version }}</span>{{ course.version || 0 }}
            </div>
            <div class="text-sm text-slate-500">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.updatedShort }}</span>{{ formatDate(String(course.updated_at || course.created_at || "")) }}
            </div>
            <span class="justify-self-start rounded-full border px-3 py-1 text-xs font-black lg:justify-self-end" :class="badgeClass(course.is_published ? 'COMPLETED' : 'PENDING')">
              {{ course.is_published ? copy.published : copy.draft }}
            </span>
          </div>
        </button>
      </div>
      <div v-if="nextPageToken" class="border-t border-slate-200 p-4">
        <button class="w-full rounded-xl border px-4 py-3 font-bold transition hover:bg-slate-50 disabled:cursor-default disabled:border-slate-200 disabled:bg-slate-50 disabled:text-slate-400 disabled:opacity-100" type="button" :disabled="!nextPageToken || loading" @click="loadCourses(nextPageToken)">
          {{ copy.loadMore }}
        </button>
      </div>
    </section>

    <main v-else class="space-y-6">
      <section class="rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">{{ selectedCourseId ? copy.courseTopData : copy.newCourse }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ selectedCourseId || copy.fillCourseHint }}</p>
          </div>
          <span v-if="selectedCourseId" class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(selectedCourseStatus)">
            {{ selectedCoursePublished ? copy.published : selectedCourseStatus }}
          </span>
        </div>

        <div class="grid gap-4 p-5 2xl:grid-cols-[minmax(0,1fr)_420px]">
          <form class="grid gap-3 lg:grid-cols-2" @submit.prevent="saveCourse">
            <label class="block">
              <span class="text-sm font-bold">{{ copy.courseTitle }}</span>
              <input v-model="courseForm.title" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.categoryTips }}</span>
              <input v-model="courseForm.category_tips" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <label class="block lg:col-span-2">
              <span class="text-sm font-bold">{{ copy.description }}</span>
              <textarea v-model="courseForm.description" class="mt-2 min-h-20 w-full rounded-xl border border-slate-200 px-3 py-2" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.respath }}</span>
              <input v-model="courseForm.respath" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="/gcc/pipeline/..." />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.durationMin }}</span>
              <input v-model="courseForm.duration_min" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" type="number" min="0" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.thumbnailObjectKey }}</span>
              <input v-model="courseForm.thumbnail_object_key" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.thumbnailFileHash }}</span>
              <input v-model="courseForm.thumbnail_file_hash" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <label class="inline-flex items-center gap-2 text-sm font-bold text-slate-600">
              <input v-model="courseForm.certification_enabled" type="checkbox" />
              {{ copy.enableCertificate }}
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.certificateDefinitionId }}</span>
              <input v-model="courseForm.certification_def_id" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <div class="flex flex-wrap gap-3 lg:col-span-2">
              <button class="inline-flex h-10 items-center gap-2 rounded-xl bg-[#0b4ea2] px-4 font-bold text-white disabled:opacity-50" :disabled="savingCourse" type="submit">
                <Loader2 v-if="savingCourse" class="h-4 w-4 animate-spin" />
                <Save v-else class="h-4 w-4" />
                {{ savingCourse ? copy.saving : copy.saveCourse }}
              </button>
              <button class="h-10 rounded-xl border px-4 font-bold disabled:opacity-40" :disabled="!selectedCourseId || publishing" type="button" @click="publishCourse">
                {{ publishing ? copy.publishing : copy.publishCourse }}
              </button>
              <button class="inline-flex h-10 items-center gap-2 rounded-xl border border-red-200 px-4 font-bold text-red-600 disabled:opacity-40" :disabled="!selectedCourseId" type="button" @click="deleteCourse">
                <Trash2 class="h-4 w-4" />
                {{ copy.deleteCourse }}
              </button>
            </div>
          </form>

          <aside class="space-y-3">
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="rounded-xl bg-slate-50 p-3">
                <div class="text-xs font-black uppercase text-slate-400">{{ copy.stats.chapters }}</div>
                <div class="mt-1 text-xl font-black">{{ courseDetail?.chapter_count ?? chapters.length }}</div>
              </div>
              <div class="rounded-xl bg-slate-50 p-3">
                <div class="text-xs font-black uppercase text-slate-400">{{ copy.stats.lessons }}</div>
                <div class="mt-1 text-xl font-black">{{ courseDetail?.lesson_count ?? lessons.length }}</div>
              </div>
              <div class="rounded-xl bg-slate-50 p-3">
                <div class="text-xs font-black uppercase text-slate-400">{{ copy.stats.quizzes }}</div>
                <div class="mt-1 text-xl font-black">{{ courseDetail?.quiz_count ?? 0 }}</div>
              </div>
              <div class="rounded-xl bg-slate-50 p-3">
                <div class="text-xs font-black uppercase text-slate-400">{{ copy.stats.materials }}</div>
                <div class="mt-1 text-xl font-black">{{ courseDetail?.material_count ?? materials.length }}</div>
              </div>
            </div>
            <div class="rounded-xl border border-slate-200 p-3">
              <h3 class="font-black">{{ copy.readonlyFields }}</h3>
              <p class="mt-1 text-xs text-slate-500">{{ copy.readonlyFieldsHint }}</p>
              <div class="mt-3 max-h-[420px] space-y-2 overflow-y-auto overscroll-contain pr-2">
                <label v-for="entry in recordEntries(selectedCourse)" :key="`course-${entry.key}`" class="block">
                  <span class="text-xs font-black text-slate-500">{{ courseReadonlyFieldLabel(entry.key) }}</span>
                  <textarea class="mt-1 min-h-12 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" readonly />
                </label>
                <div v-if="detailLoading || completeLoading" class="text-sm text-slate-500">{{ copy.loadingCompleteCourse }}</div>
                <label v-for="entry in recordEntries(courseDetail)" :key="`detail-${entry.key}`" class="block">
                  <span class="text-xs font-black text-slate-500">{{ courseDetailReadonlyFieldLabel(entry.key) }}</span>
                  <textarea class="mt-1 min-h-12 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" readonly />
                </label>
                <label v-for="entry in recordEntries(completeCourse)" :key="`complete-${entry.key}`" class="block">
                  <span class="text-xs font-black text-slate-500">{{ completeCourseReadonlyFieldLabel(entry.key) }}</span>
                  <textarea class="mt-1 min-h-12 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" readonly />
                </label>
              </div>
            </div>
          </aside>
        </div>
      </section>

      <section class="rounded-2xl border border-slate-200 bg-white shadow-sm" :class="!selectedCourseId ? 'opacity-50' : ''">
        <div class="border-b border-slate-200 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">{{ copy.materialsTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.materialsDescription }}</p>
          </div>
        </div>

        <div class="space-y-4 p-5">
          <div class="rounded-2xl border border-slate-200">
            <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 p-4">
              <div>
                <h3 class="font-black">{{ copy.supplementaryTitle }}</h3>
                <p class="mt-1 text-sm text-slate-500">{{ copy.supplementaryDescription }}</p>
              </div>
              <div class="flex items-center gap-2">
                <span class="rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-xs font-black text-slate-600">{{ copy.countText(supplementaryMaterialItems.length) }}</span>
                <button class="h-10 rounded-xl border px-4 text-sm font-bold disabled:opacity-40" :disabled="!selectedCourseId || supplementaryMaterialLoading" type="button" @click="loadSupplementaryMaterial">
                  {{ copy.refresh }}
                </button>
                <button class="h-10 rounded-xl bg-[#0b4ea2] px-4 text-sm font-bold text-white shadow-sm disabled:opacity-40" :disabled="!selectedCourseId" type="button" @click="newSupplementaryItem()">
                  {{ copy.add }}
                </button>
              </div>
            </div>
            <div v-if="supplementaryMaterialLoading" class="px-6 py-10 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              {{ copy.loading }}
            </div>
            <div v-else-if="!supplementaryMaterialItems.length" class="px-6 py-10 text-center text-slate-500">
              {{ copy.noSupplementaryMaterials }}
              <div class="mt-2 text-xs">{{ copy.supplementaryEmptyHint }}</div>
            </div>
            <div v-else class="overflow-x-auto">
              <table class="min-w-full text-left text-sm">
                <thead class="bg-slate-50 text-xs font-black uppercase tracking-wide text-slate-500">
                  <tr>
                    <th class="px-4 py-3">{{ copy.supplementaryColumns.chapter }}</th>
                    <th class="w-24 whitespace-nowrap px-4 py-3 text-center">{{ copy.supplementaryColumns.type }}</th>
                    <th class="px-4 py-3">{{ copy.supplementaryColumns.titleDescription }}</th>
                    <th class="px-4 py-3">{{ copy.supplementaryColumns.resourceLink }}</th>
                    <th class="w-24 px-4 py-3 text-right">{{ copy.columns.action }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-100">
                  <tr
                    v-for="item in supplementaryMaterialItems"
                    :key="item.key"
                    class="transition hover:bg-sky-50"
                    :class="item.recordIndex === editingSupplementaryItemIndex ? 'bg-sky-50' : ''"
                  >
                    <td class="whitespace-nowrap px-4 py-4 font-semibold text-slate-800">
                      <span class="mr-2 rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-500">#{{ item.recordIndex + 1 }}</span>
                      {{ item.chapter }}
                    </td>
                    <td class="w-24 whitespace-nowrap px-4 py-4 text-center">
                      <span class="inline-flex whitespace-nowrap rounded-full border px-2 py-1 text-xs font-black" :class="supplementaryTypeClass(item.type)">{{ supplementaryTypeLabel(item.type) }}</span>
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
                    <td class="w-24 whitespace-nowrap px-4 py-4 text-right">
                      <button class="whitespace-nowrap rounded-xl border border-slate-200 bg-white px-3 py-2 text-xs font-bold text-[#0b4ea2] shadow-sm hover:bg-sky-50" type="button" @click.stop="editSupplementaryItem(item)">
                        {{ copy.edit }}
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <Teleport to="body">
            <div v-if="supplementaryItemDialogOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
              <form class="flex max-h-[88vh] w-full max-w-[720px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl" @submit.prevent="saveSupplementaryItem">
                <div class="flex items-start justify-between gap-3 border-b border-slate-200 px-6 py-5">
                  <div>
                    <h3 class="font-black">{{ editingSupplementaryItemIndex >= 0 ? copy.editSupplementaryItem : copy.newSupplementaryItem }}</h3>
                    <p class="mt-1 text-xs text-slate-500">{{ copy.supplementaryItemDescription }}</p>
                  </div>
                  <button
                    class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
                    type="button"
                    :aria-label="copy.close"
                    @click="closeSupplementaryItemDialog"
                  >
                    <X class="h-5 w-5" />
                  </button>
                </div>
                <div class="min-h-0 flex-1 overflow-y-auto px-6 py-5">
                  <div class="grid gap-3">
                    <label class="block">
                      <span class="text-sm font-bold">{{ copy.ownerChapter }}</span>
                      <input v-model="supplementaryItemForm.chapter" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.chapterPlaceholder" />
                    </label>
                    <div class="grid gap-3 sm:grid-cols-2">
                      <label class="block">
                        <span class="text-sm font-bold">{{ copy.materialType }}</span>
                        <select v-model="supplementaryItemForm.type" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3">
                          <option value="Article">{{ copy.supplementaryTypes.article }}</option>
                          <option value="Video">{{ copy.supplementaryTypes.video }}</option>
                          <option value="PDF">{{ copy.supplementaryTypes.pdf }}</option>
                          <option value="Link">{{ copy.supplementaryTypes.link }}</option>
                          <option value="Other">{{ copy.materialTypes.other }}</option>
                        </select>
                      </label>
                      <label class="block">
                        <span class="text-sm font-bold">{{ copy.durationNote }}</span>
                        <input v-model="supplementaryItemForm.duration" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.durationPlaceholder" />
                      </label>
                    </div>
                    <label class="block">
                      <span class="text-sm font-bold">{{ copy.titleField }}</span>
                      <input v-model="supplementaryItemForm.title" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.materialTitlePlaceholder" />
                    </label>
                    <label class="block">
                      <span class="text-sm font-bold">{{ copy.description }}</span>
                      <textarea v-model="supplementaryItemForm.description" class="mt-2 min-h-20 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.descriptionPlaceholder" />
                    </label>
                    <label class="block">
                      <span class="text-sm font-bold">{{ copy.resourceLink }}</span>
                      <input v-model="supplementaryItemForm.url" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="https://..." />
                    </label>
                  </div>
                </div>
                <div class="flex justify-end gap-3 border-t border-slate-200 px-6 py-4">
                  <button class="rounded-xl border px-4 py-2 font-bold" type="button" @click="closeSupplementaryItemDialog">
                    {{ copy.cancel }}
                  </button>
                  <button class="rounded-xl border border-red-200 px-4 py-2 font-bold text-red-600 disabled:opacity-40" :disabled="editingSupplementaryItemIndex < 0" type="button" @click="deleteSupplementaryItem">
                    {{ copy.deleteThisItem }}
                  </button>
                  <button class="rounded-xl bg-[#0b4ea2] px-4 py-2 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingSupplementaryMaterial" type="submit">
                    {{ savingSupplementaryMaterial ? copy.saving : copy.saveThisMaterial }}
                  </button>
                </div>
              </form>
            </div>
          </Teleport>

          <div class="rounded-2xl border border-slate-200 p-4">
            <details>
              <summary class="cursor-pointer font-black">{{ copy.supplementaryAdvancedTitle }}</summary>
              <div class="mt-4 grid gap-3">
                <label class="block">
                  <span class="text-sm font-bold">{{ copy.kind }}</span>
                  <input v-model="supplementaryMaterialForm.kind" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="supplementary_materials" />
                  <span class="mt-1 block text-xs text-slate-500">{{ copy.kindHint }}</span>
                </label>
                <label class="block">
                  <span class="text-sm font-bold">{{ copy.dataJson }}</span>
                  <textarea v-model="supplementaryMaterialForm.data_json" class="mt-2 min-h-64 w-full rounded-xl border border-slate-200 p-4 font-mono text-xs leading-5" :placeholder="copy.supplementaryJsonPlaceholder" />
                  <span class="mt-1 block text-xs text-slate-500">{{ copy.supplementaryAdvancedHint }}</span>
                </label>
                <div class="grid gap-2 sm:grid-cols-2">
                  <button class="rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingSupplementaryMaterial" type="button" @click="saveSupplementaryMaterial">
                    {{ savingSupplementaryMaterial ? copy.saving : copy.saveWholeJson }}
                  </button>
                  <button class="rounded-xl border border-red-200 px-5 py-3 font-bold text-red-600 disabled:opacity-40" :disabled="!supplementaryMaterialId(supplementaryMaterial)" type="button" @click="deleteSupplementaryMaterial">
                    {{ copy.deleteWholeConfig }}
                  </button>
                </div>
              </div>
            </details>

            <div v-if="supplementaryMaterial" class="mt-5 border-t border-slate-200 pt-4">
              <h4 class="font-black">{{ copy.supplementaryRawFields }}</h4>
              <div class="mt-3 max-h-48 space-y-3 overflow-y-auto pr-1">
                <label v-for="entry in recordEntries(supplementaryMaterial)" :key="`supplementary-${entry.key}`" class="block">
                  <span class="text-xs font-black text-slate-500">{{ entry.key }}</span>
                  <textarea class="mt-1 min-h-10 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" disabled />
                </label>
              </div>
            </div>
          </div>
        </div>

        <div class="border-t border-slate-200 px-5 pt-4">
          <div class="flex flex-wrap items-center justify-between gap-4">
            <div>
              <h3 class="font-black">{{ copy.normalMaterialsTitle }}</h3>
              <p class="mt-1 text-sm text-slate-500">{{ copy.normalMaterialsDescription }}</p>
            </div>
            <button class="h-10 rounded-xl border px-4 font-bold disabled:opacity-40" :disabled="!selectedCourseId" type="button" @click="newMaterial">{{ copy.newNormalMaterial }}</button>
          </div>
        </div>
        <div class="grid gap-4 p-5 xl:grid-cols-[minmax(0,1fr)_400px]">
          <div class="rounded-2xl border border-slate-200">
            <div v-if="materialsLoading" class="px-6 py-10 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              {{ copy.loading }}
            </div>
            <div v-else-if="!materials.length" class="px-6 py-10 text-center text-slate-500">{{ copy.emptyMaterials }}</div>
            <div v-else class="divide-y divide-slate-100">
              <div v-for="material in materials" :key="materialId(material)" class="grid gap-3 p-4 lg:grid-cols-[1fr_auto]" :class="materialId(material) === selectedMaterialId ? 'bg-sky-50' : ''">
                <button class="text-left" type="button" @click="editMaterial(material)">
                  <div class="font-black">{{ materialTitle(material) }}</div>
                  <div class="mt-1 text-sm text-slate-500">{{ materialTypeLabel(material.material_type) }} · {{ copy.sortMeta(material.sort_order || 0) }}</div>
                  <div class="mt-1 break-all text-xs text-slate-400">{{ material.file_object_key || "-" }}</div>
                </button>
                <button class="rounded-xl border border-red-200 px-3 py-2 text-sm font-bold text-red-600" type="button" @click="deleteMaterial(material)">{{ copy.delete }}</button>
              </div>
            </div>
          </div>
          <form class="rounded-2xl border border-slate-200 p-4" @submit.prevent="saveMaterial">
            <h3 class="font-black">{{ editingMaterialId ? copy.editMaterial : copy.createMaterial }}</h3>
            <div class="mt-4 grid gap-3">
              <input v-model="materialForm.title" class="h-10 rounded-xl border border-slate-200 px-3" :placeholder="copy.materialTitlePlaceholder" />
              <select v-model="materialForm.material_type" class="h-10 rounded-xl border border-slate-200 px-3">
                <option value="1">{{ copy.materialTypes.textbook }}</option>
                <option value="2">{{ copy.materialTypes.slides }}</option>
                <option value="3">{{ copy.materialTypes.reference }}</option>
                <option value="4">{{ copy.materialTypes.other }}</option>
              </select>
              <textarea v-model="materialForm.description" class="min-h-20 rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.materialDescriptionPlaceholder" />
              <label class="block">
                <span class="text-sm font-bold">{{ copy.fileObjectKey }}</span>
                <input v-model="materialForm.file_object_key" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.fileObjectKeyPlaceholder" />
                <span class="mt-1 block text-xs text-slate-500">{{ copy.fileObjectKeyHint }}</span>
              </label>
              <label class="block">
                <span class="text-sm font-bold">{{ copy.fileHash }}</span>
                <input v-model="materialForm.file_hash" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.fileHashPlaceholder" />
                <span class="mt-1 block text-xs text-slate-500">{{ copy.fileHashHint }}</span>
              </label>
              <div class="grid gap-3 sm:grid-cols-2">
                <label class="block">
                  <span class="text-sm font-bold">{{ copy.fileSizeBytes }}</span>
                  <input v-model="materialForm.file_size" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.fileSizePlaceholder" type="number" min="0" />
                  <span class="mt-1 block text-xs text-slate-500">{{ copy.fileSizeHint }}</span>
                </label>
                <label class="block">
                  <span class="text-sm font-bold">{{ copy.sort }}</span>
                  <input v-model="materialForm.sort_order" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="1" type="number" min="1" />
                  <span class="mt-1 block text-xs text-slate-500">{{ copy.sortHint }}</span>
                </label>
              </div>
              <button class="h-10 rounded-xl bg-[#0b4ea2] px-4 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingMaterial" type="submit">
                {{ savingMaterial ? copy.saving : copy.saveMaterial }}
              </button>
            </div>
            <div v-if="selectedMaterialRecord" class="mt-5 border-t border-slate-200 pt-4">
              <h4 class="font-black">{{ copy.materialRawFields }}</h4>
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
              <h2 class="text-xl font-black">{{ copy.chapterListTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.chapterListDescription }}</p>
            </div>
            <button class="rounded-xl border px-3 py-2 font-bold" :disabled="!selectedCourseId" type="button" @click="newChapter">{{ copy.newChapter }}</button>
          </div>
          <div v-if="chaptersLoading" class="p-8 text-center text-slate-500">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            {{ copy.loading }}
          </div>
          <div v-else-if="!chapters.length" class="p-8 text-center text-slate-500">{{ copy.emptyChapters }}</div>
          <div v-else class="max-h-[520px] divide-y divide-slate-100 overflow-y-auto">
            <div v-for="chapter in chapters" :key="chapterId(chapter)" class="flex items-center justify-between gap-3 p-4" :class="chapterId(chapter) === selectedChapterId ? 'bg-sky-50' : ''">
              <button class="flex-1 text-left" type="button" @click="editChapter(chapter)">
                <div class="font-black">{{ chapterTitle(chapter) }}</div>
                <div class="mt-1 text-sm text-slate-500">{{ copy.sortVersionMeta(chapter.sort_order || 0, chapter.version || 0) }}</div>
              </button>
              <button class="rounded-xl border border-red-200 px-3 py-2 text-sm font-bold text-red-600" type="button" @click="deleteChapter(chapter)">{{ copy.delete }}</button>
            </div>
          </div>
          <form class="border-t border-slate-200 p-5" @submit.prevent="saveChapter">
            <h3 class="font-black">{{ editingChapterId ? copy.editChapter : copy.createChapter }}</h3>
            <input v-model="chapterForm.title" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.chapterTitlePlaceholder" />
            <input v-model="chapterForm.sort_order" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.sort" type="number" min="1" />
            <button class="mt-3 rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingChapter" type="submit">
              {{ savingChapter ? copy.saving : copy.saveChapter }}
            </button>
          </form>
        </aside>

        <section class="rounded-3xl border border-slate-200 bg-white shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
            <div>
              <h2 class="text-xl font-black">{{ copy.chapterDetailTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ selectedChapterId ? chapterTitle(selectedChapter) : copy.chapterDetailEmptyHint }}</p>
            </div>
          </div>
          <div class="grid gap-6 p-5 xl:grid-cols-[minmax(0,1fr)_360px]">
            <div class="rounded-2xl border border-slate-200 p-4">
              <h3 class="font-black">{{ copy.chapterRawFields }}</h3>
              <p class="mt-1 text-xs text-slate-500">{{ copy.systemReadonlyHint }}</p>
              <div v-if="!selectedChapterId" class="p-8 text-center text-slate-500">{{ copy.noSelectedChapter }}</div>
              <div v-else class="mt-3 grid gap-3 md:grid-cols-2">
                <label v-for="entry in recordEntries(selectedChapter)" :key="`chapter-${entry.key}`" class="block">
                  <span class="text-xs font-black text-slate-500">{{ entry.key }}</span>
                  <textarea class="mt-1 min-h-10 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" disabled />
                </label>
              </div>
            </div>
            <form class="rounded-2xl border border-slate-200 p-4" @submit.prevent="saveChapter">
              <h3 class="font-black">{{ editingChapterId ? copy.editChapter : copy.createChapter }}</h3>
              <input v-model="chapterForm.title" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.chapterTitlePlaceholder" />
              <input v-model="chapterForm.sort_order" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.sort" type="number" min="1" />
              <button class="mt-3 w-full rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingChapter" type="submit">
                {{ savingChapter ? copy.saving : copy.saveChapter }}
              </button>
            </form>
          </div>
        </section>
      </section>

      <section class="grid gap-6 2xl:grid-cols-[390px_minmax(0,1fr)]" :class="!selectedCourseId ? 'opacity-50' : ''">
        <aside class="rounded-3xl border border-slate-200 bg-white shadow-sm">
          <div class="flex items-center justify-between border-b border-slate-200 p-5">
            <div>
              <h2 class="text-xl font-black">{{ copy.lessonListTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.lessonListDescription }}</p>
            </div>
            <button class="rounded-xl border px-3 py-2 font-bold disabled:opacity-40" :disabled="!selectedChapterId" type="button" @click="newLesson">{{ copy.newLesson }}</button>
          </div>
          <div v-if="completeLoading || lessonsLoading" class="p-8 text-center text-slate-500">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            {{ copy.loading }}
          </div>
          <div v-else-if="!allLessonItems.length" class="p-8 text-center text-slate-500">{{ copy.emptyLessons }}</div>
          <div v-else class="max-h-[520px] divide-y divide-slate-100 overflow-y-auto">
            <div v-for="item in allLessonItems" :key="lessonId(item.lesson)" class="grid gap-3 p-4 lg:grid-cols-[1fr_auto]" :class="lessonId(item.lesson) === editingLessonId ? 'bg-sky-50' : ''">
              <button class="text-left" type="button" @click="editLesson(item.lesson)">
                <div class="font-black">{{ lessonTitle(item.lesson) }}</div>
                <div class="mt-1 text-sm text-slate-500">{{ copy.belongsToChapter }}{{ chapterTitle(item.chapter) }}</div>
                <div class="mt-1 text-xs text-slate-500">{{ copy.sortMeta(item.lesson.sort_order || 0) }} · {{ lessonTypeLabel(item.lesson.lesson_type) }}</div>
              </button>
              <button class="rounded-xl border border-red-200 px-3 py-2 text-sm font-bold text-red-600" type="button" @click="deleteLesson(item.lesson)">{{ copy.delete }}</button>
            </div>
          </div>
        </aside>

        <section class="rounded-3xl border border-slate-200 bg-white shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
            <div>
              <h2 class="text-xl font-black">{{ copy.lessonDetailTitle }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ editingLessonId ? lessonTitle(selectedLesson) : copy.lessonDetailEmptyHint }}</p>
            </div>
          </div>
          <div class="grid gap-6 p-5 xl:grid-cols-[minmax(0,1fr)_420px]">
            <div class="space-y-6">
              <div class="grid gap-3">
                <div class="rounded-2xl bg-blue-50 p-4">
                  <div class="text-xs font-black text-blue-600">{{ copy.ownerChapter }}</div>
                  <div class="mt-1 text-lg font-black text-slate-900">{{ selectedLessonOwnerChapter ? chapterTitle(selectedLessonOwnerChapter) : (selectedChapterId ? chapterTitle(selectedChapter) : copy.unselectedChapter) }}</div>
                  <div class="mt-2 break-all font-mono text-sm font-bold text-blue-900">ID: {{ selectedLessonOwnerChapter ? chapterId(selectedLessonOwnerChapter) : (selectedChapterId || "-") }}</div>
                </div>
              </div>
              <div class="rounded-2xl border border-slate-200 p-4">
                <h3 class="font-black">{{ copy.readonlyFields }}</h3>
                <p class="mt-1 text-xs text-slate-500">{{ copy.systemReadonlyHint }}</p>
                <div v-if="!selectedLesson" class="p-8 text-center text-slate-500">{{ copy.noSelectedLesson }}</div>
                <div v-else class="mt-3 grid gap-3 md:grid-cols-2">
                  <label v-for="entry in recordEntries(selectedLesson)" :key="`lesson-${entry.key}`" class="block">
                    <span class="text-xs font-black text-slate-500">{{ entry.key }}</span>
                    <textarea class="mt-1 min-h-10 w-full resize-y rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-500" :value="entry.value" disabled />
                  </label>
                </div>
              </div>
            </div>
            <form class="rounded-2xl border border-slate-200 p-4" @submit.prevent="saveLesson">
              <h3 class="font-black">{{ editingLessonId ? copy.editLesson : copy.createLesson }}</h3>
              <div class="mt-3 rounded-2xl border border-blue-100 bg-blue-50 p-3 text-sm text-blue-900">
                {{ copy.saveBelongsToChapter }}{{ lessonForm.chapter_id ? chapterTitle(chapterById(lessonForm.chapter_id)) : copy.selectChapter }}
              </div>
              <label class="mt-3 block">
                <span class="text-sm font-bold">{{ copy.ownerChapter }}</span>
                <select v-model="lessonForm.chapter_id" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
                  <option value="">{{ copy.selectChapter }}</option>
                  <option v-for="chapter in chapters" :key="chapterId(chapter)" :value="chapterId(chapter)">{{ chapterTitle(chapter) }}</option>
                </select>
              </label>
              <input v-model="lessonForm.title" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.lessonTitlePlaceholder" />
              <div class="mt-3 grid gap-3 sm:grid-cols-2">
                <label class="block">
                  <span class="text-sm font-bold">{{ copy.sort }}</span>
                  <input v-model="lessonForm.sort_order" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="1" type="number" min="1" />
                  <span class="mt-1 block text-xs text-slate-500">{{ copy.sortOrderHint }}</span>
                </label>
                <select v-model="lessonForm.lesson_type" class="rounded-xl border border-slate-200 px-4 py-3">
                  <option value="1">{{ copy.lessonTypes.video }}</option>
                  <option value="2">{{ copy.lessonTypes.text }}</option>
                  <option value="3">{{ copy.lessonTypes.pdf }}</option>
                  <option value="4">{{ copy.lessonTypes.image }}</option>
                  <option value="5">{{ copy.lessonTypes.audio }}</option>
                  <option value="6">{{ copy.lessonTypes.file }}</option>
                  <option value="7">{{ copy.lessonTypes.link }}</option>
                </select>
              </div>
              <textarea v-model="lessonForm.body" class="mt-3 min-h-24 w-full rounded-xl border border-slate-200 p-4" :placeholder="copy.lessonBodyPlaceholder" />
              <input v-model="lessonForm.asset_object_key" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.assetObjectKeyPlaceholder" />
              <label class="mt-3 block">
                <span class="text-sm font-bold">{{ copy.assetFileHash }}</span>
                <span class="ml-2 cursor-help rounded-full border border-slate-300 px-2 py-0.5 text-xs text-slate-500" :title="copy.assetHashHint">?</span>
                <input v-model="lessonForm.asset_file_hash" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.assetFileHashPlaceholder" />
              </label>
              <button class="mt-3 w-full rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedChapterId || savingLesson" type="submit">
                {{ savingLesson ? copy.saving : copy.saveLesson }}
              </button>
            </form>
          </div>
        </section>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white shadow-sm" :class="!selectedCourseId ? 'opacity-50' : ''">
        <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
          <div>
            <h2 class="text-xl font-black">{{ copy.quizBankTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.quizBankDescription }}</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <select v-model="quizForm.scope" class="rounded-xl border border-slate-200 px-4 py-2 font-bold" @change="newQuiz(quizForm.scope)">
              <option value="course">{{ copy.quizScopes.course }}</option>
              <option value="chapter">{{ copy.quizScopes.chapter }}</option>
              <option value="lesson">{{ copy.quizScopes.lesson }}</option>
            </select>
            <button
              class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40"
              :disabled="!selectedCourseId || (quizForm.scope === 'chapter' && !selectedChapterId) || (quizForm.scope === 'lesson' && !editingLessonId)"
              type="button"
              @click="loadQuizzes()"
            >
              {{ copy.loadQuizzes }}
            </button>
            <button
              class="rounded-xl bg-[#0b4ea2] px-4 py-2 font-bold text-white disabled:opacity-40"
              :disabled="!selectedCourseId || (quizForm.scope === 'chapter' && !selectedChapterId) || (quizForm.scope === 'lesson' && !editingLessonId)"
              type="button"
              @click="newQuiz(quizForm.scope)"
            >
              {{ copy.newQuiz }}
            </button>
          </div>
        </div>

        <div class="grid gap-6 p-5 2xl:grid-cols-[390px_minmax(0,1fr)]">
          <div class="rounded-2xl border border-slate-200">
            <div class="border-b border-slate-200 p-4">
              <h3 class="font-black">{{ copy.quizListTitle }}</h3>
              <p class="mt-1 text-xs text-slate-500">{{ copy.quizListDescription }}</p>
            </div>
            <div v-if="quizzesLoading" class="p-6 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-5 w-5 animate-spin" />
              {{ copy.loading }}
            </div>
            <div v-else-if="!allQuizItems.length" class="p-6 text-center text-slate-500">{{ copy.emptyQuizzes }}</div>
            <div v-else class="max-h-96 divide-y divide-slate-100 overflow-y-auto">
              <div v-for="item in allQuizItems" :key="quizId(item.quiz)" class="flex items-center justify-between gap-3 p-4" :class="quizId(item.quiz) === selectedQuizId ? 'bg-sky-50' : ''">
                <button class="flex-1 text-left" type="button" @click="editQuiz(item.quiz)">
                  <div class="font-black">{{ quizTitle(item.quiz) }}</div>
                  <div class="mt-1 text-xs text-slate-500">{{ copy.quizMeta(item.quiz.passing_score || 0, item.quiz.question_count || 0) }}</div>
                  <div class="mt-2 inline-flex max-w-full items-center gap-2 rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-700">
                    <span>{{ quizzableTypeLabel(item.ownerType) }}</span>
                    <span class="truncate">{{ copy.belongsTo }}{{ quizItemOwnerTitle(item) }}</span>
                  </div>
                  <div class="mt-1 break-all font-mono text-[11px] text-slate-400">{{ copy.ownerIdPrefix }}{{ quizItemOwnerId(item) || "-" }}</div>
                </button>
                <button class="rounded-xl border border-red-200 px-3 py-2 text-xs font-bold text-red-600" type="button" @click="deleteQuiz(item.quiz)">{{ copy.delete }}</button>
              </div>
            </div>
          </div>

          <div class="space-y-6">
            <section class="rounded-2xl border border-slate-200">
              <div class="border-b border-slate-200 p-4">
                <h3 class="font-black">{{ copy.quizDetailTitle }}</h3>
                <p class="mt-1 text-xs text-slate-500">{{ selectedQuizId ? quizTitle(selectedQuiz) : copy.quizDetailEmptyHint }}</p>
              </div>
              <div class="grid gap-4 p-4 lg:grid-cols-3">
                <div class="rounded-2xl bg-blue-50 p-4">
                  <div class="text-xs font-black text-blue-600">{{ copy.owner }}</div>
                  <div class="mt-1 text-lg font-black text-slate-900">{{ selectedQuizItem ? quizzableTypeLabel(selectedQuizItem.ownerType) : copy.ownerLevelQuiz(quizTarget().label) }}</div>
                </div>
                <div class="rounded-2xl bg-slate-50 p-4">
                  <div class="text-xs font-black text-slate-500">{{ copy.ownerChapterLabel }}</div>
                  <div class="mt-1 text-lg font-black text-slate-900">{{ selectedQuizItem?.chapter ? chapterTitle(selectedQuizItem.chapter) : (quizForm.scope === "chapter" ? chapterTitle(selectedChapter) : "-") }}</div>
                </div>
                <div class="rounded-2xl bg-slate-50 p-4">
                  <div class="text-xs font-black text-slate-500">{{ copy.ownerLessonLabel }}</div>
                  <div class="mt-1 text-lg font-black text-slate-900">{{ selectedQuizItem?.lesson ? lessonTitle(selectedQuizItem.lesson) : (quizForm.scope === "lesson" ? lessonTitle(selectedLesson) : "-") }}</div>
                </div>
              </div>
            <form class="border-t border-slate-200 p-4" @submit.prevent="saveQuiz">
              <h4 class="font-black">{{ editingQuizId ? copy.editQuiz : copy.createQuiz }}</h4>
              <div class="mt-3 rounded-2xl border border-blue-100 bg-blue-50 p-3 text-sm text-blue-900">
                {{ copy.saveQuizOwner }}{{ quizTarget().label }} · {{ quizTarget().id ? quizTarget().title : copy.unselectedTarget(quizTarget().label) }}
              </div>
              <input v-model="quizForm.title" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.quizTitlePlaceholder" />
              <textarea v-model="quizForm.description" class="mt-3 min-h-20 w-full rounded-xl border border-slate-200 p-4" :placeholder="copy.description" />
              <div class="mt-3 grid gap-3 sm:grid-cols-2">
                <input v-model="quizForm.passing_score" class="rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.passingScorePlaceholder" type="number" />
                <input v-model="quizForm.time_limit" class="rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.timeLimitPlaceholder" type="number" />
              </div>
              <label class="mt-3 inline-flex items-center gap-2 text-sm font-bold text-slate-600">
                <input v-model="quizForm.randomize_questions" type="checkbox" />
                {{ copy.randomizeQuestions }}
              </label>
              <button class="mt-3 w-full rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="savingQuiz" type="submit">
                {{ savingQuiz ? copy.saving : copy.saveQuiz }}
              </button>
            </form>
            </section>

          <section class="grid gap-6 2xl:grid-cols-[360px_minmax(0,1fr)]">
            <aside class="rounded-2xl border border-slate-200">
              <div class="flex items-center justify-between gap-3 border-b border-slate-200 p-4">
                <div>
                  <h3 class="font-black">{{ copy.questionListTitle }}</h3>
                  <p class="mt-1 text-xs text-slate-500">{{ copy.questionListDescription }}</p>
                </div>
                <button class="rounded-xl border px-3 py-2 text-xs font-bold" :disabled="!selectedQuizId" type="button" @click="newQuestion">{{ copy.newQuestion }}</button>
              </div>
              <div v-if="questionsLoading" class="p-6 text-center text-slate-500">
                <Loader2 class="mx-auto mb-2 h-5 w-5 animate-spin" />
                {{ copy.loading }}
              </div>
              <div v-else-if="!questions.length" class="p-6 text-center text-slate-500">{{ copy.emptyQuestions }}</div>
              <div v-else class="max-h-96 divide-y divide-slate-100 overflow-y-auto">
                <div v-for="question in questions" :key="questionId(question)" class="flex items-center justify-between gap-3 p-4" :class="questionId(question) === selectedQuestionId ? 'bg-sky-50' : ''">
                  <button class="flex-1 text-left" type="button" @click="editQuestion(question)">
                    <div class="line-clamp-2 font-black">{{ questionTitle(question) }}</div>
                    <div class="mt-1 text-xs text-slate-500">{{ copy.questionMeta(questionTypeLabel(question.question_type), question.points || 0) }}</div>
                  </button>
                  <button class="rounded-xl border border-red-200 px-3 py-2 text-xs font-bold text-red-600" type="button" @click="deleteQuestion(question)">{{ copy.delete }}</button>
                </div>
              </div>
            </aside>

            <div class="space-y-6">
              <section class="rounded-2xl border border-slate-200">
                <div class="border-b border-slate-200 p-4">
                  <h3 class="font-black">{{ copy.questionDetailTitle }}</h3>
                  <p class="mt-1 text-xs text-slate-500">{{ selectedQuestionId ? questionTitle(selectedQuestion) : copy.questionDetailEmptyHint }}</p>
                </div>
                <div class="grid gap-3 p-4 sm:grid-cols-2">
                  <div class="rounded-2xl bg-blue-50 p-4">
                    <div class="text-xs font-black text-blue-600">{{ copy.ownerQuiz }}</div>
                    <div class="mt-1 text-lg font-black text-slate-900">{{ selectedQuizId ? quizTitle(selectedQuiz) : copy.unselectedQuiz }}</div>
                    <div class="mt-2 break-all font-mono text-sm font-bold text-blue-900">ID: {{ selectedQuizId || "-" }}</div>
                  </div>
                  <div class="rounded-2xl bg-slate-50 p-4">
                    <div class="text-xs font-black text-slate-500">{{ copy.questionTypePoints }}</div>
                    <div class="mt-1 text-lg font-black text-slate-900">{{ copy.questionMeta(selectedQuestion ? questionTypeLabel(selectedQuestion.question_type) : questionTypeLabel(questionForm.question_type), selectedQuestion?.points || questionForm.points || 0) }}</div>
                  </div>
                </div>
                <form class="border-t border-slate-200 p-4" @submit.prevent="saveQuestion">
                  <h4 class="font-black">{{ editingQuestionId ? copy.editQuestion : copy.createQuestion }}</h4>
                  <textarea v-model="questionForm.question_text" class="mt-3 min-h-24 w-full rounded-xl border border-slate-200 p-4" :placeholder="copy.questionTextPlaceholder" />
                  <div class="mt-3 grid gap-3 sm:grid-cols-3">
                    <select v-model="questionForm.question_type" class="rounded-xl border border-slate-200 px-4 py-3">
                      <option value="1">{{ copy.questionTypes.single }}</option>
                      <option value="2">{{ copy.questionTypes.multiple }}</option>
                      <option value="3">{{ copy.questionTypes.judgement }}</option>
                    </select>
                    <input v-model="questionForm.points" class="rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.pointsPlaceholder" type="number" />
                    <input v-model="questionForm.sort_order" class="rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.sort" type="number" />
                  </div>
                  <label class="mt-3 inline-flex items-center gap-2 text-sm font-bold text-slate-600">
                    <input v-model="questionForm.is_required" type="checkbox" />
                    {{ copy.required }}
                  </label>
                  <textarea v-model="questionForm.media_items_json" class="mt-3 min-h-20 w-full rounded-xl border border-slate-200 p-4 font-mono text-xs" :placeholder="copy.mediaJsonPlaceholder" />
                  <button class="mt-3 w-full rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedQuizId || savingQuestion" type="submit">
                    {{ savingQuestion ? copy.saving : copy.saveQuestion }}
                  </button>
                </form>
              </section>

              <section class="rounded-2xl border border-slate-200">
                <div class="border-b border-slate-200 p-4">
                  <h3 class="font-black">{{ copy.optionsTitle }}</h3>
                  <p class="mt-1 text-xs text-slate-500">{{ selectedQuestionId ? copy.optionsSelectedHint : copy.optionsNeedQuestionHint }}</p>
                </div>
                <div v-if="optionsLoading" class="p-6 text-center text-slate-500">
                  <Loader2 class="mx-auto mb-2 h-5 w-5 animate-spin" />
                  {{ copy.loading }}
                </div>
                <div v-else-if="!options.length" class="p-6 text-center text-slate-500">{{ copy.emptyOptions }}</div>
                <div v-else class="max-h-72 divide-y divide-slate-100 overflow-y-auto">
                  <div v-for="option in options" :key="optionId(option)" class="flex items-center justify-between gap-3 p-4" :class="optionId(option) === editingOptionId ? 'bg-sky-50' : ''">
                    <button class="flex-1 text-left" type="button" @click="editOption(option)">
                      <div class="font-black">{{ optionTitle(option) }}</div>
                      <div class="mt-1 text-xs" :class="option.is_correct ? 'text-emerald-600' : 'text-slate-500'">
                        {{ option.is_correct ? copy.correctAnswer : copy.normalOption }} · {{ copy.sortMeta(option.sort_order || 0) }}
                      </div>
                    </button>
                    <button class="rounded-xl border border-red-200 px-3 py-2 text-xs font-bold text-red-600" type="button" @click="deleteOption(option)">{{ copy.delete }}</button>
                  </div>
                </div>
                <form class="border-t border-slate-200 p-4" @submit.prevent="saveOption">
                  <div class="flex items-center justify-between gap-3">
                    <h4 class="font-black">{{ editingOptionId ? copy.editOption : copy.createOption }}</h4>
                    <button class="rounded-xl border px-3 py-2 text-xs font-bold" :disabled="!selectedQuestionId" type="button" @click="newOption">{{ copy.newOption }}</button>
                  </div>
                  <input v-model="optionForm.option_text" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.optionTextPlaceholder" />
                  <div class="mt-3 grid gap-3 sm:grid-cols-2">
                    <input v-model="optionForm.sort_order" class="rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.sort" type="number" />
                    <label class="inline-flex items-center gap-2 rounded-xl border border-slate-200 px-4 py-3 text-sm font-bold text-slate-600">
                      <input v-model="optionForm.is_correct" type="checkbox" />
                      {{ copy.correctAnswer }}
                    </label>
                  </div>
                  <button class="mt-3 w-full rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedQuestionId || savingOption" type="submit">
                    {{ savingOption ? copy.saving : copy.saveOption }}
                  </button>
                </form>
                <div v-if="selectedQuiz || selectedQuestion" class="border-t border-slate-200 p-4">
                  <h4 class="font-black">{{ copy.quizReadonlyFields }}</h4>
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
          <h2 class="text-2xl font-black">{{ copy.importTitle }}</h2>
          <button class="rounded-xl border px-3 py-2 font-bold" type="button" @click="importOpen = false">{{ copy.close }}</button>
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <label>
            <span class="text-sm font-bold">{{ copy.importType }}</span>
            <select v-model="importScope" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
              <option value="course">{{ copy.importCourse }}</option>
              <option value="quiz">{{ copy.importChapterQuiz }}</option>
            </select>
          </label>
          <label>
            <span class="text-sm font-bold">{{ copy.categoryTips }}</span>
            <input v-model="importCategoryTips" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.importCategoryPlaceholder" />
          </label>
        </div>
        <label class="mt-4 block">
          <span class="text-sm font-bold">{{ copy.jsonFile }}</span>
          <input class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" type="file" accept=".json,application/json" @change="loadImportFile" />
        </label>
        <textarea v-model="importJson" class="mt-4 min-h-80 w-full rounded-xl border border-slate-200 p-4 font-mono text-sm" :placeholder="copy.pasteJsonPlaceholder" />
        <button class="mt-5 inline-flex items-center gap-2 rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="importing" type="button" @click="importLmsJson">
          <Loader2 v-if="importing" class="h-4 w-4 animate-spin" />
          <UploadCloud v-else class="h-4 w-4" />
          {{ importing ? copy.importing : copy.startImport }}
        </button>
      </div>
    </div>
  </div>
</template>
