<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import {
  ArrowLeft,
  ArrowRight,
  BookOpen,
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
  courseUnitNextStepActionFromStatus,
  stageStatusHintLabel,
  statusLabel,
  timelineStatusBadgeClassForStatus,
  timelineStatusLabelWithDiagnostics,
} from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

type CourseCompleteResponse = {
  complete_course?: CompleteCourse
  quiz_progress?: Record<string, QuizProgressItem>
}

type CompleteCourse = {
  course?: Course
  chapters?: ChapterDetail[]
  materials?: CourseMaterialSummary[]
  quizzes?: any[]
}

type Course = {
  course_id?: string
  title?: string
  description?: string
  category_tips?: string
  duration_min?: number
}

type ChapterDetail = {
  chapter?: {
    chapter_id?: string
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
  title?: string
  lesson_type?: number
  body?: string
  external_url?: string
  video_embed_code?: string
}

type CourseMaterialSummary = {
  material_id?: string
  course_id?: string
  title?: string
  material_type?: number
  file_object_key?: string
  file_size?: number
  sort_order?: number
  file_hash?: string
}

type QuizProgressItem = {
  quiz_id?: string
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
}

type MaterialGroupKey = "all" | "textbook" | "slides" | "reference" | "other"

const route = useRoute()
const router = useRouter()
const { t } = useTranslation()

const payload = ref<CourseCompleteResponse | null>(null)
const loading = ref(false)
const syncing = ref(false)
const activeLessonId = ref("")
const activeChapterId = ref("")
const syncState = ref<SyncProgressRsp | null>(null)
const progressRecords = ref<ProgressRecord[]>([])
const selectedMaterialId = ref("")
const activeMaterialGroup = ref<MaterialGroupKey>("all")
const runtime = ref<any>(null)
const scheduleLoading = ref(false)
const lessonContentExpanded = ref(true)

const courseId = computed(() => String(route.query.courseId || ""))
const pipelineId = computed(() => String(route.query.pipelineId || ""))
const routeLessonId = computed(() => String(route.query.lessonId || ""))
const completeCourse = computed(() => payload.value?.complete_course)
const course = computed<Course | undefined>(() => completeCourse.value?.course)
const chapters = computed<ChapterDetail[]>(() => completeCourse.value?.chapters || [])
const materials = computed<CourseMaterialSummary[]>(() => completeCourse.value?.materials || [])
const courseQuizzes = computed<any[]>(() => completeCourse.value?.quizzes || [])
const quizProgress = computed(() => payload.value?.quiz_progress || {})

const lessons = computed<LessonDetail[]>(() =>
  chapters.value.flatMap((chapter, chapterIndex) =>
    (chapter.lessons || []).map((lessonDetail) => ({
      chapterTitle: chapter.chapter?.title || t.value.learning.chapters,
      chapterId: chapter.chapter?.chapter_id || `chapter-${chapterIndex}`,
      ...lessonDetail,
    })),
  ),
)

const activeLesson = computed(() => lessons.value.find((item) => item.lesson?.lesson_id === activeLessonId.value) || lessons.value[0])
const activeChapter = computed(() => {
  return (
    chapters.value.find((chapter) => chapter.chapter?.chapter_id === activeChapterId.value) ||
    chapters.value.find((chapter) => chapter.chapter?.chapter_id === activeLesson.value?.chapterId) ||
    chapters.value[0]
  )
})
const lesson = computed(() => activeLesson.value?.lesson)
const completedLessonIds = computed(() =>
  new Set(progressRecords.value.map((record) => record.material_id).filter((value): value is string => Boolean(value))),
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
const currentLessonRawCompleted = computed(() => Boolean(lesson.value?.lesson_id && completedLessonIds.value.has(lesson.value.lesson_id)))

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
    const quizId = quiz.quiz_id || ""
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
    const chapterId = chapter.chapter?.chapter_id || `chapter-${chapterIndex}`
    const chapterTitle = chapter.chapter?.title || `${t.value.learning.chapterPrefix} ${chapterIndex + 1}`
    ;(chapter.quizzes || []).forEach((quizDetail: any, index: number) => {
      const quiz = quizDetail.quiz || quizDetail || {}
      const quizId = quiz.quiz_id || ""
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
        const quizId = quiz.quiz_id || ""
        tasks.push({
          key: quizId || `lesson-${chapterIndex}-${lessonIndex}-quiz-${index}`,
          quizId,
          title: quiz.title || `${lessonTitle} ${t.value.learning.quizPrefix} ${index + 1}`,
          scope: "lesson",
          scopeLabel: t.value.learning.quizScopeLesson,
          ownerTitle: lessonTitle,
          chapterId,
          chapterTitle,
          lessonId: lessonDetail.lesson?.lesson_id,
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

function chapterIdFor(chapter: ChapterDetail, chapterIndex: number) {
  return chapter.chapter?.chapter_id || `chapter-${chapterIndex}`
}

function chapterCompleted(chapter: ChapterDetail, chapterIndex: number) {
  const chapterId = chapterIdFor(chapter, chapterIndex)
  const lessonTasks = (chapter.lessons || []).map((item) => ({
    completed: lessonFullyCompleted(item.lesson?.lesson_id),
  }))
  const chapterTasks = quizTasks.value.filter((task) => task.scope === "chapter" && task.chapterId === chapterId)
  const contentCount = lessonTasks.length + chapterTasks.length
  return contentCount > 0 && lessonTasks.every((item) => item.completed) && chapterTasks.every((task) => task.completed)
}

const activeChapterLessonTasks = computed(() => {
  const chapterId = activeChapter.value?.chapter?.chapter_id || activeChapterId.value
  return lessons.value
    .filter((item) => item.chapterId === chapterId)
    .map((item, index) => ({
      key: item.lesson?.lesson_id || `chapter-lesson-${index}`,
      lesson: item.lesson,
      chapterTitle: item.chapterTitle,
      completed: lessonFullyCompleted(item.lesson?.lesson_id),
    }))
})

const activeChapterQuizTasks = computed(() => {
  const chapterId = activeChapter.value?.chapter?.chapter_id || activeChapterId.value
  return quizTasks.value.filter((task) => task.scope === "chapter" && task.chapterId === chapterId)
})
const activeLessonQuizTasks = computed(() => (activeLessonId.value ? lessonQuizTasksByLessonId.value.get(activeLessonId.value) || [] : []))
const currentLessonCompleted = computed(() => currentLessonRawCompleted.value && activeLessonQuizTasks.value.every((task) => task.completed))
const visibleChapterAndLessonQuizTasks = computed(() => [...activeChapterQuizTasks.value, ...activeLessonQuizTasks.value])
const nextLearningLessonId = computed(() => {
  for (const item of lessons.value) {
    const candidate = item.lesson?.lesson_id
    if (candidate && !lessonFullyCompleted(candidate)) return candidate
  }
  return ""
})
const hasPendingQuizzes = computed(() => passedQuizzesCount.value < totalQuizzesCount.value)
const nextStepState = computed(() => {
  if (nextStep.value?.action) return nextStepDisplayFromAction(nextStep.value.action)
  return nextStepDisplay(nextUnitStatus.value, Boolean(nextLearningLessonId.value), Boolean(nextStep.value?.allow_retake), hasPendingQuizzes.value)
})

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
    filteredMaterials.value.find((item) => item.material_id === selectedMaterialId.value) ||
    materials.value.find((item) => item.material_id === selectedMaterialId.value) ||
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
        .find((item: LessonDetail) => item.lesson?.lesson_id)
      activeLessonId.value = firstLesson?.lesson?.lesson_id || ""
    }
    const firstMaterial = res?.complete_course?.materials?.find((item: CourseMaterialSummary) => item.material_id)
    if (!selectedMaterialId.value && firstMaterial?.material_id) selectedMaterialId.value = firstMaterial.material_id
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
}

async function startQuiz(quizId: string) {
  try {
    const res = await apiClient(`/api/quizzes/${quizId}/take`, { method: "POST" })
    if (res?.attempt_id) router.push(`/quizzes?attemptId=${encodeURIComponent(res.attempt_id)}`)
    else toast.error(t.value.common.error)
  } catch {
    toast.error(t.value.common.error)
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

function openExternalLesson() {
  const url = lesson.value?.external_url?.trim()
  if (!url) {
    toast.error(t.value.common.error)
    return
  }
  window.open(url, "_blank", "noopener,noreferrer")
}

async function markCompleted() {
  if (!lesson.value?.lesson_id) return
  if (currentLessonCompleted.value) {
    toast.success(t.value.learning.completedTag)
    return
  }
  if (lessonHasPendingQuizzes(lesson.value.lesson_id)) {
    toast.warning(t.value.learning.nextStepTakeQuizDesc)
    return
  }
  try {
    await apiClient(`/api/pipeline/lessons/${lesson.value.lesson_id}/complete`, { method: "POST" })
    toast.success(t.value.common.success)
    await refreshProgress(false)
  } catch {
    // apiClient handles localized errors.
  }
}

async function openMaterial(material: CourseMaterialSummary) {
  if (!material.material_id) return
  try {
    const res = await apiClient(`/api/pipeline/materials/${material.material_id}/url`)
    if (res?.url) window.open(res.url, "_blank", "noopener,noreferrer")
    else toast.error(t.value.common.error)
  } catch {
    // apiClient handles localized errors.
  }
}

async function downloadMaterial(material: CourseMaterialSummary) {
  if (!material.material_id) return
  try {
    const res = await apiClient(`/api/pipeline/materials/${material.material_id}/url`)
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
  }
}

async function selectLesson(lessonId?: string, chapterId?: string) {
  if (lessonId) activeLessonId.value = lessonId
  if (chapterId) activeChapterId.value = chapterId
  activeMaterialGroup.value = "all"
  if (materials.value.length > 0 && !selectedMaterialId.value) selectedMaterialId.value = materials.value[0].material_id || ""
  await refreshProgress(false)
}

function scrollToBottom() {
  window.scrollTo({ top: document.body.scrollHeight, behavior: "smooth" })
}

function nextStepLink() {
  if (nextStepState.value.action === "continue_learning") {
    return nextLearningLessonId.value
      ? `/courses/learn?courseId=${encodeURIComponent(nextStep.value?.course_id || courseId.value)}&pipelineId=${encodeURIComponent(pipelineId.value)}&lessonId=${encodeURIComponent(nextLearningLessonId.value)}`
      : `/courses/learn?courseId=${encodeURIComponent(courseId.value)}&pipelineId=${encodeURIComponent(pipelineId.value)}`
  }
  if (nextStepState.value.action === "view_certificate") return "/certificates"
  if (nextStepState.value.action === "signup_exam") {
    return `/exams/signup?unitId=${encodeURIComponent(nextStep.value?.course_unit_ulid || "")}&pipelineId=${encodeURIComponent(pipelineId.value)}&courseId=${encodeURIComponent(courseId.value)}`
  }
  return "/exams"
}

onMounted(async () => {
  activeLessonId.value = routeLessonId.value
  await loadCourse()
  if (courseId.value) {
    await loadProgress()
    await syncProgress(courseId.value, false)
  }
  await loadRuntime()
})

watch(courseId, async () => {
  activeLessonId.value = routeLessonId.value
  selectedMaterialId.value = ""
  await loadCourse()
  await loadProgress()
  await syncProgress(courseId.value, false)
})

watch(pipelineId, loadRuntime)
watch(lessons, () => {
  if (!activeLessonId.value && lessons.value.length > 0) activeLessonId.value = lessons.value[0].lesson?.lesson_id || ""
})
watch([activeLessonId, chapters], () => {
  if (activeChapterId.value) return
  if (activeLessonId.value) {
    const chapterFromLesson = lessons.value.find((item) => item.lesson?.lesson_id === activeLessonId.value)?.chapterId
    if (chapterFromLesson) {
      activeChapterId.value = chapterFromLesson
      return
    }
  }
  const firstChapterId = chapters.value[0]?.chapter?.chapter_id || (chapters.value.length > 0 ? "chapter-0" : "")
  if (firstChapterId) activeChapterId.value = firstChapterId
})
watch(materials, () => {
  if (!selectedMaterialId.value && materials.value.length > 0) selectedMaterialId.value = materials.value[0].material_id || ""
})
watch(selectedMaterial, () => {
  if (selectedMaterial.value?.material_id) selectedMaterialId.value = selectedMaterial.value.material_id
})
</script>

<template>
  <AppShell>
    <RouterLink :to="pipelineId ? `/courses/detail?id=${encodeURIComponent(pipelineId)}` : '/courses'" class="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
      <ArrowLeft class="h-4 w-4" />
      {{ t.learning.backToCourse }}
    </RouterLink>

    <div v-if="loading" class="text-muted-foreground">{{ t.common.loading }}</div>
    <div v-else-if="!course" class="rounded-lg border bg-card p-8 text-center text-muted-foreground">{{ t.common.na }}</div>
    <div v-else class="grid gap-6 lg:grid-cols-[340px_1fr]">
      <aside class="space-y-4">
        <div class="rounded-2xl border bg-card p-6 shadow-sm">
          <div class="mb-3 flex flex-wrap gap-2">
            <span class="badge border-0 bg-primary/10 text-primary">{{ t.learning.title }}</span>
            <span v-if="course.category_tips" class="badge">{{ course.category_tips }}</span>
          </div>
          <h1 class="text-2xl font-bold text-foreground">{{ course.title || t.common.unknownCourse }}</h1>
          <p class="mt-2 text-sm text-muted-foreground">{{ course.description || t.common.na }}</p>
          <div class="mt-4 flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
            <span class="inline-flex items-center gap-1.5"><BookOpen class="h-4 w-4" />{{ chapters.length }} {{ t.learning.chapters }}</span>
            <span class="inline-flex items-center gap-1.5"><Clock class="h-4 w-4" />{{ lessons.length }} {{ t.learning.lessons }}</span>
            <span class="inline-flex items-center gap-1.5 text-primary"><CheckCircle2 class="h-4 w-4" />{{ progressPercentage }}%</span>
          </div>

          <div class="mt-4 space-y-3">
            <div class="flex items-center justify-between text-xs text-muted-foreground">
              <span>{{ t.learning.progressLabel }}</span>
              <span>{{ completedLessonsCount }}/{{ lessons.length }} {{ t.learning.lessons }}</span>
            </div>
            <div class="h-2.5 overflow-hidden rounded-full bg-muted">
              <div class="h-full rounded-full bg-primary transition-all" :style="{ width: `${Math.max(0, Math.min(100, progressPercentage))}%` }" />
            </div>
            <div class="flex flex-wrap gap-2 text-xs text-muted-foreground">
              <span class="badge">{{ t.learning.completedLessonsBadge }} {{ completedLessonsCount }}</span>
              <span class="badge">{{ t.learning.passedQuizBadge }} {{ passedQuizzesCount }}</span>
              <span v-if="syncState?.course_status" class="badge">{{ t.learning.courseStatusLabel }}: {{ courseStatusLabel(syncState.course_status) }}</span>
            </div>
          </div>

          <div class="mt-4 flex flex-wrap gap-2">
            <RouterLink to="/exams" class="btn btn-outline py-1.5 text-xs">{{ t.learning.goToExams }}</RouterLink>
          </div>

          <div class="mt-4 rounded-xl border bg-muted/20 p-4">
            <div class="mb-2 flex items-center gap-2 text-sm font-semibold text-foreground">
              <Sparkles class="h-4 w-4 text-primary" />
              {{ t.learning.statusSummaryTitle }}
            </div>
            <p class="text-xs text-muted-foreground">{{ isPipelineTerminal ? nextStepState.desc : stageStatusHintLabel(t, currentStageStatus) }}</p>
            <div class="mt-3 flex flex-wrap gap-2 text-xs">
              <span :class="['badge', timelineStatusBadgeClassForStatus('PIPELINE', pipelineStatus)]">
                {{ t.learning.pipelineStatusLabel }}: {{ pipelineStatusLabel(pipelineStatus) }}
              </span>
              <span v-if="!isPipelineTerminal && currentStageName" class="badge">{{ t.learning.currentStageNameLabel }}: {{ currentStageName }}</span>
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
              <span class="badge">{{ t.learning.nextStepActionLabel }}: {{ nextStepState.label }}</span>
            </div>
          </div>

          <div v-if="nextStepState.action || nextUnitStatus" class="mt-4 rounded-xl border border-primary/20 bg-primary/5 p-4">
            <div class="mb-2 flex items-center gap-2 text-sm font-semibold text-primary">
              <Sparkles class="h-4 w-4" />
              {{ t.learning.nextStepTitle }}
            </div>
            <div class="text-sm text-muted-foreground">{{ nextStepState.desc }}</div>
            <div class="mt-3 flex items-center justify-between gap-3">
              <button v-if="nextStepState.action === 'schedule_exam'" class="btn btn-primary py-1.5 text-xs" :disabled="scheduleLoading" @click="handleScheduleExam">
                {{ nextStepState.label }}
                <ArrowRight class="ml-1 h-4 w-4" />
              </button>
              <button v-else-if="nextStepState.action === 'take_quiz'" class="btn btn-primary py-1.5 text-xs" @click="scrollToBottom">
                {{ nextStepState.label }}
                <ArrowRight class="ml-1 h-4 w-4" />
              </button>
              <button v-else-if="nextStepState.action === 'wait_sync'" class="btn btn-primary py-1.5 text-xs" :disabled="syncing" @click="refreshProgress(true)">
                <Loader2 v-if="syncing" class="mr-1 h-4 w-4 animate-spin" />
                {{ nextStepState.label }}
              </button>
              <RouterLink v-else :to="nextStepLink()" class="btn btn-primary py-1.5 text-xs">
                {{ nextStepState.label }}
                <ArrowRight class="ml-1 h-4 w-4" />
              </RouterLink>
            </div>
          </div>
        </div>

        <div class="rounded-2xl border bg-card shadow-sm">
          <div class="border-b px-5 py-4">
            <h2 class="text-sm font-semibold text-foreground">{{ t.learning.chapters }}</h2>
          </div>
          <div class="divide-y">
            <div
              v-for="(chapter, chapterIndex) in chapters"
              :key="chapter.chapter?.chapter_id || chapterIndex"
              :class="['px-5 py-4', (chapter.chapter?.chapter_id || `chapter-${chapterIndex}`) === activeChapterId ? 'bg-primary/5' : '']"
            >
              <button
                type="button"
                class="mb-3 flex w-full items-center gap-3 text-left"
                @click="selectLesson(chapter.lessons?.[0]?.lesson?.lesson_id, chapter.chapter?.chapter_id || `chapter-${chapterIndex}`)"
              >
                <div
                  :class="[
                    'flex h-8 w-8 items-center justify-center rounded-md text-sm font-semibold',
                    chapterCompleted(chapter, chapterIndex) ? 'bg-emerald-100 text-emerald-700' : 'bg-primary/10 text-primary',
                  ]"
                >
                  <CheckCircle2 v-if="chapterCompleted(chapter, chapterIndex)" class="h-4 w-4" />
                  <span v-else>{{ chapterIndex + 1 }}</span>
                </div>
                <div class="min-w-0 flex-1">
                  <div class="truncate font-medium text-foreground">{{ chapter.chapter?.title || `${t.learning.chapterPrefix} ${chapterIndex + 1}` }}</div>
                  <div class="text-xs text-muted-foreground">
                    {{ chapter.lessons?.length || 0 }} {{ t.learning.lessons }}
                  </div>
                </div>
                <ChevronRight class="h-4 w-4 shrink-0 text-muted-foreground" />
              </button>
              <div class="space-y-1 pl-11">
                <button
                  v-for="lessonDetail in chapter.lessons || []"
                  :key="lessonDetail.lesson?.lesson_id || lessonDetail.lesson?.title"
                  type="button"
                  :class="[
                    'flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm transition-colors',
                    lessonDetail.lesson?.lesson_id === activeLessonId ? 'bg-primary/10 text-primary' : 'hover:bg-muted',
                  ]"
                  @click="selectLesson(lessonDetail.lesson?.lesson_id, chapter.chapter?.chapter_id || `chapter-${chapterIndex}`)"
                >
                  <span class="flex min-w-0 items-center gap-2 truncate">
                    <CheckCircle2 v-if="lessonFullyCompleted(lessonDetail.lesson?.lesson_id)" class="h-3.5 w-3.5 shrink-0 text-emerald-500" />
                    <span v-else class="h-3.5 w-3.5 shrink-0 rounded-full border border-muted-foreground/30" />
                    <span class="truncate">{{ lessonDetail.lesson?.title || t.learning.unknownLesson }}</span>
                  </span>
                  <ChevronDown v-if="lessonDetail.lesson?.lesson_id === activeLessonId" class="h-4 w-4" />
                  <ChevronRight v-else class="h-4 w-4" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </aside>

      <section class="space-y-4">
        <div class="flex justify-end">
          <button class="btn btn-outline py-1.5 text-xs" :disabled="syncing" @click="refreshProgress(true)">
            <Loader2 v-if="syncing" class="h-4 w-4 animate-spin" />
            <RefreshCw v-else class="h-4 w-4" />
            {{ t.learning.syncProgress }}
          </button>
        </div>

        <div v-if="courseQuizTasks.length > 0 || visibleChapterAndLessonQuizTasks.length > 0" class="rounded-2xl border bg-card p-6 shadow-sm">
          <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <div class="mb-2 flex items-center gap-2">
                <Target class="h-5 w-5 text-primary" />
                <h2 class="text-xl font-semibold text-foreground">{{ t.learning.allQuizzesTitle }}</h2>
              </div>
            </div>
            <span class="badge">
              {{ courseQuizTasks.filter((task) => task.completed).length + visibleChapterAndLessonQuizTasks.filter((task) => task.completed).length }}/{{ courseQuizTasks.length + visibleChapterAndLessonQuizTasks.length }}
            </span>
          </div>

          <div class="grid gap-4 xl:grid-cols-2">
            <div class="rounded-xl border bg-primary/5 p-4">
              <div class="mb-3 flex items-center justify-between gap-3">
                <h3 class="font-semibold text-foreground">{{ t.learning.courseQuizzesTitle }}</h3>
                <span class="badge">{{ courseQuizTasks.filter((task) => task.completed).length }}/{{ courseQuizTasks.length }}</span>
              </div>
              <div v-if="courseQuizTasks.length === 0" class="rounded-lg border border-dashed bg-background p-4 text-sm text-muted-foreground">
                {{ t.learning.noCourseQuizzes }}
              </div>
              <div v-else class="space-y-2">
                <div v-for="(task, index) in courseQuizTasks" :key="task.key" class="rounded-lg border bg-background p-3">
                  <div class="mb-2 flex flex-wrap items-center gap-2">
                    <span class="badge">{{ task.scopeLabel }}</span>
                    <span v-if="task.completed" class="badge border-emerald-200 bg-emerald-50 text-emerald-700">
                      <CheckCircle2 class="mr-1 h-3.5 w-3.5" />{{ t.learning.completedTag }}
                    </span>
                  </div>
                  <div class="text-sm font-medium text-foreground">{{ index + 1 }}. {{ task.title }}</div>
                  <div class="mt-3">
                    <button class="btn btn-primary py-1.5 text-xs" :disabled="!task.quizId || task.completed" @click="startQuiz(task.quizId)">
                      {{ task.completed ? t.learning.completedTag : t.learning.takeQuiz }}
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <div class="rounded-xl border bg-muted/20 p-4">
              <div class="mb-3 flex items-center justify-between gap-3">
                <h3 class="font-semibold text-foreground">{{ t.learning.chapterQuizzesTitle }}</h3>
                <span class="badge">{{ visibleChapterAndLessonQuizTasks.filter((task) => task.completed).length }}/{{ visibleChapterAndLessonQuizTasks.length }}</span>
              </div>
              <div v-if="visibleChapterAndLessonQuizTasks.length === 0" class="rounded-lg border border-dashed bg-background p-4 text-sm text-muted-foreground">
                {{ t.learning.noChapterQuizzes }}
              </div>
              <div v-else class="space-y-2">
                <div v-for="(task, index) in visibleChapterAndLessonQuizTasks" :key="task.key" class="rounded-lg border bg-background p-3">
                  <div class="mb-2 flex flex-wrap items-center gap-2">
                    <span class="badge">{{ task.scopeLabel }}</span>
                    <span v-if="task.chapterTitle" class="badge">{{ t.learning.chapters }}: {{ task.chapterTitle }}</span>
                    <span v-if="task.lessonTitle" class="badge">{{ task.lessonTitle }}</span>
                    <span v-if="task.completed" class="badge border-emerald-200 bg-emerald-50 text-emerald-700">
                      <CheckCircle2 class="mr-1 h-3.5 w-3.5" />{{ t.learning.completedTag }}
                    </span>
                  </div>
                  <div class="text-sm font-medium text-foreground">{{ index + 1 }}. {{ task.title }}</div>
                  <div class="mt-3">
                    <button class="btn btn-primary py-1.5 text-xs" :disabled="!task.quizId || task.completed" @click="startQuiz(task.quizId)">
                      {{ task.completed ? t.learning.completedTag : t.learning.takeQuiz }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div id="lesson-detail" class="rounded-2xl border bg-card p-6 shadow-sm">
          <div class="grid gap-4 lg:grid-cols-[1fr_auto_1fr] lg:items-start">
            <div class="flex flex-wrap items-center gap-2">
              <span class="badge border-0 bg-primary/10 text-primary">{{ lessonTypeLabel(lesson?.lesson_type) }}</span>
              <span v-if="activeLesson?.chapterTitle" class="badge">{{ activeLesson.chapterTitle }}</span>
            </div>
            <h2 class="text-center text-2xl font-bold text-foreground">{{ lesson?.title || t.common.unknownCourse }}</h2>
            <div class="flex justify-start gap-2 lg:justify-end">
              <button
                :class="[
                  'btn',
                  currentLessonCompleted ? 'border border-emerald-200 bg-emerald-50 text-emerald-700 disabled:opacity-100' : 'btn-primary shadow-md shadow-primary/20',
                ]"
                :disabled="currentLessonCompleted"
                @click="markCompleted"
              >
                <CheckCircle2 class="h-4 w-4" />
                {{ currentLessonCompleted ? t.learning.completedTag : t.learning.completeLesson }}
              </button>
            </div>
          </div>

          <div class="mt-5 border-t pt-4">
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
              <div v-if="lesson?.video_embed_code" class="overflow-hidden rounded-xl border bg-muted" v-html="lesson.video_embed_code" />
              <div v-else-if="lesson?.external_url" class="space-y-4">
                <div class="rounded-xl border bg-muted/30 p-4 text-sm text-muted-foreground">{{ t.learning.noLessonBody }}</div>
                <button class="btn btn-primary" @click="openExternalLesson">
                  <ExternalLink class="h-4 w-4" />
                  {{ t.learning.openExternalLesson }}
                </button>
              </div>
              <div v-else class="prose max-w-none text-sm text-foreground">
                <div v-if="lesson?.body" v-html="lesson.body" />
                <div v-else class="rounded-xl border bg-muted/30 p-4 text-muted-foreground">{{ t.learning.noLessonBody }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="rounded-2xl border bg-card p-6 shadow-sm">
          <div class="mb-4 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <div class="mb-2 flex items-center gap-2">
                <Sparkles class="h-4 w-4 text-primary" />
                <h3 class="text-lg font-semibold text-foreground">{{ t.learning.materialsTitle }}</h3>
              </div>
              <p class="text-sm text-muted-foreground">{{ t.learning.materialsDesc }}</p>
            </div>
            <span class="badge">{{ materials.length }} {{ t.learning.materialsCountSuffix }}</span>
          </div>

          <div v-if="materials.length === 0" class="rounded-xl border bg-muted/30 p-6 text-center text-sm text-muted-foreground">
            {{ t.learning.materialsEmpty }}
            <div class="mt-2 text-xs text-muted-foreground">{{ t.learning.materialsEmptyHint }}</div>
          </div>
          <div v-else class="grid gap-4 xl:grid-cols-[240px_1fr]">
            <div class="rounded-xl border bg-muted/20 p-3">
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
                  :key="material.material_id || material.title"
                  type="button"
                  :class="[
                    'w-full rounded-lg border px-3 py-3 text-left transition-colors',
                    material.material_id === selectedMaterialId ? 'border-primary bg-primary/5' : 'bg-background hover:bg-muted/60',
                  ]"
                  @click="selectedMaterialId = material.material_id || ''"
                >
                  <div class="mb-1 flex items-center gap-2">
                    <span class="badge">{{ materialTypeLabel(material.material_type) }}</span>
                  </div>
                  <div class="line-clamp-2 text-sm font-medium text-foreground">{{ material.title || t.learning.unknownMaterial }}</div>
                  <div class="mt-1 text-xs text-muted-foreground">{{ formatFileSize(material.file_size) || t.learning.materialSizeUnknown }}</div>
                </button>
              </div>
            </div>

            <div class="rounded-xl border bg-background p-5">
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
                  <div class="rounded-lg border bg-muted/20 p-4">
                    <div class="text-xs text-muted-foreground">{{ t.learning.materialSizeLabel }}</div>
                    <div class="mt-1 text-sm font-medium text-foreground">{{ formatFileSize(selectedMaterial.file_size) || t.learning.materialSizeUnknown }}</div>
                  </div>
                  <div class="rounded-lg border bg-muted/20 p-4">
                    <div class="text-xs text-muted-foreground">{{ t.learning.materialHashLabel }}</div>
                    <div class="mt-1 break-all text-sm font-medium text-foreground">{{ selectedMaterial.file_hash || t.learning.materialHashUnknown }}</div>
                  </div>
                </div>

                <div class="rounded-xl border bg-muted/20 p-4">
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
                  <button class="btn btn-primary" @click="openMaterial(selectedMaterial)">
                    <Play class="h-4 w-4" />
                    {{ t.learning.openMaterial }}
                  </button>
                  <button class="btn btn-outline" @click="downloadMaterial(selectedMaterial)">
                    <Download class="h-4 w-4" />
                    {{ t.learning.downloadMaterial }}
                  </button>
                </div>
              </div>
              <div v-else class="flex min-h-[320px] items-center justify-center rounded-xl border border-dashed bg-muted/20 p-8 text-sm text-muted-foreground">
                {{ t.learning.materialPreviewEmpty }}
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  </AppShell>
</template>
