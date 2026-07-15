<script setup lang="ts">
import { ChevronDown, FileJson, Info, Loader2, Plus, RefreshCw, Save, Trash2, UploadCloud, X } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import ReadonlyField from "@/components/ReadonlyField.vue"
import LmsPrerequisitesTab from "@/components/LmsPrerequisitesTab.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
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
  course_gpath: string
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
type ChapterDialogMode = "detail" | "edit" | "create"
type LessonDialogMode = "detail" | "edit" | "create"
type QuizDialogMode = "detail" | "edit" | "create"
type MaterialDialogMode = "detail" | "edit" | "create"
type CourseCreateContext = {
  selectedCourse: JsonRecord | null
  form: CourseForm
  view: "list" | "detail"
}

type QuizForm = {
  scope: QuizScope
  owner_id: string
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

type DetailDeleteKind = "supplementaryItem" | "supplementaryConfig" | "material" | "quiz" | "question" | "option"

type PendingDetailDelete = {
  kind: DetailDeleteKind
  title: string
  description: string
  id?: string
  version?: number
  index?: number
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
  questionCount: number
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
const courseCreateOpen = ref(false)
const courseCreateContext = ref<CourseCreateContext | null>(null)
const courseDetailDialogOpen = ref(false)
const courseDetailDialogLoading = ref(false)
const courseDetailTarget = ref<JsonRecord | null>(null)
const courseDetailDialogDetail = ref<JsonRecord | null>(null)
const courseDetailDialogComplete = ref<JsonRecord | null>(null)
const courseDeleteConfirmOpen = ref(false)
const pendingDeleteCourse = ref<JsonRecord | null>(null)
const deletingCourse = ref(false)
const chapterActiveTab = ref<"basic" | "prerequisites">("basic")
const lessonActiveTab = ref<"basic" | "prerequisites">("basic")
const quizActiveTab = ref<"basic" | "prerequisites">("basic")
const chapterDialogOpen = ref(false)
const chapterDialogMode = ref<ChapterDialogMode>("detail")
const chapterDeleteConfirmOpen = ref(false)
const pendingDeleteChapter = ref<JsonRecord | null>(null)
const deletingChapter = ref(false)
const lessonDialogOpen = ref(false)
const lessonDialogMode = ref<LessonDialogMode>("detail")
const lessonDeleteConfirmOpen = ref(false)
const pendingDeleteLesson = ref<JsonRecord | null>(null)
const deletingLesson = ref(false)
const detailDeleteConfirmOpen = ref(false)
const pendingDetailDelete = ref<PendingDetailDelete | null>(null)
const deletingDetail = ref(false)
const materialDialogOpen = ref(false)
const materialDialogMode = ref<MaterialDialogMode>("create")
const quizDialogOpen = ref(false)
const quizDialogMode = ref<QuizDialogMode>("detail")
const questionDialogOpen = ref(false)
const questionDialogMode = ref<"detail" | "edit" | "create">("detail")

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
const advancedMediaDialogOpen = ref(false)
const parsedMediaItems = ref<any[]>([])
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
const isCreatingCourse = computed(() => courseCreateOpen.value && !selectedCourseId.value)
const selectedChapterId = computed(() => chapterId(selectedChapter.value))
const selectedMaterialId = computed(() => materialId(selectedMaterial.value))

const duplicateGpathWarning = computed(() => {
  const gpath = courseForm.value.course_gpath?.trim()
  if (!gpath) return false
  return courses.value.some(c => c.course_gpath === gpath && courseId(c) !== selectedCourseId.value)
})

const supplementaryMaterialItems = computed<SupplementaryMaterialItem[]>(() => parseSupplementaryMaterialItems(normalizeSupplementaryMaterials({
  ...(supplementaryMaterial.value || {}),
  kind: supplementaryMaterialForm.value.kind,
  data_json: supplementaryMaterialForm.value.data_json,
}), "Chapter"))
const selectedQuizId = computed(() => quizId(selectedQuiz.value))
const selectedQuestionId = computed(() => questionId(selectedQuestion.value))
const selectedCourseStatusBadge = computed(() => courseStatusBadgeValue(selectedCourse.value))
const canDeleteSelectedCourse = computed(() => !!selectedCourseId.value && courseStatusKey(selectedCourse.value) === "draft")
const canPublishSelectedCourse = computed(() => !!selectedCourseId.value && courseStatusKey(selectedCourse.value) === "draft")
const selectedLesson = computed(() => lessons.value.find((item) => lessonId(item) === editingLessonId.value) || null)
const selectedMaterialRecord = computed(() => materials.value.find((item) => materialId(item) === selectedMaterialId.value) || selectedMaterial.value)
const courseDetailDialogCourseId = computed(() => courseId(courseDetailTarget.value))
const completeCourseRecord = computed(() => {
  const value = completeCourse.value?.complete_course
  return value && typeof value === "object" && !Array.isArray(value) ? value as JsonRecord : completeCourse.value
})
const courseDetailDialogCompleteRecord = computed(() => {
  const value = courseDetailDialogComplete.value?.complete_course
  return value && typeof value === "object" && !Array.isArray(value) ? value as JsonRecord : courseDetailDialogComplete.value
})
const completeChapterRecords = computed(() => {
  const chapterDetails = Array.isArray(completeCourseRecord.value?.chapters) ? completeCourseRecord.value.chapters : []
  return chapterDetails
    .filter(isJsonRecord)
    .map((record) => record.chapter && isJsonRecord(record.chapter) ? record.chapter : record)
})
const courseDetailDialogChapterRecords = computed(() => {
  const chapterDetails = Array.isArray(courseDetailDialogCompleteRecord.value?.chapters) ? courseDetailDialogCompleteRecord.value.chapters : []
  return chapterDetails
    .filter(isJsonRecord)
    .map((record) => record.chapter && isJsonRecord(record.chapter) ? record.chapter : record)
})
const allLessonItems = computed<LessonListItem[]>(() => {
  const items: LessonListItem[] = []
  const chapterDetails = Array.isArray(completeCourseRecord.value?.chapters) ? completeCourseRecord.value.chapters : []
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
const selectedLessonRecord = computed(() => allLessonItems.value.find((item) => lessonId(item.lesson) === editingLessonId.value)?.lesson || selectedLesson.value)
const selectedLessonOwnerChapter = computed(() => allLessonItems.value.find((item) => lessonId(item.lesson) === editingLessonId.value)?.chapter || selectedChapter.value)
const allQuizItems = computed<QuizListItem[]>(() => {
  const items: QuizListItem[] = []
  const complete = completeCourseRecord.value || {}
  const courseQuizzes = Array.isArray(complete.quizzes) ? complete.quizzes : []
  for (const quizDetail of courseQuizzes) {
    const quiz = extractQuizRecord(quizDetail)
    const questions = quizDetail && typeof quizDetail === "object" && Array.isArray((quizDetail as JsonRecord).questions) ? (quizDetail as JsonRecord).questions as unknown[] : []
    if (quiz) items.push({ quiz, questionCount: questions.length, ownerType: 3, owner: selectedCourse.value, chapter: null, lesson: null })
  }
  const chapterDetails = Array.isArray(complete.chapters) ? complete.chapters : []
  for (const detail of chapterDetails) {
    if (!detail || typeof detail !== "object" || Array.isArray(detail)) continue
    const record = detail as JsonRecord
    const chapter = record.chapter && typeof record.chapter === "object" && !Array.isArray(record.chapter) ? record.chapter as JsonRecord : record
    const chapterQuizzes = Array.isArray(record.quizzes) ? record.quizzes : []
    for (const quizDetail of chapterQuizzes) {
      const quiz = extractQuizRecord(quizDetail)
      const questions = quizDetail && typeof quizDetail === "object" && Array.isArray((quizDetail as JsonRecord).questions) ? (quizDetail as JsonRecord).questions as unknown[] : []
      if (quiz) items.push({ quiz, questionCount: questions.length, ownerType: 2, owner: chapter, chapter, lesson: null })
    }
    const lessonDetails = Array.isArray(record.lessons) ? record.lessons : []
    for (const lessonDetail of lessonDetails) {
      if (!lessonDetail || typeof lessonDetail !== "object" || Array.isArray(lessonDetail)) continue
      const lessonRecord = lessonDetail as JsonRecord
      const lesson = lessonRecord.lesson && typeof lessonRecord.lesson === "object" && !Array.isArray(lessonRecord.lesson) ? lessonRecord.lesson as JsonRecord : lessonRecord
      const lessonQuizzes = Array.isArray(lessonRecord.quizzes) ? lessonRecord.quizzes : []
      for (const quizDetail of lessonQuizzes) {
        const quiz = extractQuizRecord(quizDetail)
        const questions = quizDetail && typeof quizDetail === "object" && Array.isArray((quizDetail as JsonRecord).questions) ? (quizDetail as JsonRecord).questions as unknown[] : []
        if (quiz) items.push({ quiz, questionCount: questions.length, ownerType: 1, owner: lesson, chapter, lesson })
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
      questionCount: Number(quiz.question_count || 0),
      ownerType,
      owner: ownerType === 3 ? selectedCourse.value : ownerType === 2 ? selectedChapter.value : selectedLesson.value,
      chapter: ownerType === 2 ? selectedChapter.value : selectedLessonOwnerChapter.value,
      lesson: ownerType === 1 ? selectedLesson.value : null,
    })
  }
  return items
})
const selectedQuizItem = computed(() => allQuizItems.value.find((item) => quizId(item.quiz) === selectedQuizId.value) || null)
const quizChapterOptions = computed(() => chapters.value.length ? chapters.value : completeChapterRecords.value)
const quizLessonOptions = computed(() => allLessonItems.value)
const courseDetailDialogChapterCount = computed(() => courseDetailDialogComplete.value ? courseDetailDialogChapterRecords.value.length : positiveCount(courseDetailDialogDetail.value?.chapter_count))
const courseDetailDialogLessonCount = computed(() => {
  if (!courseDetailDialogComplete.value) return positiveCount(courseDetailDialogDetail.value?.lesson_count)
  return (Array.isArray(courseDetailDialogCompleteRecord.value?.chapters) ? courseDetailDialogCompleteRecord.value.chapters : [])
    .filter(isJsonRecord)
    .reduce((total, chapter) => total + (Array.isArray(chapter.lessons) ? chapter.lessons.length : 0), 0)
})
const courseDetailDialogQuizCount = computed(() => {
  if (!courseDetailDialogComplete.value) return positiveCount(courseDetailDialogDetail.value?.quiz_count)
  const complete = courseDetailDialogCompleteRecord.value || {}
  let total = Array.isArray(complete.quizzes) ? complete.quizzes.length : 0
  const chapters = Array.isArray(complete.chapters) ? complete.chapters.filter(isJsonRecord) : []
  for (const chapter of chapters) {
    total += Array.isArray(chapter.quizzes) ? chapter.quizzes.length : 0
    const lessons = Array.isArray(chapter.lessons) ? chapter.lessons.filter(isJsonRecord) : []
    for (const lesson of lessons) total += Array.isArray(lesson.quizzes) ? lesson.quizzes.length : 0
  }
  return total
})
const courseDetailDialogMaterialCount = computed(() => {
  if (!courseDetailDialogComplete.value) return positiveCount(courseDetailDialogDetail.value?.material_count)
  const complete = courseDetailDialogCompleteRecord.value || {}
  return Math.max(
    Array.isArray(complete.materials) ? complete.materials.length : 0,
    positiveCount(courseDetailDialogDetail.value?.material_count),
  )
})

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
    course_gpath: "",
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
    scope: "course",
    owner_id: "",
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

function positiveCount(value: unknown) {
  const count = Number(value || 0)
  return Number.isFinite(count) && count > 0 ? count : 0
}

function courseTitle(course: JsonRecord | null | undefined) {
  return String(pickFirst(course || {}, ["title", "name", "course_title"]) || courseId(course) || copy.value.fallbacks.course)
}

function courseDescription(course: JsonRecord | null | undefined) {
  return String(pickFirst(course || {}, ["description", "desc", "summary", "course_description"]) || "")
}

function courseStatusValue(course: JsonRecord | null | undefined) {
  const status = String(pickFirst(course || {}, ["status", "raw_status"]) || "").trim()
  if (status) return status
  return course?.is_published ? "Published" : "Draft"
}

function courseStatusKey(course: JsonRecord | null | undefined) {
  const normalized = courseStatusValue(course).trim().toLowerCase()
  if (normalized === "active" || normalized === "published") return "published"
  if (normalized === "deprecated" || normalized === "deprecate") return "deprecated"
  if (normalized === "draft" || normalized === "inactive") return "draft"
  return normalized || "draft"
}

function courseStatusLabel(course: JsonRecord | null | undefined) {
  const key = courseStatusKey(course)
  return copy.value.courseStatuses[key as keyof typeof copy.value.courseStatuses] || courseStatusValue(course) || copy.value.unknown
}

function courseStatusBadgeValue(course: JsonRecord | null | undefined) {
  const key = courseStatusKey(course)
  if (key === "published") return "COMPLETED"
  if (key === "deprecated") return "DEPRECATED"
  return "PENDING"
}

function chapterTitle(chapter: JsonRecord | null | undefined) {
  return String(pickFirst(chapter || {}, ["title", "name"]) || chapterId(chapter) || copy.value.fallbacks.chapter)
}

function isChapterEmpty(chapter: JsonRecord | null | undefined) {
  if (!chapter) return false
  const cid = chapterId(chapter)
  const hasLesson = allLessonItems.value.some(item => chapterId(item.chapter) === cid)
  const hasQuiz = allQuizItems.value.some(item => chapterId(item.chapter) === cid)
  return !hasLesson && !hasQuiz
}

function isLessonEmpty(lesson: JsonRecord | null | undefined) {
  if (!lesson) return false
  const type = String(lesson.lesson_type || "")
  if (type === "2") return !lesson.body
  if (type === "7") return !lesson.external_url && !lesson.asset_object_key
  return !lesson.asset_object_key && !lesson.media_object_key && !lesson.media_file_hash
}

function isQuizEmpty(item: QuizListItem | null | undefined) {
  if (!item) return false
  return item.questionCount === 0
}

function isSupplementaryMaterialEmpty(item: SupplementaryMaterialItem | null | undefined) {
  if (!item) return false
  const type = String(item.type || "").trim().toLowerCase()
  if (type === "pdf" || type === "video") return !item.url
  return false
}

function chapterById(id: string) {
  return chapters.value.find((item) => chapterId(item) === id) || null
}

function chapterOptionById(id: string) {
  return quizChapterOptions.value.find((item) => chapterId(item) === id) || null
}

function lessonTitle(lesson: JsonRecord | null | undefined) {
  return String(pickFirst(lesson || {}, ["title", "name"]) || lessonId(lesson) || copy.value.fallbacks.lesson)
}

function lessonItemById(id: string) {
  return allLessonItems.value.find((item) => lessonId(item.lesson) === id) || null
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
  if (normalized === "pdf") return "border-orange-200 bg-orange-50 text-orange-700"
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

function displayReadonlyValue(key: string, value: unknown) {
  if (key.endsWith("_at") || key.endsWith("_time")) return formatDate(value) || displayValue(value)
  return displayValue(value)
}

function courseReadonlyValue(record: JsonRecord, key: string, value: unknown) {
  if (key === "status" || key === "raw_status") return courseStatusLabel({ ...record, status: value, raw_status: value })
  return displayReadonlyValue(key, value)
}

function courseRecordEntries(record: JsonRecord | null | undefined) {
  if (!record) return []
  return Object.entries(record).map(([key, value]) => ({
    key,
    value: courseReadonlyValue(record, key, value),
  }))
}

function courseReadonlyFieldLabel(key: string) {
  const labels: Record<string, string> = copy.value.readonlyCourseFieldLabels
  return labels[key] || key
}

function chapterReadonlyFieldLabel(key: string) {
  const labels: Record<string, string> = copy.value.chapterFieldLabels
  return labels[key] || courseReadonlyFieldLabel(key)
}

function chapterRecordEntries(record: JsonRecord | null | undefined) {
  if (!record) return []
  return Object.entries(record).map(([key, value]) => ({
    key,
    label: chapterReadonlyFieldLabel(key),
    value: displayReadonlyValue(key, value),
  }))
}

function lessonReadonlyFieldLabel(key: string) {
  const labels: Record<string, string> = copy.value.lessonFieldLabels
  return labels[key] || chapterReadonlyFieldLabel(key)
}

function lessonReadonlyValue(key: string, value: unknown) {
  if (key === "lesson_type") return lessonTypeLabel(value)
  return displayReadonlyValue(key, value)
}

function lessonRecordEntries(record: JsonRecord | null | undefined) {
  if (!record) return []
  return Object.entries(record).map(([key, value]) => ({
    key,
    label: lessonReadonlyFieldLabel(key),
    value: lessonReadonlyValue(key, value),
  }))
}

function materialReadonlyFieldLabel(key: string) {
  const labels: Record<string, string> = copy.value.materialFieldLabels
  return labels[key] || lessonReadonlyFieldLabel(key)
}

function materialReadonlyValue(key: string, value: unknown) {
  if (key === "material_type") return materialTypeLabel(value)
  return displayReadonlyValue(key, value)
}

function supplementaryReadonlyFieldLabel(key: string) {
  const labels: Record<string, string> = copy.value.supplementaryFieldLabels
  return labels[key] || materialReadonlyFieldLabel(key)
}

function supplementaryKindValue(value: unknown) {
  const kind = String(value || "")
  if (kind === "supplementary_materials") return copy.value.supplementaryKindValues.supplementaryMaterials
  return displayValue(value)
}

function supplementaryReadonlyValue(key: string, value: unknown) {
  if (key === "kind") return supplementaryKindValue(value)
  return displayReadonlyValue(key, value)
}

function supplementaryRecordEntries(record: JsonRecord | null | undefined) {
  if (!record) return []
  return Object.entries(record).map(([key, value]) => ({
    key,
    label: supplementaryReadonlyFieldLabel(key),
    value: supplementaryReadonlyValue(key, value),
  }))
}

function materialReadonlyMinHeight(key: string) {
  return key === "file_object_key" || key === "file_size" ? "84px" : "44px"
}

function materialRecordEntries(record: JsonRecord | null | undefined) {
  if (!record) return []
  return Object.entries(record).map(([key, value]) => ({
    key,
    label: materialReadonlyFieldLabel(key),
    value: materialReadonlyValue(key, value),
  }))
}

function quizReadonlyFieldLabel(key: string) {
  const labels: Record<string, string> = copy.value.quizFieldLabels
  return labels[key] || lessonReadonlyFieldLabel(key)
}

function quizReadonlyValue(key: string, value: unknown) {
  if (key === "quizzable_type") return quizzableTypeLabel(value)
  return displayReadonlyValue(key, value)
}

function quizRecordEntries(record: JsonRecord | null | undefined) {
  if (!record) return []
  return Object.entries(record).map(([key, value]) => ({
    key,
    label: quizReadonlyFieldLabel(key),
    value: quizReadonlyValue(key, value),
  }))
}

function questionReadonlyFieldLabel(key: string) {
  const labels: Record<string, string> = copy.value.questionFieldLabels
  return labels[key] || quizReadonlyFieldLabel(key)
}

function questionReadonlyValue(key: string, value: unknown) {
  if (key === "question_type") return questionTypeLabel(value)
  return displayReadonlyValue(key, value)
}

function questionRecordEntries(record: JsonRecord | null | undefined) {
  if (!record) return []
  return Object.entries(record).map(([key, value]) => ({
    key,
    label: questionReadonlyFieldLabel(key),
    value: questionReadonlyValue(key, value),
  }))
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

function isSupplementaryLinkType(type = supplementaryItemForm.value.type) {
  return type === "Article" || type === "Link"
}

function isSupplementaryAssetRequired(type = supplementaryItemForm.value.type) {
  return (type === "Video" || type === "PDF") && editingSupplementaryItemIndex.value >= 0
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

function courseFormFrom(course: JsonRecord): CourseForm {
  return {
    category_tips: String(course.category_tips || ""),
    title: String(course.title || ""),
    description: courseDescription(course),
    thumbnail_object_key: String(course.thumbnail_object_key || ""),
    thumbnail_file_hash: String(course.thumbnail_file_hash || ""),
    duration_min: String(course.duration_min || 0),
    certification_enabled: Boolean(course.certification_enabled),
    certification_def_id: String(course.certification_def_ulid || ""),
    respath: String(course.respath || course.course_gpath || ""),
    course_gpath: String(course.course_gpath || ""),
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
    course_gpath: courseForm.value.course_gpath.trim(),
  }
  if (version !== undefined) payload.version = Number(version || 0)
  return payload
}

function extractCourseRecord(record: JsonRecord | null | undefined) {
  if (!record) return null
  for (const key of ["course", "course_detail", "detail", "summary"]) {
    const value = record[key]
    if (isJsonRecord(value)) return value
  }
  return record
}

function mergeSelectedCourse(record: JsonRecord | null | undefined) {
  const course = extractCourseRecord(record)
  if (!course || !selectedCourse.value) return
  selectedCourse.value = { ...selectedCourse.value, ...course }
  courseForm.value = courseFormFrom(selectedCourse.value)
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
    toast.error(apiErrorMessage(err, copy.value.toasts.courseListLoadFailed))
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

async function openCourseDetailDialog(course: JsonRecord) {
  const id = courseId(course)
  if (!id) return
  courseDetailTarget.value = course
  courseDetailDialogDetail.value = null
  courseDetailDialogComplete.value = null
  courseDetailDialogOpen.value = true
  courseDetailDialogLoading.value = true
  try {
    const [detailResult, completeResult] = await Promise.allSettled([
      apiClient<JsonRecord>(`/api/lms/courses/${encodeURIComponent(id)}/detail`),
      apiClient<JsonRecord>(`/api/lms/courses/${encodeURIComponent(id)}/complete`),
    ])
    if (courseDetailDialogCourseId.value !== id) return
    if (detailResult.status === "fulfilled") courseDetailDialogDetail.value = detailResult.value
    else console.error(detailResult.reason)
    if (completeResult.status === "fulfilled") courseDetailDialogComplete.value = completeResult.value
    else console.error(completeResult.reason)
  } finally {
    if (courseDetailDialogCourseId.value === id) courseDetailDialogLoading.value = false
  }
}

function closeCourseDetailDialog() {
  courseDetailDialogOpen.value = false
  courseDetailDialogLoading.value = false
  courseDetailTarget.value = null
  courseDetailDialogDetail.value = null
  courseDetailDialogComplete.value = null
}

function clearCourseSelection() {
  selectedCourse.value = null
  courseForm.value = emptyCourseForm()
  resetContent()
}

function newCourse() {
  courseCreateContext.value = {
    selectedCourse: selectedCourse.value,
    form: { ...courseForm.value },
    view: courseView.value,
  }
  selectedCourse.value = null
  courseForm.value = emptyCourseForm()
  courseView.value = "list"
  courseCreateOpen.value = true
}

function closeCourseCreate() {
  if (savingCourse.value) return
  const context = courseCreateContext.value
  courseCreateOpen.value = false
  selectedCourse.value = context?.selectedCourse || null
  courseForm.value = context ? { ...context.form } : emptyCourseForm()
  courseView.value = context?.view || "list"
  courseCreateContext.value = null
}

function backToCourseList() {
  courseView.value = "list"
}

async function loadCourseDetail() {
  if (!selectedCourseId.value) return
  detailLoading.value = true
  try {
    courseDetail.value = await apiClient<JsonRecord>(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/detail`)
    mergeSelectedCourse(courseDetail.value)
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
    mergeSelectedCourse(extractCourseRecord(completeCourseRecord.value))
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
  if (!courseForm.value.course_gpath.trim()) {
    toast.error(copy.value.toasts.courseGpathRequired)
    return
  }

  savingCourse.value = true
  const creating = !selectedCourseId.value
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
      resetContent()
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
    if (creating) {
      courseCreateOpen.value = false
      courseCreateContext.value = null
      courseView.value = "detail"
    }
    await Promise.all([loadCourseDetail(), loadCompleteCourse(), loadMaterials(), loadSupplementaryMaterial()])
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.courseSaveFailed))
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
  } catch (err: any) {
    console.error(err)
    if (err?.status === 409) {
      toast.error((copy.value.toasts as any).coursePublishMissingConfig || "课程发布失败，请检查下方带有「缺少内容」标签的章节或课时并完善配置")
    } else {
      toast.error(apiErrorMessage(err, copy.value.toasts.coursePublishFailed))
    }
  } finally {
    publishing.value = false
  }
}

function deleteCourse() {
  if (!selectedCourseId.value || !selectedCourse.value) return
  if (courseStatusKey(selectedCourse.value) === "deprecated") return
  pendingDeleteCourse.value = selectedCourse.value
  courseDeleteConfirmOpen.value = true
}

function closeCourseDeleteConfirm() {
  if (deletingCourse.value) return
  courseDeleteConfirmOpen.value = false
  pendingDeleteCourse.value = null
}

async function confirmDeleteCourse() {
  const course = pendingDeleteCourse.value
  const id = courseId(course)
  if (!course || !id) return
  deletingCourse.value = true
  try {
    await apiClient(`/api/lms/courses/${encodeURIComponent(id)}?version=${versionOf(course)}`, { method: "DELETE" })
    toast.success(copy.value.toasts.courseDeleted)
    courseDeleteConfirmOpen.value = false
    pendingDeleteCourse.value = null
    clearCourseSelection()
    courseView.value = "list"
    await loadCourses()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.courseDeleteFailed))
  } finally {
    deletingCourse.value = false
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
    toast.error(apiErrorMessage(err, copy.value.toasts.chaptersLoadFailed))
  } finally {
    chaptersLoading.value = false
  }
}

function selectChapterForContext(chapter: JsonRecord, editing = false) {
  selectedChapter.value = chapter
  editingChapterId.value = editing ? chapterId(chapter) : ""
  editingLessonId.value = ""
  lessonForm.value = emptyLessonForm()
  chapterForm.value = {
    title: String(chapter.title || ""),
    sort_order: String(chapter.sort_order || 1),
  }
  void loadLessons()
  newQuiz("chapter")
  void loadQuizzes("chapter")
}

function resetChapterState() {
  selectedChapter.value = null
  editingChapterId.value = ""
  chapterForm.value = emptyChapterForm()
  lessons.value = []
  editingLessonId.value = ""
  lessonForm.value = emptyLessonForm()
  newQuiz("chapter")
  quizzes.value = []
}

function openChapterDetail(chapter: JsonRecord) {
  selectChapterForContext(chapter)
  chapterActiveTab.value = "basic"
  chapterDialogMode.value = "detail"
  chapterDialogOpen.value = true
}

function editChapter(chapter: JsonRecord) {
  selectChapterForContext(chapter, true)
  chapterActiveTab.value = "basic"
  chapterDialogMode.value = "edit"
  chapterDialogOpen.value = true
}

function newChapter() {
  resetChapterState()
  chapterDialogMode.value = "create"
  const maxSort = chapters.value.reduce((max, c) => Math.max(max, Number(c.sort_order) || 0), 0)
  chapterForm.value.sort_order = String(maxSort + 1)
  chapterDialogOpen.value = true
}

function closeChapterDialog() {
  chapterDialogOpen.value = false
}

async function saveChapter() {
  if (!selectedCourseId.value || !chapterForm.value.title.trim()) {
    toast.error(copy.value.toasts.chapterRequired)
    return
  }

  const targetSort = Number(chapterForm.value.sort_order || 1)
  const isConflict = chapters.value.some(c => Number(c.sort_order || 0) === targetSort && chapterId(c) !== editingChapterId.value)
  if (isConflict) {
    toast.error((copy.value.toasts as any)?.duplicateChapterSort || "该排序序号已被其他章节使用，请更换")
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
    await loadChapters()
    resetChapterState()
    closeChapterDialog()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.chapterSaveFailed))
  } finally {
    savingChapter.value = false
  }
}

function deleteChapter(chapter: JsonRecord) {
  pendingDeleteChapter.value = chapter
  chapterDeleteConfirmOpen.value = true
}

function closeChapterDeleteConfirm() {
  if (deletingChapter.value) return
  chapterDeleteConfirmOpen.value = false
  pendingDeleteChapter.value = null
}

async function confirmDeleteChapter() {
  const chapter = pendingDeleteChapter.value
  const id = chapterId(chapter)
  if (!chapter || !id) return
  deletingChapter.value = true
  try {
    await apiClient(`/api/lms/chapters/${encodeURIComponent(id)}?version=${versionOf(chapter)}`, { method: "DELETE" })
    toast.success(copy.value.toasts.chapterDeleted)
    chapterDeleteConfirmOpen.value = false
    pendingDeleteChapter.value = null
    resetChapterState()
    await loadChapters()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.chapterDeleteFailed))
  } finally {
    deletingChapter.value = false
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
    toast.error(apiErrorMessage(err, copy.value.toasts.lessonsLoadFailed))
  } finally {
    lessonsLoading.value = false
  }
}

function editLesson(lesson: JsonRecord, openDialog = true) {
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
  if (openDialog) {
    lessonDialogMode.value = "edit"
    lessonDialogOpen.value = true
  }
}

function newLesson() {
  editingLessonId.value = ""
  lessonForm.value = emptyLessonForm()
  lessonForm.value.chapter_id = selectedChapterId.value
  const targetChapterId = selectedChapterId.value
  if (targetChapterId) {
    const maxSort = allLessonItems.value
      .filter(item => chapterId(item.chapter) === targetChapterId)
      .reduce((max, item) => Math.max(max, Number(item.lesson.sort_order) || 0), 0)
    lessonForm.value.sort_order = String(maxSort + 1)
  } else {
    lessonForm.value.sort_order = "1"
  }
}

function openLessonDetail(lesson: JsonRecord) {
  editLesson(lesson, false)
  lessonActiveTab.value = "basic"
  lessonDialogMode.value = "detail"
  lessonDialogOpen.value = true
}

function openNewLesson() {
  newLesson()
  lessonActiveTab.value = "basic"
  lessonDialogMode.value = "create"
  lessonDialogOpen.value = true
}

function closeLessonDialog() {
  lessonDialogOpen.value = false
}

const lessonFileInput = ref<HTMLInputElement | null>(null)
const uploadingLesson = ref(false)

async function handleLessonFileUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!selectedCourseId.value || !lessonForm.value.chapter_id || !editingLessonId.value) return

  uploadingLesson.value = true
  try {
    const arrayBuffer = await file.arrayBuffer()
    const hashBuffer = await crypto.subtle.digest('SHA-256', arrayBuffer)
    const hashArray = Array.from(new Uint8Array(hashBuffer))
    const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('')

    const uploadUrlReq = {
      upload_type: 3,
      course_ulid: selectedCourseId.value,
      chapter_ulid: lessonForm.value.chapter_id,
      lesson_ulid: editingLessonId.value,
      file_name: file.name,
      content_type: file.type || "application/octet-stream",
      file_hash: hashHex
    }
    const uploadRes = await apiClient<JsonRecord>("/api/lms/upload-url", { method: "POST", body: JSON.stringify(uploadUrlReq) })
    
    if (!uploadRes.upload_url) throw new Error("Missing upload URL")
    const uploadResponse = await fetch(String(uploadRes.upload_url), {
      method: "PUT",
      body: file,
      headers: uploadRes.signed_headers as Record<string, string> || {}
    })
    if (!uploadResponse.ok) {
      throw new Error(`Upload failed: ${uploadResponse.status}`)
    }

    lessonForm.value.asset_object_key = String(uploadRes.object_key)
    lessonForm.value.asset_file_hash = hashHex
    
    await saveLesson()
    toast.success((copy.value.toasts as any)?.uploadSuccess || "课时资产直传并配置成功")
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, (copy.value.toasts as any)?.uploadFailed || "上传失败"))
  } finally {
    uploadingLesson.value = false
    if (lessonFileInput.value) lessonFileInput.value.value = ""
  }
}

async function saveLesson() {
  const targetChapterId = lessonForm.value.chapter_id
  const type = Number(lessonForm.value.lesson_type || 2)
  if (!targetChapterId || !lessonForm.value.title.trim()) {
    toast.error(copy.value.toasts.lessonRequired)
    return
  }
  if (type === 2 && !lessonForm.value.body.trim()) {
    toast.error(copy.value.toasts.lessonBodyRequired)
    return
  }
  if (lessonForm.value.lesson_type === '7' && !lessonForm.value.asset_object_key.trim()) {
    toast.error((copy.value.toasts as any)?.externalUrlRequired || "外部链接不能为空 (External URL required)")
    return
  }

  const targetSort = Number(lessonForm.value.sort_order || 1)
  const isConflict = allLessonItems.value.some(item => chapterId(item.chapter) === targetChapterId && Number(item.lesson.sort_order || 0) === targetSort && lessonId(item.lesson) !== editingLessonId.value)
  if (isConflict) {
    toast.error((copy.value.toasts as any)?.duplicateLessonSort || "该章节下已有相同排序的课时，请更换")
    return
  }

  savingLesson.value = true
  try {
    const body = JSON.stringify({
      chapter_id: targetChapterId,
      title: lessonForm.value.title.trim(),
      sort_order: Number(lessonForm.value.sort_order || 1),
      lesson_type: type,
      body: lessonForm.value.body,
      media_object_key: (type === 7 || type === 2) ? "" : lessonForm.value.asset_object_key.trim(),
      media_file_hash: (type === 7 || type === 2) ? "" : lessonForm.value.asset_file_hash.trim(),
      external_url: type === 7 ? lessonForm.value.asset_object_key.trim() : "",
      version: selectedLessonRecord.value?.version || 0,
    })
    if (editingLessonId.value) {
      await apiClient(`/api/lms/lessons/${encodeURIComponent(editingLessonId.value)}`, { method: "PUT", body })
      toast.success(copy.value.toasts.lessonUpdated)
      await Promise.all([loadLessons(), loadCompleteCourse(), loadCourseDetail()])
      newLesson()
      closeLessonDialog()
    } else {
      const res = await apiClient<JsonRecord>(`/api/lms/chapters/${encodeURIComponent(targetChapterId)}/lessons`, { method: "POST", body })
      toast.success(copy.value.toasts.lessonCreated)
      if ([1, 3, 4, 5, 6].includes(type)) {
        editingLessonId.value = String(res.lesson_ulid)
        toast.info("请继续点击下方按钮上传课时文件 (视频/PDF等)")
        await Promise.all([loadLessons(), loadCompleteCourse(), loadCourseDetail()])
      } else {
        await Promise.all([loadLessons(), loadCompleteCourse(), loadCourseDetail()])
        newLesson()
        closeLessonDialog()
      }
    }
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.lessonSaveFailed))
  } finally {
    savingLesson.value = false
  }
}

function deleteLesson(lesson: JsonRecord) {
  pendingDeleteLesson.value = lesson
  lessonDeleteConfirmOpen.value = true
}

function closeLessonDeleteConfirm() {
  if (deletingLesson.value) return
  lessonDeleteConfirmOpen.value = false
  pendingDeleteLesson.value = null
}

async function confirmDeletePendingLesson() {
  const lesson = pendingDeleteLesson.value
  const id = lessonId(lesson)
  if (!lesson || !id) return
  deletingLesson.value = true
  try {
    await apiClient(`/api/lms/lessons/${encodeURIComponent(id)}?version=${versionOf(lesson)}`, { method: "DELETE" })
    toast.success(copy.value.toasts.lessonDeleted)
    lessonDeleteConfirmOpen.value = false
    pendingDeleteLesson.value = null
    await Promise.all([loadLessons(), loadCompleteCourse(), loadCourseDetail()])
    newLesson()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.lessonDeleteFailed))
  } finally {
    deletingLesson.value = false
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
    toast.error(apiErrorMessage(err, copy.value.toasts.materialsLoadFailed))
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

function resetMaterialEditor() {
  selectedMaterial.value = null
  editingMaterialId.value = ""
  materialForm.value = emptyMaterialForm()
  if (materialFileInput.value) materialFileInput.value.value = ""
}

function viewMaterial(material: JsonRecord) {
  selectedMaterial.value = material
  editingMaterialId.value = ""
  materialDialogMode.value = "detail"
  materialDialogOpen.value = true
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
  materialDialogMode.value = "edit"
  materialDialogOpen.value = true
}

function newMaterial() {
  resetMaterialEditor()
  const maxSort = materials.value.reduce((max, m) => Math.max(max, Number(m.sort_order) || 0), 0)
  materialForm.value.sort_order = String(maxSort + 1)
  materialDialogMode.value = "create"
  materialDialogOpen.value = true
}

function closeMaterialDialog() {
  if (savingMaterial.value || uploadingMaterial.value) return
  materialDialogOpen.value = false
  resetMaterialEditor()
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

function openDetailDeleteConfirm(deleteInfo: PendingDetailDelete) {
  pendingDetailDelete.value = deleteInfo
  detailDeleteConfirmOpen.value = true
}

function closeDetailDeleteConfirm() {
  if (deletingDetail.value) return
  detailDeleteConfirmOpen.value = false
  pendingDetailDelete.value = null
}

async function saveSupplementaryItem() {
  if (!selectedCourseId.value) return
  if (!supplementaryItemForm.value.title.trim()) {
    toast.error(copy.value.toasts.materialTitleRequired)
    return
  }
  const type = supplementaryItemForm.value.type
  if (isSupplementaryLinkType(type) && !supplementaryItemForm.value.url.trim()) {
    toast.error(copy.value.toasts.materialUrlRequired)
    return
  }
  if (isSupplementaryAssetRequired(type) && !supplementaryItemForm.value.url.trim()) {
    toast.error((copy.value.toasts as any)?.externalUrlRequired || "外部链接不能为空 (URL required)")
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

function deleteSupplementaryItem() {
  if (editingSupplementaryItemIndex.value < 0) return
  const title = supplementaryItemForm.value.title || copy.value.fallbacks.material
  openDetailDeleteConfirm({
    kind: "supplementaryItem",
    title,
    description: copy.value.confirmDeleteMaterial(title),
    index: editingSupplementaryItemIndex.value,
  })
}

async function confirmDeleteSupplementaryItem(deleteInfo: PendingDetailDelete) {
  if (typeof deleteInfo.index !== "number" || deleteInfo.index < 0) return
  const records = supplementaryEditableRecords()
  records.splice(deleteInfo.index, 1)
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
    toast.error(apiErrorMessage(err, copy.value.toasts.supplementarySaveFailed))
  } finally {
    savingSupplementaryMaterial.value = false
  }
}

function deleteSupplementaryMaterial() {
  const id = supplementaryMaterialId(supplementaryMaterial.value)
  if (!selectedCourseId.value || !id) return
  openDetailDeleteConfirm({
    kind: "supplementaryConfig",
    title: supplementaryMaterialForm.value.kind || copy.value.supplementaryTitle,
    description: copy.value.confirmDeleteSupplementaryConfig,
    id,
    version: versionOf(supplementaryMaterial.value),
  })
}

async function confirmDeleteSupplementaryMaterial(deleteInfo: PendingDetailDelete) {
  if (!selectedCourseId.value || !deleteInfo.id) return
  try {
    await apiClient(`/api/lms/courses/${encodeURIComponent(selectedCourseId.value)}/supplementary-material/${encodeURIComponent(deleteInfo.id)}?version=${deleteInfo.version || 0}`, { method: "DELETE" })
    toast.success(copy.value.toasts.supplementaryDeleted)
    await Promise.all([loadSupplementaryMaterial(), loadCompleteCourse(), loadCourseDetail()])
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.supplementaryDeleteFailed))
    throw err
  }
}

const materialFileInput = ref<HTMLInputElement | null>(null)
const uploadingMaterial = ref(false)

async function handleMaterialFileUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!selectedCourseId.value || !editingMaterialId.value) return

  uploadingMaterial.value = true
  try {
    const arrayBuffer = await file.arrayBuffer()
    const hashBuffer = await crypto.subtle.digest('SHA-256', arrayBuffer)
    const hashArray = Array.from(new Uint8Array(hashBuffer))
    const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('')

    const uploadUrlReq = {
      upload_type: 2,
      course_ulid: selectedCourseId.value,
      material_ulid: editingMaterialId.value,
      file_name: file.name,
      content_type: file.type || "application/octet-stream",
      file_hash: hashHex
    }
    const uploadRes = await apiClient<JsonRecord>("/api/lms/upload-url", { method: "POST", body: JSON.stringify(uploadUrlReq) })
    
    if (!uploadRes.upload_url) throw new Error("Missing upload URL")
    const uploadResponse = await fetch(String(uploadRes.upload_url), {
      method: "PUT",
      body: file,
      headers: uploadRes.signed_headers as Record<string, string> || {}
    })
    if (!uploadResponse.ok) {
      throw new Error(`Upload failed: ${uploadResponse.status}`)
    }

    materialForm.value.file_object_key = String(uploadRes.object_key)
    materialForm.value.file_hash = hashHex
    materialForm.value.file_size = String(file.size)
    
    await saveMaterial()
    toast.success((copy.value.toasts as any)?.uploadSuccess || "文件直传并配置成功")
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, (copy.value.toasts as any)?.uploadFailed || "上传失败"))
  } finally {
    uploadingMaterial.value = false
    if (materialFileInput.value) materialFileInput.value.value = ""
  }
}

const supplementaryFileInput = ref<HTMLInputElement | null>(null)
const uploadingSupplementary = ref(false)

async function handleSupplementaryFileUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!selectedCourseId.value) return

  uploadingSupplementary.value = true
  try {
    const arrayBuffer = await file.arrayBuffer()
    const hashBuffer = await crypto.subtle.digest('SHA-256', arrayBuffer)
    const hashArray = Array.from(new Uint8Array(hashBuffer))
    const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('')

    const uploadUrlReq = {
      upload_type: 1,
      course_ulid: selectedCourseId.value,
      file_name: file.name,
      content_type: file.type || "application/octet-stream",
      file_hash: hashHex
    }
    const uploadRes = await apiClient<JsonRecord>("/api/lms/upload-url", { method: "POST", body: JSON.stringify(uploadUrlReq) })
    
    if (!uploadRes.upload_url) throw new Error("Missing upload URL")
    const uploadResponse = await fetch(String(uploadRes.upload_url), {
      method: "PUT",
      body: file,
      headers: uploadRes.signed_headers as Record<string, string> || {}
    })
    if (!uploadResponse.ok) {
      throw new Error(`Upload failed: ${uploadResponse.status}`)
    }

    supplementaryItemForm.value.url = String(uploadRes.object_key)
    
    await saveSupplementaryItem()
    toast.success((copy.value.toasts as any)?.uploadSuccess || "上传成功")
  } catch (err) {
    console.error(err)
    toast.error((copy.value.toasts as any)?.uploadFailed || "上传失败")
  } finally {
    uploadingSupplementary.value = false
    if (supplementaryFileInput.value) supplementaryFileInput.value.value = ""
  }
}

async function saveMaterial() {
  if (!selectedCourseId.value || !materialForm.value.title.trim()) {
    toast.error(copy.value.toasts.materialFileRequired || "Title required")
    return
  }

  const targetSort = Number(materialForm.value.sort_order || 1)
  const isConflict = materials.value.some(m => Number(m.sort_order || 0) === targetSort && materialId(m) !== editingMaterialId.value)
  if (isConflict) {
    toast.error((copy.value.toasts as any)?.duplicateMaterialSort || "该课程下已有相同排序的资料，请更换")
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
    materialDialogOpen.value = false
    resetMaterialEditor()
    await Promise.all([loadMaterials(), loadCourseDetail(), loadCompleteCourse()])
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.materialSaveFailed))
  } finally {
    savingMaterial.value = false
  }
}

function deleteMaterial(material: JsonRecord) {
  const id = materialId(material)
  if (!id) return
  const title = materialTitle(material)
  openDetailDeleteConfirm({
    kind: "material",
    title,
    description: copy.value.confirmDeleteMaterial(title),
    id,
    version: versionOf(material),
  })
}

async function confirmDeleteMaterial(deleteInfo: PendingDetailDelete) {
  if (!deleteInfo.id) return
  try {
    await apiClient(`/api/lms/materials/${encodeURIComponent(deleteInfo.id)}?version=${deleteInfo.version || 0}`, { method: "DELETE" })
    toast.success(copy.value.toasts.materialDeleted)
    if (deleteInfo.id === selectedMaterialId.value) {
      materialDialogOpen.value = false
      resetMaterialEditor()
    }
    await Promise.all([loadMaterials(), loadCourseDetail(), loadCompleteCourse()])
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.materialDeleteFailed))
    throw err
  }
}

function defaultQuizOwnerId(scope: QuizScope) {
  if (scope === "course") return selectedCourseId.value
  if (scope === "chapter") {
    const current = selectedChapterId.value
    return current && quizChapterOptions.value.some((chapter) => chapterId(chapter) === current) ? current : ""
  }
  const current = editingLessonId.value
  return current && quizLessonOptions.value.some((item) => lessonId(item.lesson) === current) ? current : ""
}

function quizTarget(scope: QuizScope = quizForm.value.scope, ownerId = quizForm.value.owner_id) {
  if (scope === "course") {
    return { type: 3, id: selectedCourseId.value, label: copy.value.fallbacks.course, title: courseTitle(selectedCourse.value) }
  }
  if (scope === "chapter") {
    const chapter = chapterOptionById(ownerId)
    return { type: 2, id: ownerId, label: copy.value.fallbacks.chapter, title: chapterTitle(chapter) }
  }
  const lessonItem = lessonItemById(ownerId)
  return { type: 1, id: ownerId, label: copy.value.fallbacks.lesson, title: lessonTitle(lessonItem?.lesson) }
}

function quizTargetChapterTitle() {
  if (selectedQuizItem.value?.chapter) return chapterTitle(selectedQuizItem.value.chapter)
  if (quizForm.value.scope === "chapter") return chapterTitle(chapterOptionById(quizForm.value.owner_id))
  if (quizForm.value.scope === "lesson") return chapterTitle(lessonItemById(quizForm.value.owner_id)?.chapter)
  return "-"
}

function quizTargetLessonTitle() {
  if (selectedQuizItem.value?.lesson) return lessonTitle(selectedQuizItem.value.lesson)
  if (quizForm.value.scope === "lesson") return lessonTitle(lessonItemById(quizForm.value.owner_id)?.lesson)
  return "-"
}

function clearQuestionState() {
  questionDialogOpen.value = false
  questionDialogMode.value = "detail"
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
    toast.error(apiErrorMessage(err, copy.value.toasts.quizzesLoadFailed))
  } finally {
    quizzesLoading.value = false
  }
}

async function refreshQuizBank() {
  if (!selectedCourseId.value) return
  quizzesLoading.value = true
  try {
    quizzes.value = []
    selectedQuiz.value = null
    editingQuizId.value = ""
    clearQuestionState()
    await loadCompleteCourse()
  } finally {
    quizzesLoading.value = false
  }
}

function editQuiz(quiz: JsonRecord, openDialog = true) {
  const item = allQuizItems.value.find((entry) => quizId(entry.quiz) === quizId(quiz))
  const scope = scopeFromQuizzableType(item?.ownerType || quiz.quizzable_type || quizForm.value.scope)
  if (item?.chapter) selectedChapter.value = item.chapter
  if (item?.lesson) editLesson(item.lesson, false)
  selectedQuiz.value = quiz
  editingQuizId.value = quizId(quiz)
  quizForm.value = {
    scope,
    owner_id: String(quiz.quizzable_ulid || quiz.quizzable_id || (scope === "course" ? selectedCourseId.value : scope === "chapter" ? chapterId(item?.chapter) : lessonId(item?.lesson)) || ""),
    title: String(quiz.title || ""),
    description: String(quiz.description || ""),
    passing_score: String(quiz.passing_score || 70),
    time_limit: String(quiz.time_limit || 0),
    randomize_questions: Boolean(quiz.randomize_questions),
  }
  questionDialogOpen.value = false
  void loadQuestions()
  if (openDialog) {
    quizDialogMode.value = "edit"
    quizDialogOpen.value = true
  }
}

function newQuiz(scope: QuizScope = "course") {
  selectedQuiz.value = null
  editingQuizId.value = ""
  quizForm.value = { ...emptyQuizForm(), scope, owner_id: defaultQuizOwnerId(scope) }
  clearQuestionState()
}

function openQuizDetail(quiz: JsonRecord) {
  editQuiz(quiz, false)
  quizActiveTab.value = "basic"
  quizDialogMode.value = "detail"
  quizDialogOpen.value = true
}

function openNewQuiz() {
  newQuiz()
  quizActiveTab.value = "basic"
  quizDialogMode.value = "create"
  quizDialogOpen.value = true
}

function changeQuizFormScope() {
  quizForm.value.owner_id = defaultQuizOwnerId(quizForm.value.scope)
}

function closeQuizDialog() {
  quizDialogOpen.value = false
  questionDialogOpen.value = false
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
      const scope = quizForm.value.scope
      newQuiz(scope)
      closeQuizDialog()
      await Promise.all([loadQuizzes(scope), loadCourseDetail(), loadCompleteCourse()])
    } else {
      const created = await apiClient<JsonRecord>("/api/lms/quizzes", { method: "POST", body: JSON.stringify(body) })
      const createdId = quizId(created)
      const scope = quizForm.value.scope
      await Promise.all([loadQuizzes(scope), loadCourseDetail(), loadCompleteCourse()])
      const createdQuiz = allQuizItems.value.find((item) => quizId(item.quiz) === createdId)?.quiz
        || quizzes.value.find((item) => quizId(item) === createdId)
        || { ...body, quiz_id: createdId, quiz_ulid: createdId, version: 1 }
      selectedQuiz.value = createdQuiz
      editingQuizId.value = createdId
      quizDialogMode.value = "edit"
      quizDialogOpen.value = true
      await loadQuestions(createdId)
      newQuestion()
      toast.success(copy.value.toasts.quizCreatedAddQuestions)
    }
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.quizSaveFailed))
  } finally {
    savingQuiz.value = false
  }
}

function deleteQuiz(quiz: JsonRecord) {
  const id = quizId(quiz)
  if (!id) return
  const title = quizTitle(quiz)
  openDetailDeleteConfirm({
    kind: "quiz",
    title,
    description: copy.value.confirmDeleteQuiz(title),
    id,
    version: versionOf(quiz),
  })
}

async function confirmDeleteQuiz(deleteInfo: PendingDetailDelete) {
  if (!deleteInfo.id) return
  try {
    await apiClient(`/api/lms/quizzes/${encodeURIComponent(deleteInfo.id)}?version=${deleteInfo.version || 0}`, { method: "DELETE" })
    toast.success(copy.value.toasts.quizDeleted)
    const scope = quizForm.value.scope
    newQuiz(scope)
    await Promise.all([loadQuizzes(scope), loadCourseDetail(), loadCompleteCourse()])
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.quizDeleteFailed))
    throw err
  }
}

async function loadQuestions(id = selectedQuizId.value) {
  if (!id) return
  questionsLoading.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/lms/quizzes/${encodeURIComponent(id)}/questions`)
    let list = Array.isArray(data.questions) ? data.questions : []
    list = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    
    const details = await Promise.all(list.map(async (item) => {
      try {
        const detail = await apiClient<JsonRecord>(`/api/lms/questions/${encodeURIComponent(String(item.question_ulid || item.id))}`)
        return detail.question && typeof detail.question === "object" ? { ...item, ...detail.question } : item
      } catch (e) {
        return item
      }
    }))
    questions.value = details
    selectedQuestion.value = null
    options.value = []
    editingQuestionId.value = ""
    editingOptionId.value = ""
    questionForm.value = emptyQuestionForm()
    optionForm.value = emptyOptionForm()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.questionsLoadFailed))
  } finally {
    questionsLoading.value = false
  }
}

function editQuestion(question: JsonRecord) {
  selectedQuestion.value = question
  editingQuestionId.value = questionId(question)
  questionDialogMode.value = "edit"
  questionDialogOpen.value = true
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

function resetQuestionEditor() {
  selectedQuestion.value = null
  editingQuestionId.value = ""
  questionForm.value = emptyQuestionForm()
  const maxSort = questions.value.reduce((max, q) => Math.max(max, Number(q.sort_order) || 0), 0)
  questionForm.value.sort_order = String(maxSort + 1)
  options.value = []
  editingOptionId.value = ""
  optionForm.value = emptyOptionForm()
}

function newQuestion() {
  resetQuestionEditor()
  questionDialogMode.value = "create"
  questionDialogOpen.value = true
}

function viewQuestion(question: JsonRecord) {
  selectedQuestion.value = question
  editingQuestionId.value = ""
  editingOptionId.value = ""
  questionForm.value = emptyQuestionForm()
  optionForm.value = emptyOptionForm()
  questionDialogMode.value = "detail"
  questionDialogOpen.value = true
  void loadOptions()
}

function closeQuestionDialog() {
  questionDialogOpen.value = false
}

function openMediaConfig() {
  try {
    parsedMediaItems.value = JSON.parse(questionForm.value.media_items_json || "[]")
    if (!Array.isArray(parsedMediaItems.value)) parsedMediaItems.value = []
  } catch (e) {
    parsedMediaItems.value = []
  }
  advancedMediaDialogOpen.value = true
}

function saveMediaConfig() {
  questionForm.value.media_items_json = JSON.stringify(parsedMediaItems.value)
  advancedMediaDialogOpen.value = false
}

function addMediaItem() {
  parsedMediaItems.value.push({ type: "image", url: "" })
}

async function saveQuestion() {
  if (!selectedQuizId.value || !questionForm.value.question_text.trim()) {
    toast.error(copy.value.toasts.questionRequired)
    return
  }
  const targetSort = Number(questionForm.value.sort_order || 1)
  const isConflict = questions.value.some(q => Number(q.sort_order || 0) === targetSort && questionId(q) !== editingQuestionId.value)
  if (isConflict) {
    toast.error((copy.value.toasts as any)?.duplicateQuestionSort || "该课检下已有相同排序的题目，请更换")
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
    await loadQuestions()
    resetQuestionEditor()
    questionDialogOpen.value = false
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.questionSaveFailed))
  } finally {
    savingQuestion.value = false
  }
}

function deleteQuestion(question: JsonRecord) {
  const id = questionId(question)
  if (!id) return
  const title = questionTitle(question)
  openDetailDeleteConfirm({
    kind: "question",
    title,
    description: copy.value.confirmDeleteQuestion(title),
    id,
    version: versionOf(question),
  })
}

async function confirmDeleteQuestion(deleteInfo: PendingDetailDelete) {
  if (!deleteInfo.id) return
  try {
    await apiClient(`/api/lms/questions/${encodeURIComponent(deleteInfo.id)}?version=${deleteInfo.version || 0}`, { method: "DELETE" })
    toast.success(copy.value.toasts.questionDeleted)
    await loadQuestions()
    resetQuestionEditor()
    questionDialogOpen.value = false
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.questionDeleteFailed))
    throw err
  }
}

async function loadOptions(id = selectedQuestionId.value) {
  if (!id) return
  optionsLoading.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/lms/questions/${encodeURIComponent(id)}/options`)
    let list = Array.isArray(data.options) ? data.options : []
    list = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))

    const details = await Promise.all(list.map(async (item) => {
      try {
        const detail = await apiClient<JsonRecord>(`/api/lms/options/${encodeURIComponent(String(item.option_ulid || item.id))}`)
        return detail.option && typeof detail.option === "object" ? { ...item, ...detail.option } : item
      } catch (e) {
        return item
      }
    }))
    options.value = details
    newOption()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.optionsLoadFailed))
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
  const maxSort = options.value.reduce((max, o) => Math.max(max, Number(o.sort_order) || 0), 0)
  optionForm.value.sort_order = String(maxSort + 1)
}

async function saveOption() {
  if (!selectedQuestionId.value || !optionForm.value.option_text.trim()) {
    toast.error(copy.value.toasts.optionRequired)
    return
  }
  const targetSort = Number(optionForm.value.sort_order || 1)
  const isConflict = options.value.some(o => Number(o.sort_order || 0) === targetSort && optionId(o) !== editingOptionId.value)
  if (isConflict) {
    toast.error((copy.value.toasts as any)?.duplicateOptionSort || "该题目下已有相同排序的选项，请更换")
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
    await loadOptions()
    newOption()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.optionSaveFailed))
  } finally {
    savingOption.value = false
  }
}

function deleteOption(option: JsonRecord) {
  const id = optionId(option)
  if (!id) return
  const title = optionTitle(option)
  openDetailDeleteConfirm({
    kind: "option",
    title,
    description: copy.value.confirmDeleteOption(title),
    id,
    version: versionOf(option),
  })
}

async function confirmDeleteOption(deleteInfo: PendingDetailDelete) {
  if (!deleteInfo.id) return
  try {
    await apiClient(`/api/lms/options/${encodeURIComponent(deleteInfo.id)}?version=${deleteInfo.version || 0}`, { method: "DELETE" })
    toast.success(copy.value.toasts.optionDeleted)
    await loadOptions()
    newOption()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.optionDeleteFailed))
    throw err
  }
}

async function confirmDetailDelete() {
  const deleteInfo = pendingDetailDelete.value
  if (!deleteInfo) return
  deletingDetail.value = true
  try {
    if (deleteInfo.kind === "supplementaryItem") await confirmDeleteSupplementaryItem(deleteInfo)
    else if (deleteInfo.kind === "supplementaryConfig") await confirmDeleteSupplementaryMaterial(deleteInfo)
    else if (deleteInfo.kind === "material") await confirmDeleteMaterial(deleteInfo)
    else if (deleteInfo.kind === "quiz") await confirmDeleteQuiz(deleteInfo)
    else if (deleteInfo.kind === "question") await confirmDeleteQuestion(deleteInfo)
    else if (deleteInfo.kind === "option") await confirmDeleteOption(deleteInfo)
    detailDeleteConfirmOpen.value = false
    pendingDetailDelete.value = null
  } catch {
    // Each delete helper shows its own toast and keeps the confirmation open.
  } finally {
    deletingDetail.value = false
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
    toast.error(apiErrorMessage(err, copy.value.toasts.importFailed))
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

watch(() => courseForm.value.title, (newTitle, oldTitle) => {
  if (!isCreatingCourse.value) return
  const generatePath = (t: string) => t ? `/gcc/pipeline/core/${t.trim().toLowerCase().replace(/\s+/g, '_')}` : ''
  const newPath = generatePath(newTitle)
  const oldPath = generatePath(oldTitle)
  
  if (!courseForm.value.respath || courseForm.value.respath === oldPath) {
    courseForm.value.respath = newPath
  }
  if (!courseForm.value.course_gpath || courseForm.value.course_gpath === oldPath) {
    courseForm.value.course_gpath = newPath
  }
})

onMounted(() => {
  void loadCourses()
})
</script>

<template>
  <div class="space-y-5 px-4 py-5 md:space-y-6 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div class="min-w-0">
        <h1 class="text-3xl font-black tracking-tight md:text-4xl">{{ copy.title }}</h1>
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
        <button v-if="!isCreatingCourse" class="inline-flex items-center gap-2 rounded-xl bg-blue-700 px-4 py-3 font-bold text-white shadow-sm" type="button" @click="newCourse">
          <Plus class="h-4 w-4" />
          {{ copy.newCourse }}
        </button>
      </div>
    </header>

    <Teleport to="body">
      <section v-if="courseCreateOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-0 md:p-6" role="dialog" aria-modal="true">
        <div class="flex h-full max-h-none w-full max-w-[980px] flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-6 md:py-5">
            <div class="min-w-0">
              <h2 class="text-xl font-black text-slate-950 md:text-2xl">{{ copy.newCourse }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.fillCourseHint }}</p>
            </div>
            <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900 disabled:cursor-not-allowed disabled:opacity-50" type="button" :aria-label="copy.close" :disabled="savingCourse" @click="closeCourseCreate">
              <X class="h-5 w-5" />
            </button>
          </div>

          <form class="grid min-h-0 flex-1 gap-4 overflow-y-auto p-4 md:grid-cols-2 md:p-6" @submit.prevent="saveCourse">
            <label class="block">
              <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.courseTitle }}</span>
              <input v-model="courseForm.title" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <label v-if="false" class="block">
              <span class="text-sm font-bold">{{ copy.categoryTips }}</span>
              <input v-model="courseForm.category_tips" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <label class="block md:col-span-2">
              <span class="text-sm font-bold">{{ copy.description }}</span>
              <textarea v-model="courseForm.description" class="mt-2 min-h-20 w-full rounded-xl border border-slate-200 px-3 py-2" />
            </label>
            <details class="group md:col-span-2">
              <summary class="inline-flex cursor-pointer select-none items-center gap-1 rounded-lg text-sm font-bold text-slate-500 transition-colors hover:text-slate-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2">
                {{ (copy as any).advancedConfig || '高级配置' }}
                <ChevronDown class="h-4 w-4 transition-transform group-open:rotate-180" />
              </summary>
              <div class="mt-4 grid gap-3 md:grid-cols-2">
                <label class="block">
                  <span class="flex items-center gap-1.5 text-sm font-bold">
                    <span><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.respath }}</span>
                    <span class="group/tooltip relative inline-flex cursor-help rounded-full text-slate-600 outline-none transition-colors hover:text-slate-900 focus-visible:text-slate-900 focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-1" tabindex="0" :aria-label="copy.respathHint">
                      <Info class="h-4 w-4" aria-hidden="true" />
                      <span role="tooltip" class="pointer-events-none absolute bottom-full left-0 z-30 mb-2 w-72 max-w-[calc(100vw-2rem)] rounded-md bg-slate-900 px-3 py-2 text-xs font-medium leading-5 text-white opacity-0 shadow-lg transition-opacity group-hover/tooltip:opacity-100 group-focus/tooltip:opacity-100">
                        {{ copy.respathHint }}
                      </span>
                    </span>
                  </span>
                  <input v-model="courseForm.respath" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="/gcc/pipeline/..." />
                </label>
                <label class="block">
                  <span class="flex items-center gap-1.5 text-sm font-bold">
                    <span><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.courseGpath }}</span>
                    <span class="group/tooltip relative inline-flex cursor-help rounded-full text-slate-600 outline-none transition-colors hover:text-slate-900 focus-visible:text-slate-900 focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-1" tabindex="0" :aria-label="copy.courseGpathHint">
                      <Info class="h-4 w-4" aria-hidden="true" />
                      <span role="tooltip" class="pointer-events-none absolute bottom-full left-0 z-30 mb-2 w-72 max-w-[calc(100vw-2rem)] rounded-md bg-slate-900 px-3 py-2 text-xs font-medium leading-5 text-white opacity-0 shadow-lg transition-opacity group-hover/tooltip:opacity-100 group-focus/tooltip:opacity-100">
                        {{ copy.courseGpathHint }}
                      </span>
                    </span>
                  </span>
                  <input v-model="courseForm.course_gpath" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="/gcc/pipeline/..." />
                  <p v-if="duplicateGpathWarning" class="mt-2 text-xs font-semibold text-red-500">{{ copy.duplicateGpathWarning }}</p>
                </label>
              </div>
            </details>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.durationMin }}</span>
              <input v-model="courseForm.duration_min" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" type="number" min="0" />
            </label>
            <label v-if="false" class="block">
              <span class="text-sm font-bold">{{ copy.thumbnailObjectKey }}</span>
              <input v-model="courseForm.thumbnail_object_key" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <label v-if="false" class="block">
              <span class="text-sm font-bold">{{ copy.thumbnailFileHash }}</span>
              <input v-model="courseForm.thumbnail_file_hash" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <div class="flex flex-col gap-3 sm:flex-row md:col-span-2">
              <button class="inline-flex h-10 w-full items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white disabled:opacity-50 sm:w-auto" :disabled="savingCourse" type="submit">
                <Loader2 v-if="savingCourse" class="h-4 w-4 animate-spin" />
                <Save v-else class="h-4 w-4" />
                {{ savingCourse ? copy.saving : copy.saveCourse }}
              </button>
              <button v-if="canPublishSelectedCourse" class="h-10 w-full rounded-xl border px-4 font-bold disabled:opacity-40 sm:w-auto" :disabled="!selectedCourseId || publishing" type="button" @click="publishCourse">
                {{ publishing ? copy.publishing : copy.publishCourse }}
              </button>
            </div>
          </form>
        </div>
      </section>
    </Teleport>

    <Teleport to="body">
      <section v-if="courseDetailDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-0 md:p-6" role="dialog" aria-modal="true">
        <div class="flex h-full max-h-none w-full max-w-[980px] flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-6 md:py-5">
            <div class="min-w-0">
              <h2 class="text-xl font-black text-slate-950 md:text-2xl">{{ copy.courseTopData }}</h2>
              <p class="mt-1 break-all text-sm text-slate-500">{{ courseDetailDialogCourseId || "-" }}</p>
            </div>
            <div class="flex shrink-0 items-center gap-3">
              <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeCourseDetailDialog">
                <X class="h-5 w-5" />
              </button>
            </div>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto p-4 md:p-6">
            <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
              <div class="rounded-xl bg-slate-50 p-3">
                <div class="text-xs font-black uppercase text-slate-400">{{ copy.stats.chapters }}</div>
                <div class="mt-1 text-xl font-black">{{ courseDetailDialogChapterCount }}</div>
              </div>
              <div class="rounded-xl bg-slate-50 p-3">
                <div class="text-xs font-black uppercase text-slate-400">{{ copy.stats.lessons }}</div>
                <div class="mt-1 text-xl font-black">{{ courseDetailDialogLessonCount }}</div>
              </div>
              <div class="rounded-xl bg-slate-50 p-3">
                <div class="text-xs font-black uppercase text-slate-400">{{ copy.stats.quizzes }}</div>
                <div class="mt-1 text-xl font-black">{{ courseDetailDialogQuizCount }}</div>
              </div>
              <div class="rounded-xl bg-slate-50 p-3">
                <div class="text-xs font-black uppercase text-slate-400">{{ copy.stats.materials }}</div>
                <div class="mt-1 text-xl font-black">{{ courseDetailDialogMaterialCount }}</div>
              </div>
            </div>

            <div class="mt-4 rounded-xl border border-slate-200 p-4">
              <h3 class="font-black">{{ copy.readonlyFields }}</h3>
              <p class="mt-1 text-xs text-slate-500">{{ copy.readonlyFieldsHint }}</p>
              <div class="mt-3 max-h-[56vh] space-y-3 overflow-y-auto overscroll-contain pr-0 md:pr-2">
                <ReadonlyField v-for="entry in courseRecordEntries(courseDetailTarget)" :key="`course-dialog-${entry.key}`" :label="courseReadonlyFieldLabel(entry.key)" :text="entry.value" min-height="48px" />
                <div v-if="courseDetailDialogLoading" class="flex items-center gap-2 rounded-xl bg-slate-50 px-4 py-3 text-sm font-semibold text-slate-500">
                  <Loader2 class="h-4 w-4 animate-spin" />
                  {{ copy.loadingCompleteCourse }}
                </div>
                <ReadonlyField v-for="entry in courseRecordEntries(courseDetailDialogDetail)" :key="`course-dialog-detail-${entry.key}`" :label="courseDetailReadonlyFieldLabel(entry.key)" :text="entry.value" min-height="48px" />
                <ReadonlyField v-for="entry in courseRecordEntries(courseDetailDialogComplete)" :key="`course-dialog-complete-${entry.key}`" :label="completeCourseReadonlyFieldLabel(entry.key)" :text="entry.value" min-height="48px" />
              </div>
            </div>
          </div>
        </div>
      </section>
    </Teleport>

    <section v-if="courseView === 'list'" class="rounded-2xl border border-slate-200 bg-white shadow-sm md:rounded-3xl">
      <div class="grid gap-3 border-b border-slate-200 bg-slate-50/60 p-4 lg:grid-cols-[1fr_auto]">
        <div class="relative min-w-0">
          <input v-model="categoryFilter" class="h-10 w-full rounded-xl border border-slate-200 bg-white px-4 pr-10 text-sm shadow-sm outline-none transition focus:border-sky-300 focus:ring-2 focus:ring-sky-100" :placeholder="copy.categoryPlaceholder" />
          <button
            v-if="categoryFilter"
            type="button"
            class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
            :aria-label="copy.clearCategoryFilter"
            :title="copy.clearCategoryFilter"
            @click="categoryFilter = ''"
          >
            <X class="h-4 w-4" />
          </button>
        </div>
        <label class="inline-flex h-10 items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 text-sm font-bold text-slate-600 shadow-sm">
          <input v-model="publishedOnly" type="checkbox" />
          {{ copy.publishedOnly }}
        </label>
      </div>

      <div v-if="loading && !courses.length" class="px-4 py-10 text-center text-slate-500 md:p-12">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        {{ copy.loading }}
      </div>
      <div v-else-if="!courses.length" class="px-4 py-10 text-center text-slate-500 md:p-12">{{ copy.emptyCourses }}</div>
      <div v-else>
        <div class="hidden grid-cols-[minmax(0,1fr)_120px_260px_120px_180px] gap-6 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-400 lg:grid">
          <span>{{ copy.columns.course }}</span>
          <span>{{ copy.columns.version }}</span>
          <span>{{ copy.columns.updatedAt }}</span>
          <span class="text-right">{{ copy.columns.status }}</span>
          <span class="text-right">{{ copy.columns.action }}</span>
        </div>
        <div
          v-for="course in courses"
          :key="courseId(course)"
          class="block w-full cursor-pointer border-b border-slate-100 px-4 py-4 text-left transition last:border-b-0 hover:bg-slate-50 md:px-5 md:py-3"
          :class="courseId(course) === selectedCourseId ? 'bg-sky-50/70' : ''"
          role="button"
          tabindex="0"
          @click="selectCourse(course)"
          @keyup.enter="selectCourse(course)"
        >
          <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_120px_260px_120px_180px] lg:items-center lg:gap-6">
            <div class="min-w-0">
              <div class="break-words text-base font-black text-slate-950 lg:truncate">{{ courseTitle(course) }}</div>
              <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500">
                <span class="max-w-full break-words">{{ course.category_tips || copy.uncategorized }}</span>
                <span class="max-w-full break-all font-mono">ID: {{ courseId(course) || "-" }}</span>
              </div>
            </div>
            <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 text-sm font-bold text-slate-700 lg:block lg:rounded-none lg:bg-transparent lg:p-0">
              <span class="text-xs font-bold text-slate-400 lg:hidden">{{ copy.columns.version }}</span>
              <span>{{ course.version || 0 }}</span>
            </div>
            <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 text-sm text-slate-500 lg:block lg:rounded-none lg:bg-transparent lg:p-0">
              <span class="text-xs font-bold text-slate-400 lg:hidden">{{ copy.updatedShort }}</span>
              <span>{{ formatDate(String(course.updated_at || course.created_at || "")) }}</span>
            </div>
            <span class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 lg:block lg:justify-self-end lg:rounded-none lg:bg-transparent lg:p-0">
              <span class="text-xs font-bold text-slate-400 lg:hidden">{{ copy.columns.status }}</span>
              <span class="inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(courseStatusBadgeValue(course))">
                {{ courseStatusLabel(course) }}
              </span>
            </span>
            <div class="flex flex-col gap-3 sm:flex-row lg:items-center lg:justify-end">
              <button class="inline-flex items-center justify-center rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm font-bold text-[#1890ff] transition hover:underline lg:border-0 lg:bg-transparent lg:px-0 lg:py-0" type="button" @click.stop="openCourseDetailDialog(course)">
                {{ copy.viewDetails }}
              </button>
              <button class="inline-flex items-center justify-center rounded-xl border border-amber-100 bg-amber-50 px-3 py-2 text-sm font-bold text-[#ffba00] transition hover:underline lg:border-0 lg:bg-transparent lg:px-0 lg:py-0" type="button" @click.stop="selectCourse(course)">
                {{ copy.edit }}
              </button>
            </div>
          </div>
        </div>
      </div>
      <div v-if="nextPageToken" class="border-t border-slate-200 p-4">
        <button class="w-full rounded-xl border px-4 py-3 font-bold transition hover:bg-slate-50 disabled:cursor-default disabled:border-slate-200 disabled:bg-slate-50 disabled:text-slate-400 disabled:opacity-100" type="button" :disabled="!nextPageToken || loading" @click="loadCourses(nextPageToken)">
          {{ copy.loadMore }}
        </button>
      </div>
    </section>

    <main v-else class="space-y-6">
      <section class="rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-4 md:px-5">
          <div class="min-w-0">
            <h2 class="text-xl font-black">{{ selectedCourseId ? copy.courseTopData : copy.newCourse }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ selectedCourseId || copy.fillCourseHint }}</p>
          </div>
          <span v-if="selectedCourseId" class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(selectedCourseStatusBadge)">
            {{ courseStatusLabel(selectedCourse) }}
          </span>
        </div>

        <div class="p-4 md:p-5">
          <form class="grid gap-3 lg:grid-cols-2" @submit.prevent="saveCourse">
            <label class="block">
              <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.courseTitle }}</span>
              <input v-model="courseForm.title" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <label v-if="false" class="block">
              <span class="text-sm font-bold">{{ copy.categoryTips }}</span>
              <input v-model="courseForm.category_tips" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <label class="block lg:col-span-2">
              <span class="text-sm font-bold">{{ copy.description }}</span>
              <textarea v-model="courseForm.description" class="mt-2 min-h-20 w-full rounded-xl border border-slate-200 px-3 py-2" />
            </label>
            <details class="group lg:col-span-2">
              <summary class="inline-flex cursor-pointer select-none items-center gap-1 rounded-lg text-sm font-bold text-slate-500 transition-colors hover:text-slate-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2">
                {{ (copy as any).advancedConfig || '高级配置' }}
                <ChevronDown class="h-4 w-4 transition-transform group-open:rotate-180" />
              </summary>
              <div class="mt-4 grid gap-3 lg:grid-cols-2">
                <label class="block">
                  <span class="flex items-center gap-1.5 text-sm font-bold">
                    <span><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.respath }}</span>
                    <span class="group/tooltip relative inline-flex cursor-help rounded-full text-slate-600 outline-none transition-colors hover:text-slate-900 focus-visible:text-slate-900 focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-1" tabindex="0" :aria-label="copy.respathHint">
                      <Info class="h-4 w-4" aria-hidden="true" />
                      <span role="tooltip" class="pointer-events-none absolute bottom-full left-1/2 z-30 mb-2 w-72 max-w-[calc(100vw-2rem)] -translate-x-1/2 rounded-md bg-slate-900 px-3 py-2 text-xs font-medium leading-5 text-white opacity-0 shadow-lg transition-opacity group-hover/tooltip:opacity-100 group-focus/tooltip:opacity-100">
                        {{ copy.respathHint }}
                      </span>
                    </span>
                  </span>
                  <input v-model="courseForm.respath" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="/gcc/pipeline/..." />
                </label>
                <label class="block">
                  <span class="flex items-center gap-1.5 text-sm font-bold">
                    <span><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.courseGpath }}</span>
                    <span class="group/tooltip relative inline-flex cursor-help rounded-full text-slate-600 outline-none transition-colors hover:text-slate-900 focus-visible:text-slate-900 focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-1" tabindex="0" :aria-label="copy.courseGpathHint">
                      <Info class="h-4 w-4" aria-hidden="true" />
                      <span role="tooltip" class="pointer-events-none absolute bottom-full left-1/2 z-30 mb-2 w-72 max-w-[calc(100vw-2rem)] -translate-x-1/2 rounded-md bg-slate-900 px-3 py-2 text-xs font-medium leading-5 text-white opacity-0 shadow-lg transition-opacity group-hover/tooltip:opacity-100 group-focus/tooltip:opacity-100">
                        {{ copy.courseGpathHint }}
                      </span>
                    </span>
                  </span>
                  <input v-model="courseForm.course_gpath" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="/gcc/pipeline/..." />
                  <p v-if="duplicateGpathWarning" class="mt-2 text-xs font-semibold text-red-500">{{ copy.duplicateGpathWarning }}</p>
                </label>
              </div>
            </details>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.durationMin }}</span>
              <input v-model="courseForm.duration_min" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" type="number" min="0" />
            </label>
            <label v-if="false" class="block">
              <span class="text-sm font-bold">{{ copy.thumbnailObjectKey }}</span>
              <input v-model="courseForm.thumbnail_object_key" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <label v-if="false" class="block">
              <span class="text-sm font-bold">{{ copy.thumbnailFileHash }}</span>
              <input v-model="courseForm.thumbnail_file_hash" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" />
            </label>
            <div class="flex flex-col gap-3 sm:flex-row lg:col-span-2">
              <button class="inline-flex h-10 w-full items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white disabled:opacity-50 sm:w-auto" :disabled="savingCourse" type="submit">
                <Loader2 v-if="savingCourse" class="h-4 w-4 animate-spin" />
                <Save v-else class="h-4 w-4" />
                {{ savingCourse ? copy.saving : copy.saveCourse }}
              </button>
              <button v-if="canPublishSelectedCourse" class="h-10 w-full rounded-xl border px-4 font-bold disabled:opacity-40 sm:w-auto" :disabled="!selectedCourseId || publishing" type="button" @click="publishCourse">
                {{ publishing ? copy.publishing : copy.publishCourse }}
              </button>
              <button v-if="canDeleteSelectedCourse" class="inline-flex h-10 w-full items-center justify-center gap-2 rounded-xl border border-red-200 px-4 font-bold text-red-600 disabled:opacity-40 sm:w-auto" :disabled="!selectedCourseId" type="button" @click="deleteCourse">
                <Trash2 class="h-4 w-4" />
                {{ copy.deleteCourse }}
              </button>
            </div>
          </form>
        </div>
      </section>

      <Teleport to="body">
        <section v-if="courseDeleteConfirmOpen && pendingDeleteCourse" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4 md:p-6">
          <div class="w-full max-w-[460px] rounded-2xl bg-white p-4 shadow-2xl md:rounded-3xl md:p-6">
            <h2 class="text-xl font-black text-slate-950 md:text-2xl">{{ copy.courseDeleteConfirmTitle }}</h2>
            <p class="mt-3 text-sm font-semibold text-slate-500">{{ copy.courseDeleteConfirmDescription }}</p>
            <div class="mt-5 rounded-2xl bg-slate-50 p-4">
              <div class="break-words font-black text-slate-950">{{ courseTitle(pendingDeleteCourse) }}</div>
              <div class="mt-1 break-all text-sm font-semibold text-slate-500">{{ courseId(pendingDeleteCourse) }}</div>
              <div class="mt-1 text-sm font-semibold text-slate-500">{{ copy.readonlyCourseFieldLabels.version }}: {{ versionOf(pendingDeleteCourse) }}</div>
            </div>
            <div class="mt-6 flex flex-col justify-end gap-3 sm:flex-row">
              <button class="rounded-xl border border-slate-900 px-5 py-3 font-bold text-slate-950 disabled:cursor-not-allowed disabled:opacity-50" type="button" :disabled="deletingCourse" @click="closeCourseDeleteConfirm">{{ copy.cancel }}</button>
              <button class="rounded-xl bg-red-600 px-5 py-3 font-bold text-white disabled:cursor-not-allowed disabled:opacity-50" type="button" :disabled="deletingCourse" @click="confirmDeleteCourse">
                {{ deletingCourse ? copy.deleting : copy.confirmDeleteAction }}
              </button>
            </div>
          </div>
        </section>
      </Teleport>

      <section class="rounded-3xl border border-slate-200 bg-white shadow-sm" :class="!selectedCourseId ? 'opacity-50' : ''">
        <div class="flex items-center justify-between border-b border-slate-200 p-5">
          <div>
            <h2 class="text-xl font-black">{{ copy.chapterListTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.chapterListDescription }}</p>
          </div>
          <button class="inline-flex h-10 items-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white shadow-sm disabled:opacity-40" :disabled="!selectedCourseId" type="button" @click="newChapter">
            <Plus class="h-4 w-4" />
            {{ copy.newChapter }}
          </button>
        </div>
        <div v-if="chaptersLoading" class="p-8 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          {{ copy.loading }}
        </div>
        <div v-else-if="!chapters.length" class="p-8 text-center text-slate-500">{{ copy.emptyChapters }}</div>
        <div v-else>
          <div class="hidden grid-cols-[minmax(0,1fr)_110px_110px_240px] gap-6 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-400 lg:grid">
            <span>{{ copy.chapterColumns.chapter }}</span>
            <span class="text-center">{{ copy.chapterColumns.sort }}</span>
            <span class="text-center">{{ copy.chapterColumns.version }}</span>
            <span class="text-center">{{ copy.chapterColumns.action }}</span>
          </div>
          <div
            v-for="chapter in chapters"
            :key="chapterId(chapter)"
            class="grid w-full gap-3 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50 lg:grid-cols-[minmax(0,1fr)_110px_110px_240px] lg:items-center lg:gap-6"
            :class="chapterId(chapter) === selectedChapterId ? 'bg-sky-50/70' : ''"
          >
            <div class="min-w-0">
              <div class="flex items-center gap-2 overflow-hidden">
                <span class="truncate text-lg font-black text-slate-950">{{ chapterTitle(chapter) }}</span>
                <span v-if="isChapterEmpty(chapter)" class="shrink-0 rounded border border-red-200 bg-red-50 px-1.5 py-0.5 text-[10px] font-bold text-red-600">{{ (copy as any).missingConfig || '缺少内容' }}</span>
              </div>
              <div class="mt-1 truncate font-mono text-xs font-semibold text-slate-500">ID: {{ chapterId(chapter) || "-" }}</div>
            </div>
            <div class="text-sm font-bold text-slate-700 lg:text-center">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.chapterColumns.sort }}</span>{{ chapter.sort_order || 0 }}
            </div>
            <div class="text-sm font-bold text-slate-700 lg:text-center">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.chapterColumns.version }}</span>{{ chapter.version || 0 }}
            </div>
            <div class="flex items-center justify-start gap-4 lg:justify-center">
              <button class="text-sm font-bold text-[#1890ff] transition hover:underline" type="button" @click="openChapterDetail(chapter)">
                {{ copy.viewDetails }}
              </button>
              <button class="text-sm font-bold text-[#ffba00] transition hover:underline" type="button" @click="editChapter(chapter)">
                {{ copy.editChapter }}
              </button>
              <button class="text-sm font-bold text-[#ff4949] transition hover:underline" type="button" @click="deleteChapter(chapter)">
                {{ copy.delete }}
              </button>
            </div>
          </div>
        </div>
      </section>

      <Teleport to="body">
        <section v-if="chapterDialogOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
          <div class="flex max-h-[88vh] w-full max-w-[980px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
            <div class="flex items-start justify-between gap-4 border-b border-slate-200 p-5">
              <div>
                <h2 class="text-xl font-black">{{ chapterDialogMode === "create" ? copy.createChapter : chapterDialogMode === "edit" ? copy.editChapter : copy.chapterDetailTitle }}</h2>
                <p class="mt-1 text-sm text-slate-500">{{ chapterDialogMode === "detail" ? chapterTitle(selectedChapter) : copy.chapterDetailEmptyHint }}</p>
              </div>
              <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeChapterDialog">
                <X class="h-5 w-5" />
              </button>
            </div>

            <div class="flex-1 overflow-y-auto p-5">
              <div v-if="chapterDialogMode !== 'create'" class="mb-4 flex gap-4 border-b border-slate-200">
                <button :class="chapterActiveTab === 'basic' ? 'border-blue-500 text-blue-600' : 'border-transparent text-slate-500'" class="border-b-2 px-1 pb-2 font-bold transition-colors" type="button" @click="chapterActiveTab = 'basic'">{{ (copy as any).basicInfo || '基本信息' }}</button>
                <button :class="chapterActiveTab === 'prerequisites' ? 'border-blue-500 text-blue-600' : 'border-transparent text-slate-500'" class="border-b-2 px-1 pb-2 font-bold transition-colors" type="button" @click="chapterActiveTab = 'prerequisites'">{{ (copy as any).prerequisites || '前置条件' }}</button>
              </div>
              <div v-show="chapterActiveTab === 'basic'">
                <div v-if="chapterDialogMode === 'detail'" class="rounded-2xl border border-slate-200 p-4">
                <h3 class="font-black">{{ copy.chapterRawFields }}</h3>
                <p class="mt-1 text-xs text-slate-500">{{ copy.systemReadonlyHint }}</p>
                <div v-if="!selectedChapterId" class="p-8 text-center text-slate-500">{{ copy.noSelectedChapter }}</div>
                <div v-else class="mt-3 grid gap-3 md:grid-cols-2">
                  <ReadonlyField v-for="entry in chapterRecordEntries(selectedChapter)" :key="`chapter-dialog-${entry.key}`" :label="entry.label" :text="entry.value" />
                </div>
              </div>

              <form v-else class="space-y-4" @submit.prevent="saveChapter">
                <label class="grid gap-2 text-sm font-bold">
                  <span><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.chapterTitlePlaceholder }}</span>
                  <input v-model="chapterForm.title" class="w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.chapterTitlePlaceholder" />
                </label>
                <label class="grid gap-2 text-sm font-bold">
                  {{ copy.sort }}
                  <input v-model="chapterForm.sort_order" class="w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.sort" type="number" min="1" />
                </label>
              </form>
              </div>
              <div v-if="chapterActiveTab === 'prerequisites' && chapterDialogMode !== 'create'" class="mt-2">
                <LmsPrerequisitesTab :targetEntityType="3" :targetEntityId="selectedChapterId" :course="completeCourseRecord" :copy="copy" />
              </div>
            </div>

            <div v-if="chapterDialogMode !== 'detail' && chapterActiveTab === 'basic'" class="flex shrink-0 justify-end border-t border-slate-200 bg-white px-5 py-4">
              <button class="inline-flex h-10 min-w-[180px] items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingChapter" type="button" @click="saveChapter">
                <Loader2 v-if="savingChapter" class="h-4 w-4 animate-spin" />
                <Save v-else class="h-4 w-4" />
                {{ savingChapter ? copy.saving : copy.saveChapter }}
              </button>
            </div>
          </div>
        </section>
      </Teleport>

      <Teleport to="body">
        <section v-if="chapterDeleteConfirmOpen && pendingDeleteChapter" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-6">
          <div class="w-full max-w-[460px] rounded-3xl bg-white p-6 shadow-2xl">
            <h2 class="text-2xl font-black text-slate-950">{{ copy.chapterDeleteConfirmTitle }}</h2>
            <p class="mt-3 text-sm font-semibold text-slate-500">{{ copy.chapterDeleteConfirmDescription }}</p>
            <div class="mt-5 rounded-2xl bg-slate-50 p-4">
              <div class="break-words font-black text-slate-950">{{ chapterTitle(pendingDeleteChapter) }}</div>
              <div class="mt-1 break-all text-sm font-semibold text-slate-500">{{ chapterId(pendingDeleteChapter) }}</div>
            </div>
            <div class="mt-6 flex justify-end gap-3">
              <button class="rounded-xl border border-slate-900 px-5 py-3 font-bold text-slate-950 disabled:cursor-not-allowed disabled:opacity-50" type="button" :disabled="deletingChapter" @click="closeChapterDeleteConfirm">{{ copy.cancel }}</button>
              <button class="rounded-xl bg-red-600 px-5 py-3 font-bold text-white disabled:cursor-not-allowed disabled:opacity-50" type="button" :disabled="deletingChapter" @click="confirmDeleteChapter">
                {{ deletingChapter ? copy.deleting : copy.confirmDeleteAction }}
              </button>
            </div>
          </div>
        </section>
      </Teleport>

      <section class="rounded-3xl border border-slate-200 bg-white shadow-sm" :class="!selectedCourseId ? 'opacity-50' : ''">
        <div class="flex items-center justify-between border-b border-slate-200 p-5">
          <div>
            <h2 class="text-xl font-black">{{ copy.lessonListTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.lessonListDescription }}</p>
          </div>
          <button class="inline-flex h-10 items-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white shadow-sm disabled:opacity-40" :disabled="!selectedCourseId" type="button" @click="openNewLesson">
            <Plus class="h-4 w-4" />
            {{ copy.newLesson }}
          </button>
        </div>
        <div v-if="completeLoading || lessonsLoading" class="p-8 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          {{ copy.loading }}
        </div>
        <div v-else-if="!allLessonItems.length" class="p-8 text-center text-slate-500">{{ copy.emptyLessons }}</div>
        <div v-else>
          <div class="hidden grid-cols-[minmax(0,1fr)_220px_100px_100px_240px] gap-6 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-400 lg:grid">
            <span>{{ copy.lessonColumns.lesson }}</span>
            <span>{{ copy.lessonColumns.chapter }}</span>
            <span class="text-center">{{ copy.lessonColumns.sort }}</span>
            <span class="text-center">{{ copy.lessonColumns.type }}</span>
            <span class="text-center">{{ copy.lessonColumns.action }}</span>
          </div>
          <div
            v-for="item in allLessonItems"
            :key="lessonId(item.lesson)"
            class="grid w-full gap-3 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50 lg:grid-cols-[minmax(0,1fr)_220px_100px_100px_240px] lg:items-center lg:gap-6"
            :class="lessonId(item.lesson) === editingLessonId ? 'bg-sky-50/70' : ''"
          >
            <div class="min-w-0">
              <div class="flex items-center gap-2 overflow-hidden">
                <span class="truncate text-lg font-black text-slate-950">{{ lessonTitle(item.lesson) }}</span>
                <span v-if="isLessonEmpty(item.lesson)" class="shrink-0 rounded border border-red-200 bg-red-50 px-1.5 py-0.5 text-[10px] font-bold text-red-600">{{ (copy as any).missingConfig || '缺少内容' }}</span>
              </div>
              <div class="mt-1 truncate font-mono text-xs font-semibold text-slate-500">ID: {{ lessonId(item.lesson) || "-" }}</div>
            </div>
            <div class="min-w-0 text-sm font-bold text-slate-700">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.lessonColumns.chapter }}</span>
              <span class="truncate">{{ chapterTitle(item.chapter) }}</span>
            </div>
            <div class="text-sm font-bold text-slate-700 lg:text-center">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.lessonColumns.sort }}</span>{{ item.lesson.sort_order || 0 }}
            </div>
            <div class="text-sm font-bold text-slate-700 lg:text-center">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.lessonColumns.type }}</span>{{ lessonTypeLabel(item.lesson.lesson_type) }}
            </div>
            <div class="flex items-center justify-start gap-4 lg:justify-center">
              <button class="text-sm font-bold text-[#1890ff] transition hover:underline" type="button" @click="openLessonDetail(item.lesson)">
                {{ copy.viewDetails }}
              </button>
              <button class="text-sm font-bold text-[#ffba00] transition hover:underline" type="button" @click="editLesson(item.lesson)">
                {{ copy.editLesson }}
              </button>
              <button class="text-sm font-bold text-[#ff4949] transition hover:underline" type="button" @click="deleteLesson(item.lesson)">
                {{ copy.delete }}
              </button>
            </div>
          </div>
        </div>
      </section>

      <Teleport to="body">
        <section v-if="lessonDialogOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
          <div class="flex max-h-[88vh] w-full max-w-[980px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
            <div class="flex items-start justify-between gap-4 border-b border-slate-200 p-5">
              <div>
                <h2 class="text-xl font-black">{{ lessonDialogMode === "create" ? copy.createLesson : lessonDialogMode === "edit" ? copy.editLesson : copy.lessonDetailTitle }}</h2>
                <p class="mt-1 text-sm text-slate-500">{{ lessonDialogMode === "detail" ? lessonTitle(selectedLessonRecord) : copy.lessonDetailEmptyHint }}</p>
              </div>
              <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeLessonDialog">
                <X class="h-5 w-5" />
              </button>
            </div>

            <div class="flex-1 overflow-y-auto p-5">
              <div v-if="lessonDialogMode !== 'create'" class="mb-4 flex gap-4 border-b border-slate-200">
                <button :class="lessonActiveTab === 'basic' ? 'border-blue-500 text-blue-600' : 'border-transparent text-slate-500'" class="border-b-2 px-1 pb-2 font-bold transition-colors" type="button" @click="lessonActiveTab = 'basic'">{{ (copy as any).basicInfo || '基本信息' }}</button>
                <button :class="lessonActiveTab === 'prerequisites' ? 'border-blue-500 text-blue-600' : 'border-transparent text-slate-500'" class="border-b-2 px-1 pb-2 font-bold transition-colors" type="button" @click="lessonActiveTab = 'prerequisites'">{{ (copy as any).prerequisites || '前置条件' }}</button>
              </div>
              <div v-show="lessonActiveTab === 'basic'">
                <div v-if="lessonDialogMode === 'detail'" class="space-y-5">
                <div class="rounded-2xl bg-blue-50 p-4">
                  <div class="text-xs font-black text-blue-600">{{ copy.ownerChapter }}</div>
                  <div class="mt-1 text-lg font-black text-slate-900">{{ selectedLessonOwnerChapter ? chapterTitle(selectedLessonOwnerChapter) : (selectedChapterId ? chapterTitle(selectedChapter) : copy.unselectedChapter) }}</div>
                  <div class="mt-2 break-all font-mono text-sm font-bold text-blue-900">ID: {{ selectedLessonOwnerChapter ? chapterId(selectedLessonOwnerChapter) : (selectedChapterId || "-") }}</div>
                </div>
                <div class="rounded-2xl border border-slate-200 p-4">
                  <h3 class="font-black">{{ copy.readonlyFields }}</h3>
                  <p class="mt-1 text-xs text-slate-500">{{ copy.systemReadonlyHint }}</p>
                  <div v-if="!selectedLessonRecord" class="p-8 text-center text-slate-500">{{ copy.noSelectedLesson }}</div>
                  <div v-else class="mt-3 grid gap-3 md:grid-cols-2">
                    <ReadonlyField v-for="entry in lessonRecordEntries(selectedLessonRecord)" :key="`lesson-dialog-${entry.key}`" :label="entry.label" :text="entry.value" />
                  </div>
                </div>
              </div>

              <form v-else class="space-y-4" @submit.prevent="saveLesson">
                <div class="rounded-2xl border border-blue-100 bg-blue-50 p-3 text-sm text-blue-900">
                  {{ copy.saveBelongsToChapter }}{{ lessonForm.chapter_id ? chapterTitle(chapterById(lessonForm.chapter_id)) : copy.selectChapter }}
                </div>
                <label class="block">
                  <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.ownerChapter }}</span>
                  <select v-model="lessonForm.chapter_id" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
                    <option value="">{{ copy.selectChapter }}</option>
                    <option v-for="chapter in chapters" :key="chapterId(chapter)" :value="chapterId(chapter)">{{ chapterTitle(chapter) }}</option>
                  </select>
                  <span v-if="!chapters.length" class="mt-2 block text-xs font-semibold text-amber-600">{{ copy.noChapterOwnerOptions }}</span>
                </label>
                <label class="block">
                  <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.lessonTitlePlaceholder }}</span>
                  <input v-model="lessonForm.title" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.lessonTitlePlaceholder" />
                </label>
                <div class="grid items-start gap-3 sm:grid-cols-2">
                  <label class="block">
                    <span class="text-sm font-bold">{{ copy.sort }}</span>
                    <input v-model="lessonForm.sort_order" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="1" type="number" min="1" />
                    <span class="mt-1 block text-xs text-slate-500">{{ copy.sortOrderHint }}</span>
                  </label>
                  <label class="block">
                    <span class="text-sm font-bold">{{ copy.lessonColumns.type }}</span>
                    <select v-model="lessonForm.lesson_type" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
                      <option value="1">{{ copy.lessonTypes.video }}</option>
                      <option value="2">{{ copy.lessonTypes.text }}</option>
                      <option value="3">{{ copy.lessonTypes.pdf }}</option>
                      <option value="4">{{ copy.lessonTypes.image }}</option>
                      <option value="5">{{ copy.lessonTypes.audio }}</option>
                      <option value="6">{{ copy.lessonTypes.file }}</option>
                      <option value="7">{{ copy.lessonTypes.link }}</option>
                    </select>
                    <span class="mt-1 block text-xs text-transparent select-none">{{ copy.sortOrderHint }}</span>
                  </label>
                </div>
                <label class="block">
                  <span class="text-sm font-bold">
                    <span v-if="lessonForm.lesson_type === '2'" class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.lessonFieldLabels.body }}
                  </span>
                  <textarea v-model="lessonForm.body" class="mt-2 min-h-24 w-full rounded-xl border border-slate-200 p-4" :placeholder="copy.lessonBodyPlaceholder" />
                </label>
                <template v-if="lessonForm.lesson_type === '7'">
                  <label class="block">
                    <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ (copy as any).externalUrl }}</span>
                    <input v-model="lessonForm.asset_object_key" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="https://" />
                  </label>
                </template>
                <template v-else-if="lessonForm.lesson_type !== '2'">
                  <label class="block">
                    <span class="text-sm font-bold">{{ (copy as any).assetObjectKeyLabel }}</span>
                    <input v-model="lessonForm.asset_object_key" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.assetObjectKeyPlaceholder" />
                    <span v-if="!editingLessonId" class="mt-2 block text-xs font-semibold text-amber-600">
                      {{ (copy as any).uploadAfterSaveHint }}
                    </span>
                  </label>
                  <label class="block">
                    <span class="text-sm font-bold">{{ copy.assetFileHash }}</span>
                    <span class="ml-2 cursor-help rounded-full border border-slate-300 px-2 py-0.5 text-xs text-slate-500" :title="copy.assetHashHint">?</span>
                    <input v-model="lessonForm.asset_file_hash" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.assetFileHashPlaceholder" />
                  </label>
                </template>
              </form>
              </div>
              <div v-if="lessonActiveTab === 'prerequisites' && lessonDialogMode !== 'create'" class="mt-2">
                <LmsPrerequisitesTab :targetEntityType="1" :targetEntityId="editingLessonId" :course="completeCourseRecord" :copy="copy" />
              </div>
            </div>

            <div v-if="lessonDialogMode !== 'detail' && lessonActiveTab === 'basic'" class="flex shrink-0 items-center justify-between border-t border-slate-200 bg-white px-5 py-4 gap-4">
              <div class="flex-1" v-if="editingLessonId && lessonForm.lesson_type !== '7' && lessonForm.lesson_type !== '2'">
                <input type="file" ref="lessonFileInput" class="hidden" @change="handleLessonFileUpload" />
                <button type="button" class="flex w-full items-center justify-center gap-2 rounded-xl border border-blue-500 bg-blue-50 px-4 h-10 font-bold text-blue-700 shadow-sm transition hover:bg-blue-100 disabled:opacity-50" :disabled="uploadingLesson" @click="lessonFileInput?.click()">
                  <Loader2 v-if="uploadingLesson" class="h-4 w-4 animate-spin" />
                  <UploadCloud v-else class="h-4 w-4" />
                  {{ uploadingLesson ? (copy as any).uploading : (copy as any).uploadLessonFile }}
                </button>
              </div>
              <div class="flex-1" v-else></div>
              <button class="inline-flex h-10 min-w-[180px] items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white disabled:opacity-50" :disabled="savingLesson || uploadingLesson || !lessonForm.chapter_id" type="button" @click="saveLesson">
                <Loader2 v-if="savingLesson" class="h-4 w-4 animate-spin" />
                <Save v-else class="h-4 w-4" />
                {{ savingLesson ? copy.saving : copy.saveLesson }}
              </button>
            </div>
          </div>
        </section>
      </Teleport>

      <Teleport to="body">
        <section v-if="lessonDeleteConfirmOpen && pendingDeleteLesson" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-6">
          <div class="w-full max-w-[460px] rounded-3xl bg-white p-6 shadow-2xl">
            <h2 class="text-2xl font-black text-slate-950">{{ copy.lessonDeleteConfirmTitle }}</h2>
            <p class="mt-3 text-sm font-semibold text-slate-500">{{ copy.lessonDeleteConfirmDescription }}</p>
            <div class="mt-5 rounded-2xl bg-slate-50 p-4">
              <div class="break-words font-black text-slate-950">{{ lessonTitle(pendingDeleteLesson) }}</div>
              <div class="mt-1 break-all text-sm font-semibold text-slate-500">{{ lessonId(pendingDeleteLesson) }}</div>
            </div>
            <div class="mt-6 flex justify-end gap-3">
              <button class="rounded-xl border border-slate-900 px-5 py-3 font-bold text-slate-950 disabled:cursor-not-allowed disabled:opacity-50" type="button" :disabled="deletingLesson" @click="closeLessonDeleteConfirm">{{ copy.cancel }}</button>
              <button class="rounded-xl bg-red-600 px-5 py-3 font-bold text-white disabled:cursor-not-allowed disabled:opacity-50" type="button" :disabled="deletingLesson" @click="confirmDeletePendingLesson">
                {{ deletingLesson ? copy.deleting : copy.confirmDeleteAction }}
              </button>
            </div>
          </div>
        </section>
      </Teleport>

      <Teleport to="body">
        <section v-if="detailDeleteConfirmOpen && pendingDetailDelete" class="fixed inset-0 z-[60] flex items-center justify-center bg-slate-950/50 p-6">
          <div class="w-full max-w-[460px] rounded-3xl bg-white p-6 shadow-2xl">
            <h2 class="text-2xl font-black text-slate-950">{{ copy.confirmDeleteAction }}</h2>
            <p class="mt-3 text-sm font-semibold text-slate-500">{{ pendingDetailDelete.description }}</p>
            <div class="mt-5 rounded-2xl bg-slate-50 p-4">
              <div class="break-words font-black text-slate-950">{{ pendingDetailDelete.title }}</div>
              <div v-if="pendingDetailDelete.id" class="mt-1 break-all text-sm font-semibold text-slate-500">{{ pendingDetailDelete.id }}</div>
              <div v-if="pendingDetailDelete.version !== undefined" class="mt-1 text-sm font-semibold text-slate-500">{{ copy.readonlyCourseFieldLabels.version }}: {{ pendingDetailDelete.version }}</div>
            </div>
            <div class="mt-6 flex justify-end gap-3">
              <button class="rounded-xl border border-slate-900 px-5 py-3 font-bold text-slate-950 disabled:cursor-not-allowed disabled:opacity-50" type="button" :disabled="deletingDetail" @click="closeDetailDeleteConfirm">{{ copy.cancel }}</button>
              <button class="rounded-xl bg-red-600 px-5 py-3 font-bold text-white disabled:cursor-not-allowed disabled:opacity-50" type="button" :disabled="deletingDetail" @click="confirmDetailDelete">
                {{ deletingDetail ? copy.deleting : copy.confirmDeleteAction }}
              </button>
            </div>
          </div>
        </section>
      </Teleport>

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
                <button class="inline-flex h-10 items-center gap-2 rounded-xl bg-blue-700 px-4 text-sm font-bold text-white shadow-sm disabled:opacity-40" :disabled="!selectedCourseId" type="button" @click="newSupplementaryItem()">
                  <Plus class="h-4 w-4" />
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
                      <div class="flex items-center gap-2 overflow-hidden">
                        <span class="truncate font-black text-slate-950">{{ item.title }}</span>
                        <span v-if="isSupplementaryMaterialEmpty(item)" class="shrink-0 rounded border border-red-200 bg-red-50 px-1.5 py-0.5 text-[10px] font-bold text-red-600">{{ (copy as any).missingConfig || '缺少内容' }}</span>
                      </div>
                      <div v-if="item.description" class="mt-1 max-w-2xl text-sm text-slate-500">{{ item.description }}</div>
                    </td>
                    <td class="px-4 py-4">
                      <a v-if="item.url" class="inline-flex max-w-xs items-center rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-xs font-bold text-blue-700 hover:bg-blue-100" :href="item.url" target="_blank" rel="noreferrer">
                        <span class="truncate">{{ item.url }}</span>
                      </a>
                      <span v-else class="text-slate-400">-</span>
                    </td>
                    <td class="w-24 whitespace-nowrap px-4 py-4 text-right">
                      <button class="whitespace-nowrap text-sm font-bold text-[#ffba00] transition hover:underline" type="button" @click.stop="editSupplementaryItem(item)">
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
                      <select v-model="supplementaryItemForm.chapter" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3">
                        <option value="">{{ copy.globalSupplementaryOption }}</option>
                        <option v-for="ch in chapters" :key="chapterId(ch)" :value="chapterTitle(ch)">
                          {{ chapterTitle(ch) }}
                        </option>
                      </select>
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
                      <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.titleField }}</span>
                      <input v-model="supplementaryItemForm.title" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.materialTitlePlaceholder" />
                    </label>
                    <label class="block">
                      <span class="text-sm font-bold">{{ copy.description }}</span>
                      <textarea v-model="supplementaryItemForm.description" class="mt-2 min-h-20 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.descriptionPlaceholder" />
                    </label>
                    <template v-if="isSupplementaryLinkType()">
                      <label class="block">
                        <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ (copy as any).externalUrl }}</span>
                        <input v-model="supplementaryItemForm.url" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" placeholder="https://..." />
                      </label>
                    </template>
                    <template v-else>
                      <label class="block">
                        <span class="text-sm font-bold"><span v-if="isSupplementaryAssetRequired()" class="mr-1 text-red-500" aria-hidden="true">*</span>{{ (copy as any).assetObjectKeyLabel }}</span>
                        <input v-model="supplementaryItemForm.url" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.assetObjectKeyPlaceholder" />
                        <span v-if="editingSupplementaryItemIndex < 0" class="mt-2 block text-xs font-semibold text-amber-600">
                          {{ (copy as any).uploadAfterSaveHintSupplementary }}
                        </span>
                      </label>
                      <div class="mt-3 flex gap-3" v-if="editingSupplementaryItemIndex >= 0">
                        <input type="file" ref="supplementaryFileInput" class="hidden" @change="handleSupplementaryFileUpload" />
                        <button type="button" class="flex w-full items-center justify-center gap-2 rounded-xl border border-blue-500 bg-blue-50 px-4 py-3 font-bold text-blue-700 shadow-sm transition hover:bg-blue-100 disabled:opacity-50" :disabled="uploadingSupplementary" @click="supplementaryFileInput?.click()">
                          <Loader2 v-if="uploadingSupplementary" class="h-4 w-4 animate-spin" />
                          <UploadCloud v-else class="h-4 w-4" />
                          {{ uploadingSupplementary ? (copy as any).uploading : (copy as any).uploadSupplementaryFile }}
                        </button>
                      </div>
                    </template>
                  </div>
                </div>
                <div class="flex justify-end gap-3 border-t border-slate-200 px-6 py-4">
                  <button class="rounded-xl border px-4 py-2 font-bold" type="button" @click="closeSupplementaryItemDialog">
                    {{ copy.cancel }}
                  </button>
                  <button class="rounded-xl border border-red-200 px-4 py-2 font-bold text-red-600 disabled:opacity-40" :disabled="editingSupplementaryItemIndex < 0" type="button" @click="deleteSupplementaryItem">
                    {{ copy.deleteThisItem }}
                  </button>
                  <button class="rounded-xl bg-blue-700 px-4 py-2 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingSupplementaryMaterial" type="submit">
                    {{ savingSupplementaryMaterial ? copy.saving : copy.saveThisMaterial }}
                  </button>
                </div>
              </form>
            </div>
          </Teleport>

          <div v-if="false" class="rounded-2xl border border-slate-200 p-4">
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
                  <button class="rounded-xl bg-blue-700 px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingSupplementaryMaterial" type="button" @click="saveSupplementaryMaterial">
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
                <ReadonlyField v-for="entry in supplementaryRecordEntries(supplementaryMaterial)" :key="`supplementary-${entry.key}`" :label="entry.label" :text="entry.value" />
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
            <button class="inline-flex h-10 items-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white shadow-sm disabled:opacity-40" :disabled="!selectedCourseId" type="button" @click="newMaterial">
              <Plus class="h-4 w-4" />
              {{ copy.newNormalMaterial }}
            </button>
          </div>
        </div>
        <div class="p-5">
          <div class="overflow-hidden rounded-2xl border border-slate-200">
            <div v-if="materialsLoading" class="px-6 py-10 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              {{ copy.loading }}
            </div>
            <div v-else-if="!materials.length" class="px-6 py-10 text-center text-slate-500">{{ copy.emptyMaterials }}</div>
            <div v-else class="overflow-x-auto">
              <table class="min-w-full text-left text-sm">
                <thead class="bg-slate-50 text-xs font-black uppercase tracking-wide text-slate-500">
                  <tr>
                    <th class="px-5 py-3">{{ copy.materialTitlePlaceholder }}</th>
                    <th class="px-5 py-3">{{ copy.materialType }}</th>
                    <th class="px-5 py-3">{{ copy.fileObjectKey }}</th>
                    <th class="px-5 py-3">{{ copy.sort }}</th>
                    <th class="w-48 px-5 py-3 text-right">{{ copy.chapterColumns.action }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-100">
                  <tr v-for="material in materials" :key="materialId(material)" class="transition hover:bg-sky-50">
                    <td class="px-5 py-4">
                      <div class="font-black text-slate-950">{{ materialTitle(material) }}</div>
                      <div v-if="material.description" class="mt-1 line-clamp-2 text-xs font-semibold text-slate-500">{{ material.description }}</div>
                    </td>
                    <td class="px-5 py-4 font-semibold text-slate-700">{{ materialTypeLabel(material.material_type) }}</td>
                    <td class="max-w-[520px] px-5 py-4">
                      <div class="break-all font-mono text-xs font-semibold text-slate-500">{{ material.file_object_key || "-" }}</div>
                    </td>
                    <td class="px-5 py-4 font-semibold text-slate-700">{{ material.sort_order || 0 }}</td>
                    <td class="w-48 whitespace-nowrap px-5 py-4 text-right">
                      <div class="inline-flex items-center justify-end gap-3">
                        <button class="text-sm font-bold text-[#1890ff] transition hover:underline" type="button" @click="viewMaterial(material)">{{ copy.viewDetails }}</button>
                        <button class="text-sm font-bold text-[#ffba00] transition hover:underline" type="button" @click="editMaterial(material)">{{ copy.edit }}</button>
                        <button class="text-sm font-bold text-[#ff4949] transition hover:underline" type="button" @click="deleteMaterial(material)">{{ copy.delete }}</button>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <Teleport to="body">
          <section v-if="materialDialogOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
            <div v-if="materialDialogMode === 'detail'" class="flex max-h-[88vh] w-full max-w-[860px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
              <div class="flex items-start justify-between gap-4 border-b border-slate-200 p-5">
                <div>
                  <h2 class="text-xl font-black">{{ copy.materialRawFields }}</h2>
                  <p class="mt-1 break-all text-sm text-slate-500">{{ materialTitle(selectedMaterialRecord) }}</p>
                </div>
                <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeMaterialDialog">
                  <X class="h-5 w-5" />
                </button>
              </div>
              <div class="min-h-0 flex-1 overflow-y-auto p-5">
                <div v-if="selectedMaterialRecord" class="grid gap-3 md:grid-cols-2">
                  <ReadonlyField
                    v-for="entry in materialRecordEntries(selectedMaterialRecord)"
                    :key="`material-detail-${entry.key}`"
                    :label="entry.label"
                    :text="entry.value"
                    :min-height="materialReadonlyMinHeight(entry.key)"
                  />
                </div>
                <div v-else class="p-10 text-center text-slate-500">{{ copy.emptyMaterials }}</div>
              </div>
            </div>

            <form v-else class="flex max-h-[88vh] w-full max-w-[760px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl" @submit.prevent="saveMaterial">
              <div class="flex items-start justify-between gap-4 border-b border-slate-200 p-5">
                <div>
                  <h2 class="text-xl font-black">{{ materialDialogMode === "edit" ? copy.editMaterial : copy.createMaterial }}</h2>
                  <p class="mt-1 text-sm text-slate-500">{{ copy.normalMaterialsDescription }}</p>
                </div>
                <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeMaterialDialog">
                  <X class="h-5 w-5" />
                </button>
              </div>
              <div class="min-h-0 flex-1 overflow-y-auto p-5">
                <div class="grid gap-3">
                  <label class="block">
                    <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.materialTitlePlaceholder }}</span>
                    <input v-model="materialForm.title" class="mt-2 h-10 w-full rounded-xl border border-slate-200 px-3" :placeholder="copy.materialTitlePlaceholder" />
                  </label>
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
                    <span v-if="!editingMaterialId" class="mt-2 block text-xs font-semibold text-amber-600">
                      {{ (copy as any).uploadAfterSaveHintMaterial }}
                    </span>
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
                  <div v-if="selectedMaterialRecord" class="border-t border-slate-200 pt-4">
                    <h4 class="font-black">{{ copy.materialRawFields }}</h4>
                    <div class="mt-3 max-h-56 space-y-3 overflow-y-auto pr-1">
                      <ReadonlyField
                        v-for="entry in materialRecordEntries(selectedMaterialRecord)"
                        :key="`material-edit-${entry.key}`"
                        :label="entry.label"
                        :text="entry.value"
                        :min-height="materialReadonlyMinHeight(entry.key)"
                      />
                    </div>
                  </div>
                </div>
              </div>
              <div class="flex shrink-0 flex-wrap items-center justify-between gap-3 border-t border-slate-200 bg-white px-5 py-4">
                <div class="min-w-0 flex-1">
                  <div v-if="editingMaterialId" class="flex gap-3">
                    <input type="file" ref="materialFileInput" class="hidden" @change="handleMaterialFileUpload" />
                    <button type="button" class="flex w-full items-center justify-center gap-2 rounded-xl border border-blue-500 bg-blue-50 px-4 py-3 font-bold text-blue-700 shadow-sm transition hover:bg-blue-100 disabled:opacity-50" :disabled="uploadingMaterial" @click="materialFileInput?.click()">
                      <Loader2 v-if="uploadingMaterial" class="h-4 w-4 animate-spin" />
                      <UploadCloud v-else class="h-4 w-4" />
                      {{ uploadingMaterial ? (copy as any).uploading : (copy as any).uploadFile }}
                    </button>
                  </div>
                </div>
                <button class="h-10 min-w-[140px] rounded-xl bg-blue-700 px-4 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingMaterial || uploadingMaterial" type="submit">
                  <Loader2 v-if="savingMaterial" class="mr-2 inline-block h-4 w-4 animate-spin" />
                  {{ savingMaterial ? copy.saving : copy.saveMaterial }}
                </button>
              </div>
            </form>
          </section>
        </Teleport>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white shadow-sm" :class="!selectedCourseId ? 'opacity-50' : ''">
        <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
          <div>
            <h2 class="text-xl font-black">{{ copy.quizBankTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.quizBankDescription }}</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button
              class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40"
              :disabled="!selectedCourseId"
              type="button"
              @click="refreshQuizBank"
            >
              {{ copy.loadQuizzes }}
            </button>
            <button
              class="inline-flex h-10 items-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white shadow-sm disabled:opacity-40"
              :disabled="!selectedCourseId"
              type="button"
              @click="openNewQuiz"
            >
              <Plus class="h-4 w-4" />
              {{ copy.newQuiz }}
            </button>
          </div>
        </div>

        <div v-if="quizzesLoading" class="p-8 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          {{ copy.loading }}
        </div>
        <div v-else-if="!allQuizItems.length" class="p-8 text-center text-slate-500">{{ copy.emptyQuizzes }}</div>
        <div v-else>
          <div class="hidden grid-cols-[minmax(0,1fr)_170px_220px_110px_110px_240px] gap-6 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-400 lg:grid">
            <span>{{ copy.quizColumns.quiz }}</span>
            <span>{{ copy.quizColumns.ownerType }}</span>
            <span>{{ copy.quizColumns.owner }}</span>
            <span class="text-center">{{ copy.quizColumns.passingScore }}</span>
            <span class="text-center">{{ copy.quizColumns.questions }}</span>
            <span class="text-center">{{ copy.quizColumns.action }}</span>
          </div>
          <div
            v-for="item in allQuizItems"
            :key="quizId(item.quiz)"
            class="grid w-full gap-3 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50 lg:grid-cols-[minmax(0,1fr)_170px_220px_110px_110px_240px] lg:items-center lg:gap-6"
            :class="quizId(item.quiz) === selectedQuizId ? 'bg-sky-50/70' : ''"
          >
            <div class="min-w-0">
              <div class="flex items-center gap-2 overflow-hidden">
                <span class="truncate text-lg font-black text-slate-950">{{ quizTitle(item.quiz) }}</span>
                <span v-if="isQuizEmpty(item)" class="shrink-0 rounded border border-red-200 bg-red-50 px-1.5 py-0.5 text-[10px] font-bold text-red-600">{{ (copy as any).missingConfig || '缺少内容' }}</span>
              </div>
              <div class="mt-1 truncate font-mono text-xs font-semibold text-slate-500">ID: {{ quizId(item.quiz) || "-" }}</div>
            </div>
            <div class="text-sm font-bold text-slate-700">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.quizColumns.ownerType }}</span>{{ quizzableTypeLabel(item.ownerType) }}
            </div>
            <div class="min-w-0 text-sm font-bold text-slate-700">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.quizColumns.owner }}</span><span class="truncate">{{ quizItemOwnerTitle(item) }}</span>
            </div>
            <div class="text-sm font-bold text-slate-700 lg:text-center">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.quizColumns.passingScore }}</span>{{ item.quiz.passing_score || 0 }}
            </div>
            <div class="text-sm font-bold text-slate-700 lg:text-center">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.quizColumns.questions }}</span>{{ item.questionCount }}
            </div>
            <div class="flex items-center justify-start gap-4 lg:justify-center">
              <button class="text-sm font-bold text-[#1890ff] transition hover:underline" type="button" @click="openQuizDetail(item.quiz)">
                {{ copy.viewDetails }}
              </button>
              <button class="text-sm font-bold text-[#ffba00] transition hover:underline" type="button" @click="editQuiz(item.quiz)">
                {{ copy.editQuiz }}
              </button>
              <button class="text-sm font-bold text-[#ff4949] transition hover:underline" type="button" @click="deleteQuiz(item.quiz)">
                {{ copy.delete }}
              </button>
            </div>
          </div>
        </div>
      </section>

      <Teleport to="body">
        <section v-if="quizDialogOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
          <div class="flex max-h-[88vh] w-full max-w-[1180px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
            <div class="flex items-start justify-between gap-4 border-b border-slate-200 p-5">
              <div>
                <h2 class="text-xl font-black">{{ quizDialogMode === "create" ? copy.createQuiz : quizDialogMode === "edit" ? copy.editQuiz : copy.quizDetailTitle }}</h2>
                <p class="mt-1 text-sm text-slate-500">{{ quizDialogMode === "create" ? copy.quizDetailEmptyHint : quizTitle(selectedQuiz) }}</p>
              </div>
              <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeQuizDialog">
                <X class="h-5 w-5" />
              </button>
            </div>

            <div class="flex-1 space-y-5 overflow-y-auto p-5">
              <div v-if="quizDialogMode !== 'create'" class="mb-4 flex gap-4 border-b border-slate-200">
                <button :class="quizActiveTab === 'basic' ? 'border-blue-500 text-blue-600' : 'border-transparent text-slate-500'" class="border-b-2 px-1 pb-2 font-bold transition-colors" type="button" @click="quizActiveTab = 'basic'">{{ (copy as any).basicInfo || '基本信息' }}</button>
                <button :class="quizActiveTab === 'prerequisites' ? 'border-blue-500 text-blue-600' : 'border-transparent text-slate-500'" class="border-b-2 px-1 pb-2 font-bold transition-colors" type="button" @click="quizActiveTab = 'prerequisites'">{{ (copy as any).prerequisites || '前置条件' }}</button>
              </div>
              <div v-show="quizActiveTab === 'basic'">
                <div class="grid gap-4 lg:grid-cols-3">
                <div class="rounded-2xl bg-blue-50 p-4">
                  <div class="text-xs font-black text-blue-600">{{ copy.owner }}</div>
                  <div class="mt-1 text-lg font-black text-slate-900">{{ selectedQuizItem ? quizzableTypeLabel(selectedQuizItem.ownerType) : copy.ownerLevelQuiz(quizTarget().label) }}</div>
                </div>
                <div class="rounded-2xl bg-slate-50 p-4">
                  <div class="text-xs font-black text-slate-500">{{ copy.ownerChapterLabel }}</div>
                  <div class="mt-1 text-lg font-black text-slate-900">{{ quizTargetChapterTitle() }}</div>
                </div>
                <div class="rounded-2xl bg-slate-50 p-4">
                  <div class="text-xs font-black text-slate-500">{{ copy.ownerLessonLabel }}</div>
                  <div class="mt-1 text-lg font-black text-slate-900">{{ quizTargetLessonTitle() }}</div>
                </div>
              </div>

              <section v-if="quizDialogMode === 'detail'" class="rounded-2xl border border-slate-200 p-4">
                <h3 class="font-black">{{ copy.quizReadonlyFields }}</h3>
                <div class="mt-3 grid gap-3 md:grid-cols-2">
                  <ReadonlyField v-for="entry in quizRecordEntries(selectedQuiz)" :key="`quiz-detail-${entry.key}`" :label="entry.label" :text="entry.value" />
                </div>
              </section>

              <form v-else class="rounded-2xl border border-slate-200 p-4" @submit.prevent="saveQuiz">
                <h3 class="font-black">{{ editingQuizId ? copy.editQuiz : copy.createQuiz }}</h3>
                <div class="mt-3 rounded-2xl border border-blue-100 bg-blue-50 p-3 text-sm text-blue-900">
                  {{ copy.saveQuizOwner }}{{ quizTarget().label }} 路 {{ quizTarget().id ? quizTarget().title : copy.unselectedTarget(quizTarget().label) }}
                </div>
                <div class="mt-3 grid gap-3 sm:grid-cols-2">
                  <label class="block">
                    <span class="text-sm font-bold">{{ copy.quizOwnerType }}</span>
                    <select v-model="quizForm.scope" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" @change="changeQuizFormScope">
                      <option value="course">{{ copy.quizScopes.course }}</option>
                      <option value="chapter">{{ copy.quizScopes.chapter }}</option>
                      <option value="lesson">{{ copy.quizScopes.lesson }}</option>
                    </select>
                  </label>
                  <label v-if="quizForm.scope === 'chapter'" class="block">
                    <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.quizOwnerObject }}</span>
                    <select v-model="quizForm.owner_id" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
                      <option value="">{{ copy.selectChapter }}</option>
                      <option v-for="chapter in quizChapterOptions" :key="chapterId(chapter)" :value="chapterId(chapter)">{{ chapterTitle(chapter) }}</option>
                    </select>
                  </label>
                  <label v-else-if="quizForm.scope === 'lesson'" class="block">
                    <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.quizOwnerObject }}</span>
                    <select v-model="quizForm.owner_id" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
                      <option value="">{{ copy.selectLesson }}</option>
                      <option v-for="item in quizLessonOptions" :key="lessonId(item.lesson)" :value="lessonId(item.lesson)">{{ chapterTitle(item.chapter) }} 路 {{ lessonTitle(item.lesson) }}</option>
                    </select>
                  </label>
                  <div v-else class="block">
                    <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.quizOwnerObject }}</span>
                    <div class="mt-2 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 font-semibold text-slate-700">{{ courseTitle(selectedCourse) }}</div>
                  </div>
                </div>
                <p v-if="quizForm.scope === 'chapter' && !quizChapterOptions.length" class="mt-3 rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm font-semibold text-amber-800">
                  {{ copy.noChapterOwnerOptions }}
                </p>
                <p v-if="quizForm.scope === 'lesson' && !quizLessonOptions.length" class="mt-3 rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm font-semibold text-amber-800">
                  {{ copy.noLessonOwnerOptions }}
                </p>
                <label class="mt-3 block">
                  <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.quizTitlePlaceholder }}</span>
                  <input v-model="quizForm.title" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.quizTitlePlaceholder" />
                </label>
                <textarea v-model="quizForm.description" class="mt-3 min-h-20 w-full rounded-xl border border-slate-200 p-4" :placeholder="copy.description" />
                <div class="mt-3 grid gap-3 sm:grid-cols-2">
                  <input v-model="quizForm.passing_score" class="rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.passingScorePlaceholder" type="number" />
                  <input v-model="quizForm.time_limit" class="rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.timeLimitPlaceholder" type="number" />
                </div>
                <label class="mt-3 inline-flex items-center gap-2 text-sm font-bold text-slate-600">
                  <input v-model="quizForm.randomize_questions" type="checkbox" />
                  {{ copy.randomizeQuestions }}
                </label>
                <p v-if="quizDialogMode === 'create'" class="mt-3 rounded-2xl border border-dashed border-sky-200 bg-sky-50 px-4 py-3 text-sm font-semibold text-sky-800">
                  {{ copy.quizQuestionsAfterSaveHint }}
                </p>
              </form>

              <section v-if="selectedQuizId && quizDialogMode !== 'create'" class="rounded-2xl border border-slate-200">
                <div class="flex items-center justify-between gap-3 border-b border-slate-200 p-4">
                  <div>
                    <h3 class="font-black">{{ copy.questionListTitle }}</h3>
                    <p class="mt-1 text-xs text-slate-500">{{ copy.questionListDescription }}</p>
                  </div>
                  <button v-if="quizDialogMode !== 'detail'" class="inline-flex h-9 items-center gap-2 rounded-xl bg-blue-700 px-3 text-xs font-bold text-white shadow-sm disabled:opacity-40" :disabled="!selectedQuizId" type="button" @click="newQuestion">
                    <Plus class="h-3.5 w-3.5" />
                    {{ copy.newQuestion }}
                  </button>
                </div>
                <div v-if="questionsLoading" class="p-6 text-center text-slate-500">
                  <Loader2 class="mx-auto mb-2 h-5 w-5 animate-spin" />
                  {{ copy.loading }}
                </div>
                <div v-else-if="!questions.length" class="p-6 text-center text-slate-500">{{ copy.emptyQuestions }}</div>
                <div v-else class="overflow-hidden">
                  <div class="hidden grid-cols-[minmax(0,1fr)_180px_110px_220px] gap-5 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-400 lg:grid">
                    <span>{{ copy.questionStemLabel }}</span>
                    <span>{{ copy.questionTypePoints }}</span>
                    <span class="text-center">{{ copy.sort }}</span>
                    <span class="text-right">{{ copy.quizColumns.action }}</span>
                  </div>
                  <div class="max-h-96 divide-y divide-slate-100 overflow-y-auto">
                    <div
                      v-for="question in questions"
                      :key="questionId(question)"
                      class="grid gap-3 px-5 py-4 transition hover:bg-slate-50 lg:grid-cols-[minmax(0,1fr)_180px_110px_220px] lg:items-center lg:gap-5"
                      :class="questionId(question) === selectedQuestionId ? 'bg-sky-50/70' : ''"
                    >
                      <div class="min-w-0">
                        <div class="line-clamp-2 font-black text-slate-950">{{ questionTitle(question) }}</div>
                        <div class="mt-1 break-all font-mono text-xs font-semibold text-slate-500">ID: {{ questionId(question) || "-" }}</div>
                      </div>
                      <div class="text-sm font-bold text-slate-700">
                        <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.questionTypePoints }}</span>{{ copy.questionMeta(questionTypeLabel(question.question_type), question.points || 0) }}
                      </div>
                      <div class="text-sm font-bold text-slate-700 lg:text-center">
                        <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.sort }}</span>{{ question.sort_order || 0 }}
                      </div>
                      <div class="flex flex-wrap items-center justify-start gap-3 lg:justify-end">
                        <button class="text-sm font-bold text-[#1890ff] transition hover:underline" type="button" @click="viewQuestion(question)">
                          {{ copy.viewDetails }}
                        </button>
                        <button v-if="quizDialogMode !== 'detail'" class="text-sm font-bold text-[#ffba00] transition hover:underline" type="button" @click="editQuestion(question)">
                          {{ copy.edit }}
                        </button>
                        <button v-if="quizDialogMode !== 'detail'" class="text-sm font-bold text-[#ff4949] transition hover:underline" type="button" @click="deleteQuestion(question)">
                          {{ copy.delete }}
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </section>
              </div>
              <div v-if="quizActiveTab === 'prerequisites' && quizDialogMode !== 'create'" class="mt-2">
                <LmsPrerequisitesTab :targetEntityType="2" :targetEntityId="selectedQuizId" :course="completeCourseRecord" :copy="copy" />
              </div>
            </div>

            <div v-if="quizDialogMode !== 'detail' && quizActiveTab === 'basic'" class="flex shrink-0 justify-end border-t border-slate-200 bg-white px-5 py-4">
              <button class="inline-flex h-10 min-w-[180px] items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white disabled:opacity-50" :disabled="savingQuiz || !quizTarget().id" type="button" @click="saveQuiz">
                <Loader2 v-if="savingQuiz" class="h-4 w-4 animate-spin" />
                <Save v-else class="h-4 w-4" />
                {{ savingQuiz ? copy.saving : quizDialogMode === "create" ? copy.saveQuizAndAddQuestions : copy.saveQuiz }}
              </button>
            </div>
          </div>
        </section>
      </Teleport>

      <Teleport to="body">
        <section v-if="questionDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-6">
          <div class="flex max-h-[88vh] w-full max-w-[980px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
            <div class="flex items-start justify-between gap-4 border-b border-slate-200 p-5">
              <div>
                <h2 class="text-xl font-black">{{ questionDialogMode === "create" ? copy.createQuestion : questionDialogMode === "edit" ? copy.editQuestion : copy.questionDetailTitle }}</h2>
                <p class="mt-1 text-sm text-slate-500">{{ selectedQuestionId ? questionTitle(selectedQuestion) : copy.questionDetailEmptyHint }}</p>
              </div>
              <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeQuestionDialog">
                <X class="h-5 w-5" />
              </button>
            </div>

            <div class="min-h-0 flex-1 space-y-6 overflow-y-auto p-5">
              <section class="rounded-2xl border border-slate-200">
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
                <form v-if="quizDialogMode !== 'detail' && questionDialogMode !== 'detail'" class="border-t border-slate-200 p-4" @submit.prevent="saveQuestion">
                  <h4 class="font-black">{{ editingQuestionId ? copy.editQuestion : copy.createQuestion }}</h4>
                  <label class="mt-3 block">
                    <span class="text-sm font-bold">{{ copy.questionStemLabel }} <span class="text-red-500">*</span></span>
                    <textarea v-model="questionForm.question_text" class="mt-2 min-h-24 w-full rounded-xl border border-slate-200 p-4" :placeholder="copy.questionTextPlaceholder" />
                  </label>
                  <div class="mt-3 grid gap-3 sm:grid-cols-3">
                    <select v-model="questionForm.question_type" class="w-full rounded-xl border border-slate-200 px-4 py-3" :title="copy.questionTypePoints">
                      <option value="1">{{ copy.questionTypes.single }}</option>
                      <option value="2">{{ copy.questionTypes.multiple }}</option>
                      <option value="3">{{ copy.questionTypes.judgement }}</option>
                    </select>
                    <div class="relative w-full">
                      <div class="absolute -top-2 left-3 bg-white px-1 text-[10px] font-bold leading-none text-slate-500">{{ copy.pointsPlaceholder }}</div>
                      <input v-model="questionForm.points" class="w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.pointsPlaceholder" type="number" />
                    </div>
                    <div class="relative w-full">
                      <div class="absolute -top-2 left-3 bg-white px-1 text-[10px] font-bold leading-none text-slate-500">{{ copy.sort }}</div>
                      <input v-model="questionForm.sort_order" class="w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.sort" type="number" />
                    </div>
                  </div>
                  <label class="mt-3 inline-flex items-center gap-2 text-sm font-bold text-slate-600">
                    <input v-model="questionForm.is_required" type="checkbox" />
                    {{ copy.required }}
                  </label>
                  <div class="mt-4 rounded-xl border border-slate-100 bg-slate-50 p-4">
                    <div class="flex items-center justify-between">
                      <div>
                        <div class="text-sm font-bold text-slate-700">{{ copy.mediaJsonLabel }}</div>
                        <div class="mt-0.5 text-xs text-slate-500">{{ copy.mediaJsonHint }}</div>
                      </div>
                      <button type="button" class="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-bold text-blue-600 shadow-sm hover:bg-slate-50" @click="openMediaConfig">
                        {{ copy.configureMediaJson }}
                      </button>
                    </div>
                  </div>
                  <button class="mt-3 w-full rounded-xl bg-blue-700 px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedQuizId || savingQuestion" type="submit">
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
                    <button v-if="questionDialogMode !== 'detail'" class="flex-1 text-left" type="button" @click="editOption(option)">
                      <div class="font-black">{{ optionTitle(option) }}</div>
                      <div class="mt-1 text-xs" :class="option.is_correct ? 'text-emerald-600' : 'text-slate-500'">
                        {{ option.is_correct ? copy.correctAnswer : copy.normalOption }} 路 {{ copy.sortMeta(option.sort_order || 0) }}
                      </div>
                    </button>
                    <div v-else class="flex-1">
                      <div class="font-black">{{ optionTitle(option) }}</div>
                      <div class="mt-1 text-xs" :class="option.is_correct ? 'text-emerald-600' : 'text-slate-500'">
                        {{ option.is_correct ? copy.correctAnswer : copy.normalOption }} 路 {{ copy.sortMeta(option.sort_order || 0) }}
                      </div>
                    </div>
                    <button v-if="quizDialogMode !== 'detail' && questionDialogMode !== 'detail'" class="text-xs font-bold text-[#ff4949] transition hover:underline" type="button" @click="deleteOption(option)">{{ copy.delete }}</button>
                  </div>
                </div>
                <form v-if="quizDialogMode !== 'detail' && questionDialogMode !== 'detail'" class="border-t border-slate-200 p-4" @submit.prevent="saveOption">
                  <div class="flex items-center justify-between gap-3">
                    <h4 class="font-black">{{ editingOptionId ? copy.editOption : copy.createOption }}</h4>
                    <button class="inline-flex h-9 items-center gap-2 rounded-xl bg-blue-700 px-3 text-xs font-bold text-white shadow-sm disabled:opacity-40" :disabled="!selectedQuestionId" type="button" @click="newOption">
                      <Plus class="h-3.5 w-3.5" />
                      {{ copy.newOption }}
                    </button>
                  </div>
                  <input v-model="optionForm.option_text" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" :placeholder="copy.optionTextPlaceholder" />
                  <div class="mt-3 flex items-center gap-3">
                    <div class="relative flex-1">
                      <div class="absolute -top-2 left-3 bg-white px-1 text-[10px] font-bold leading-none text-slate-500">{{ copy.sort }}</div>
                      <input v-model="optionForm.sort_order" class="h-11 w-full rounded-xl border border-slate-200 px-4" :placeholder="copy.sort" type="number" />
                    </div>
                    <label class="inline-flex h-11 shrink-0 items-center gap-2 rounded-xl border border-slate-200 bg-slate-50 px-4 text-sm font-bold text-slate-600">
                      <input v-model="optionForm.is_correct" type="checkbox" />
                      {{ copy.correctAnswer }}
                    </label>
                  </div>
                  <button class="mt-3 w-full rounded-xl bg-blue-700 px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedQuestionId || savingOption" type="submit">
                    {{ savingOption ? copy.saving : copy.saveOption }}
                  </button>
                </form>
                <div v-if="selectedQuestion" class="border-t border-slate-200 p-4">
                  <h4 class="font-black">{{ copy.quizReadonlyFields }}</h4>
                  <div class="mt-3 max-h-72 space-y-3 overflow-y-auto pr-1">
                    <ReadonlyField v-for="entry in questionRecordEntries(selectedQuestion)" :key="`question-dialog-${entry.key}`" :label="entry.label" :text="entry.value" />
                  </div>
                </div>
              </section>
            </div>
          </div>
        </section>
      </Teleport>
    </main>

    <div v-if="importOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-0 md:p-6" @click.self="importOpen = false">
      <div class="flex h-full max-h-none w-full max-w-3xl flex-col overflow-hidden rounded-none bg-white p-4 shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl md:p-6">
        <div class="mb-5 flex items-center justify-between gap-4">
          <h2 class="min-w-0 text-xl font-black md:text-2xl">{{ copy.importTitle }}</h2>
          <button class="rounded-xl border px-3 py-2 font-bold" type="button" @click="importOpen = false">{{ copy.close }}</button>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto">
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
        <textarea v-model="importJson" class="mt-4 min-h-64 w-full rounded-xl border border-slate-200 p-4 font-mono text-sm md:min-h-80" :placeholder="copy.pasteJsonPlaceholder" />
        </div>
        <button class="mt-5 inline-flex w-full items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 py-3 font-bold text-white disabled:opacity-50 sm:w-auto" :disabled="importing" type="button" @click="importLmsJson">
          <Loader2 v-if="importing" class="h-4 w-4 animate-spin" />
          <UploadCloud v-else class="h-4 w-4" />
          {{ importing ? copy.importing : copy.startImport }}
        </button>
      </div>
    </div>
  </div>
  <teleport to="body">
    <div v-if="advancedMediaDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 p-4 backdrop-blur-sm">
      <div class="flex max-h-full w-full max-w-2xl flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
        <div class="flex items-center justify-between border-b border-slate-100 px-6 py-4">
          <h2 class="text-lg font-black">{{ copy.mediaJsonLabel }}</h2>
          <button class="rounded-full p-2 text-slate-400 hover:bg-slate-100 hover:text-slate-600" @click="advancedMediaDialogOpen = false">
            <X class="h-5 w-5" />
          </button>
        </div>
        <div class="overflow-y-auto bg-slate-50 px-6 py-5">
          <div v-if="!parsedMediaItems.length" class="text-center text-sm text-slate-400 py-4">{{ copy.mediaJsonHint }}</div>
          <div class="grid gap-3">
            <div v-for="(item, index) in parsedMediaItems" :key="index" class="flex gap-2 items-start bg-white p-3 rounded-xl border border-slate-200">
              <label class="block w-32 shrink-0">
                <span class="text-xs font-bold text-slate-500">{{ copy.mediaType }}</span>
                <select v-model="item.type" class="mt-1 h-9 w-full rounded-lg border border-slate-200 px-2 text-sm">
                  <option value="image">Image (图片)</option>
                  <option value="video">Video (视频)</option>
                </select>
              </label>
              <label class="block flex-1 min-w-0">
                <span class="text-xs font-bold text-slate-500">{{ copy.mediaUrl }}</span>
                <input v-model="item.url" class="mt-1 h-9 w-full rounded-lg border border-slate-200 px-2 text-sm" placeholder="https://" />
              </label>
              <button type="button" class="mt-5 rounded-lg p-2 text-red-500 hover:bg-red-50" @click="parsedMediaItems.splice(index, 1)" :title="copy.deleteMedia">
                <Trash2 class="h-4 w-4" />
              </button>
            </div>
          </div>
          <button type="button" @click="addMediaItem" class="mt-4 flex w-full items-center justify-center gap-2 rounded-xl border border-dashed border-slate-300 bg-white py-3 text-sm font-bold text-slate-500 hover:border-slate-400 hover:bg-slate-50 hover:text-slate-700">
            <Plus class="h-4 w-4" />
            {{ copy.addMedia }}
          </button>
        </div>
        <div class="flex justify-end gap-3 border-t border-slate-100 bg-white px-6 py-4">
          <button class="rounded-xl border border-slate-200 bg-white px-5 py-2.5 font-bold text-slate-600 hover:bg-slate-50" @click="advancedMediaDialogOpen = false">{{ copy.cancelConfig }}</button>
          <button class="rounded-xl bg-blue-700 px-5 py-2.5 font-bold text-white hover:bg-blue-800" @click="saveMediaConfig">{{ copy.confirmConfig }}</button>
        </div>
      </div>
    </div>
  </teleport>
</template>

