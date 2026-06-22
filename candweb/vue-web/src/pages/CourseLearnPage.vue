<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import {
  AlertCircle,
  ArrowLeft,
  ArrowRight,
  Award,
  BookOpen,
  CalendarClock,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Clock,
  Download,
  ExternalLink,
  FileText,
  Loader2,
  Play,
  RefreshCw,
  Sparkles,
  Target,
  Video,
} from "lucide-vue-next"
import {
  CANDIDATE_COURSE_STATUS_LABELS,
  EXAM_STATUS_LABELS,
  courseUnitNextStepActionFromStatus,
  normalizeEnumValueUpper,
  stageStatusHintLabel,
  statusBadgeClassForStatusValue,
  statusLabel,
  timelineStatusBadgeClassForStatus,
  timelineStatusLabelWithDiagnostics,
} from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import PaymentSessionDialog from "@/components/PaymentSessionDialog.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"
import { formatBackendDate, formatBackendDateOnly } from "@/lib/utils"
import {
  normalizeSupplementaryMaterials,
  parseSupplementaryMaterialItems,
  isPdfResourceUrl,
  type SupplementaryMaterial,
  type SupplementaryMaterialItem,
} from "@/lib/supplementaryMaterials"

type CourseCompleteResponse = {
  complete_course?: CompleteCourse
  supplementary_material?: SupplementaryMaterial | SupplementaryMaterial[]
  supplementaryMaterial?: SupplementaryMaterial | SupplementaryMaterial[]
  quiz_progress?: Record<string, QuizProgressItem>
}

type CompleteCourse = {
  course?: Course
  chapters?: ChapterDetail[]
  materials?: CourseMaterialSummary[]
  supplementary_material?: SupplementaryMaterial | SupplementaryMaterial[]
  supplementaryMaterial?: SupplementaryMaterial | SupplementaryMaterial[]
  quizzes?: any[]
}

type Course = {
  course_id?: string
  course_ulid?: string
  courseUlid?: string
  title?: string
  description?: string
  category_tips?: string
  duration_min?: number
}

type ChapterDetail = {
  chapter?: {
    chapter_id?: string
    chapter_ulid?: string
    chapterUlid?: string
    title?: string
    sort_order?: number
  }
  lessons?: LessonDetail[]
  quizzes?: any[]
}

type LessonDetail = {
  lesson?: Lesson
  quizzes?: any[]
  chapterTitle?: string
  chapterId?: string
}

type Lesson = {
  lesson_id?: string
  lesson_ulid?: string
  lessonUlid?: string
  title?: string
  lesson_type?: number
  body?: string
  external_url?: string
  video_embed_code?: string
}

type CourseMaterialSummary = {
  material_id?: string
  material_ulid?: string
  materialUlid?: string
  course_id?: string
  course_ulid?: string
  courseUlid?: string
  title?: string
  material_type?: number
  file_object_key?: string
  file_size?: number
  sort_order?: number
  file_hash?: string
}

type QuizProgressItem = {
  quiz_id?: string
  quiz_ulid?: string
  quizUlid?: string
  is_passed?: boolean
  status?: string
  attempt_id?: string
}

type QuizTask = {
  key: string
  quizId: string
  title: string
  scope: "course" | "chapter" | "lesson"
  scopeLabel: string
  ownerTitle: string
  chapterId?: string
  chapterTitle?: string
  lessonId?: string
  lessonTitle?: string
  completed?: boolean
}

type SyncProgressRsp = {
  success?: boolean
  course_status?: string
  progress_percentage?: number
  completed_lessons_count?: number
  passed_quizzes_count?: number
}

type ProgressRecord = {
  material_id?: string
  material_ulid?: string
  materialUlid?: string
}

type MaterialGroupKey = "all" | "textbook" | "slides" | "reference" | "other"
type LearnContentTabKey = "lesson" | "quiz" | "materials" | "exam" | "certificate"

const route = useRoute()
const router = useRouter()
const { t } = useTranslation()

const payload = ref<CourseCompleteResponse | null>(null)
const loading = ref(false)
const initializing = ref(false)
const syncing = ref(false)
const activeLessonId = ref("")
const syncState = ref<SyncProgressRsp | null>(null)
const progressRecords = ref<ProgressRecord[]>([])
const selectedMaterialId = ref("")
const openingMaterialId = ref("")
const downloadingMaterialId = ref("")
const startingQuizId = ref("")
const markingLessonComplete = ref(false)
const activeMaterialGroup = ref<MaterialGroupKey>("all")
const runtime = ref<any>(null)
const scheduleLoading = ref(false)
const retakeLoadingUnitId = ref<string | null>(null)
const lessonContentExpanded = ref(true)
const activeContentTab = ref<LearnContentTabKey>("lesson")
const courseExamsLoading = ref(false)
const courseExamsLoaded = ref(false)
const courseExams = ref<any[]>([])
const courseCertificateLoading = ref(false)
const courseCertificateUrl = ref("")
const courseCertificateError = ref("")
const retakePaymentSession = ref<{
  paymentKey?: string
  orderId?: string
  bizType: string
  bizRefUlid: string
  source: string
  returnPath: string
  extraReturnParams?: Record<string, string>
} | null>(null)
const retakePaymentDialogOpen = ref(false)

const courseId = computed(() => String(route.params.courseId || route.query.courseId || ""))
const pipelineId = computed(() => String(route.params.pipelineId || route.query.pipelineId || ""))
const routeLessonId = computed(() => String(route.params.lessonId || route.query.lessonId || ""))
const pageLoading = computed(() => loading.value || initializing.value)
const completeCourse = computed(() => payload.value?.complete_course)
const course = computed<Course | undefined>(() => completeCourse.value?.course)
const chapters = computed<ChapterDetail[]>(() => completeCourse.value?.chapters || [])
const materials = computed<CourseMaterialSummary[]>(() => completeCourse.value?.materials || [])
const supplementaryMaterials = computed<SupplementaryMaterial[]>(() => {
  const raw =
    completeCourse.value?.supplementary_material ??
    completeCourse.value?.supplementaryMaterial ??
    payload.value?.supplementary_material ??
    payload.value?.supplementaryMaterial
  return normalizeSupplementaryMaterials(raw)
})
const supplementaryMaterialItems = computed<SupplementaryMaterialItem[]>(() =>
  parseSupplementaryMaterialItems(supplementaryMaterials.value, t.value.learning.chapters),
)
const totalMaterialCount = computed(() => materials.value.length + supplementaryMaterialItems.value.length)
const courseQuizzes = computed<any[]>(() => completeCourse.value?.quizzes || [])
const quizProgress = computed(() => payload.value?.quiz_progress || {})

function firstString(...values: unknown[]) {
  for (const value of values) {
    const normalized = String(value || "").trim()
    if (normalized) return normalized
  }
  return ""
}

function courseIdOf(value?: Course | Record<string, unknown>) {
  return firstString(value?.course_id, value?.course_ulid, value?.courseUlid)
}

function chapterIdOf(value?: ChapterDetail["chapter"] | Record<string, unknown>) {
  return firstString(value?.chapter_id, value?.chapter_ulid, value?.chapterUlid)
}

function lessonIdOf(value?: Lesson | Record<string, unknown>) {
  return firstString(value?.lesson_id, value?.lesson_ulid, value?.lessonUlid)
}

function materialIdOf(value?: CourseMaterialSummary | Record<string, unknown>) {
  return firstString(value?.material_id, value?.material_ulid, value?.materialUlid)
}

function quizIdOf(value?: Record<string, unknown>) {
  return firstString(value?.quiz_id, value?.quiz_ulid, value?.quizUlid)
}

function progressLessonIdOf(value?: ProgressRecord | Record<string, unknown>) {
  return firstString(value?.material_id, value?.material_ulid, value?.materialUlid)
}

const lessons = computed<LessonDetail[]>(() =>
  chapters.value.flatMap((chapter, chapterIndex) =>
    (chapter.lessons || []).map((lessonDetail) => ({
      chapterTitle: chapter.chapter?.title || t.value.learning.chapters,
      chapterId: chapterIdOf(chapter.chapter) || `chapter-${chapterIndex}`,
      ...lessonDetail,
    })),
  ),
)

const activeLesson = computed(() => lessons.value.find((item) => lessonIdOf(item.lesson) === activeLessonId.value) || lessons.value[0])
const lesson = computed(() => activeLesson.value?.lesson)
const completedLessonIds = computed(() =>
  new Set(progressRecords.value.map(progressLessonIdOf).filter((value): value is string => Boolean(value))),
)
const progressPercentage = computed(() => syncState.value?.progress_percentage ?? 0)
const completedLessonsCount = computed(() => syncState.value?.completed_lessons_count ?? completedLessonIds.value.size)
const passedQuizzesCount = computed(() => syncState.value?.passed_quizzes_count ?? 0)
const nextStep = computed(() => runtime.value?.next_step || {})
const pipelineStatus = computed(() => runtime.value?.pipeline_status)
const isPipelineTerminal = computed(() => pipelineIsTerminal(pipelineStatus.value))
const currentStageName = computed(() => runtime.value?.current_stage_name || "")
const currentStageStatus = computed(() => runtime.value?.current_stage_status)
const currentUnitStatus = computed(() => runtime.value?.current_unit_status)
const nextUnitStatus = computed(() => nextStep.value?.status || currentUnitStatus.value)
const currentLessonId = computed(() => lessonIdOf(lesson.value))
const currentLessonRawCompleted = computed(() => Boolean(currentLessonId.value && completedLessonIds.value.has(currentLessonId.value)))
const pipelineHasCertificate = computed(() => {
  const certQuals = runtime.value?.config?.cert_quals || runtime.value?.config?.final_quals || []
  return Array.isArray(certQuals) && certQuals.some((qual) => firstString(qual?.qual_id, qual?.qualId))
})
const courseRuntimeUnit = computed(() => {
  const stages = runtime.value?.config?.stages || []
  for (const stage of stages) {
    for (const unit of stage.units || []) {
      if (unit.glms_course_id === courseId.value || unit.course_id === courseId.value || unit.course_ulid === courseId.value || unit.courseUlid === courseId.value) return unit
    }
  }
  return null
})
const courseRuntimeUnitStatus = computed(() => courseRuntimeUnit.value?.runtime_status || nextUnitStatus.value)
const courseRuntimeUnitUlid = computed(() => {
  const nextCourseId = firstString(nextStep.value?.course_id, nextStep.value?.course_ulid, nextStep.value?.courseUlid)
  if (nextCourseId && nextCourseId !== courseId.value) return ""
  return nextStep.value?.course_unit_ulid || ""
})
const hasCertificateTab = computed(() => pipelineHasCertificate.value && (nextStepState.value.action === "view_certificate" || pipelineIsTerminal(pipelineStatus.value)))
const courseCertificateSummary = computed(() => {
  const instance = runtime.value?.instance || {}
  const config = runtime.value?.config || {}
  const issuedAt = instance.completed_at || instance.updated_at || instance.created_at || config.created_at || ""
  return {
    name: config.name || course.value?.title || t.value.learning.actionViewCertificate,
    description: config.description || course.value?.description || t.value.learning.nextStepViewCertificateDesc,
    issueDate: issuedAt ? formatBackendDateOnly(issuedAt) : t.value.common.na,
    expiryDate: t.value.common.permanent,
    credentialId: instance.pipeline_ulid || pipelineId.value || t.value.common.na,
  }
})

const courseHasExam = computed(() => {
  const stages = runtime.value?.config?.stages || []
  for (const stage of stages) {
    for (const unit of stage.units || []) {
      if (unit.glms_course_id === courseId.value || unit.course_id === courseId.value || unit.course_ulid === courseId.value || unit.courseUlid === courseId.value) {
        return Boolean(unit.exam_id || unit.program)
      }
    }
  }
  return false
})
const hasExamTab = computed(() => courseHasExam.value || ["signup_exam", "schedule_exam", "view_exam_schedule", "apply_retake", "view_exam_result"].includes(nextStepState.value.action))
const courseExamTabCount = computed(() => {
  if (courseExams.value.length > 0) return courseExams.value.length
  if (!courseExamsLoaded.value && hasExamTab.value) return 1
  return 0
})

const totalQuizzesCount = computed(() => {
  let count = courseQuizzes.value.length
  for (const chapter of chapters.value) {
    if (chapter.quizzes) count += chapter.quizzes.length
    for (const lessonDetail of chapter.lessons || []) {
      if (lessonDetail.quizzes) count += lessonDetail.quizzes.length
    }
  }
  return count
})

function quizCompleted(quizId?: string) {
  return Boolean(quizId && quizProgress.value[quizId]?.is_passed)
}

const quizTasks = computed<QuizTask[]>(() => {
  const tasks: QuizTask[] = []
  courseQuizzes.value.forEach((quizDetail: any, index: number) => {
    const quiz = quizDetail.quiz || quizDetail || {}
    const quizId = quizIdOf(quiz)
    tasks.push({
      key: quizId || `course-quiz-${index}`,
      quizId,
      title: quiz.title || `${t.value.learning.quizPrefix} ${index + 1}`,
      scope: "course",
      scopeLabel: t.value.learning.quizScopeCourse,
      ownerTitle: course.value?.title || t.value.common.unknownCourse,
      completed: quizCompleted(quizId),
    })
  })
  chapters.value.forEach((chapter, chapterIndex) => {
    const chapterId = chapterIdOf(chapter.chapter) || `chapter-${chapterIndex}`
    const chapterTitle = chapter.chapter?.title || `${t.value.learning.chapterPrefix} ${chapterIndex + 1}`
    ;(chapter.quizzes || []).forEach((quizDetail: any, index: number) => {
      const quiz = quizDetail.quiz || quizDetail || {}
      const quizId = quizIdOf(quiz)
      tasks.push({
        key: quizId || `chapter-${chapterIndex}-quiz-${index}`,
        quizId,
        title: quiz.title || `${chapterTitle} ${t.value.learning.quizPrefix} ${index + 1}`,
        scope: "chapter",
        scopeLabel: t.value.learning.quizScopeChapter,
        ownerTitle: chapterTitle,
        chapterId,
        chapterTitle,
        completed: quizCompleted(quizId),
      })
    })
    ;(chapter.lessons || []).forEach((lessonDetail, lessonIndex) => {
      const lessonTitle = lessonDetail.lesson?.title || `${t.value.learning.unknownLesson} ${lessonIndex + 1}`
      ;(lessonDetail.quizzes || []).forEach((quizDetail: any, index: number) => {
        const quiz = quizDetail.quiz || quizDetail || {}
        const quizId = quizIdOf(quiz)
        tasks.push({
          key: quizId || `lesson-${chapterIndex}-${lessonIndex}-quiz-${index}`,
          quizId,
          title: quiz.title || `${lessonTitle} ${t.value.learning.quizPrefix} ${index + 1}`,
          scope: "lesson",
          scopeLabel: t.value.learning.quizScopeLesson,
          ownerTitle: lessonTitle,
          chapterId,
          chapterTitle,
          lessonId: lessonIdOf(lessonDetail.lesson),
          lessonTitle,
          completed: quizCompleted(quizId),
        })
      })
    })
  })
  return tasks
})

const courseQuizTasks = computed(() => quizTasks.value.filter((task) => task.scope === "course"))
const lessonQuizTasksByLessonId = computed(() => {
  const map = new Map<string, QuizTask[]>()
  quizTasks.value.forEach((task) => {
    if (task.scope !== "lesson" || !task.lessonId) return
    map.set(task.lessonId, [...(map.get(task.lessonId) || []), task])
  })
  return map
})

function lessonFullyCompleted(lessonId?: string) {
  if (!lessonId || !completedLessonIds.value.has(lessonId)) return false
  return (lessonQuizTasksByLessonId.value.get(lessonId) || []).every((task) => task.completed)
}

function lessonHasPendingQuizzes(lessonId?: string) {
  return Boolean(lessonId && (lessonQuizTasksByLessonId.value.get(lessonId) || []).some((task) => !task.completed))
}

const nonCourseQuizTasks = computed(() => quizTasks.value.filter((task) => task.scope !== "course"))
const activeLessonQuizTasks = computed(() => (activeLessonId.value ? lessonQuizTasksByLessonId.value.get(activeLessonId.value) || [] : []))
const currentLessonCompleted = computed(() => currentLessonRawCompleted.value && activeLessonQuizTasks.value.every((task) => task.completed))
const completedQuizTaskCount = computed(() => quizTasks.value.filter((task) => task.completed).length)
const learnContentTabs = computed(() => [
  {
    id: "lesson" as const,
    label: t.value.learning.lessonContentTitle,
    icon: BookOpen,
    count: lessons.value.length,
  },
  ...(quizTasks.value.length > 0
    ? [{
        id: "quiz" as const,
        label: t.value.learning.allQuizzesTitle,
        icon: Target,
        count: quizTasks.value.length,
      }]
    : []),
  ...(hasExamTab.value
    ? [{
        id: "exam" as const,
        label: t.value.sidebar.exams,
        icon: CalendarClock,
        count: courseExamTabCount.value,
      }]
    : []),
  ...(hasCertificateTab.value
    ? [{
        id: "certificate" as const,
        label: t.value.learning.actionViewCertificate,
        icon: Award,
        count: courseCertificateUrl.value ? 1 : 0,
      }]
    : []),
  {
    id: "materials" as const,
    label: t.value.learning.materialsTitle,
    icon: FileText,
    count: totalMaterialCount.value,
  },
])
const nextLearningLessonId = computed(() => {
  for (const item of lessons.value) {
    const candidate = lessonIdOf(item.lesson)
    if (candidate && !lessonFullyCompleted(candidate)) return candidate
  }
  return ""
})
const hasPendingQuizzes = computed(() => passedQuizzesCount.value < totalQuizzesCount.value)
const nextStepState = computed(() => {
  if (nextStep.value?.action) return nextStepDisplayFromAction(nextStep.value.action)
  return nextStepDisplay(nextUnitStatus.value, Boolean(nextLearningLessonId.value), Boolean(nextStep.value?.allow_retake), hasPendingQuizzes.value)
})
const sidebarNextActions = new Set(["signup_exam", "schedule_exam", "view_exam_schedule", "apply_retake", "view_exam_result", "view_certificate"])
const showSidebarNextAction = computed(() => sidebarNextActions.has(nextStepState.value.action))

const filteredMaterials = computed(() => {
  if (activeMaterialGroup.value === "all") return materials.value
  return materials.value.filter((material) => materialGroupKey(material.material_type) === activeMaterialGroup.value)
})
const groupedMaterials = computed(() => {
  const groups: Array<{ key: MaterialGroupKey; label: string; items: CourseMaterialSummary[] }> = [
    { key: "textbook", label: t.value.learning.materialTypeTextbook, items: [] },
    { key: "slides", label: t.value.learning.materialTypeSlides, items: [] },
    { key: "reference", label: t.value.learning.materialTypeReference, items: [] },
    { key: "other", label: t.value.learning.materialTypeOther, items: [] },
  ]
  for (const material of materials.value) {
    const target = groups.find((item) => item.key === materialGroupKey(material.material_type))
    target?.items.push(material)
  }
  return groups.filter((item) => item.items.length > 0)
})
const selectedMaterial = computed(() => {
  return (
    filteredMaterials.value.find((item) => materialIdOf(item) === selectedMaterialId.value) ||
    materials.value.find((item) => materialIdOf(item) === selectedMaterialId.value) ||
    filteredMaterials.value[0] ||
    materials.value[0]
  )
})

function pipelineIsTerminal(status?: string | number | null) {
  const normalized = String(status ?? "").trim()
  return normalized === "3" || normalized === "4"
}

function nextStepDisplayFromAction(action?: string) {
  switch (action) {
    case "continue_learning":
      return { action, label: t.value.learning.actionContinueLearning, desc: t.value.learning.nextStepContinueLearningDesc }
    case "wait_candidate":
      return { action, label: t.value.learning.actionWaitCandidate, desc: t.value.learning.nextStepWaitCandidateDesc }
    case "signup_exam":
      return { action, label: t.value.learning.actionSignupExam, desc: t.value.learning.nextStepGoToExamsDesc }
    case "schedule_exam":
      return { action, label: t.value.learning.actionScheduleExam, desc: t.value.learning.nextStepGoToExamsDesc }
    case "view_exam_schedule":
      return { action, label: t.value.learning.actionViewExamSchedule, desc: t.value.learning.nextStepGoToExamsDesc }
    case "apply_retake":
      return { action, label: t.value.learning.actionApplyRetake, desc: t.value.learning.nextStepGoToExamsDesc }
    case "view_exam_result":
      return { action, label: t.value.learning.actionViewExamResult, desc: t.value.learning.nextStepGoToExamsDesc }
    case "view_certificate":
      return { action, label: t.value.learning.actionViewCertificate, desc: t.value.learning.nextStepViewCertificateDesc }
    case "completed":
      return { action, label: t.value.learning.completedTag, desc: t.value.learning.nextStepDesc }
    default:
      return { action: "", label: t.value.common.unknown, desc: t.value.learning.nextStepDesc }
  }
}

function nextStepDisplay(status?: string | number | null, hasNextLesson = false, allowRetake = false, pendingQuizzes = false) {
  const action = courseUnitNextStepActionFromStatus(status, allowRetake)
  if (action === "continue_learning") {
    if (!hasNextLesson && pendingQuizzes) return { action: "take_quiz", label: t.value.learning.takeQuiz, desc: t.value.learning.nextStepTakeQuizDesc }
    if (!hasNextLesson) return { action: "wait_sync", label: t.value.learning.timelineRefresh, desc: t.value.learning.nextStepWaitSyncDesc }
    return { action, label: t.value.learning.actionContinueLearning, desc: t.value.learning.nextStepContinueLearningDesc }
  }
  if (action) return nextStepDisplayFromAction(action)
  return { action: "", label: t.value.common.unknown, desc: t.value.learning.nextStepDesc }
}

function formatFileSize(size?: number) {
  if (!size || size <= 0) return ""
  if (size < 1024) return `${size} B`
  const kb = size / 1024
  if (kb < 1024) return `${kb.toFixed(kb >= 100 ? 0 : 1)} KB`
  const mb = kb / 1024
  return `${mb.toFixed(mb >= 100 ? 0 : 1)} MB`
}

function courseStatusLabel(status?: string | number | null) {
  return statusLabel(t.value, CANDIDATE_COURSE_STATUS_LABELS, status)
}

function stageStatusLabel(status?: string | number | null) {
  return timelineStatusLabelWithDiagnostics(t.value, "STAGE", status)
}

function courseUnitStatusLabel(status?: string | number | null) {
  return timelineStatusLabelWithDiagnostics(t.value, "COURSE_UNIT", status)
}

function pipelineStatusLabel(status?: string | number | null) {
  return timelineStatusLabelWithDiagnostics(t.value, "PIPELINE", status)
}

function normalizedExamStatus(status?: string | number | null) {
  return normalizeEnumValueUpper(status)
}

function shouldShowExamStatus(status?: string | number | null) {
  const normalized = normalizedExamStatus(status)
  return Boolean(normalized && !["NONE", "UNKNOWN", "UNSPECIFIED"].some((item) => normalized.includes(item)))
}

function hasExamResult(exam: any) {
  const normalized = normalizedExamStatus(exam?.result_status)
  return typeof exam?.total_score === "number" || exam?.is_passed === true || ["DONE", "PASSED", "FAILED", "RESULT_STATUS_PASSED", "RESULT_STATUS_FAILED"].includes(normalized)
}

function hasExplicitPassStatus(exam: any) {
  return typeof exam?.is_passed === "boolean"
}

function hasText(value?: string | null) {
  return Boolean(value?.trim())
}

function hasAppointmentDetails(exam: any) {
  return hasText(exam?.confirmation_number) || hasText(exam?.site_name) || hasText(exam?.appointment_start_time) || hasText(exam?.appointment_end_time)
}

function canScheduleExam(exam: any) {
  const status = normalizedExamStatus(exam?.exam_status)
  return Boolean(exam?.exam_id && status && status.includes("OPEN"))
}

function isWaitingExamConfirmation(exam: any) {
  return normalizedExamStatus(exam?.exam_status) === "WAITING_EXAM_CONFIRMATION"
}

function isExamFailedUnit(exam: any) {
  return normalizeEnumValueUpper(exam?.course_unit_status).includes("EXAM_FAILED")
}

function canApplyRetake(exam: any) {
  return Boolean(exam?.course_unit_ulid && exam?.course_unit_cc_ulid && isExamFailedUnit(exam) && exam?.retake_eligible)
}

function noResultLabel() {
  return (t.value.examsPage as any).statusNoResult || t.value.examsPage.statusPending
}

function resultPublishedLabel() {
  return (t.value.examsPage as any).statusResultPublished || t.value.examsPage.statusPending
}

function passStatusLabel(exam: any) {
  return exam.is_passed ? (t.value.examsPage as any).statusQualified || t.value.examsPage.statusPassed : (t.value.examsPage as any).statusUnqualified || t.value.examsPage.statusFailed
}

function examStatusBadgeClass(status?: string | number | null) {
  const normalized = normalizedExamStatus(status)
  if (normalized.includes("PASSED") || normalized.includes("DONE") || normalized.includes("SUCCESS")) {
    return "border-[#6CE9A6] bg-[#ECFDF3] text-[#027A48]"
  }
  return statusBadgeClassForStatusValue(status)
}

function lessonTypeLabel(lessonType?: number) {
  switch (lessonType) {
    case 1:
      return t.value.learning.lessonTypeVideo
    case 2:
      return t.value.learning.lessonTypeText
    case 3:
      return t.value.learning.lessonTypePdf
    case 4:
      return t.value.learning.lessonTypeImage
    case 5:
      return t.value.learning.lessonTypeAudio
    case 6:
      return t.value.learning.lessonTypeFile
    case 7:
      return t.value.learning.lessonTypeLink
    default:
      return t.value.learning.lessonTypeUnknown
  }
}

function materialGroupKey(materialType?: number): MaterialGroupKey {
  switch (materialType) {
    case 1:
      return "textbook"
    case 2:
      return "slides"
    case 3:
      return "reference"
    case 4:
      return "other"
    default:
      return "other"
  }
}

function materialTypeLabel(materialType?: number) {
  switch (materialType) {
    case 1:
      return t.value.learning.materialTypeTextbook
    case 2:
      return t.value.learning.materialTypeSlides
    case 3:
      return t.value.learning.materialTypeReference
    case 4:
      return t.value.learning.materialTypeOther
    default:
      return t.value.learning.materialTypeUnknown
  }
}

function supplementaryChapterLabel(item: SupplementaryMaterialItem, index: number) {
  return supplementaryMaterialItems.value[index - 1]?.chapter === item.chapter ? "" : item.chapter
}

function supplementaryTypeLabel(type: string) {
  const normalized = type.trim().toLowerCase()
  if (normalized === "article") return "Article"
  if (normalized === "video") return "Video"
  if (normalized === "pdf") return "PDF"
  if (normalized === "link") return "Link"
  return type || t.value.learning.materialTypeUnknown
}

function supplementaryTypeClass(type: string) {
  const normalized = type.trim().toLowerCase()
  if (normalized === "video") return "border-violet-200 bg-violet-100 text-violet-700"
  if (normalized === "article" || normalized === "pdf") return "border-blue-200 bg-blue-100 text-blue-700"
  return "border-slate-200 bg-slate-100 text-slate-700"
}

function supplementaryTypeIcon(type: string) {
  return type.trim().toLowerCase() === "video" ? Play : FileText
}

function openSupplementaryPreview(item: SupplementaryMaterialItem) {
  if (!item.url) return
  if (!isPdfResourceUrl(item.url)) {
    window.open(item.url, "_blank", "noopener,noreferrer")
    return
  }

  openExternalPdfPreview(item.url, item.title || "Supplementary Material")
}

async function loadCourse() {
  if (!courseId.value) {
    payload.value = null
    loading.value = false
    return
  }
  loading.value = true
  try {
    const res = await apiClient(`/api/pipeline/courses/${courseId.value}/complete`)
    payload.value = res
    if (!activeLessonId.value) {
      const firstLesson = res?.complete_course?.chapters
        ?.flatMap((chapter: ChapterDetail) => chapter.lessons || [])
        .find((item: LessonDetail) => lessonIdOf(item.lesson))
      activeLessonId.value = lessonIdOf(firstLesson?.lesson)
    }
    const firstMaterial = res?.complete_course?.materials?.find((item: CourseMaterialSummary) => materialIdOf(item))
    const firstMaterialId = materialIdOf(firstMaterial)
    if (!selectedMaterialId.value && firstMaterialId) selectedMaterialId.value = firstMaterialId
  } finally {
    loading.value = false
  }
}

async function loadProgress() {
  if (!courseId.value) {
    progressRecords.value = []
    return
  }
  try {
    const res = await apiClient("/api/progress")
    progressRecords.value = res?.records || []
  } catch {
    progressRecords.value = []
  }
}

async function loadRuntime() {
  if (!pipelineId.value) {
    runtime.value = null
    return
  }
  try {
    runtime.value = await apiClient(`/api/mall/pipelines/${pipelineId.value}/runtime`)
  } catch {
    runtime.value = null
  }
}

async function loadCourseExams() {
  if (!hasExamTab.value) {
    courseExams.value = []
    courseExamsLoaded.value = false
    return
  }
  if (!courseRuntimeUnitUlid.value) {
    courseExams.value = []
    courseExamsLoaded.value = true
    return
  }
  courseExamsLoading.value = true
  try {
    const params = new URLSearchParams({
      page: "1",
      page_size: "20",
      course_unit_ulid: courseRuntimeUnitUlid.value,
    })
    const res = await apiClient(`/api/exams?${params.toString()}`)
    courseExams.value = res?.exams || []
  } catch {
    courseExams.value = []
  } finally {
    courseExamsLoaded.value = true
    courseExamsLoading.value = false
  }
}

async function loadCourseCertificate() {
  if (!hasCertificateTab.value || !runtime.value?.instance?.pipeline_ulid) {
    courseCertificateUrl.value = ""
    courseCertificateError.value = ""
    return
  }
  courseCertificateLoading.value = true
  courseCertificateError.value = ""
  try {
    const res = await apiClient(`/api/pipeline/${encodeURIComponent(runtime.value.instance.pipeline_ulid)}/certificate-url`)
    courseCertificateUrl.value = res?.view_url || ""
    if (!courseCertificateUrl.value) courseCertificateError.value = t.value.certificatesPage.certificateGenerating
  } catch {
    courseCertificateUrl.value = ""
    courseCertificateError.value = t.value.certificatesPage.certificateGenerating
  } finally {
    courseCertificateLoading.value = false
  }
}

function openCourseCertificate() {
  if (!courseCertificateUrl.value) return
  window.open(courseCertificateUrl.value, "_blank", "noopener,noreferrer")
}

async function syncProgress(targetCourseId = courseId.value, showToast = false) {
  if (!targetCourseId) return
  syncing.value = true
  try {
    syncState.value = await apiClient(`/api/progress/courses/${targetCourseId}/sync`, { method: "POST" })
    if (showToast) toast.success(t.value.common.success)
  } catch {
    // apiClient handles localized errors.
  } finally {
    syncing.value = false
  }
}

async function refreshProgress(showToast = false) {
  await syncProgress(courseId.value, showToast)
  await loadProgress()
  await loadCourse()
  await loadRuntime()
  if (activeContentTab.value === "exam") await loadCourseExams()
  if (activeContentTab.value === "certificate") await loadCourseCertificate()
}

async function startQuiz(quizId: string) {
  if (!quizId || startingQuizId.value) {
    if (!quizId) toast.error(t.value.common.error)
    return
  }
  startingQuizId.value = quizId
  try {
    const res = await apiClient(`/api/quizzes/${quizId}/take`, { method: "POST" })
    if (res?.attempt_id) await router.push(`/quizzes?attemptId=${encodeURIComponent(res.attempt_id)}`)
    else toast.error(t.value.common.error)
  } catch {
    toast.error(t.value.common.error)
  } finally {
    startingQuizId.value = ""
  }
}

async function handleScheduleExam() {
  const targetPipelineUlid = runtime.value?.instance?.pipeline_ulid
  if (!nextStep.value?.exam_id || !targetPipelineUlid) return
  scheduleLoading.value = true
  try {
    const termUrlBase = window.location.origin + "/api/public/webhooks/exams/callback"
    const res = await apiClient(`/api/exams/${encodeURIComponent(nextStep.value.exam_id)}/schedule-url?pipeline_ulid=${encodeURIComponent(targetPipelineUlid)}&course_ulid=${encodeURIComponent(nextStep.value.course_unit_ulid || "")}&url_type=1&term_url_base=${encodeURIComponent(termUrlBase)}`)
    if (res?.url) window.open(res.url, "_blank", "noopener,noreferrer")
    else toast.error(t.value.common.error)
  } finally {
    scheduleLoading.value = false
  }
}

async function handleInlineScheduleExam(exam: any) {
  if (!exam?.exam_id || scheduleLoading.value) return
  scheduleLoading.value = true
  try {
    const termUrlBase = window.location.origin + "/api/public/webhooks/exams/callback"
    const params = new URLSearchParams({ url_type: "schd", term_url_base: termUrlBase })
    if (runtime.value?.instance?.pipeline_ulid) params.set("pipeline_ulid", runtime.value.instance.pipeline_ulid)
    if (exam.course_unit_ulid || courseRuntimeUnitUlid.value) params.set("course_ulid", exam.course_unit_ulid || courseRuntimeUnitUlid.value)
    const res = await apiClient(`/api/exams/${encodeURIComponent(exam.exam_id)}/schedule-url?${params.toString()}`)
    if (res?.url) {
      toast.info(t.value.examsPage.scheduleRedirecting)
      window.open(res.url, "_blank", "noopener,noreferrer")
    } else {
      toast.error(t.value.examsPage.scheduleURLMissing)
    }
  } catch {
    toast.error(t.value.examsPage.scheduleFailed)
  } finally {
    scheduleLoading.value = false
  }
}

async function handleInlineApplyRetake(exam: any) {
  if (!canApplyRetake(exam) || retakeLoadingUnitId.value) return
  if (!exam.bundle_order_ulid) {
    toast.error(t.value.common.error)
    return
  }
  retakeLoadingUnitId.value = exam.course_unit_ulid
  try {
    const currentUrl = window.location.href
    const payment = await apiClient(`/api/exams/units/${encodeURIComponent(exam.course_unit_ulid)}/retake-payment`, {
      method: "POST",
      body: JSON.stringify({
        course_unit_cc_ulid: exam.course_unit_cc_ulid,
        bundle_order_ulid: exam.bundle_order_ulid,
        retried_count: exam.next_retried_count || exam.retried_count || 0,
        success_url: currentUrl,
        cancel_url: currentUrl,
      }),
    })
    if (payment?.payment_required && !payment?.paid) {
      retakePaymentSession.value = {
        paymentKey: payment.payment_key,
        orderId: payment.course_retake_order_ulid,
        bizType: "COURSE_RETAKE_PAYMENT",
        bizRefUlid: payment.course_retake_order_ulid,
        source: "retake",
        returnPath: window.location.pathname,
        extraReturnParams: {
          courseId: courseId.value,
          pipelineId: pipelineId.value,
        },
      }
      retakePaymentDialogOpen.value = true
      return
    }
    await apiClient(`/api/exams/units/${encodeURIComponent(exam.course_unit_ulid)}/retake`, { method: "POST" })
    toast.success(t.value.examsPage.retakeApplied)
    await router.push(`/exams/signup?unitId=${encodeURIComponent(exam.course_unit_ulid)}&pipelineId=${encodeURIComponent(exam.pipeline_ulid || pipelineId.value)}&courseId=${encodeURIComponent(courseId.value)}`)
  } catch {
    // apiClient handles localized errors.
  } finally {
    retakeLoadingUnitId.value = null
  }
}

function openExternalLesson() {
  const url = lesson.value?.external_url?.trim()
  if (!url) {
    toast.error(t.value.common.error)
    return
  }
  window.open(url, "_blank", "noopener,noreferrer")
}

async function markCompleted() {
  if (markingLessonComplete.value) return
  const lessonId = currentLessonId.value
  if (!lessonId) {
    toast.error(t.value.common.error)
    return
  }
  if (currentLessonCompleted.value) {
    toast.success(t.value.learning.completedTag)
    return
  }
  if (lessonHasPendingQuizzes(lessonId)) {
    toast.warning(t.value.learning.nextStepTakeQuizDesc)
    return
  }
  markingLessonComplete.value = true
  try {
    await apiClient(`/api/pipeline/lessons/${lessonId}/complete`, { method: "POST" })
    toast.success(t.value.common.success)
    await refreshProgress(false)
  } catch {
    // apiClient handles localized errors.
  } finally {
    markingLessonComplete.value = false
  }
}


async function openLessonPdf() {
  const lessonId = currentLessonId.value
  if (!lessonId) {
    toast.error(t.value.common.error)
    return
  }
  sessionStorage.setItem(`lesson-pdf-preview-title:${lessonId}`, lesson.value?.title || "PDF Preview")
  openPreviewTab(`/pdf-preview/lessons/${encodeURIComponent(lessonId)}`)
}

async function openInlinePdf(url: string) {
  openPreviewTab(url)
}

function openPreviewTab(url: string) {
  const resolved = router.resolve(url)
  const target = window.open(resolved.href, "_blank", "noopener,noreferrer")
  if (!target) {
    router.push(url)
  }
}

function openExternalPdfPreview(src: string, title: string) {
  const resourceKey = crypto.randomUUID()
  sessionStorage.setItem(`external-pdf-preview-src:${resourceKey}`, src)
  sessionStorage.setItem(`external-pdf-preview-title:${resourceKey}`, title)
  openPreviewTab(`/pdf-preview/resources/${encodeURIComponent(resourceKey)}`)
}

async function openMaterial(material: CourseMaterialSummary) {
  const materialId = materialIdOf(material)
  if (!materialId) return
  if (openingMaterialId.value) return
  openingMaterialId.value = materialId
  try {
    const res = await apiClient(`/api/pipeline/materials/${materialId}/url`)
    if (res?.url) {
      if (material.material_type === 3) {
        await openInlinePdf(res.url)
      } else {
        window.open(res.url, "_blank", "noopener,noreferrer")
      }
    } else toast.error(t.value.common.error)
  } catch {
    // apiClient handles localized errors.
  } finally {
    openingMaterialId.value = ""
  }
}

async function downloadMaterial(material: CourseMaterialSummary) {
  const materialId = materialIdOf(material)
  if (!materialId) return
  if (downloadingMaterialId.value) return
  downloadingMaterialId.value = materialId
  try {
    const res = await apiClient(`/api/pipeline/materials/${materialId}/url`)
    if (!res?.url) {
      toast.error(t.value.common.error)
      return
    }
    const link = document.createElement("a")
    link.href = res.url
    link.download = material.title || "material"
    link.rel = "noopener noreferrer"
    document.body.appendChild(link)
    link.click()
    link.remove()
  } catch {
    // apiClient handles localized errors.
  } finally {
    downloadingMaterialId.value = ""
  }
}

async function selectLesson(lessonId?: string) {
  if (lessonId) activeLessonId.value = lessonId
  activeContentTab.value = "lesson"
  activeMaterialGroup.value = "all"
  if (materials.value.length > 0 && !selectedMaterialId.value) selectedMaterialId.value = materialIdOf(materials.value[0])
  await refreshProgress(false)
}

function scrollToBottom() {
  activeContentTab.value = "quiz"
  requestAnimationFrame(() => {
    document.getElementById("course-learn-content")?.scrollIntoView({ behavior: "smooth", block: "start" })
  })
}

function nextStepLink() {
  if (nextStepState.value.action === "continue_learning") {
    const nextCourseId = firstString(nextStep.value?.course_id, nextStep.value?.course_ulid, nextStep.value?.courseUlid) || courseId.value
    return nextLearningLessonId.value
      ? `/certifications/${encodeURIComponent(pipelineId.value)}/learn/${encodeURIComponent(nextCourseId)}/lessons/${encodeURIComponent(nextLearningLessonId.value)}`
      : `/certifications/${encodeURIComponent(pipelineId.value)}/learn/${encodeURIComponent(courseId.value)}`
  }
  if (nextStepState.value.action === "view_certificate") return "/certificates"
  if (nextStepState.value.action === "signup_exam") {
    return `/exams/signup?unitId=${encodeURIComponent(nextStep.value?.course_unit_ulid || "")}&pipelineId=${encodeURIComponent(pipelineId.value)}&courseId=${encodeURIComponent(courseId.value)}`
  }
  return "/exams"
}

onMounted(async () => {
  initializing.value = true
  try {
    activeLessonId.value = routeLessonId.value
    await loadCourse()
    if (courseId.value) {
      await loadProgress()
      await syncProgress(courseId.value, false)
    }
    await loadRuntime()
  } finally {
    initializing.value = false
  }
})

watch(courseId, async () => {
  initializing.value = true
  try {
    activeLessonId.value = routeLessonId.value
    selectedMaterialId.value = ""
    courseExamsLoaded.value = false
    courseExams.value = []
    await loadCourse()
    await loadProgress()
    await syncProgress(courseId.value, false)
  } finally {
    initializing.value = false
  }
})

watch(pipelineId, loadRuntime)
watch(activeContentTab, async (tab) => {
  if (tab === "exam") await loadCourseExams()
  if (tab === "certificate") await loadCourseCertificate()
})
watch([runtime, courseId], async () => {
  if (activeContentTab.value === "exam") await loadCourseExams()
  if (activeContentTab.value === "certificate") await loadCourseCertificate()
})
watch(lessons, () => {
  if (!activeLessonId.value && lessons.value.length > 0) activeLessonId.value = lessonIdOf(lessons.value[0].lesson)
})
watch(materials, () => {
  if (!selectedMaterialId.value && materials.value.length > 0) selectedMaterialId.value = materialIdOf(materials.value[0])
})
watch(selectedMaterial, () => {
  const materialId = materialIdOf(selectedMaterial.value)
  if (materialId) selectedMaterialId.value = materialId
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <BookOpen class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ course?.title || t.common.unknownCourse }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <div class="mb-6 flex items-center justify-between gap-4">
          <RouterLink :to="pipelineId ? `/certifications/${encodeURIComponent(pipelineId)}` : '/certifications'" class="inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
            <ArrowLeft class="h-4 w-4" />
            {{ t.learning.backToCourse }}
          </RouterLink>
          <button v-if="course" class="btn btn-outline rounded-lg py-1.5 text-xs" :disabled="syncing" @click="refreshProgress(true)">
            <Loader2 v-if="syncing" class="h-4 w-4 animate-spin" />
            <RefreshCw v-else class="h-4 w-4" />
            {{ t.learning.syncProgress }}
          </button>
        </div>

    <div v-if="pageLoading" class="flex items-center justify-center gap-2 rounded-[16px] bg-white py-16 text-muted-foreground shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <Loader2 class="h-5 w-5 animate-spin" />
      <span>{{ t.common.loading }}</span>
    </div>
    <div v-else-if="!course" class="rounded-md bg-white p-8 text-center text-muted-foreground">
      <div class="mx-auto max-w-md space-y-4">
        <div>
          <h2 class="text-lg font-semibold text-foreground">{{ t.learning.courseUnavailableTitle }}</h2>
          <p class="mt-2 text-sm">{{ t.learning.courseUnavailableDesc }}</p>
        </div>
        <RouterLink :to="pipelineId ? `/certifications/${encodeURIComponent(pipelineId)}` : '/certifications'" class="btn btn-primary mx-auto w-fit rounded-lg">
          {{ pipelineId ? t.learning.backToCourse : t.courses.backToPipelines }}
        </RouterLink>
      </div>
    </div>
    <div v-else class="space-y-6">
      <section class="rounded-md bg-white p-6">
        <div class="min-w-0">
          <h1 class="text-2xl font-bold text-foreground">{{ course.title || t.common.unknownCourse }}</h1>
          <p class="mt-2 text-sm text-muted-foreground">{{ course.description || t.common.na }}</p>
          <div class="mt-4 flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
            <span class="inline-flex items-center gap-1.5"><BookOpen class="h-4 w-4" />{{ chapters.length }} {{ t.learning.chapters }}</span>
            <span class="inline-flex items-center gap-1.5"><Clock class="h-4 w-4" />{{ lessons.length }} {{ t.learning.lessons }}</span>
            <span class="inline-flex items-center gap-1.5 text-primary"><CheckCircle2 class="h-4 w-4" />{{ progressPercentage }}%</span>
            <span v-if="courseHasExam" class="inline-flex items-center gap-1.5 text-amber-600"><FileText class="h-4 w-4" />{{ t.learning.phaseExam }}</span>
            <span v-else class="inline-flex items-center gap-1.5 text-slate-500"><FileText class="h-4 w-4" />{{ t.learning.phaseLearning }}</span>
          </div>
          <div class="mt-4 flex flex-wrap gap-2 text-xs">
            <span :class="['badge', timelineStatusBadgeClassForStatus('PIPELINE', pipelineStatus)]">
              {{ t.learning.pipelineStatusLabel }}: {{ pipelineStatusLabel(pipelineStatus) }}
            </span>
            <span v-if="!isPipelineTerminal && currentStageName" class="badge border-slate-200 bg-white text-slate-700">{{ t.learning.currentStageNameLabel }}: {{ currentStageName }}</span>
            <span
              v-if="!isPipelineTerminal && currentStageStatus !== undefined && currentStageStatus !== ''"
              :class="['badge', timelineStatusBadgeClassForStatus('STAGE', currentStageStatus)]"
            >
              {{ t.learning.currentStageStatusLabel }}: {{ stageStatusLabel(currentStageStatus) }}
            </span>
            <span
              v-if="!isPipelineTerminal && currentUnitStatus !== undefined && currentUnitStatus !== ''"
              :class="['badge', timelineStatusBadgeClassForStatus('COURSE_UNIT', currentUnitStatus)]"
            >
              {{ t.learning.unitStatusLabel }}: {{ courseUnitStatusLabel(currentUnitStatus) }}
            </span>
            <span v-if="showSidebarNextAction" class="badge border-slate-200 bg-white text-slate-700">{{ t.learning.nextStepActionLabel }}: {{ nextStepState.label }}</span>
          </div>

          <div class="mt-5 space-y-3 rounded-md bg-slate-50 p-4">
            <div class="flex items-center justify-between text-xs text-muted-foreground">
              <span>{{ t.learning.progressLabel }}</span>
              <span>{{ completedLessonsCount }}/{{ lessons.length }} {{ t.learning.lessons }}</span>
            </div>
            <div class="h-2.5 overflow-hidden rounded-full bg-white">
              <div class="h-full rounded-full bg-primary transition-all" :style="{ width: `${Math.max(0, Math.min(100, progressPercentage))}%` }" />
            </div>
            <div class="flex flex-wrap gap-2 text-xs text-muted-foreground">
              <span class="badge border-slate-200 bg-white text-slate-700">{{ t.learning.completedLessonsBadge }} {{ completedLessonsCount }}</span>
              <span class="badge border-slate-200 bg-white text-slate-700">{{ t.learning.passedQuizBadge }} {{ passedQuizzesCount }}</span>
              <span v-if="syncState?.course_status" class="badge border-slate-200 bg-white text-slate-700">{{ t.learning.courseStatusLabel }}: {{ courseStatusLabel(syncState.course_status) }}</span>
            </div>
          </div>
        </div>
      </section>
      <section id="course-learn-content" class="grid gap-4 xl:grid-cols-[220px_minmax(0,1fr)]">
        <aside class="rounded-md bg-white p-4 xl:sticky xl:top-4 xl:self-start">
          <div class="space-y-2">
            <button
              v-for="tab in learnContentTabs"
              :key="tab.id"
              type="button"
              :class="[
                'flex w-full items-center gap-3 rounded-md border px-3 py-3 text-left text-sm transition-all',
                activeContentTab === tab.id
                  ? 'border-primary/30 bg-primary/10 text-primary shadow-sm'
                  : 'border-transparent bg-transparent text-slate-600 hover:border-slate-200 hover:bg-slate-50',
              ]"
              @click="activeContentTab = tab.id"
            >
              <span
                :class="[
                  'flex h-9 w-9 shrink-0 items-center justify-center rounded-md',
                  activeContentTab === tab.id ? 'bg-white text-primary' : 'bg-slate-50 text-slate-400',
                ]"
              >
                <component :is="tab.icon" class="h-4 w-4" />
              </span>
              <span class="min-w-0 flex-1 font-semibold">{{ tab.label }}</span>
              <span v-if="tab.count > 0" class="badge shrink-0 border-slate-200 bg-white text-slate-700">{{ tab.count }}</span>
            </button>
          </div>
        </aside>

        <div class="min-w-0 space-y-4">
        <div v-if="activeContentTab === 'exam'" class="rounded-md bg-white p-6">
          <div class="mb-4 flex items-start justify-between gap-4">
            <div>
              <div class="mb-2 flex items-center gap-2">
                <CalendarClock class="h-5 w-5 text-primary" />
                <h2 class="text-xl font-semibold text-foreground">{{ t.sidebar.exams }}</h2>
              </div>
              <p class="text-sm text-muted-foreground">{{ t.learning.nextStepGoToExamsDesc }}</p>
            </div>
            <span v-if="courseRuntimeUnitStatus" :class="['badge shrink-0', timelineStatusBadgeClassForStatus('COURSE_UNIT', courseRuntimeUnitStatus)]">
              {{ courseUnitStatusLabel(courseRuntimeUnitStatus) }}
            </span>
          </div>

          <div v-if="courseExamsLoading" class="flex items-center justify-center gap-2 rounded-md bg-slate-50 py-12 text-muted-foreground">
            <Loader2 class="h-5 w-5 animate-spin" />
            <span>{{ t.common.loading }}</span>
          </div>
          <div v-else-if="!courseRuntimeUnitUlid" class="rounded-md border border-dashed border-slate-200 bg-slate-50 p-8 text-center">
            <AlertCircle class="mx-auto mb-3 h-8 w-8 text-muted-foreground" />
            <h3 class="font-semibold text-foreground">{{ nextStepState.label }}</h3>
            <p class="mt-2 text-sm text-muted-foreground">{{ nextStepState.desc }}</p>
          </div>
          <div v-else-if="courseExams.length === 0" class="rounded-md border border-dashed border-slate-200 bg-slate-50 p-8 text-center">
            <CalendarClock class="mx-auto mb-3 h-8 w-8 text-primary" />
            <h3 class="font-semibold text-foreground">{{ t.examsPage.noExams }}</h3>
            <p class="mt-2 text-sm text-muted-foreground">{{ t.examsPage.noExamsDesc }}</p>
            <RouterLink
              v-if="nextStepState.action === 'signup_exam'"
              :to="nextStepLink()"
              class="btn btn-primary mx-auto mt-4 w-fit rounded-lg"
            >
              {{ nextStepState.label }}
              <ArrowRight class="ml-1 h-4 w-4" />
            </RouterLink>
          </div>
          <div v-else class="space-y-3">
            <div v-for="exam in courseExams" :key="exam.exam_id" class="rounded-md border border-slate-100 bg-slate-50 p-4">
              <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div class="min-w-0 space-y-3">
                  <div class="flex flex-wrap items-center gap-2">
                    <span v-if="shouldShowExamStatus(exam.exam_status)" :class="['badge', examStatusBadgeClass(exam.exam_status)]">{{ statusLabel(t, EXAM_STATUS_LABELS, normalizedExamStatus(exam.exam_status)) }}</span>
                    <span v-if="hasExamResult(exam)" :class="['badge', examStatusBadgeClass('DONE')]">{{ resultPublishedLabel() }}</span>
                    <span v-else :class="['badge', statusBadgeClassForStatusValue('PENDING')]">{{ noResultLabel() }}</span>
                    <span v-if="hasExplicitPassStatus(exam)" :class="['badge gap-1', exam.is_passed ? examStatusBadgeClass('SUCCESS') : statusBadgeClassForStatusValue('FAILED')]">
                      <CheckCircle2 v-if="exam.is_passed" class="h-3 w-3" />
                      {{ passStatusLabel(exam) }}
                    </span>
                  </div>
                  <h3 class="text-lg font-semibold text-foreground">{{ exam.exam_code || exam.program_code || exam.exam_id || t.common.unknown }}</h3>
                  <div class="grid gap-2 text-sm text-muted-foreground sm:grid-cols-2">
                    <div v-if="hasText(exam.confirmation_number)"><span class="font-medium text-foreground">{{ t.examsPage.confirmationNumber }}:</span> {{ exam.confirmation_number }}</div>
                    <div v-if="hasText(exam.site_name)"><span class="font-medium text-foreground">{{ t.examsPage.site }}:</span> {{ exam.site_name }}</div>
                    <div v-if="hasText(exam.appointment_start_time)"><span class="font-medium text-foreground">{{ t.examsPage.appointmentStart }}:</span> {{ formatBackendDate(exam.appointment_start_time) }}</div>
                    <div v-if="hasText(exam.appointment_end_time)"><span class="font-medium text-foreground">{{ t.examsPage.appointmentEnd }}:</span> {{ formatBackendDate(exam.appointment_end_time) }}</div>
                    <div v-if="isWaitingExamConfirmation(exam)" class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-amber-800 sm:col-span-2">
                      <div class="flex items-start gap-2">
                        <CalendarClock class="mt-0.5 h-4 w-4 shrink-0" />
                        <div class="text-xs">{{ t.examsPage.waitingExamConfirmationDesc }}</div>
                      </div>
                    </div>
                    <div v-if="!isWaitingExamConfirmation(exam) && !hasAppointmentDetails(exam) && !hasExamResult(exam)" class="rounded-lg border border-blue-200 bg-blue-50 px-3 py-2 text-blue-700 sm:col-span-2">
                      <div class="flex items-start gap-2">
                        <CalendarClock class="mt-0.5 h-4 w-4 shrink-0" />
                        <div>
                          <div class="font-medium text-blue-800">{{ t.examsPage.notScheduledTitle }}</div>
                          <div class="mt-1 text-xs">{{ t.examsPage.notScheduledDesc }}</div>
                        </div>
                      </div>
                    </div>
                    <div v-if="isExamFailedUnit(exam)" class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-red-700 sm:col-span-2">
                      <div class="flex items-start gap-2">
                        <AlertCircle class="mt-0.5 h-4 w-4 shrink-0" />
                        <div>
                          <div class="font-medium text-red-800">{{ t.examsPage.examFailedTitle }}</div>
                          <div class="mt-1 text-xs">{{ exam.retake_message || t.examsPage.examFailedDesc }}</div>
                        </div>
                      </div>
                    </div>
                    <div v-if="hasExamResult(exam)"><span class="font-medium text-foreground">{{ t.examsPage.score }}:</span> {{ typeof exam.total_score === 'number' ? exam.total_score.toFixed(2) : t.common.unknown }}</div>
                  </div>
                </div>
                <div class="flex shrink-0 flex-wrap gap-2">
                  <button v-if="canApplyRetake(exam)" class="btn btn-primary rounded-lg" :disabled="retakeLoadingUnitId === exam.course_unit_ulid" @click="handleInlineApplyRetake(exam)">
                    <Loader2 v-if="retakeLoadingUnitId === exam.course_unit_ulid" class="h-4 w-4 animate-spin" />
                    <RefreshCw v-else class="h-4 w-4" />
                    {{ t.examsPage.applyRetake }}
                  </button>
                  <button v-if="canScheduleExam(exam)" class="btn btn-primary rounded-lg" :disabled="scheduleLoading" @click="handleInlineScheduleExam(exam)">
                    <Loader2 v-if="scheduleLoading" class="h-4 w-4 animate-spin" />
                    <ExternalLink v-else class="h-4 w-4" />
                    {{ t.learning.actionScheduleExam }}
                  </button>
                  <RouterLink v-if="hasExamResult(exam)" :to="`/exams/result?examId=${encodeURIComponent(exam.exam_id)}`" class="btn btn-primary rounded-lg">{{ t.examsPage.viewResult }}</RouterLink>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="activeContentTab === 'certificate'" class="rounded-md bg-white p-6">
          <div class="mb-4 flex items-center justify-between gap-4">
            <div>
              <div class="mb-2 flex items-center gap-2">
                <Award class="h-5 w-5 text-primary" />
                <h2 class="text-xl font-semibold text-foreground">{{ t.learning.actionViewCertificate }}</h2>
              </div>
              <p class="text-sm text-muted-foreground">{{ t.learning.nextStepViewCertificateDesc }}</p>
            </div>
            <button class="btn btn-outline rounded-lg py-1.5 text-xs" :disabled="courseCertificateLoading" @click="loadCourseCertificate">
              <Loader2 v-if="courseCertificateLoading" class="h-4 w-4 animate-spin" />
              <RefreshCw v-else class="h-4 w-4" />
              {{ t.examsPage.refresh }}
            </button>
          </div>
          <div v-if="courseCertificateLoading" class="flex items-center justify-center gap-2 rounded-md bg-slate-50 py-12 text-muted-foreground">
            <Loader2 class="h-5 w-5 animate-spin" />
            <span>{{ t.common.loading }}</span>
          </div>
          <div v-else-if="courseCertificateUrl" class="overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.06)]">
            <div class="relative bg-primary p-4 text-white">
              <div class="relative flex items-start justify-between">
                <div>
                  <span class="badge mb-3 border-0 bg-white/20 text-white">
                    <CheckCircle2 class="mr-1 h-3 w-3" />
                    {{ t.certificatesPage.active }}
                  </span>
                  <h3 class="mb-1 text-xl font-bold">{{ courseCertificateSummary.name }}</h3>
                  <p class="text-sm text-white/80">{{ courseCertificateSummary.description }}</p>
                </div>
                <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-white/20 backdrop-blur-sm">
                  <Award class="h-6 w-6" />
                </div>
              </div>
            </div>
            <div class="p-4">
              <div class="mb-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div class="rounded-lg bg-[#f7fbfc] p-3">
                  <p class="mb-1 text-xs text-muted-foreground">{{ t.certificatesPage.issueDate }}</p>
                  <p class="flex items-center gap-1.5 font-medium text-card-foreground"><CalendarClock class="h-4 w-4 text-muted-foreground" /> {{ courseCertificateSummary.issueDate }}</p>
                </div>
                <div class="rounded-lg bg-[#f7fbfc] p-3">
                  <p class="mb-1 text-xs text-muted-foreground">{{ t.certificatesPage.expiryDate }}</p>
                  <p class="flex items-center gap-1.5 font-medium text-card-foreground"><CalendarClock class="h-4 w-4 text-muted-foreground" /> {{ courseCertificateSummary.expiryDate }}</p>
                </div>
              </div>
              <div class="mb-4 rounded-lg bg-[#f7fbfc] p-3">
                <p class="mb-1 text-xs text-muted-foreground">{{ t.certificatesPage.certificateId }}</p>
                <p class="break-all font-mono text-sm text-card-foreground">{{ courseCertificateSummary.credentialId }}</p>
              </div>
              <div class="flex flex-wrap gap-3">
                <button class="btn btn-primary flex-1 rounded-lg shadow-sm shadow-primary/20" @click="openCourseCertificate">
                  <Download class="h-4 w-4" />
                  {{ t.certificatesPage.downloadCertificate }}
                </button>
                <button class="btn btn-outline rounded-lg px-3" @click="openCourseCertificate">
                  <ExternalLink class="h-4 w-4" />
                </button>
              </div>
            </div>
          </div>
          <div v-else class="rounded-md border border-dashed border-slate-200 bg-slate-50 p-8 text-center">
            <Award class="mx-auto mb-3 h-8 w-8 text-primary" />
            <h3 class="font-semibold text-foreground">{{ t.certificatesPage.certificateGenerating }}</h3>
            <p class="mt-2 text-sm text-muted-foreground">{{ courseCertificateError || t.learning.nextStepViewCertificateDesc }}</p>
          </div>
        </div>

        <div v-if="activeContentTab === 'quiz'" class="rounded-md bg-white p-6">
          <div class="mb-4 flex items-center justify-between gap-4">
            <div>
              <div class="mb-2 flex items-center gap-2">
                <Target class="h-5 w-5 text-primary" />
                <h2 class="text-xl font-semibold text-foreground">{{ t.learning.allQuizzesTitle }}</h2>
              </div>
            </div>
            <span class="badge shrink-0 border-slate-200 bg-slate-50 text-slate-700">
              {{ completedQuizTaskCount }}/{{ quizTasks.length }}
            </span>
          </div>

          <div class="grid items-stretch gap-4 xl:grid-cols-2">
            <div class="flex min-h-[214px] flex-col rounded-md bg-slate-50 p-4">
              <div class="mb-3 flex items-center justify-between gap-3">
                <h3 class="font-semibold text-foreground">{{ t.learning.courseQuizzesTitle }}</h3>
                <span class="badge shrink-0 border-slate-200 bg-white text-slate-700">{{ courseQuizTasks.filter((task) => task.completed).length }}/{{ courseQuizTasks.length }}</span>
              </div>
              <div v-if="courseQuizTasks.length === 0" class="rounded-md border border-dashed border-slate-200 bg-white p-4 text-sm text-muted-foreground">
                {{ t.learning.noCourseQuizzes }}
              </div>
              <div v-else class="flex flex-1 flex-col gap-2">
                <div v-for="(task, index) in courseQuizTasks" :key="task.key" class="flex min-h-[142px] flex-1 flex-col rounded-md border border-slate-100 bg-white p-3">
                  <div class="mb-2 flex flex-wrap items-center gap-2">
                    <span class="badge border-slate-200 bg-slate-50 text-slate-700">{{ task.scopeLabel }}</span>
                    <span v-if="task.completed" class="badge border-emerald-200 bg-emerald-50 text-emerald-700">
                      <CheckCircle2 class="mr-1 h-3.5 w-3.5" />{{ t.learning.completedTag }}
                    </span>
                  </div>
                  <div class="text-sm font-medium text-foreground">{{ index + 1 }}. {{ task.title }}</div>
                  <div class="mt-auto pt-3">
                    <button class="btn btn-primary rounded-lg py-1.5 text-xs" :disabled="!task.quizId || task.completed || Boolean(startingQuizId)" @click="startQuiz(task.quizId)">
                      <Loader2 v-if="task.quizId && startingQuizId === task.quizId" class="h-4 w-4 animate-spin" />
                      {{ task.completed ? t.learning.completedTag : t.learning.takeQuiz }}
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <div class="flex min-h-[214px] flex-col rounded-md bg-slate-50 p-4">
              <div class="mb-3 flex items-center justify-between gap-3">
                <h3 class="font-semibold text-foreground">{{ t.learning.chapterQuizzesTitle }}</h3>
                <span class="badge shrink-0 border-slate-200 bg-white text-slate-700">{{ nonCourseQuizTasks.filter((task) => task.completed).length }}/{{ nonCourseQuizTasks.length }}</span>
              </div>
              <div v-if="nonCourseQuizTasks.length === 0" class="rounded-md border border-dashed border-slate-200 bg-white p-4 text-sm text-muted-foreground">
                {{ t.learning.noChapterQuizzes }}
              </div>
              <div v-else class="flex flex-1 flex-col gap-2">
                <div v-for="(task, index) in nonCourseQuizTasks" :key="task.key" class="flex min-h-[142px] flex-1 flex-col rounded-md border border-slate-100 bg-white p-3">
                  <div class="mb-2 flex flex-wrap items-center gap-2">
                    <span class="badge border-slate-200 bg-slate-50 text-slate-700">{{ task.scopeLabel }}</span>
                    <span v-if="task.chapterTitle" class="badge border-slate-200 bg-slate-50 text-slate-700">{{ t.learning.chapters }}: {{ task.chapterTitle }}</span>
                    <span v-if="task.lessonTitle" class="badge border-slate-200 bg-slate-50 text-slate-700">{{ task.lessonTitle }}</span>
                    <span v-if="task.completed" class="badge border-emerald-200 bg-emerald-50 text-emerald-700">
                      <CheckCircle2 class="mr-1 h-3.5 w-3.5" />{{ t.learning.completedTag }}
                    </span>
                  </div>
                  <div class="text-sm font-medium text-foreground">{{ index + 1 }}. {{ task.title }}</div>
                  <div class="mt-auto pt-3">
                    <button class="btn btn-primary rounded-lg py-1.5 text-xs" :disabled="!task.quizId || task.completed || Boolean(startingQuizId)" @click="startQuiz(task.quizId)">
                      <Loader2 v-if="task.quizId && startingQuizId === task.quizId" class="h-4 w-4 animate-spin" />
                      {{ task.completed ? t.learning.completedTag : t.learning.takeQuiz }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="activeContentTab === 'lesson'" class="grid gap-4 xl:grid-cols-[280px_minmax(0,1fr)]">
          <div class="rounded-md bg-white p-4">
            <div class="mb-3 flex items-center justify-between gap-3">
              <div class="flex items-center gap-2">
                <BookOpen class="h-4 w-4 text-primary" />
                <h2 class="text-sm font-semibold text-foreground">{{ t.learning.lessonContentTitle }}</h2>
              </div>
              <span class="badge shrink-0 border-slate-200 bg-slate-50 text-slate-700">{{ lessons.length }}</span>
            </div>
            <div v-if="lessons.length === 0" class="rounded-md border border-dashed border-slate-200 bg-slate-50 p-4 text-center text-sm text-muted-foreground">
              {{ t.learning.noChaptersDesc }}
            </div>
            <div v-else class="space-y-2">
              <button
                v-for="(lessonDetail, index) in lessons"
                :key="lessonIdOf(lessonDetail.lesson) || `lesson-${index}`"
                type="button"
                :class="[
                  'flex w-full items-start gap-3 rounded-md border px-3 py-3 text-left text-sm transition-all',
                  lessonIdOf(lessonDetail.lesson) === activeLessonId
                    ? 'border-primary/30 bg-primary/10 text-primary shadow-sm'
                    : 'border-slate-100 bg-slate-50 text-slate-700 hover:border-primary/20 hover:bg-white',
                ]"
                @click="selectLesson(lessonIdOf(lessonDetail.lesson))"
              >
                <span
                  :class="[
                    'mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-xs font-semibold',
                    lessonFullyCompleted(lessonIdOf(lessonDetail.lesson)) ? 'bg-emerald-100 text-emerald-700' : 'bg-white text-primary',
                  ]"
                >
                  <CheckCircle2 v-if="lessonFullyCompleted(lessonIdOf(lessonDetail.lesson))" class="h-4 w-4" />
                  <span v-else>{{ index + 1 }}</span>
                </span>
                <span class="min-w-0 flex-1">
                  <span class="block truncate font-semibold">{{ lessonDetail.lesson?.title || t.learning.unknownLesson }}</span>
                  <span v-if="lessonDetail.chapterTitle" class="mt-1 block truncate text-xs text-muted-foreground">{{ lessonDetail.chapterTitle }}</span>
                  <span class="mt-2 flex flex-wrap gap-1.5">
                    <span class="badge border-primary/15 bg-primary/10 text-[11px] text-primary">{{ lessonTypeLabel(lessonDetail.lesson?.lesson_type) }}</span>
                    <span v-if="lessonHasPendingQuizzes(lessonIdOf(lessonDetail.lesson))" class="badge border-amber-200 bg-amber-50 text-[11px] text-amber-700">{{ t.learning.quizScopeLesson }}</span>
                  </span>
                </span>
              </button>
            </div>
          </div>

          <div id="lesson-detail" class="rounded-md bg-white p-6">
            <div v-if="lesson" class="grid gap-4 lg:grid-cols-[1fr_auto_1fr] lg:items-start">
              <div class="flex flex-wrap items-center gap-2">
                <span class="badge border-primary/15 bg-primary/10 text-primary">{{ lessonTypeLabel(lesson?.lesson_type) }}</span>
                <span v-if="activeLesson?.chapterTitle" class="badge border-slate-200 bg-slate-50 text-slate-700">{{ activeLesson.chapterTitle }}</span>
              </div>
              <h2 class="text-center text-[20px] font-bold text-foreground">{{ lesson?.title || t.common.unknownCourse }}</h2>
              <div class="flex justify-start gap-2 lg:justify-end">
                <button
                  :class="[
                    'btn',
                    currentLessonCompleted ? 'border border-emerald-200 bg-emerald-50 text-emerald-700 disabled:opacity-100' : 'btn-primary',
                  ]"
                  :disabled="currentLessonCompleted || markingLessonComplete"
                  @click="markCompleted"
                >
                  <Loader2 v-if="markingLessonComplete" class="h-4 w-4 animate-spin" />
                  <CheckCircle2 v-else class="h-4 w-4" />
                  {{ currentLessonCompleted ? t.learning.completedTag : t.learning.completeLesson }}
                </button>
              </div>
            </div>

            <div v-if="lesson" class="mt-5 border-t pt-4">
              <button
                type="button"
                class="flex w-full items-center justify-between rounded-lg px-2 py-2 text-left text-sm font-semibold text-foreground transition-colors hover:bg-muted/60"
                @click="lessonContentExpanded = !lessonContentExpanded"
              >
                <span>{{ t.learning.lessonContentTitle }}</span>
                <ChevronDown v-if="lessonContentExpanded" class="h-4 w-4 text-muted-foreground" />
                <ChevronRight v-else class="h-4 w-4 text-muted-foreground" />
              </button>

              <div v-if="lessonContentExpanded" class="mt-3">
                <div v-if="lesson?.video_embed_code" class="overflow-hidden rounded-md bg-muted" v-html="lesson.video_embed_code" />
                <div v-else-if="lesson?.lesson_type === 3" class="space-y-4">
                  <div class="rounded-md bg-slate-50 p-4 text-sm text-muted-foreground">
                    <div v-if="lesson?.body" class="prose max-w-none text-sm text-foreground" v-html="lesson.body" />
                    <p v-else>{{ t.learning.lessonPdfHint }}</p>
                  </div>
                  <button class="btn btn-primary rounded-lg" @click="openLessonPdf">
                    <FileText class="mr-2 h-4 w-4" />
                    {{ t.learning.openLessonPdf }} <span v-if="lesson?.title" class="ml-1 font-normal opacity-90">- {{ lesson.title }}</span>
                  </button>
                </div>
                <div v-else-if="lesson?.external_url" class="space-y-4">
                  <div class="rounded-md bg-slate-50 p-4 text-sm text-muted-foreground">
                    <div v-if="lesson?.body" class="prose max-w-none text-sm text-foreground" v-html="lesson.body" />
                    <p v-else>{{ t.learning.noLessonBody }}</p>
                  </div>
                  <button class="btn btn-primary rounded-lg" @click="openExternalLesson">
                    <ExternalLink class="mr-2 h-4 w-4" />
                    {{ t.learning.openExternalLesson }} <span v-if="lesson?.title" class="ml-1 font-normal opacity-90">- {{ lesson.title }}</span>
                  </button>
                </div>
                <div v-else class="prose max-w-none text-sm text-foreground">
                  <div v-if="lesson?.body" v-html="lesson.body" />
                  <div v-else class="rounded-md bg-slate-50 p-4 text-muted-foreground">{{ t.learning.noLessonBody }}</div>
                </div>
              </div>
            </div>
            <div v-else class="rounded-md border border-dashed border-slate-200 bg-slate-50 p-8 text-center text-sm text-muted-foreground">
              {{ t.learning.noChaptersDesc }}
            </div>
          </div>
        </div>

        <div v-if="activeContentTab === 'materials'" class="rounded-md bg-white p-6">
          <div class="mb-4 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <div class="mb-2 flex items-center gap-2">
                <Sparkles class="h-4 w-4 text-primary" />
                <h3 class="text-lg font-semibold text-foreground">{{ t.learning.materialsTitle }}</h3>
              </div>
              <p class="text-sm text-muted-foreground">{{ t.learning.materialsDesc }}</p>
            </div>
            <span class="badge border-slate-200 bg-slate-50 text-slate-700">{{ totalMaterialCount }} {{ t.learning.materialsCountSuffix }}</span>
          </div>

          <div v-if="totalMaterialCount === 0" class="rounded-md bg-slate-50 p-6 text-center text-sm text-muted-foreground">
            {{ t.learning.materialsEmpty }}
            <div class="mt-2 text-xs text-muted-foreground">{{ t.learning.materialsEmptyHint }}</div>
          </div>
          <div v-else class="space-y-4">
            <div v-if="supplementaryMaterialItems.length > 0" class="rounded-md border border-slate-100 bg-slate-50 p-4">
              <div class="border-b border-slate-100 pb-3">
                <div class="flex items-center gap-2 text-sm font-semibold text-foreground">
                  <BookOpen class="h-4 w-4 text-primary" />
                  <span>Supplementary Materials</span>
                  <span class="badge border-slate-200 bg-white text-slate-700">{{ supplementaryMaterialItems.length }} {{ t.learning.materialsCountSuffix }}</span>
                </div>
                <p class="mt-1 text-xs text-muted-foreground">Additional learning resources organized by chapter</p>
              </div>

              <div class="hidden grid-cols-[minmax(160px,0.9fr)_120px_minmax(260px,1.4fr)_180px] border-b border-slate-100 px-3 py-3 text-sm font-medium text-muted-foreground md:grid">
                <div>Chapter</div>
                <div>Type</div>
                <div>Title & Description</div>
                <div>Resource Link</div>
              </div>

              <div class="divide-y divide-slate-100">
                <div
                  v-for="(item, index) in supplementaryMaterialItems"
                  :key="item.key"
                  class="grid gap-3 px-3 py-4 text-sm md:grid-cols-[minmax(160px,0.9fr)_120px_minmax(260px,1.4fr)_180px]"
                >
                  <div class="font-medium text-foreground">
                    <span class="md:hidden text-xs text-muted-foreground">Chapter: </span>
                    {{ supplementaryChapterLabel(item, index) }}
                  </div>
                  <div>
                    <span class="badge gap-1 border text-xs" :class="supplementaryTypeClass(item.type)">
                      <component :is="supplementaryTypeIcon(item.type)" class="h-3 w-3" />
                      {{ supplementaryTypeLabel(item.type) }}
                    </span>
                  </div>
                  <div>
                    <div class="font-semibold text-foreground">{{ item.title }}</div>
                    <p v-if="item.description" class="mt-1 text-xs leading-relaxed text-muted-foreground">{{ item.description }}</p>
                  </div>
                  <div class="min-w-0">
                    <button
                      v-if="item.url"
                      class="inline-flex max-w-full items-center gap-1 rounded-lg border border-primary/20 bg-white px-3 py-2 text-left text-xs font-semibold text-primary transition-colors hover:bg-primary/10"
                      :title="item.url"
                      @click="openSupplementaryPreview(item)"
                    >
                      <ExternalLink class="h-3.5 w-3.5 shrink-0" />
                      <span class="truncate">{{ item.url }}</span>
                    </button>
                    <span v-else class="text-xs text-muted-foreground">No resource_link</span>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="materials.length > 0" class="grid gap-4 xl:grid-cols-[240px_1fr]">
              <div class="rounded-md bg-slate-50 p-3">
                <div class="mb-3 flex flex-wrap gap-2">
                  <button :class="['btn py-1.5 text-xs', activeMaterialGroup === 'all' ? 'btn-primary' : 'btn-outline']" @click="activeMaterialGroup = 'all'">
                    {{ t.learning.materialGroupAll }}
                  </button>
                  <button
                    v-for="group in groupedMaterials"
                    :key="group.key"
                    :class="['btn py-1.5 text-xs', activeMaterialGroup === group.key ? 'btn-primary' : 'btn-outline']"
                    @click="activeMaterialGroup = group.key"
                  >
                    {{ group.label }}
                  </button>
                </div>

                <div class="space-y-2">
                  <button
                    v-for="material in filteredMaterials"
                    :key="materialIdOf(material) || material.title"
                    type="button"
                    :class="[
                      'w-full rounded-md border px-3 py-3 text-left transition-colors',
                      materialIdOf(material) === selectedMaterialId ? 'border-primary bg-white' : 'border-slate-100 bg-white hover:bg-slate-100',
                    ]"
                    @click="selectedMaterialId = materialIdOf(material)"
                  >
                    <div class="mb-1 flex items-center gap-2">
                      <span class="badge border-slate-200 bg-slate-50 text-slate-700">{{ materialTypeLabel(material.material_type) }}</span>
                    </div>
                    <div class="line-clamp-2 text-sm font-medium text-foreground">{{ material.title || t.learning.unknownMaterial }}</div>
                    <div class="mt-1 text-xs text-muted-foreground">{{ formatFileSize(material.file_size) || t.learning.materialSizeUnknown }}</div>
                  </button>
                </div>
              </div>

              <div class="rounded-md bg-slate-50 p-5">
                <div v-if="selectedMaterial" class="space-y-5">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="badge">{{ materialTypeLabel(selectedMaterial.material_type) }}</span>
                    <span v-if="selectedMaterial.sort_order !== undefined" class="badge">{{ t.learning.materialSortOrder }} {{ selectedMaterial.sort_order }}</span>
                  </div>
                  <div>
                    <h4 class="text-xl font-semibold text-foreground">{{ selectedMaterial.title || t.learning.unknownMaterial }}</h4>
                    <p class="mt-2 text-sm text-muted-foreground">{{ selectedMaterial.file_object_key || t.learning.materialFileKeyUnknown }}</p>
                  </div>

                  <div class="grid gap-3 sm:grid-cols-2">
                    <div class="rounded-lg bg-muted/20 p-4">
                      <div class="text-xs text-muted-foreground">{{ t.learning.materialSizeLabel }}</div>
                      <div class="mt-1 text-sm font-medium text-foreground">{{ formatFileSize(selectedMaterial.file_size) || t.learning.materialSizeUnknown }}</div>
                    </div>
                    <div class="rounded-lg bg-muted/20 p-4">
                      <div class="text-xs text-muted-foreground">{{ t.learning.materialHashLabel }}</div>
                      <div class="mt-1 break-all text-sm font-medium text-foreground">{{ selectedMaterial.file_hash || t.learning.materialHashUnknown }}</div>
                    </div>
                  </div>

                  <div class="rounded-lg bg-muted/20 p-4">
                    <div class="mb-2 text-xs font-medium uppercase text-muted-foreground">{{ t.learning.materialPreviewLabel }}</div>
                    <div class="flex min-h-[240px] items-center justify-center rounded-lg border border-dashed bg-background p-6 text-center text-sm text-muted-foreground">
                      <div class="space-y-3">
                        <FileText v-if="selectedMaterial.material_type === 1 || selectedMaterial.material_type === 2" class="mx-auto h-10 w-10 text-primary" />
                        <BookOpen v-else-if="selectedMaterial.material_type === 3" class="mx-auto h-10 w-10 text-primary" />
                        <Video v-else class="mx-auto h-10 w-10 text-primary" />
                        <div>{{ t.learning.materialPreviewHint }}</div>
                      </div>
                    </div>
                  </div>

                  <div class="flex flex-wrap gap-2">
                    <button class="btn btn-primary" :disabled="openingMaterialId === materialIdOf(selectedMaterial)" @click="openMaterial(selectedMaterial)">
                      <Loader2 v-if="openingMaterialId === materialIdOf(selectedMaterial)" class="h-4 w-4 animate-spin" />
                      <Play v-else class="h-4 w-4" />
                      {{ t.learning.openMaterial }}
                    </button>
                    <button class="btn btn-outline" :disabled="downloadingMaterialId === materialIdOf(selectedMaterial)" @click="downloadMaterial(selectedMaterial)">
                      <Loader2 v-if="downloadingMaterialId === materialIdOf(selectedMaterial)" class="h-4 w-4 animate-spin" />
                      <Download v-else class="h-4 w-4" />
                      {{ t.learning.downloadMaterial }}
                    </button>
                  </div>
                </div>
                <div v-else class="flex min-h-[320px] items-center justify-center rounded-lg border border-dashed bg-muted/20 p-8 text-sm text-muted-foreground">
                  {{ t.learning.materialPreviewEmpty }}
                </div>
              </div>
            </div>
          </div>
        </div>
        </div>
      </section>
    </div>
    <PaymentSessionDialog
      v-if="retakePaymentSession"
      v-model:open="retakePaymentDialogOpen"
      :title="t.examsPage.applyRetake"
      :subtitle="retakePaymentSession.orderId"
      :payment-key="retakePaymentSession.paymentKey"
      :biz-type="retakePaymentSession.bizType"
      :biz-ref-ulid="retakePaymentSession.bizRefUlid"
      :order-id="retakePaymentSession.orderId"
      :source="retakePaymentSession.source"
      :return-path="retakePaymentSession.returnPath"
      :extra-return-params="retakePaymentSession.extraReturnParams"
    />
      </main>
    </div>
  </AppShell>
</template>
