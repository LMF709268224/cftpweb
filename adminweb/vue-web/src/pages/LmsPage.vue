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
  title: string
  sort_order: string
  lesson_type: string
  body: string
  asset_object_key: string
  asset_file_hash: string
}

const pageSize = 20

const courses = ref<JsonRecord[]>([])
const selectedCourse = ref<JsonRecord | null>(null)
const courseDetail = ref<JsonRecord | null>(null)
const chapters = ref<JsonRecord[]>([])
const selectedChapter = ref<JsonRecord | null>(null)
const lessons = ref<JsonRecord[]>([])

const loading = ref(false)
const detailLoading = ref(false)
const chaptersLoading = ref(false)
const lessonsLoading = ref(false)
const savingCourse = ref(false)
const savingChapter = ref(false)
const savingLesson = ref(false)
const publishing = ref(false)
const importing = ref(false)

const categoryFilter = ref("")
const publishedOnly = ref(false)
const nextPageToken = ref("")
const courseForm = ref<CourseForm>(emptyCourseForm())
const chapterForm = ref<ChapterForm>(emptyChapterForm())
const lessonForm = ref<LessonForm>(emptyLessonForm())
const editingChapterId = ref("")
const editingLessonId = ref("")
const importOpen = ref(false)
const importScope = ref<"course" | "quiz">("course")
const importCategoryTips = ref("")
const importJson = ref("")

const selectedCourseId = computed(() => courseId(selectedCourse.value))
const selectedChapterId = computed(() => chapterId(selectedChapter.value))
const selectedCoursePublished = computed(() => Boolean(selectedCourse.value?.is_published))
const selectedCourseStatus = computed(() => selectedCourse.value?.status || (selectedCoursePublished.value ? "Published" : "Draft"))

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
    title: "",
    sort_order: "1",
    lesson_type: "2",
    body: "",
    asset_object_key: "",
    asset_file_hash: "",
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

function versionOf(record: JsonRecord | null | undefined) {
  return Number(record?.version || 0)
}

function courseTitle(course: JsonRecord | null | undefined) {
  return String(pickFirst(course || {}, ["title", "name", "course_title"]) || courseId(course) || "课程")
}

function chapterTitle(chapter: JsonRecord | null | undefined) {
  return String(pickFirst(chapter || {}, ["title", "name"]) || chapterId(chapter) || "章节")
}

function lessonTitle(lesson: JsonRecord | null | undefined) {
  return String(pickFirst(lesson || {}, ["title", "name"]) || lessonId(lesson) || "课时")
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
  chapters.value = []
  selectedChapter.value = null
  lessons.value = []
  editingChapterId.value = ""
  editingLessonId.value = ""
  chapterForm.value = emptyChapterForm()
  lessonForm.value = emptyLessonForm()
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
  await Promise.all([loadCourseDetail(), loadChapters()])
}

function newCourse() {
  selectedCourse.value = null
  courseForm.value = emptyCourseForm()
  resetContent()
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
}

function newChapter() {
  selectedChapter.value = null
  editingChapterId.value = ""
  chapterForm.value = emptyChapterForm()
  lessons.value = []
  editingLessonId.value = ""
  lessonForm.value = emptyLessonForm()
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
  lessonForm.value = {
    title: String(lesson.title || ""),
    sort_order: String(lesson.sort_order || 1),
    lesson_type: String(lesson.lesson_type || 2),
    body: String(lesson.body || ""),
    asset_object_key: String(lesson.media_object_key || lesson.asset_object_key || lesson.file_object_key || ""),
    asset_file_hash: String(lesson.media_file_hash || lesson.asset_file_hash || lesson.file_hash || ""),
  }
}

function newLesson() {
  editingLessonId.value = ""
  lessonForm.value = emptyLessonForm()
}

async function saveLesson() {
  if (!selectedChapterId.value || !lessonForm.value.title.trim()) {
    toast.error("请先选择章节并填写课时标题")
    return
  }

  savingLesson.value = true
  try {
    const type = Number(lessonForm.value.lesson_type || 2)
    const body = JSON.stringify({
      chapter_id: selectedChapterId.value,
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
      await apiClient(`/api/lms/chapters/${encodeURIComponent(selectedChapterId.value)}/lessons`, { method: "POST", body })
      toast.success("课时已创建")
    }
    newLesson()
    await loadLessons()
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
    await loadLessons()
  } catch (err) {
    console.error(err)
    toast.error("课时删除失败")
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
        <p class="mt-2 text-slate-600">维护 GLMS 课程内容、发布状态、章节和课时。</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 font-bold shadow-sm" type="button" @click="loadCourses()">
          <RefreshCw class="h-4 w-4" />
          刷新
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 font-bold shadow-sm" type="button" @click="importOpen = true">
          <FileJson class="h-4 w-4" />
          导入 JSON
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-4 py-3 font-bold text-white shadow-lg shadow-sky-200" type="button" @click="newCourse">
          <Plus class="h-4 w-4" />
          新建课程
        </button>
      </div>
    </header>

    <section class="grid gap-6 xl:grid-cols-[420px_1fr]">
      <aside class="rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="border-b border-slate-200 p-5">
          <div class="grid gap-3">
            <input v-model="categoryFilter" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="分类筛选，例如 CFtP/CFtP" />
            <label class="inline-flex items-center gap-2 text-sm font-bold text-slate-600">
              <input v-model="publishedOnly" type="checkbox" />
              仅看已发布
            </label>
          </div>
        </div>

        <div v-if="loading && !courses.length" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载...
        </div>
        <div v-else-if="!courses.length" class="p-12 text-center text-slate-500">暂无课程</div>
        <div v-else class="max-h-[calc(100vh-260px)] divide-y divide-slate-100 overflow-y-auto">
          <button
            v-for="course in courses"
            :key="courseId(course)"
            class="block w-full p-5 text-left transition hover:bg-slate-50"
            :class="courseId(course) === selectedCourseId ? 'bg-sky-50' : ''"
            type="button"
            @click="selectCourse(course)"
          >
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="text-lg font-black">{{ courseTitle(course) }}</div>
                <div class="mt-1 text-sm text-slate-500">{{ course.category_tips || "未分类" }}</div>
              </div>
              <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(course.is_published ? 'COMPLETED' : 'PENDING')">
                {{ course.is_published ? "已发布" : "草稿" }}
              </span>
            </div>
            <div class="mt-3 text-xs text-slate-400">版本 {{ course.version || 0 }} · {{ formatDate(String(course.updated_at || course.created_at || "")) }}</div>
          </button>
        </div>
        <div class="border-t border-slate-200 p-4">
          <button class="w-full rounded-xl border px-4 py-3 font-bold disabled:opacity-40" type="button" :disabled="!nextPageToken || loading" @click="loadCourses(nextPageToken)">
            {{ nextPageToken ? "加载更多" : "没有更多了" }}
          </button>
        </div>
      </aside>

      <main class="space-y-6">
        <section class="rounded-3xl border border-slate-200 bg-white shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
            <div>
              <h2 class="text-xl font-black">{{ selectedCourseId ? "课程详情" : "新建课程" }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ selectedCourseId || "填写课程基础信息后保存。" }}</p>
            </div>
            <span v-if="selectedCourseId" class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(selectedCourseStatus)">
              {{ selectedCoursePublished ? "已发布" : selectedCourseStatus }}
            </span>
          </div>

          <form class="grid gap-4 p-5 lg:grid-cols-2" @submit.prevent="saveCourse">
            <label class="block">
              <span class="text-sm font-bold">课程标题</span>
              <input v-model="courseForm.title" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">分类提示</span>
              <input v-model="courseForm.category_tips" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
            </label>
            <label class="block lg:col-span-2">
              <span class="text-sm font-bold">描述</span>
              <textarea v-model="courseForm.description" class="mt-2 min-h-24 w-full rounded-xl border border-slate-200 p-4" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">Respath</span>
              <input v-model="courseForm.respath" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="/gcc/pipeline/..." />
            </label>
            <label class="block">
              <span class="text-sm font-bold">时长分钟</span>
              <input v-model="courseForm.duration_min" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" type="number" min="0" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">封面 Object Key</span>
              <input v-model="courseForm.thumbnail_object_key" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">封面 File Hash</span>
              <input v-model="courseForm.thumbnail_file_hash" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
            </label>
            <label class="inline-flex items-center gap-2 text-sm font-bold text-slate-600">
              <input v-model="courseForm.certification_enabled" type="checkbox" />
              启用证书
            </label>
            <label class="block">
              <span class="text-sm font-bold">证书定义 ID</span>
              <input v-model="courseForm.certification_def_id" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
            </label>

            <div class="flex flex-wrap gap-3 lg:col-span-2">
              <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="savingCourse" type="submit">
                <Loader2 v-if="savingCourse" class="h-4 w-4 animate-spin" />
                <Save v-else class="h-4 w-4" />
                {{ savingCourse ? "保存中..." : "保存课程" }}
              </button>
              <button class="rounded-xl border px-5 py-3 font-bold disabled:opacity-40" :disabled="!selectedCourseId || publishing" type="button" @click="publishCourse">
                {{ publishing ? "发布中..." : "发布课程" }}
              </button>
              <button class="inline-flex items-center gap-2 rounded-xl border border-red-200 px-5 py-3 font-bold text-red-600 disabled:opacity-40" :disabled="!selectedCourseId" type="button" @click="deleteCourse">
                <Trash2 class="h-4 w-4" />
                删除课程
              </button>
            </div>
          </form>

          <div v-if="selectedCourseId" class="border-t border-slate-200 p-5">
            <div v-if="detailLoading" class="text-sm text-slate-500">正在加载课程统计...</div>
            <div v-else class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <div class="rounded-2xl bg-slate-50 p-4">
                <div class="text-xs font-black uppercase text-slate-400">章节</div>
                <div class="mt-2 text-2xl font-black">{{ courseDetail?.chapter_count ?? chapters.length }}</div>
              </div>
              <div class="rounded-2xl bg-slate-50 p-4">
                <div class="text-xs font-black uppercase text-slate-400">课时</div>
                <div class="mt-2 text-2xl font-black">{{ courseDetail?.lesson_count ?? lessons.length }}</div>
              </div>
              <div class="rounded-2xl bg-slate-50 p-4">
                <div class="text-xs font-black uppercase text-slate-400">测验</div>
                <div class="mt-2 text-2xl font-black">{{ courseDetail?.quiz_count ?? 0 }}</div>
              </div>
              <div class="rounded-2xl bg-slate-50 p-4">
                <div class="text-xs font-black uppercase text-slate-400">资料</div>
                <div class="mt-2 text-2xl font-black">{{ courseDetail?.material_count ?? 0 }}</div>
              </div>
            </div>
          </div>
        </section>

        <section class="grid gap-6 xl:grid-cols-[1fr_1fr]" :class="!selectedCourseId ? 'opacity-50' : ''">
          <div class="rounded-3xl border border-slate-200 bg-white shadow-sm">
            <div class="flex items-center justify-between border-b border-slate-200 p-5">
              <div>
                <h2 class="text-xl font-black">章节</h2>
                <p class="mt-1 text-sm text-slate-500">选择章节后可维护课时。</p>
              </div>
              <button class="rounded-xl border px-3 py-2 font-bold" :disabled="!selectedCourseId" type="button" @click="newChapter">新章节</button>
            </div>
            <div v-if="chaptersLoading" class="p-8 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              正在加载...
            </div>
            <div v-else-if="!chapters.length" class="p-8 text-center text-slate-500">暂无章节</div>
            <div v-else class="divide-y divide-slate-100">
              <div v-for="chapter in chapters" :key="chapterId(chapter)" class="flex items-center justify-between gap-3 p-4" :class="chapterId(chapter) === selectedChapterId ? 'bg-sky-50' : ''">
                <button class="flex-1 text-left" type="button" @click="editChapter(chapter)">
                  <div class="font-black">{{ chapterTitle(chapter) }}</div>
                  <div class="mt-1 text-sm text-slate-500">排序 {{ chapter.sort_order || 0 }}</div>
                </button>
                <button class="rounded-xl border border-red-200 px-3 py-2 text-sm font-bold text-red-600" type="button" @click="deleteChapter(chapter)">删除</button>
              </div>
            </div>
            <form class="border-t border-slate-200 p-5" @submit.prevent="saveChapter">
              <h3 class="font-black">{{ editingChapterId ? "编辑章节" : "创建章节" }}</h3>
              <input v-model="chapterForm.title" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="章节标题" />
              <input v-model="chapterForm.sort_order" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="排序" type="number" />
              <button class="mt-3 rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedCourseId || savingChapter" type="submit">
                {{ savingChapter ? "保存中..." : "保存章节" }}
              </button>
            </form>
          </div>

          <div class="rounded-3xl border border-slate-200 bg-white shadow-sm">
            <div class="flex items-center justify-between border-b border-slate-200 p-5">
              <div>
                <h2 class="text-xl font-black">课时</h2>
                <p class="mt-1 text-sm text-slate-500">{{ selectedChapterId ? chapterTitle(selectedChapter) : "请先选择章节。" }}</p>
              </div>
              <button class="rounded-xl border px-3 py-2 font-bold" :disabled="!selectedChapterId" type="button" @click="newLesson">新课时</button>
            </div>
            <div v-if="lessonsLoading" class="p-8 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              正在加载...
            </div>
            <div v-else-if="!lessons.length" class="p-8 text-center text-slate-500">暂无课时</div>
            <div v-else class="divide-y divide-slate-100">
              <div v-for="lesson in lessons" :key="lessonId(lesson)" class="flex items-center justify-between gap-3 p-4">
                <button class="flex-1 text-left" type="button" @click="editLesson(lesson)">
                  <div class="font-black">{{ lessonTitle(lesson) }}</div>
                  <div class="mt-1 text-sm text-slate-500">排序 {{ lesson.sort_order || 0 }} · 类型 {{ lesson.lesson_type || "-" }}</div>
                </button>
                <button class="rounded-xl border border-red-200 px-3 py-2 text-sm font-bold text-red-600" type="button" @click="deleteLesson(lesson)">删除</button>
              </div>
            </div>
            <form class="border-t border-slate-200 p-5" @submit.prevent="saveLesson">
              <h3 class="font-black">{{ editingLessonId ? "编辑课时" : "创建课时" }}</h3>
              <input v-model="lessonForm.title" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="课时标题" />
              <div class="mt-3 grid gap-3 sm:grid-cols-2">
                <input v-model="lessonForm.sort_order" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="排序" type="number" />
                <select v-model="lessonForm.lesson_type" class="rounded-xl border border-slate-200 px-4 py-3">
                  <option value="1">文本</option>
                  <option value="2">PDF</option>
                  <option value="3">视频</option>
                  <option value="4">图片</option>
                  <option value="5">音频</option>
                  <option value="6">链接</option>
                  <option value="7">文件</option>
                </select>
              </div>
              <textarea v-model="lessonForm.body" class="mt-3 min-h-24 w-full rounded-xl border border-slate-200 p-4" placeholder="正文 / 链接说明" />
              <input v-model="lessonForm.asset_object_key" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="资产 Object Key" />
              <input v-model="lessonForm.asset_file_hash" class="mt-3 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="资产 File Hash" />
              <button class="mt-3 rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="!selectedChapterId || savingLesson" type="submit">
                {{ savingLesson ? "保存中..." : "保存课时" }}
              </button>
            </form>
          </div>
        </section>
      </main>
    </section>

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
