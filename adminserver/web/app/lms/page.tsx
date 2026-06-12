"use client"

import React, { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { AlertTriangle, ArrowLeft, BookOpen, CheckCircle2, ClipboardList, Eye, FileJson, FileText, Plus, RefreshCw, Save, Trash2, UploadCloud, Users } from "lucide-react"
import { Tooltip, TooltipTrigger, TooltipContent } from "@/components/ui/tooltip"
import { toast } from "sonner"

import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { cn, formatBackendDate } from "@/lib/utils"
import {
  LMS_ASSET_STATUS_LABELS,
  LMS_CHAPTER_PROGRESS_STATUS_LABELS,
  LMS_COURSE_STATUS_LABELS,
  LMS_ENROLLMENT_STATUS_LABELS,
  LMS_LESSON_PROGRESS_STATUS_LABELS,
  LMS_QUIZ_ATTEMPT_STATUS_LABELS,
  normalizeEnumValue,
  statusBadgeClassForStatusValue,
  statusLabel,
} from "@cftpweb/shared"

type LmsCourse = {
  course_id: string
  course_guid?: string
  category_tips?: string
  title?: string
  description?: string
  thumbnail_object_key?: string
  thumbnail_file_hash?: string
  duration_min?: number
  certification_enabled?: boolean
  certification_def_id?: string
  is_published?: boolean
  published_at?: string
  version?: number
  status?: string
  is_current?: boolean
  created_at?: string
  updated_at?: string
}

type CourseForm = {
  category_tips: string
  title: string
  description: string
  thumbnail_object_key: string
  thumbnail_file_hash: string
  duration_min: string
  certification_enabled: boolean
  certification_def_id: string
}

type CatalogOption = {
  catalog_id: string
  name?: string
  description?: string
}

type CredentialDefinitionOption = {
  cred_def_id: string
  name?: string
  category?: string
}

type CourseEnrollment = {
  enrollment_id: string
  candidate_id: string
  course_id?: string
  biz_unit?: string
  status?: string
  progress_percentage?: number
  total_lessons?: number
  total_quizzes?: number
  completed_lessons_count?: number
  passed_quizzes_count?: number
  joined_at?: string
  completed_at?: string
  version?: number
  created_at?: string
  updated_at?: string
}

type CandidateProgress = {
  enrollment_id: string
  candidate_id: string
  course_id: string
  status?: string
  progress_percentage?: number
  completed_lesson_ids?: string[]
  passed_quiz_ids?: string[]
  joined_at?: string
  completed_at?: string
}

type BrokenAsset = {
  object_key: string
  asset_type?: string
  associated_id?: string
  file_hash?: string
  status?: string
  error_message?: string
  reconciled_at?: string
  created_at?: string
  updated_at?: string
  course_id?: string
  course_title?: string
  chapter_id?: string
  chapter_title?: string
  lesson_id?: string
  lesson_title?: string
  material_id?: string
  material_title?: string
}

type CourseDetailCounts = {
  course?: LmsCourse
  chapter_count?: number
  lesson_count?: number
  quiz_count?: number
  material_count?: number
  asset_count?: number
  enrollment_count?: number
}

type CourseSupplementaryMaterial = {
  material_id: string
  course_id?: string
  kind?: string
  data_json?: string
  version?: number
  created_at?: string
  updated_at?: string
}

type SupplementaryMaterialForm = {
  material_id: string
  kind: string
  data_json: string
}

type CourseMaterial = {
  material_id: string
  course_id?: string
  title?: string
  material_type?: number
  file_object_key?: string
  file_size?: number
  sort_order?: number
  file_hash?: string
  version?: number
  created_at?: string
  updated_at?: string
}

type LessonProgress = {
  user_id?: string
  candidate_id?: string
  lesson_id?: string
  lesson_title?: string
  status?: string
  started_at?: string
  completed_at?: string
  created_at?: string
  updated_at?: string
}

type ChapterProgress = {
  candidate_id?: string
  chapter_id?: string
  chapter_title?: string
  course_id?: string
  total_lessons?: number
  total_quizzes?: number
  completed_lessons_count?: number
  passed_quizzes_count?: number
  status?: string
  created_at?: string
  updated_at?: string
}

type QuizAttempt = {
  attempt_id?: string
  quiz_id?: string
  quiz_title?: string
  user_id?: string
  status?: string
  score?: number
  max_score?: number
  is_passed?: boolean
  started_at?: string
  completed_at?: string
}

type Chapter = {
  chapter_id: string
  course_id: string
  title?: string
  sort_order?: number
  version?: number
  created_at?: string
  updated_at?: string
}

type Lesson = {
  lesson_id: string
  chapter_id: string
  title?: string
  sort_order?: number
  lesson_type?: number
  body?: string
  version?: number
  created_at?: string
  updated_at?: string
}

type Quiz = {
  quiz_id: string
  quizzable_type?: number
  quizzable_id?: string
  title?: string
  description?: string
  passing_score?: number
  time_limit?: number
  max_attempts?: number
  allow_retake?: boolean
  randomize_questions?: boolean
  is_active?: boolean
  version?: number
  created_at?: string
  updated_at?: string
}

type QuizQuestion = {
  question_id: string
  quiz_id: string
  question_text?: string
  question_type?: number
  points?: number
  sort_order?: number
  is_required?: boolean
  version?: number
  created_at?: string
  updated_at?: string
}

type QuizOption = {
  option_id: string
  question_id: string
  option_text?: string
  is_correct?: boolean
  sort_order?: number
  version?: number
  created_at?: string
  updated_at?: string
}

type MaterialForm = {
  material_id: string
  title: string
  material_type: string
  sort_order: string
  file_object_key: string
  file_hash: string
  file_size: string
}

const MATERIAL_TYPE_ACCEPTS: Record<string, string> = {
  "1":
    ".pdf,.doc,.docx,.txt,.epub,application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document,text/plain,application/epub+zip",
  "2":
    ".pdf,.ppt,.pptx,application/pdf,application/vnd.ms-powerpoint,application/vnd.openxmlformats-officedocument.presentationml.presentation",
  "3":
    ".pdf,.doc,.docx,.txt,.xls,.xlsx,.csv,.zip,.rar,.7z,application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document,text/plain,application/vnd.ms-excel,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/zip,application/x-7z-compressed,application/vnd.rar",
  "4":
    ".pdf,.ppt,.pptx,.doc,.docx,.txt,.epub,.png,.jpg,.jpeg,.gif,.mp3,.wav,.flac,.mp4,.webm,.mov,.m4v,.avi,.mkv,.zip,.rar,.7z,application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document,text/plain,application/epub+zip,image/*,audio/*,video/*,application/zip,application/x-7z-compressed,application/vnd.rar",
}

function getMaterialTypeAccept(materialType: string) {
  return MATERIAL_TYPE_ACCEPTS[materialType] || MATERIAL_TYPE_ACCEPTS["4"]
}

function getMaterialTypeName(page: any, materialType: string) {
  switch (materialType) {
    case "1":
      return page.materialTypeTextbook || "教材"
    case "2":
      return page.materialTypeSlides || "课件"
    case "3":
      return page.materialTypeReference || "参考资料"
    default:
      return page.materialTypeOther || "其他"
  }
}

function materialTypeKey(materialType?: number) {
  return String(materialType || 4)
}

function normalizeMaterialFileName(name: string) {
  return name.trim().toLowerCase()
}

function isPdfFile(file: File) {
  const fileName = normalizeMaterialFileName(file.name)
  return file.type === "application/pdf" || fileName.endsWith(".pdf")
}

function isPresentationFile(file: File) {
  const fileName = normalizeMaterialFileName(file.name)
  return (
    file.type === "application/pdf" ||
    file.type === "application/vnd.ms-powerpoint" ||
    file.type === "application/vnd.openxmlformats-officedocument.presentationml.presentation" ||
    fileName.endsWith(".pdf") ||
    fileName.endsWith(".ppt") ||
    fileName.endsWith(".pptx")
  )
}

function isReferenceFile(file: File) {
  const fileName = normalizeMaterialFileName(file.name)
  return (
    isPdfFile(file) ||
    file.type === "application/msword" ||
    file.type === "application/vnd.openxmlformats-officedocument.wordprocessingml.document" ||
    file.type === "text/plain" ||
    file.type === "application/vnd.ms-excel" ||
    file.type === "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" ||
    file.type === "application/zip" ||
    file.type === "application/x-7z-compressed" ||
    file.type === "application/vnd.rar" ||
    fileName.endsWith(".doc") ||
    fileName.endsWith(".docx") ||
    fileName.endsWith(".txt") ||
    fileName.endsWith(".xls") ||
    fileName.endsWith(".xlsx") ||
    fileName.endsWith(".csv") ||
    fileName.endsWith(".zip") ||
    fileName.endsWith(".rar") ||
    fileName.endsWith(".7z")
  )
}

function isOpenMaterialFile(file: File) {
  const fileName = normalizeMaterialFileName(file.name)
  return (
    isReferenceFile(file) ||
    file.type.startsWith("image/") ||
    file.type.startsWith("audio/") ||
    file.type.startsWith("video/") ||
    fileName.endsWith(".png") ||
    fileName.endsWith(".jpg") ||
    fileName.endsWith(".jpeg") ||
    fileName.endsWith(".gif") ||
    fileName.endsWith(".webp") ||
    fileName.endsWith(".mp3") ||
    fileName.endsWith(".wav") ||
    fileName.endsWith(".flac") ||
    fileName.endsWith(".mp4") ||
    fileName.endsWith(".webm") ||
    fileName.endsWith(".mov") ||
    fileName.endsWith(".m4v") ||
    fileName.endsWith(".avi") ||
    fileName.endsWith(".mkv")
  )
}

const emptyForm: CourseForm = {
  category_tips: "",
  title: "",
  description: "",
  thumbnail_object_key: "",
  thumbnail_file_hash: "",
  duration_min: "",
  certification_enabled: false,
  certification_def_id: "",
}

const emptyChapterForm = {
  title: "",
  sort_order: "",
}

const emptyLessonForm = {
  title: "",
  body: "",
  sort_order: "",
}

const emptyQuizForm = {
  title: "",
  description: "",
  passing_score: "60",
  time_limit: "",
  max_attempts: "1",
  allow_retake: true,
  randomize_questions: false,
  is_active: true,
}

const emptyQuestionForm = {
  question_text: "",
  question_type: "1",
  points: "1",
  sort_order: "",
  is_required: true,
}

const emptyOptionForm = {
  option_text: "",
  sort_order: "",
  is_correct: false,
}

const emptyMaterialForm: MaterialForm = {
  material_id: "",
  title: "",
  material_type: "1",
  sort_order: "",
  file_object_key: "",
  file_hash: "",
  file_size: "",
}

const courseListPageSize = 20

async function sha256Hex(file: File) {
  const buffer = await file.arrayBuffer()
  const hash = await crypto.subtle.digest("SHA-256", buffer)
  return Array.from(new Uint8Array(hash)).map((byte) => byte.toString(16).padStart(2, "0")).join("")
}

function newMaterialDraftId() {
  const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
  const time = Date.now()
  const timeChars = Array.from({ length: 10 }, (_, index) => {
    const shift = (9 - index) * 5
    return alphabet[Math.floor(time / 2 ** shift) % 32]
  }).join("")
  const randomBytes = new Uint8Array(16)
  crypto.getRandomValues(randomBytes)
  const randomChars = Array.from(randomBytes, (byte) => alphabet[byte % 32]).join("")
  return `${timeChars}${randomChars}`.slice(0, 26)
}

function uploadHeaders(signedHeaders?: Record<string, string>) {
  return new Headers(signedHeaders || {})
}

function formFromCourse(course: LmsCourse | null): CourseForm {
  if (!course) return emptyForm
  return {
    category_tips: course.category_tips || "",
    title: course.title || "",
    description: course.description || "",
    thumbnail_object_key: course.thumbnail_object_key || "",
    thumbnail_file_hash: course.thumbnail_file_hash || "",
    duration_min: course.duration_min ? String(course.duration_min) : "",
    certification_enabled: Boolean(course.certification_enabled),
    certification_def_id: course.certification_def_id || "",
  }
}

function formToPayload(form: CourseForm, version?: number) {
  return {
    category_tips: form.category_tips.trim(),
    title: form.title.trim(),
    description: form.description.trim(),
    thumbnail_object_key: form.thumbnail_object_key.trim(),
    thumbnail_file_hash: form.thumbnail_file_hash.trim(),
    duration_min: Number(form.duration_min || 0),
    certification_enabled: form.certification_enabled,
    certification_def_id: form.certification_enabled ? form.certification_def_id.trim() : "",
    version: version || 0,
  }
}

function ProtectedButton({ disabled, isPublished, tooltipText, children, className, ...props }: any) {
  if (!disabled) {
    return <Button disabled={false} className={className} {...props}>{children}</Button>
  }
  let hint = tooltipText || "前置条件不足或正在处理"
  if (isPublished) hint = "已发布的课程无法进行修改或删除。若需操作，请基于此版本新建草稿。"
  
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span tabIndex={0} className="inline-block cursor-not-allowed">
          <Button disabled={true} className={cn("pointer-events-none", className)} {...props}>
            {children}
          </Button>
        </span>
      </TooltipTrigger>
      <TooltipContent side="top">
        <p>{hint}</p>
      </TooltipContent>
    </Tooltip>
  )
}

export default function LmsCoursesPage() {
  const { t } = useTranslation()
  const page = t.lmsCoursesPage
  const [courses, setCourses] = useState<LmsCourse[]>([])
  const [catalogs, setCatalogs] = useState<CatalogOption[]>([])
  const [credentialDefinitions, setCredentialDefinitions] = useState<CredentialDefinitionOption[]>([])
  const [selectedId, setSelectedId] = useState("")
  const selectedIdRef = useRef("")
  const [form, setForm] = useState<CourseForm>(emptyForm)
  const [categoryFilter, setCategoryFilter] = useState("")
  const [publishedOnly, setPublishedOnly] = useState(false)
  const [loading, setLoading] = useState(true)
  const [courseListLoadingMore, setCourseListLoadingMore] = useState(false)
  const [courseListNextPageToken, setCourseListNextPageToken] = useState("")
  const [saving, setSaving] = useState(false)
  const [thumbnailUploading, setThumbnailUploading] = useState(false)
  const [preview, setPreview] = useState<any>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [enrollmentStatus, setEnrollmentStatus] = useState("all")
  const [batchEnrollCandidateId, setBatchEnrollCandidateId] = useState("")
  const [batchEnrolling, setBatchEnrolling] = useState(false)
  const [enrollments, setEnrollments] = useState<CourseEnrollment[]>([])
  const [enrollmentsLoading, setEnrollmentsLoading] = useState(false)
  const [progressDetail, setProgressDetail] = useState<CandidateProgress | null>(null)
  const [progressLoadingFor, setProgressLoadingFor] = useState("")
  const [syncProgressLoadingFor, setSyncProgressLoadingFor] = useState("")
  const [assetStatus, setAssetStatus] = useState("all")
  const [brokenAssetType, setBrokenAssetType] = useState("all")
  const [brokenAssets, setBrokenAssets] = useState<BrokenAsset[]>([])
  const [brokenAssetsLoading, setBrokenAssetsLoading] = useState(false)
  const [brokenAssetsNextPageToken, setBrokenAssetsNextPageToken] = useState("")
  const [courseDetail, setCourseDetail] = useState<CourseDetailCounts | null>(null)
  const [courseDetailLoading, setCourseDetailLoading] = useState(false)
  const [enrollmentDetail, setEnrollmentDetail] = useState<CourseEnrollment | null>(null)
  const [enrollmentDetailLoadingFor, setEnrollmentDetailLoadingFor] = useState("")
  const [lessonProgress, setLessonProgress] = useState<LessonProgress[]>([])
  const [lessonProgressDetail, setLessonProgressDetail] = useState<any>(null)
  const [lessonProgressLoadingFor, setLessonProgressLoadingFor] = useState("")
  const [chapterProgress, setChapterProgress] = useState<ChapterProgress | null>(null)
  const [chapterProgressLoadingFor, setChapterProgressLoadingFor] = useState("")
  const [quizAttempts, setQuizAttempts] = useState<QuizAttempt[]>([])
  const [quizAttemptDetail, setQuizAttemptDetail] = useState<any>(null)
  const [quizAttemptsLoadingFor, setQuizAttemptsLoadingFor] = useState("")
  const [assetDetail, setAssetDetail] = useState<BrokenAsset | null>(null)
  const [assetDetailLoadingFor, setAssetDetailLoadingFor] = useState("")
  const [chapters, setChapters] = useState<Chapter[]>([])
  const [selectedChapterId, setSelectedChapterId] = useState("")
  const [chapterForm, setChapterForm] = useState(emptyChapterForm)
  const [chapterSaving, setChapterSaving] = useState(false)
  const [chaptersLoading, setChaptersLoading] = useState(false)
  const [lessons, setLessons] = useState<Lesson[]>([])
  const [lessonForm, setLessonForm] = useState(emptyLessonForm)
  const [lessonSaving, setLessonSaving] = useState(false)
  const [lessonsLoading, setLessonsLoading] = useState(false)
  const [quizzes, setQuizzes] = useState<Quiz[]>([])
  const [selectedQuizId, setSelectedQuizId] = useState("")
  const [quizForm, setQuizForm] = useState(emptyQuizForm)
  const [quizSaving, setQuizSaving] = useState(false)
  const [quizzesLoading, setQuizzesLoading] = useState(false)
  const [questions, setQuestions] = useState<QuizQuestion[]>([])
  const [selectedQuestionId, setSelectedQuestionId] = useState("")
  const [questionForm, setQuestionForm] = useState(emptyQuestionForm)
  const [questionSaving, setQuestionSaving] = useState(false)
  const [questionsLoading, setQuestionsLoading] = useState(false)
  const [options, setOptions] = useState<QuizOption[]>([])
  const [optionForm, setOptionForm] = useState(emptyOptionForm)
  const [optionSaving, setOptionSaving] = useState(false)
  const [optionsLoading, setOptionsLoading] = useState(false)
  const [materials, setMaterials] = useState<CourseMaterial[]>([])
  const [supplementaryMaterial, setSupplementaryMaterial] = useState<CourseSupplementaryMaterial | null>(null)
  const [supplementaryMaterialLoading, setSupplementaryMaterialLoading] = useState(false)
  const [supplementaryMaterialForm, setSupplementaryMaterialForm] = useState<SupplementaryMaterialForm>({ material_id: "", kind: "", data_json: "" })
  const [supplementaryMaterialSaving, setSupplementaryMaterialSaving] = useState(false)
  const [materialsLoading, setMaterialsLoading] = useState(false)
  const [materialForm, setMaterialForm] = useState<MaterialForm>(emptyMaterialForm)
  const [materialSaving, setMaterialSaving] = useState(false)
  const [materialUploading, setMaterialUploading] = useState(false)
  const [selectedMaterialId, setSelectedMaterialId] = useState("")
  const [importOpen, setImportOpen] = useState(false)
  const [importScope, setImportScope] = useState<"course" | "quiz">("course")
  const [importCategoryTips, setImportCategoryTips] = useState("")
  const [importJson, setImportJson] = useState("")
  const [importing, setImporting] = useState(false)
  const [chapterModalOpen, setChapterModalOpen] = useState(false)
  const [materialModalOpen, setMaterialModalOpen] = useState(false)
  const [suppMaterialModalOpen, setSuppMaterialModalOpen] = useState(false)

  const selectedCourse = useMemo(
    () => courses.find((course) => course.course_id === selectedId) || null,
    [courses, selectedId]
  )

  useEffect(() => {
    selectedIdRef.current = selectedId
  }, [selectedId])
  const selectedCoursePublished = Boolean(
    selectedCourse?.is_published || selectedCourse?.status?.toLowerCase() === "active"
  )

  const selectedChapter = useMemo(
    () => chapters.find((chapter) => chapter.chapter_id === selectedChapterId) || null,
    [chapters, selectedChapterId]
  )

  const selectedQuiz = useMemo(
    () => quizzes.find((quiz) => quiz.quiz_id === selectedQuizId) || null,
    [quizzes, selectedQuizId]
  )

  const selectedQuestion = useMemo(
    () => questions.find((question) => question.question_id === selectedQuestionId) || null,
    [questions, selectedQuestionId]
  )

  const selectedMaterial = useMemo(
    () => materials.find((material) => material.material_id === selectedMaterialId) || null,
    [materials, selectedMaterialId]
  )

  const materialTypeLabel = useCallback(
    (materialType?: number) => getMaterialTypeName(page, materialTypeKey(materialType)),
    [page]
  )

  useEffect(() => {
    if (!selectedCourse?.course_id) return
    void loadMaterials(selectedCourse.course_id)
    void loadSupplementaryMaterial(selectedCourse.course_id)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedCourse?.course_id])

  const resetCourseInspectionState = () => {
    setPreview(null)
    setEnrollments([])
    setProgressDetail(null)
    setCourseDetail(null)
    setEnrollmentDetail(null)
    setLessonProgress([])
    setLessonProgressDetail(null)
    setChapterProgress(null)
    setQuizAttempts([])
    setQuizAttemptDetail(null)
    setAssetDetail(null)
  }

  const resetCourseContentState = () => {
    setChapters([])
    setSelectedChapterId("")
    setChapterForm(emptyChapterForm)
    setLessons([])
    setLessonForm(emptyLessonForm)
    setQuizzes([])
    setSelectedQuizId("")
    setQuizForm(emptyQuizForm)
    setQuestions([])
    setSelectedQuestionId("")
    setQuestionForm(emptyQuestionForm)
    setOptions([])
    setOptionForm(emptyOptionForm)
    setMaterials([])
    setSupplementaryMaterial(null)
    setSupplementaryMaterialForm({ material_id: "", kind: "", data_json: "" })
    setSelectedMaterialId("")
    setMaterialForm(emptyMaterialForm)
  }

  const loadCourses = useCallback(async (pageToken = "") => {
    if (pageToken) {
      setCourseListLoadingMore(true)
    } else {
      setLoading(true)
    }
    try {
      const params = new URLSearchParams()
      if (categoryFilter.trim()) params.set("category_tips", categoryFilter.trim())
      if (publishedOnly) params.set("published_only", "true")
      params.set("page_size", String(courseListPageSize))
      if (pageToken) params.set("page_token", pageToken)
      const query = params.toString()
      const res = await apiClient(`/api/lms/courses${query ? `?${query}` : ""}`)
      const nextCourses = res?.courses || []
      setCourseListNextPageToken(res?.next_page_token || "")
      setCourses((prevCourses) => {
        const mergedCourses = pageToken ? [...prevCourses, ...nextCourses] : nextCourses
        const currentSelectedId = selectedIdRef.current
        if (!pageToken && currentSelectedId && !mergedCourses.some((course: LmsCourse) => course.course_id === currentSelectedId)) {
          setSelectedId("")
          setForm(emptyForm)
          resetCourseInspectionState()
          resetCourseContentState()
        }
        return mergedCourses
      })
    } finally {
      setLoading(false)
      setCourseListLoadingMore(false)
    }
  }, [categoryFilter, publishedOnly])

  useEffect(() => {
    loadCourses()
  }, [loadCourses])

  useEffect(() => {
    const loadRelations = async () => {
      const [catalogRes, credentialRes] = await Promise.all([
        apiClient("/api/catalogs"),
        apiClient("/api/credentials/definitions"),
      ])
      setCatalogs(catalogRes?.catalogs || [])
      setCredentialDefinitions(credentialRes?.definitions || [])
    }

    loadRelations().catch((error) => {
      console.error(error)
      toast.error(page.loadRelationsFailed)
    })
  }, [page.loadRelationsFailed])

  const selectCourse = (course: LmsCourse) => {
    setSelectedId(course.course_id)
    setForm(formFromCourse(course))
    resetCourseInspectionState()
    resetCourseContentState()
  }

  const startNewCourse = () => {
    setSelectedId("")
    setForm(emptyForm)
    resetCourseInspectionState()
    resetCourseContentState()
  }

  const clonePublishedCourseDraft = async (course: LmsCourse) => {
    const res = await apiClient("/api/lms/courses", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        ...formToPayload(form),
        from_course_guid: course.course_guid || "",
      }),
    })

    const draftCourse = res?.course_id ? res : null
    if (!draftCourse?.course_id) {
      throw new Error("create draft failed")
    }

    setSelectedId(draftCourse.course_id)
    resetCourseInspectionState()
    resetCourseContentState()
    return draftCourse as LmsCourse
  }

  const switchImportScope = (scope: "course" | "quiz") => {
    setImportScope(scope)
    setImportJson("")
  }

  const importLmsJson = async () => {
    if (!importJson.trim()) {
      toast.error(page.importInvalidJson)
      return
    }
    try {
      JSON.parse(importJson)
    } catch {
      toast.error(page.importInvalidJson)
      return
    }
    if (importScope === "quiz" && !selectedChapter) {
      toast.error(page.importSelectChapter)
      return
    }

    setImporting(true)
    try {
      const res = await apiClient("/api/lms/import", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(
          importScope === "course"
            ? {
                scope: "course",
                category_tips: importCategoryTips.trim(),
                course_json: importJson,
              }
            : {
                scope: "quiz",
                quizzable_type: 2,
                quizzable_id: selectedChapter?.chapter_id || "",
                quiz_json: importJson,
              }
        ),
      })
      if (importScope === "course") {
        toast.success(
          page.importCourseSuccess
            .replace("{{chapters}}", String(res?.chapter_count || 0))
            .replace("{{lessons}}", String(res?.lesson_count || 0))
            .replace("{{materials}}", String(res?.material_count || 0))
        )
      } else {
        toast.success(
          page.importQuizSuccess
            .replace("{{questions}}", String(res?.question_count || 0))
            .replace("{{options}}", String(res?.option_count || 0))
        )
      }
      setImportOpen(false)
      if (res?.course_id) {
        setSelectedId(res.course_id)
      }
      await loadCourses()
      if (importScope === "quiz" && selectedChapter) {
        await loadQuizzes(selectedChapter.chapter_id)
      }
    } finally {
      setImporting(false)
    }
  }

  const loadImportFile = async (file: File | null) => {
    if (!file) return
    try {
      setImportJson(await file.text())
    } catch {
      toast.error(page.importReadFileFailed)
    }
  }

  const saveCourse = async () => {
    if (!form.title.trim()) {
      toast.error(page.fillRequired)
      return
    }
    if (form.certification_enabled && !form.certification_def_id.trim()) {
      toast.error(page.selectCertificationDef)
      return
    }

    setSaving(true)
    try {
      let targetCourse = selectedCourse
      if (selectedCourse) {
        targetCourse = selectedCoursePublished ? await clonePublishedCourseDraft(selectedCourse) : selectedCourse
        await apiClient(`/api/lms/courses/${targetCourse.course_id}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(formToPayload(form, targetCourse.version)),
        })
        toast.success(page.updateSuccess)
      } else {
        const res = await apiClient("/api/lms/courses", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(formToPayload(form)),
        })
        setSelectedId(res?.course_id || "")
        toast.success(page.createSuccess)
      }
      await loadCourses()
      if (selectedCoursePublished && targetCourse?.course_id) {
        await loadChapters(targetCourse.course_id)
      }
    } finally {
      setSaving(false)
    }
  }

  const loadSupplementaryMaterial = async (courseId = selectedCourse?.course_id) => {
    if (!courseId) return
    setSupplementaryMaterialLoading(true)
    try {
      const res = await apiClient(`/api/lms/courses/${courseId}/supplementary-material`)
      setSupplementaryMaterial(res?.material || null)
      if (res?.material) {
        setSupplementaryMaterialForm({
          material_id: res.material.material_id || "",
          kind: res.material.kind ? String(res.material.kind) : "",
          data_json: res.material.data_json || "",
        })
      } else {
        setSupplementaryMaterialForm({ material_id: "", kind: "", data_json: "" })
      }
    } catch {
      setSupplementaryMaterial(null)
      setSupplementaryMaterialForm({ material_id: "", kind: "", data_json: "" })
    } finally {
      setSupplementaryMaterialLoading(false)
    }
  }

  const saveSupplementaryMaterial = async () => {
    if (!selectedCourse?.course_id) return
    setSupplementaryMaterialSaving(true)
    try {
      const isEditing = !!supplementaryMaterial?.material_id
      const payload = {
        material_id: supplementaryMaterialForm.material_id.trim() || newMaterialDraftId(),
        course_id: selectedCourse.course_id,
        kind: Number(supplementaryMaterialForm.kind || 0),
        data_json: supplementaryMaterialForm.data_json.trim(),
        version: supplementaryMaterial?.version || 0,
      }
      
      const method = isEditing ? "PUT" : "POST"
      const url = isEditing 
        ? `/api/lms/courses/${selectedCourse.course_id}/supplementary-material/${supplementaryMaterial.material_id}`
        : `/api/lms/courses/${selectedCourse.course_id}/supplementary-material`
      
      const res = await apiClient(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      })
      toast.success((page as any).saveSuccess || "保存成功" || "保存成功")
      await loadSupplementaryMaterial(selectedCourse.course_id)
    } finally {
      setSupplementaryMaterialSaving(false)
    }
  }

  const deleteSupplementaryMaterial = async () => {
    if (!selectedCourse?.course_id || !supplementaryMaterial?.material_id) return
    if (!confirm(page.confirmDelete || "确认删除?")) return
    setSupplementaryMaterialSaving(true)
    try {
      await apiClient(`/api/lms/courses/${selectedCourse.course_id}/supplementary-material/${supplementaryMaterial.material_id}?version=${supplementaryMaterial.version || 0}`, {
        method: "DELETE",
      })
      toast.success(page.deleteSuccess || "删除成功")
      await loadSupplementaryMaterial(selectedCourse.course_id)
    } finally {
      setSupplementaryMaterialSaving(false)
    }
  }

  const loadMaterials = async (courseId = selectedCourse?.course_id, preferredMaterialId = selectedMaterialId) => {
    if (!courseId) return
    setMaterialsLoading(true)
    try {
      const res = await apiClient(`/api/lms/courses/${courseId}/materials`)
      const nextMaterials = res?.materials || []
      setMaterials(nextMaterials)
      const matchedMaterial = nextMaterials.find((material: CourseMaterial) => material.material_id === preferredMaterialId)
      if (matchedMaterial) {
        selectMaterial(matchedMaterial)
      } else if (preferredMaterialId && nextMaterials.length > 0) {
        selectMaterial(nextMaterials[0])
      } else if (!preferredMaterialId && nextMaterials.length > 0) {
        selectMaterial(nextMaterials[0])
      } else if (preferredMaterialId) {
        resetMaterialForm()
      }
    } finally {
      setMaterialsLoading(false)
    }
  }

  const selectMaterial = (material: CourseMaterial) => {
    setSelectedMaterialId(material.material_id)
    setMaterialForm({
      material_id: material.material_id,
      title: material.title || "",
      material_type: String(material.material_type || 1),
      sort_order: String(material.sort_order || ""),
      file_object_key: material.file_object_key || "",
      file_hash: material.file_hash || "",
      file_size: material.file_size ? String(material.file_size) : "",
    })
  }

  const loadMaterialDetail = async (materialId: string) => {
    const res = await apiClient(`/api/lms/materials/${materialId}`)
    return (res?.material || null) as CourseMaterial | null
  }

  const loadMaterialForEdit = async (material: CourseMaterial) => {
    try {
      const detail = await loadMaterialDetail(material.material_id)
      selectMaterial(detail || material)
    } catch (error) {
      console.error(error)
      selectMaterial(material)
    }
  }

  const resetMaterialForm = () => {
    setMaterialForm(emptyMaterialForm)
    setSelectedMaterialId("")
  }

  const materialUploadAccept = useMemo(() => getMaterialTypeAccept(materialForm.material_type), [materialForm.material_type])

  const validateMaterialFile = (file: File) => {
    const materialType = materialForm.material_type || "4"
    const ok =
      materialType === "1"
        ? isPdfFile(file) || file.type === "application/msword" || file.type === "application/vnd.openxmlformats-officedocument.wordprocessingml.document" || file.type === "text/plain" || normalizeMaterialFileName(file.name).endsWith(".doc") || normalizeMaterialFileName(file.name).endsWith(".docx") || normalizeMaterialFileName(file.name).endsWith(".txt") || normalizeMaterialFileName(file.name).endsWith(".epub")
        : materialType === "2"
          ? isPresentationFile(file)
          : materialType === "3"
            ? isReferenceFile(file)
            : isOpenMaterialFile(file)

    if (!ok) {
      toast.error(page.materialFileTypeMismatch.replace("{{type}}", getMaterialTypeName(page, materialType)))
      return false
    }

    return true
  }

  const uploadMaterialFile = async (file: File | null) => {
    if (!file) return
    if (!selectedCourse) {
      toast.error(page.createBeforeMaterial)
      return
    }
    if (!validateMaterialFile(file)) {
      return
    }

    setMaterialUploading(true)
    try {
      const materialId = materialForm.material_id || newMaterialDraftId()
      const contentType = file.type || "application/octet-stream"
      const fileHash = await sha256Hex(file)
      const upload = await apiClient("/api/lms/upload-url", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          upload_type: 2,
          file_name: file.name,
          content_type: contentType,
          course_id: selectedCourse.course_id,
          material_id: materialId,
          file_hash: fileHash,
        }),
      })

      const uploadRes = await fetch(upload.upload_url, {
        method: "PUT",
        headers: uploadHeaders(upload.signed_headers),
        body: file,
      })
      if (!uploadRes.ok) throw new Error(`material upload failed: ${uploadRes.status}`)

      setMaterialForm((current) => ({
        ...current,
        material_id: materialId,
        file_object_key: upload.object_key || "",
        file_hash: fileHash,
        file_size: String(file.size),
      }))
      toast.success(page.materialFileUploadSuccess)
    } catch (error) {
      console.error(error)
      toast.error(page.materialFileUploadFailed)
    } finally {
      setMaterialUploading(false)
    }
  }

  const createMaterial = async () => {
    if (!selectedCourse) return
    if (!materialForm.title.trim()) {
      toast.error(page.fillMaterialTitle)
      return
    }
    if (!materialForm.file_object_key.trim()) {
      toast.error(page.fillMaterialFileKey)
      return
    }

    setMaterialSaving(true)
    try {
      const isEditing = Boolean(selectedMaterialId)
      const materialId = materialForm.material_id || newMaterialDraftId()
      const payload = {
        material_id: materialId,
        title: materialForm.title.trim(),
        material_type: Number(materialForm.material_type || 0),
        sort_order: Number(materialForm.sort_order || (materials.length + 1) * 1000),
        file_object_key: materialForm.file_object_key.trim(),
        file_hash: materialForm.file_hash.trim(),
        file_size: Number(materialForm.file_size || 0),
      }
      const res = await apiClient(
        isEditing ? `/api/lms/materials/${selectedMaterialId}` : `/api/lms/courses/${selectedCourse.course_id}/materials`,
        {
          method: isEditing ? "PUT" : "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(
            isEditing
              ? {
                  ...payload,
                  version: selectedMaterial?.version || 0,
                }
              : payload
          ),
        }
      )
      toast.success(isEditing ? page.materialUpdateSuccess : page.materialCreateSuccess)
      const nextMaterialId = res?.material_id || materialId
      setMaterialForm(emptyMaterialForm)
      await loadMaterials(selectedCourse.course_id, nextMaterialId)
      await loadCourseDetail()
    } finally {
      setMaterialSaving(false)
    }
  }

  const deleteMaterial = async () => {
    if (!selectedMaterial) return
    if (!window.confirm(page.confirmDeleteMaterial)) return
    if (!selectedMaterial.version) {
      toast.error(page.materialVersionRequired)
      return
    }

    setMaterialSaving(true)
    try {
      await apiClient(`/api/lms/materials/${selectedMaterial.material_id}`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ version: selectedMaterial.version }),
      })
      toast.success(page.materialDeleteSuccess)
      resetMaterialForm()
      await loadMaterials(selectedCourse?.course_id)
      await loadCourseDetail()
    } finally {
      setMaterialSaving(false)
    }
  }

  const persistCourse = async (nextForm: CourseForm, course: LmsCourse) => {
    const clonedPublishedCourse = course.is_published || course.status?.toLowerCase() === "active"
    const targetCourse = clonedPublishedCourse ? await clonePublishedCourseDraft(course) : course
    await apiClient(`/api/lms/courses/${targetCourse.course_id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(formToPayload(nextForm, targetCourse.version)),
    })
    setSelectedId(targetCourse.course_id)
    setForm(nextForm)
    if (clonedPublishedCourse) {
      await loadChapters(targetCourse.course_id)
    }
    return targetCourse
  }

  const uploadThumbnail = async (file: File | null) => {
    if (!file) return
    if (!selectedCourse) {
      toast.error(page.createBeforeThumbnail)
      return
    }

    setThumbnailUploading(true)
    try {
      const contentType = file.type || "application/octet-stream"
      const fileHash = await sha256Hex(file)
      const upload = await apiClient("/api/lms/upload-url", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          upload_type: 1,
          file_name: file.name,
          content_type: contentType,
          course_id: selectedCourse.course_id,
          file_hash: fileHash,
        }),
      })

      const uploadRes = await fetch(upload.upload_url, {
        method: "PUT",
        headers: uploadHeaders(upload.signed_headers),
        body: file,
      })
      if (!uploadRes.ok) throw new Error(`thumbnail upload failed: ${uploadRes.status}`)

      const nextForm = { ...form, thumbnail_object_key: upload.object_key || "", thumbnail_file_hash: fileHash }
      setForm(nextForm)
      await persistCourse(nextForm, selectedCourse)
      toast.success(page.thumbnailUploadSuccess)
      await loadCourses()
    } catch (error) {
      console.error(error)
      toast.error(page.thumbnailUploadFailed)
    } finally {
      setThumbnailUploading(false)
    }
  }

  const showChapterContent = (chapter: Chapter, nextLessons: Lesson[], nextQuizzes: Quiz[]) => {
    setSelectedChapterId(chapter.chapter_id)
    setLessons(nextLessons)
    setQuizzes(nextQuizzes)
    setSelectedQuizId("")
    setQuestions([])
    setSelectedQuestionId("")
    setOptions([])
  }

  const showQuizQuestions = (chapter: Chapter, nextLessons: Lesson[], nextQuizzes: Quiz[], quiz: Quiz, nextQuestions: QuizQuestion[]) => {
    showChapterContent(chapter, nextLessons, nextQuizzes)
    setSelectedQuizId(quiz.quiz_id)
    setQuestions(nextQuestions)
  }

  const showQuestionOptions = (
    chapter: Chapter,
    nextLessons: Lesson[],
    nextQuizzes: Quiz[],
    quiz: Quiz,
    nextQuestions: QuizQuestion[],
    question: QuizQuestion,
    nextOptions: QuizOption[]
  ) => {
    showQuizQuestions(chapter, nextLessons, nextQuizzes, quiz, nextQuestions)
    setSelectedQuestionId(question.question_id)
    setOptions(nextOptions)
  }

  const getChapterName = (chapter: Chapter) => chapter.title || chapter.chapter_id

  const validatePublishReadiness = async (courseId: string) => {
    const chapterRes = await apiClient(`/api/lms/courses/${courseId}/chapters`)
    const nextChapters: Chapter[] = chapterRes?.chapters || []
    setChapters(nextChapters)

    if (nextChapters.length === 0) {
      toast.error(page.courseMissingChapters)
      return false
    }

    for (const chapter of nextChapters) {
      const [lessonRes, quizRes] = await Promise.all([
        apiClient(`/api/lms/chapters/${chapter.chapter_id}/lessons`),
        apiClient(`/api/lms/quizzes?${new URLSearchParams({ quizzable_type: "2", quizzable_id: chapter.chapter_id }).toString()}`),
      ])
      const nextLessons: Lesson[] = lessonRes?.lessons || []
      const nextQuizzes: Quiz[] = quizRes?.quizzes || []
      const activeQuizzes = nextQuizzes.filter((quiz) => quiz.is_active !== false)

      if (nextLessons.length === 0 && activeQuizzes.length === 0) {
        showChapterContent(chapter, nextLessons, nextQuizzes)
        toast.error(page.chapterMissingContent.replace("{{chapter}}", getChapterName(chapter)))
        return false
      }

      for (const quiz of activeQuizzes) {
        const questionRes = await apiClient(`/api/lms/quizzes/${quiz.quiz_id}/questions`)
        const nextQuestions: QuizQuestion[] = questionRes?.questions || []
        if (nextQuestions.length === 0) {
          showQuizQuestions(chapter, nextLessons, nextQuizzes, quiz, nextQuestions)
          toast.error(page.quizMissingQuestions.replace("{{quiz}}", quiz.title || quiz.quiz_id))
          return false
        }

        for (const question of nextQuestions) {
          const isChoiceQuestion = question.question_type === 1 || question.question_type === 2
          if (!isChoiceQuestion) continue

          const optionRes = await apiClient(`/api/lms/questions/${question.question_id}/options`)
          const nextOptions: QuizOption[] = optionRes?.options || []
          const correctCount = nextOptions.filter((option) => option.is_correct).length

          if (nextOptions.length < 2) {
            showQuestionOptions(chapter, nextLessons, nextQuizzes, quiz, nextQuestions, question, nextOptions)
            toast.error(page.questionMissingOptions.replace("{{question}}", question.question_text || question.question_id))
            return false
          }
          if (question.question_type === 1 && correctCount !== 1) {
            showQuestionOptions(chapter, nextLessons, nextQuizzes, quiz, nextQuestions, question, nextOptions)
            toast.error(page.singleChoiceCorrectOption.replace("{{question}}", question.question_text || question.question_id))
            return false
          }
          if (question.question_type === 2 && correctCount < 1) {
            showQuestionOptions(chapter, nextLessons, nextQuizzes, quiz, nextQuestions, question, nextOptions)
            toast.error(page.multipleChoiceCorrectOption.replace("{{question}}", question.question_text || question.question_id))
            return false
          }
        }
      }
    }

    return true
  }

  const publishCourse = async () => {
    if (!selectedCourse) return
    const ready = await validatePublishReadiness(selectedCourse.course_id)
    if (!ready) return

    try {
      await apiClient(`/api/lms/courses/${selectedCourse.course_id}/publish`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ version: selectedCourse.version || 0 }),
      })
      toast.success(page.publishSuccess)
      await loadCourses()
    } catch (error) {
      await loadChapters(selectedCourse.course_id)
      throw error
    }
  }

  const deleteCourse = async () => {
    if (!selectedCourse) return
    if (selectedCoursePublished) {
      toast.error(page.deletePublishedBlocked)
      return
    }
    if (!window.confirm(page.confirmDelete)) return
    await apiClient(`/api/lms/courses/${selectedCourse.course_id}`, {
      method: "DELETE",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ version: selectedCourse.version || 0 }),
    })
    toast.success(page.deleteSuccess)
    setSelectedId("")
    setForm(emptyForm)
    resetCourseInspectionState()
    resetCourseContentState()
    await loadCourses()
  }

  const loadCompleteCourse = async () => {
    if (!selectedCourse) return
    setPreviewLoading(true)
    try {
      const res = await apiClient(`/api/lms/courses/${selectedCourse.course_id}/complete`)
      setPreview(res?.complete_course || res)
    } finally {
      setPreviewLoading(false)
    }
  }

  const loadCourseDetail = async () => {
    if (!selectedCourse) return
    setCourseDetailLoading(true)
    try {
      const res = await apiClient(`/api/lms/courses/${selectedCourse.course_id}/detail`)
      setCourseDetail(res?.course_detail || res)
    } finally {
      setCourseDetailLoading(false)
    }
  }

  const loadChapters = async (courseId = selectedCourse?.course_id) => {
    if (!courseId) return
    setChaptersLoading(true)
    try {
      const res = await apiClient(`/api/lms/courses/${courseId}/chapters`)
      const nextChapters = res?.chapters || []
      setChapters(nextChapters)
      if (selectedChapterId && !nextChapters.some((chapter: Chapter) => chapter.chapter_id === selectedChapterId)) {
        setSelectedChapterId("")
        setLessons([])
        setQuizzes([])
        setSelectedQuizId("")
        setQuestions([])
        setSelectedQuestionId("")
        setOptions([])
      }
    } finally {
      setChaptersLoading(false)
    }
  }

  const createChapter = async () => {
    if (!selectedCourse) return
    if (!chapterForm.title.trim()) {
      toast.error(page.fillChapterTitle)
      return
    }

    setChapterSaving(true)
    try {
      await apiClient(`/api/lms/courses/${selectedCourse.course_id}/chapters`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title: chapterForm.title.trim(),
          sort_order: Number(chapterForm.sort_order || (chapters.length + 1) * 1000),
        }),
      })
      toast.success(page.chapterCreateSuccess)
      setChapterForm(emptyChapterForm)
      await loadChapters(selectedCourse.course_id)
    } finally {
      setChapterSaving(false)
    }
  }

  const selectChapter = async (chapter: Chapter) => {
    setChapterModalOpen(true)
    setSelectedChapterId(chapter.chapter_id)
    setSelectedQuizId("")
    setQuestions([])
    setSelectedQuestionId("")
    setOptions([])
    await Promise.all([loadLessons(chapter.chapter_id), loadQuizzes(chapter.chapter_id)])
  }

  const loadLessons = async (chapterId = selectedChapterId) => {
    if (!chapterId) return
    setLessonsLoading(true)
    try {
      const res = await apiClient(`/api/lms/chapters/${chapterId}/lessons`)
      setLessons(res?.lessons || [])
    } finally {
      setLessonsLoading(false)
    }
  }

  const createTextLesson = async () => {
    if (!selectedChapter) return
    if (!lessonForm.title.trim()) {
      toast.error(page.fillLessonTitle)
      return
    }
    if (!lessonForm.body.trim()) {
      toast.error(page.fillLessonBody)
      return
    }

    setLessonSaving(true)
    try {
      await apiClient(`/api/lms/chapters/${selectedChapter.chapter_id}/lessons`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title: lessonForm.title.trim(),
          body: lessonForm.body.trim(),
          sort_order: Number(lessonForm.sort_order || (lessons.length + 1) * 1000),
          lesson_type: 2,
          meta_json: "{}",
        }),
      })
      toast.success(page.lessonCreateSuccess)
      setLessonForm(emptyLessonForm)
      await loadLessons(selectedChapter.chapter_id)
    } finally {
      setLessonSaving(false)
    }
  }

  const loadQuizzes = async (chapterId = selectedChapterId) => {
    if (!chapterId) return
    setQuizzesLoading(true)
    try {
      const params = new URLSearchParams({
        quizzable_type: "2",
        quizzable_id: chapterId,
      })
      const res = await apiClient(`/api/lms/quizzes?${params.toString()}`)
      const nextQuizzes = res?.quizzes || []
      setQuizzes(nextQuizzes)
      if (selectedQuizId && !nextQuizzes.some((quiz: Quiz) => quiz.quiz_id === selectedQuizId)) {
        setSelectedQuizId("")
        setQuestions([])
        setSelectedQuestionId("")
        setOptions([])
      }
    } finally {
      setQuizzesLoading(false)
    }
  }

  const createQuiz = async () => {
    if (!selectedChapter) return
    if (!quizForm.title.trim()) {
      toast.error(page.fillQuizTitle)
      return
    }

    setQuizSaving(true)
    try {
      const isEditing = Boolean(selectedQuizId)
      const payload = {
        title: quizForm.title.trim(),
        description: quizForm.description.trim(),
        passing_score: Number(quizForm.passing_score || 0),
        time_limit: Number(quizForm.time_limit || 0),
        max_attempts: Number(quizForm.max_attempts || 0),
        allow_retake: quizForm.allow_retake,
        randomize_questions: quizForm.randomize_questions,
        is_active: quizForm.is_active,
        quizzable_type: 2,
        quizzable_id: selectedChapter.chapter_id,
      }

      let nextQuizId = selectedQuizId
      if (isEditing) {
        await apiClient(`/api/lms/quizzes/${selectedQuizId}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            ...payload,
            version: selectedQuiz?.version || 0,
          }),
        })
        toast.success(page.quizUpdateSuccess || page.quizCreateSuccess)
      } else {
        const res = await apiClient("/api/lms/quizzes", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        })
        toast.success(page.quizCreateSuccess)
        nextQuizId = res?.quiz_id || ""
      }
      
      setQuizForm(emptyQuizForm)
      setSelectedQuizId(nextQuizId)
      await loadQuizzes(selectedChapter.chapter_id)
    } finally {
      setQuizSaving(false)
    }
  }

  const resetQuizForm = () => {
    setQuizForm(emptyQuizForm)
    setSelectedQuizId("")
    setQuestions([])
    setSelectedQuestionId("")
    setOptions([])
  }

  const selectQuiz = async (quiz: Quiz) => {
    setSelectedQuizId(quiz.quiz_id)
    setQuizForm({
      title: quiz.title || "",
      description: quiz.description || "",
      passing_score: String(quiz.passing_score || 0),
      time_limit: String(quiz.time_limit || 0),
      max_attempts: String(quiz.max_attempts || 0),
      allow_retake: quiz.allow_retake || false,
      randomize_questions: quiz.randomize_questions || false,
      is_active: quiz.is_active ?? true,
    })
    setSelectedQuestionId("")
    setOptions([])
    await loadQuestions(quiz.quiz_id)
  }

  const loadQuestions = async (quizId = selectedQuizId) => {
    if (!quizId) return
    setQuestionsLoading(true)
    try {
      const res = await apiClient(`/api/lms/quizzes/${quizId}/questions`)
      const nextQuestions = res?.questions || []
      setQuestions(nextQuestions)
      if (selectedQuestionId && !nextQuestions.some((question: QuizQuestion) => question.question_id === selectedQuestionId)) {
        setSelectedQuestionId("")
        setOptions([])
      }
    } finally {
      setQuestionsLoading(false)
    }
  }

  const createQuestion = async () => {
    if (!selectedQuiz) return
    if (!questionForm.question_text.trim()) {
      toast.error(page.fillQuestionText)
      return
    }

    setQuestionSaving(true)
    try {
      const res = await apiClient(`/api/lms/quizzes/${selectedQuiz.quiz_id}/questions`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          question_text: questionForm.question_text.trim(),
          question_type: Number(questionForm.question_type),
          points: Number(questionForm.points || 1),
          sort_order: Number(questionForm.sort_order || (questions.length + 1) * 1000),
          is_required: questionForm.is_required,
          media_items_json: "[]",
        }),
      })
      toast.success(page.questionCreateSuccess)
      setQuestionForm(emptyQuestionForm)
      setSelectedQuestionId(res?.question_id || "")
      await loadQuestions(selectedQuiz.quiz_id)
    } finally {
      setQuestionSaving(false)
    }
  }

  const selectQuestion = async (question: QuizQuestion) => {
    setSelectedQuestionId(question.question_id)
    await loadOptions(question.question_id)
  }

  const loadOptions = async (questionId = selectedQuestionId) => {
    if (!questionId) return
    setOptionsLoading(true)
    try {
      const res = await apiClient(`/api/lms/questions/${questionId}/options`)
      setOptions(res?.options || [])
    } finally {
      setOptionsLoading(false)
    }
  }

  const createOption = async () => {
    if (!selectedQuestion) return
    if (!optionForm.option_text.trim()) {
      toast.error(page.fillOptionText)
      return
    }

    setOptionSaving(true)
    try {
      await apiClient(`/api/lms/questions/${selectedQuestion.question_id}/options`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          option_text: optionForm.option_text.trim(),
          sort_order: Number(optionForm.sort_order || (options.length + 1) * 1000),
          is_correct: optionForm.is_correct,
        }),
      })
      toast.success(page.optionCreateSuccess)
      setOptionForm(emptyOptionForm)
      await loadOptions(selectedQuestion.question_id)
    } finally {
      setOptionSaving(false)
    }
  }

  const loadCourseEnrollments = async () => {
    if (!selectedCourse) return
    setEnrollmentsLoading(true)
    try {
      const params = new URLSearchParams({ page_size: "50" })
      if (enrollmentStatus !== "all") params.set("status", enrollmentStatus)
      const res = await apiClient(`/api/lms/courses/${selectedCourse.course_id}/enrollments?${params.toString()}`)
      setEnrollments(res?.enrollments || [])
      setProgressDetail(null)
      setEnrollmentDetail(null)
      setLessonProgress([])
      setLessonProgressDetail(null)
      setChapterProgress(null)
      setQuizAttempts([])
      setQuizAttemptDetail(null)
    } finally {
      setEnrollmentsLoading(false)
    }
  }

  const batchEnrollCandidate = async () => {
    if (!selectedCourse) return
    const candidateId = batchEnrollCandidateId.trim()
    if (!candidateId) {
      toast.error(page.fillCandidateId)
      return
    }
    setBatchEnrolling(true)
    try {
      await apiClient("/api/lms/enrollments/batch", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          candidate_id: candidateId,
          course_ids: [selectedCourse.course_id],
          biz_unit: "adminserver",
        }),
      })
      toast.success(page.batchEnrollSuccess)
      setBatchEnrollCandidateId("")
      await loadCourseEnrollments()
    } finally {
      setBatchEnrolling(false)
    }
  }

  const grantCourseAccess = async () => {
    if (!selectedCourse) return
    const candidateId = batchEnrollCandidateId.trim()
    if (!candidateId) {
      toast.error(page.fillCandidateId || "请输入 Candidate ID")
      return
    }
    setBatchEnrolling(true)
    try {
      await apiClient(`/api/lms/courses/${selectedCourse.course_id}/permissions/grant`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          candidate_id: candidateId,
          biz_unit: "adminserver",
          operator_id: "admin",
        }),
      })
      toast.success(page.grantSuccess || "授权成功")
    } catch (err: any) {
      console.error(err)
    } finally {
      setBatchEnrolling(false)
    }
  }

  const revokeCourseAccess = async () => {
    if (!selectedCourse) return
    const candidateId = batchEnrollCandidateId.trim()
    if (!candidateId) {
      toast.error(page.fillCandidateId || "请输入 Candidate ID")
      return
    }
    setBatchEnrolling(true)
    try {
      await apiClient(`/api/lms/courses/${selectedCourse.course_id}/permissions/revoke`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          candidate_id: candidateId,
          biz_unit: "adminserver",
          operator_id: "admin",
        }),
      })
      toast.success(page.revokeSuccess || "撤销授权成功")
    } catch (err: any) {
      console.error(err)
    } finally {
      setBatchEnrolling(false)
    }
  }

  const loadEnrollmentDetail = async (enrollment: CourseEnrollment) => {
    if (!enrollment.enrollment_id) return
    setEnrollmentDetailLoadingFor(enrollment.enrollment_id)
    try {
      const res = await apiClient(`/api/lms/enrollments/${enrollment.enrollment_id}`)
      setEnrollmentDetail(res?.enrollment || res)
    } finally {
      setEnrollmentDetailLoadingFor("")
    }
  }

  const loadCandidateProgress = async (candidateId: string) => {
    if (!selectedCourse || !candidateId) return
    setProgressLoadingFor(candidateId)
    try {
      const res = await apiClient(`/api/lms/courses/${selectedCourse.course_id}/candidates/${candidateId}/progress`)
      setProgressDetail(res)
    } finally {
      setProgressLoadingFor("")
    }
  }

  const syncCandidateProgress = async (candidateId: string) => {
    if (!selectedCourse || !candidateId) return
    setSyncProgressLoadingFor(candidateId)
    try {
      const res = await apiClient(`/api/lms/courses/${selectedCourse.course_id}/progress/sync`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ candidate_id: candidateId }),
      })
      setProgressDetail(res)
      toast.success(page.syncProgressSuccess)
      await loadCourseEnrollments()
    } finally {
      setSyncProgressLoadingFor("")
    }
  }

  const loadLessonProgressForCandidate = async (candidateId: string) => {
    if (!candidateId) return
    setLessonProgressLoadingFor(candidateId)
    try {
      const params = new URLSearchParams({ candidate_id: candidateId, page_size: "50" })
      const res = await apiClient(`/api/lms/lesson-progress?${params.toString()}`)
      setLessonProgress(res?.progress || [])
      setLessonProgressDetail(null)
    } finally {
      setLessonProgressLoadingFor("")
    }
  }

  const loadLessonProgressDetail = async (item: LessonProgress) => {
    const candidateID = item.user_id || item.candidate_id || ""
    if (!candidateID || !item.lesson_id) return
    setLessonProgressLoadingFor(`${candidateID}:${item.lesson_id}`)
    try {
      const params = new URLSearchParams({ candidate_id: candidateID })
      const res = await apiClient(`/api/lms/lessons/${item.lesson_id}/progress?${params.toString()}`)
      setLessonProgressDetail(res?.progress || res)
    } finally {
      setLessonProgressLoadingFor("")
    }
  }

  const loadChapterProgressForCandidate = async (candidateId: string, chapterId = selectedChapterId) => {
    if (!selectedCourse || !candidateId || !chapterId) {
      toast.error(page.selectChapterForProgress)
      return
    }
    setChapterProgressLoadingFor(candidateId)
    try {
      const params = new URLSearchParams({ candidate_id: candidateId })
      const res = await apiClient(`/api/lms/chapters/${chapterId}/progress?${params.toString()}`)
      setChapterProgress(res?.progress || res?.chapter_progress || res)
    } finally {
      setChapterProgressLoadingFor("")
    }
  }

  const loadQuizAttemptsForCandidate = async (candidateId: string, quizId = selectedQuizId) => {
    if (!candidateId || !quizId) {
      toast.error(page.selectQuizForAttempts)
      return
    }
    setQuizAttemptsLoadingFor(candidateId)
    try {
      const params = new URLSearchParams({ candidate_id: candidateId, page_size: "20" })
      const res = await apiClient(`/api/lms/quizzes/${quizId}/attempts?${params.toString()}`)
      setQuizAttempts(res?.attempts || [])
      setQuizAttemptDetail(null)
    } finally {
      setQuizAttemptsLoadingFor("")
    }
  }

  const loadQuizAttemptDetail = async (attempt: QuizAttempt) => {
    if (!attempt.attempt_id) return
    setQuizAttemptsLoadingFor(attempt.attempt_id)
    try {
      const res = await apiClient(`/api/lms/quiz-attempts/${attempt.attempt_id}`)
      setQuizAttemptDetail(res?.attempt || res)
    } finally {
      setQuizAttemptsLoadingFor("")
    }
  }

  const loadBrokenAssets = async (pageToken = "") => {
    setBrokenAssetsLoading(true)
    try {
      const params = new URLSearchParams({ page_size: "20" })
      if (pageToken) params.set("page_token", pageToken)
      if (brokenAssetType !== "all") params.set("asset_type", brokenAssetType)
      if (assetStatus !== "all") params.set("status", assetStatus)
      const res = await apiClient(`/api/lms/assets?${params.toString()}`)
      setBrokenAssets(pageToken ? [...brokenAssets, ...(res?.assets || [])] : res?.assets || [])
      setBrokenAssetsNextPageToken(res?.next_page_token || "")
    } finally {
      setBrokenAssetsLoading(false)
    }
  }

  const loadAssetDetail = async (asset: BrokenAsset) => {
    if (!asset.object_key || !asset.associated_id) {
      toast.error(page.assetDetailMissingFields)
      return
    }
    setAssetDetailLoadingFor(`${asset.object_key}:${asset.associated_id}`)
    try {
      const params = new URLSearchParams({ object_key: asset.object_key, associated_id: asset.associated_id })
      const res = await apiClient(`/api/lms/assets/detail?${params.toString()}`)
      setAssetDetail(res?.asset || res)
    } finally {
      setAssetDetailLoadingFor("")
    }
  }

  const describeBrokenAssetOwner = (asset: BrokenAsset) => {
    const owner =
      asset.lesson_title ||
      asset.material_title ||
      asset.chapter_title ||
      asset.course_title ||
      asset.lesson_id ||
      asset.material_id ||
      asset.chapter_id ||
      asset.course_id
    return owner || t.common.na
  }

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
            <div>
              <h1 className="text-3xl font-bold text-foreground">{page.title}</h1>
              <p className="mt-2 text-muted-foreground">{page.subtitle}</p>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" onClick={() => loadCourses()} disabled={loading}>
                <RefreshCw className={cn("mr-2 h-4 w-4", loading && "animate-spin")} />
                {page.refresh}
              </Button>
              <Button variant="outline" onClick={() => setImportOpen(true)}>
                <FileJson className="mr-2 h-4 w-4" />
                {page.importJson}
              </Button>
              <Button onClick={startNewCourse}>
                <Plus className="mr-2 h-4 w-4" />
                {page.newCourse}
              </Button>
            </div>
          </div>

          <Dialog open={importOpen} onOpenChange={setImportOpen}>
            <DialogContent className="flex max-h-[90vh] max-w-4xl grid-rows-none flex-col gap-0 overflow-hidden p-0">
              <DialogHeader className="border-b px-6 py-4">
                <DialogTitle>{page.importJson}</DialogTitle>
              </DialogHeader>
              <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-6 py-4">
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant={importScope === "course" ? "default" : "outline"}
                    onClick={() => switchImportScope("course")}
                  >
                    {page.importCourse}
                  </Button>
                  <Button
                    type="button"
                    variant={importScope === "quiz" ? "default" : "outline"}
                    onClick={() => switchImportScope("quiz")}
                  >
                    {page.importQuiz}
                  </Button>
                </div>
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <p className="text-sm text-muted-foreground">
                    {importScope === "course"
                      ? page.importCourseHint
                      : page.importQuizHint.replace("{{chapter}}", selectedChapter ? getChapterName(selectedChapter) : page.selectChapterHint)}
                  </p>
                  <Input
                    type="file"
                    accept="application/json,.json"
                    className="min-w-0 max-w-full shrink sm:max-w-xs"
                    onChange={(event) => {
                      loadImportFile(event.target.files?.[0] || null)
                      event.currentTarget.value = ""
                    }}
                  />
                </div>
                {importScope === "course" && (
                  <div className="space-y-2">
                    <Label htmlFor="importCategoryTips">{page.importCategoryTips}</Label>
                    <Input
                      id="importCategoryTips"
                      value={importCategoryTips}
                      onChange={(event) => setImportCategoryTips(event.target.value)}
                    />
                  </div>
                )}
                <Textarea
                  value={importJson}
                  onChange={(event) => setImportJson(event.target.value)}
                  className="min-h-[460px] font-mono text-xs"
                  spellCheck={false}
                />
              </div>
              <div className="flex shrink-0 justify-end gap-2 border-t bg-background px-6 py-4">
                <Button variant="outline" onClick={() => setImportOpen(false)} disabled={importing}>
                  {t.common.cancel}
                </Button>
                <Button onClick={importLmsJson} disabled={importing}>
                  <UploadCloud className={cn("mr-2 h-4 w-4", importing && "animate-spin")} />
                  {importScope === "course" ? page.importCourse : page.importQuiz}
                </Button>
              </div>
            </DialogContent>
          </Dialog>

          <div className="mb-4 flex flex-wrap items-end gap-3">
            <div className="w-72 space-y-2">
              <Label htmlFor="categoryFilter">{page.categoryFilter}</Label>
              <Select value={categoryFilter || "all"} onValueChange={(value) => setCategoryFilter(value === "all" ? "" : value)}>
                <SelectTrigger id="categoryFilter" className="w-full">
                  <SelectValue placeholder={page.categoryPlaceholder} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{page.categoryPlaceholder}</SelectItem>
                  {catalogs.map((catalog) => (
                    <SelectItem key={catalog.catalog_id} value={catalog.catalog_id}>
                      {catalog.name || catalog.catalog_id}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <label className="flex h-10 items-center gap-2 rounded-md border px-3 text-sm">
              <Checkbox checked={publishedOnly} onCheckedChange={(checked) => setPublishedOnly(Boolean(checked))} />
              {page.publishedOnly}
            </label>
          </div>

          <div>
            {!selectedCourse ? (
            <section className="rounded-lg border bg-card">
              <div className="flex items-center justify-between border-b px-4 py-3">
                <h2 className="font-semibold">{page.courseList}</h2>
                <Badge variant="outline">{courses.length}</Badge>
              </div>
              <div className="max-h-[680px] overflow-y-auto">
                {loading ? (
                  <div className="p-4 text-sm text-muted-foreground">{t.common.loading}</div>
                ) : courses.length === 0 ? (
                  <div className="p-8 text-center text-sm text-muted-foreground">{page.noCourses}</div>
                ) : (
                  courses.map((course) => (
                    <button
                      key={course.course_id}
                      onClick={() => selectCourse(course)}
                      className={cn(
                        "flex w-full flex-col gap-2 border-b px-4 py-3 text-left transition-colors last:border-b-0 hover:bg-accent",
                        selectedId === course.course_id && "bg-accent"
                      )}
                    >
                      <div className="flex items-center justify-between gap-3">
                        <span className="truncate font-medium text-foreground">{course.title || t.common.unknownCourse}</span>
                        <Badge variant={course.is_published ? "default" : "outline"}>
                          {course.is_published ? page.published : page.draft}
                        </Badge>
                      </div>
                      <div className="truncate text-xs text-muted-foreground">{course.course_guid || course.course_id}</div>
                      <div className="truncate text-xs text-muted-foreground">{course.course_id}</div>
                      <div className="flex items-center justify-between text-xs text-muted-foreground">
                        <span>{statusLabel(t, LMS_COURSE_STATUS_LABELS, course.status)}</span>
                        <span>
                          {page.version} {course.version || 0}
                        </span>
                      </div>
                    </button>
                  ))
                )}
              </div>
              {courseListNextPageToken && (
                <div className="border-t p-3">
                  <Button variant="outline" size="sm" className="w-full" onClick={() => loadCourses(courseListNextPageToken)} disabled={courseListLoadingMore}>
                    <RefreshCw className={cn("mr-2 h-4 w-4", courseListLoadingMore && "animate-spin")} />
                    {page.loadMoreCourses}
                  </Button>
                </div>
              )}
            </section>
            ) : (
            <section className="space-y-4">
              <div className="mb-2">
                <Button variant="ghost" onClick={() => setSelectedId("")}>
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  {(t.common as any).back || "返回"}
                </Button>
              </div>
              <div className="rounded-lg border bg-card">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
                  <h2 className="font-semibold">{page.courseEditor}</h2>
                  {selectedCourse && (
                    <div className="flex gap-2">
                      {selectedCoursePublished ? (
                        <Button variant="outline" size="sm" disabled>
                          <CheckCircle2 className="mr-2 h-4 w-4" />
                          {page.published}
                        </Button>
                      ) : (
                        <Button variant="outline" size="sm" onClick={publishCourse}>
                          <UploadCloud className="mr-2 h-4 w-4" />
                          {page.publish}
                        </Button>
                      )}
                      <ProtectedButton variant="destructive" size="sm" onClick={deleteCourse} disabled={selectedCoursePublished} isPublished={selectedCoursePublished}>
                        <Trash2 className="mr-2 h-4 w-4" />
                        {page.delete}
                      </ProtectedButton>
                    </div>
                  )}
                </div>

                <div className="border-b bg-muted/40 px-4 py-3 text-sm text-muted-foreground">{page.courseFlowHint}</div>

                <div className="grid gap-4 p-4 lg:grid-cols-[minmax(0,1fr)_220px]">
                  <div className="space-y-2">
                    <Label htmlFor="title">{page.titleLabel}</Label>
                    <Input id="title" value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="duration">{page.durationMin}</Label>
                    <Input
                      id="duration"
                      type="number"
                      min="0"
                      value={form.duration_min}
                      onChange={(event) => setForm({ ...form, duration_min: event.target.value })}
                    />
                  </div>
                  <details className="rounded-md border bg-muted/20 p-3 lg:col-span-2">
                    <summary className="cursor-pointer select-none text-sm font-medium text-foreground">
                      {page.optionalSettings || "可选设置"}
                    </summary>
                    <div className="mt-4 grid gap-4 lg:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="categoryId">{page.categorySelect}</Label>
                    <Select value={form.category_tips || "none"} onValueChange={(value) => setForm({ ...form, category_tips: value === "none" ? "" : value })}>
                      <SelectTrigger id="categoryId" className="w-full">
                        <SelectValue placeholder={page.noCategory} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">{page.noCategory}</SelectItem>
                        {catalogs.map((catalog) => (
                          <SelectItem key={catalog.catalog_id} value={catalog.catalog_id}>
                            {catalog.name || catalog.catalog_id}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="thumbnail">{page.thumbnailUpload}</Label>
                    <Input
                      id="thumbnail"
                      type="file"
                      accept="image/*"
                      disabled={!selectedCourse || thumbnailUploading}
                      onChange={(event) => {
                        uploadThumbnail(event.target.files?.[0] || null)
                        event.currentTarget.value = ""
                      }}
                    />
                    <div className="truncate text-xs text-muted-foreground">
                      {form.thumbnail_object_key || (selectedCourse ? page.noThumbnail : page.createBeforeThumbnail)}
                    </div>
                  </div>
                  <div className="space-y-2 lg:col-span-2">
                    <Label htmlFor="description">{page.description}</Label>
                    <Textarea
                      id="description"
                      value={form.description}
                      onChange={(event) => setForm({ ...form, description: event.target.value })}
                    />
                  </div>
                  <label className="flex items-center gap-2 rounded-md border px-3 py-2 text-sm">
                    <Checkbox
                      checked={form.certification_enabled}
                      onCheckedChange={(checked) =>
                        setForm({
                          ...form,
                          certification_enabled: Boolean(checked),
                          certification_def_id: checked ? form.certification_def_id : "",
                        })
                      }
                    />
                    {page.certificationEnabled}
                  </label>
                  <div className="space-y-2">
                    <Label htmlFor="certificationDefId">{page.certificationDefId}</Label>
                    <Select
                      value={form.certification_def_id || "none"}
                      disabled={!form.certification_enabled}
                      onValueChange={(value) => setForm({ ...form, certification_def_id: value === "none" ? "" : value })}
                    >
                      <SelectTrigger id="certificationDefId" className="w-full">
                        <SelectValue placeholder={page.noCertificationDef} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">{page.noCertificationDef}</SelectItem>
                        {credentialDefinitions.map((definition) => (
                          <SelectItem key={definition.cred_def_id} value={definition.cred_def_id}>
                            {definition.name || definition.cred_def_id}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                    </div>
                  </details>
                </div>

                {selectedCourse && (
                  <div className="border-t px-4 py-3 text-xs text-muted-foreground">
                    <span className="mr-4">GUID: {selectedCourse.course_guid || t.common.na}</span>
                    <span className="mr-4">ID: {selectedCourse.course_id}</span>
                    <span className="mr-4">
                      {page.version}: {selectedCourse.version || 0}
                    </span>
                    <span className="mr-4">{statusLabel(t, LMS_COURSE_STATUS_LABELS, selectedCourse.status)}</span>
                    <span className="mr-4">{selectedCourse.is_current ? page.current : page.historical}</span>
                    <span>{formatBackendDate(selectedCourse.updated_at || selectedCourse.created_at)}</span>
                  </div>
                )}

                <div className="flex justify-end border-t px-4 py-3">
                  <Button onClick={saveCourse} disabled={saving}>
                    {selectedCourse ? <Save className="mr-2 h-4 w-4" /> : <Plus className="mr-2 h-4 w-4" />}
                    {selectedCourse ? page.saveCourse : page.createCourse}
                  </Button>
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
                  <div>
                    <h2 className="font-semibold">{page.courseContent}</h2>
                    <p className="mt-1 text-sm text-muted-foreground">{page.publishRequirements}</p>
                  </div>
                  <Button variant="outline" size="sm" onClick={() => loadChapters()} disabled={!selectedCourse || chaptersLoading}>
                    <RefreshCw className={cn("mr-2 h-4 w-4", chaptersLoading && "animate-spin")} />
                    {page.loadChapters}
                  </Button>
                </div>
                <div className="p-4">
                  {!selectedCourse ? (
                    <div className="xl:col-span-2 flex items-center gap-2 text-sm text-muted-foreground">
                      <BookOpen className="h-4 w-4" />
                      {page.selectCourseHint}
                    </div>
                  ) : (
                    <>
                      <div className="space-y-3">
                        <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_120px]">
                          <div className="space-y-2">
                            <Label htmlFor="chapterTitle">{page.chapterTitle}</Label>
                            <Input
                              id="chapterTitle"
                              value={chapterForm.title}
                              disabled={selectedCourse.is_published}
                              onChange={(event) => setChapterForm({ ...chapterForm, title: event.target.value })}
                            />
                          </div>
                          <div className="space-y-2">
                            <Label htmlFor="chapterSortOrder">{page.chapterSortOrder}</Label>
                            <Input
                              id="chapterSortOrder"
                              type="number"
                              min="0"
                              value={chapterForm.sort_order}
                              disabled={selectedCourse.is_published}
                              onChange={(event) => setChapterForm({ ...chapterForm, sort_order: event.target.value })}
                            />
                          </div>
                        </div>
                        <ProtectedButton
                          size="sm"
                          onClick={createChapter}
                          disabled={chapterSaving || selectedCourse.is_published}
                          isPublished={selectedCourse.is_published}
                        >
                          <Plus className="mr-2 h-4 w-4" />
                          {page.createChapter}
                        </ProtectedButton>
                        {selectedCourse.is_published && <div className="text-xs text-muted-foreground">{page.publishedContentLocked}</div>}

                        <div className="overflow-hidden rounded-md border">
                          {chapters.length === 0 ? (
                            <div className="px-3 py-6 text-center text-sm text-muted-foreground">{page.noChapters}</div>
                          ) : (
                            chapters.map((chapter) => (
                              <button
                                key={chapter.chapter_id}
                                type="button"
                                onClick={() => selectChapter(chapter)}
                                className={cn(
                                  "block w-full border-b px-3 py-3 text-left text-sm last:border-b-0 hover:bg-muted/60",
                                  selectedChapterId === chapter.chapter_id && "bg-muted"
                                )}
                              >
                                <div className="truncate font-medium">{chapter.title || chapter.chapter_id}</div>
                                <div className="mt-1 flex items-center justify-between gap-3 text-xs text-muted-foreground">
                                  <span className="truncate">{chapter.chapter_id}</span>
                                  <span>{chapter.sort_order || 0}</span>
                                </div>
                              </button>
                            ))
                          )}
                        </div>
                      </div>

                      <Dialog open={chapterModalOpen} onOpenChange={setChapterModalOpen}>
                        <DialogContent className="max-w-5xl max-h-[90vh] overflow-y-auto flex flex-col">
                          <DialogHeader className="shrink-0">
                            <DialogTitle>{selectedChapter ? getChapterName(selectedChapter) : page.selectChapterHint}</DialogTitle>
                          </DialogHeader>
                          <div className="space-y-4 flex-1 overflow-y-auto">
                        <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_120px]">
                          <div className="space-y-2">
                            <Label htmlFor="lessonTitle">{page.lessonTitle}</Label>
                            <Input
                              id="lessonTitle"
                              value={lessonForm.title}
                              disabled={!selectedChapter || selectedCourse.is_published}
                              onChange={(event) => setLessonForm({ ...lessonForm, title: event.target.value })}
                            />
                          </div>
                          <div className="space-y-2">
                            <Label htmlFor="lessonSortOrder">{page.lessonSortOrder}</Label>
                            <Input
                              id="lessonSortOrder"
                              type="number"
                              min="0"
                              value={lessonForm.sort_order}
                              disabled={!selectedChapter || selectedCourse.is_published}
                              onChange={(event) => setLessonForm({ ...lessonForm, sort_order: event.target.value })}
                            />
                          </div>
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="lessonBody">{page.lessonBody}</Label>
                          <Textarea
                            id="lessonBody"
                            rows={4}
                            value={lessonForm.body}
                            disabled={!selectedChapter || selectedCourse.is_published}
                            onChange={(event) => setLessonForm({ ...lessonForm, body: event.target.value })}
                          />
                        </div>
                        <ProtectedButton
                          size="sm"
                          onClick={createTextLesson}
                          disabled={!selectedChapter || lessonSaving || lessonsLoading || selectedCourse.is_published}
                          isPublished={selectedCourse.is_published}
                        >
                          <FileText className="mr-2 h-4 w-4" />
                          {page.createTextLesson}
                        </ProtectedButton>

                        {!selectedChapter ? (
                          <div className="rounded-md border px-3 py-6 text-center text-sm text-muted-foreground">{page.selectChapterHint}</div>
                        ) : lessons.length === 0 ? (
                          <div className="rounded-md border px-3 py-6 text-center text-sm text-muted-foreground">{page.noLessons}</div>
                        ) : (
                          <div className="overflow-hidden rounded-md border">
                            {lessons.map((lesson) => (
                              <div key={lesson.lesson_id} className="border-b px-3 py-3 text-sm last:border-b-0">
                                <div className="flex items-center justify-between gap-3">
                                  <div className="truncate font-medium">{lesson.title || lesson.lesson_id}</div>
                                  <Badge variant="outline">{page.textLesson}</Badge>
                                </div>
                                <div className="mt-1 truncate text-xs text-muted-foreground">{lesson.lesson_id}</div>
                              </div>
                            ))}
                          </div>
                        )}

                        <div className="border-t pt-4">
                          <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                            <div>
                              <h3 className="text-sm font-semibold">{page.chapterQuizzes}</h3>
                              <p className="mt-1 text-xs text-muted-foreground">{page.quizHint}</p>
                            </div>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => loadQuizzes()}
                              disabled={!selectedChapter || quizzesLoading}
                            >
                              <RefreshCw className={cn("mr-2 h-4 w-4", quizzesLoading && "animate-spin")} />
                              {page.loadQuizzes}
                            </Button>
                          </div>

                          <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_120px_120px]">
                            <div className="space-y-2">
                              <Label htmlFor="quizTitle">{page.quizTitle}</Label>
                              <Input
                                id="quizTitle"
                                value={quizForm.title}
                                disabled={!selectedChapter || selectedCourse.is_published}
                                onChange={(event) => setQuizForm({ ...quizForm, title: event.target.value })}
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="quizPassingScore">{page.passingScore}</Label>
                              <Input
                                id="quizPassingScore"
                                type="number"
                                min="0"
                                value={quizForm.passing_score}
                                disabled={!selectedChapter || selectedCourse.is_published}
                                onChange={(event) => setQuizForm({ ...quizForm, passing_score: event.target.value })}
                              />
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="quizMaxAttempts">{page.maxAttempts}</Label>
                              <Input
                                id="quizMaxAttempts"
                                type="number"
                                min="0"
                                value={quizForm.max_attempts}
                                disabled={!selectedChapter || selectedCourse.is_published}
                                onChange={(event) => setQuizForm({ ...quizForm, max_attempts: event.target.value })}
                              />
                            </div>
                          </div>
                          <div className="mt-3 space-y-2">
                            <Label htmlFor="quizDescription">{page.quizDescription}</Label>
                            <Textarea
                              id="quizDescription"
                              rows={2}
                              value={quizForm.description}
                              disabled={!selectedChapter || selectedCourse.is_published}
                              onChange={(event) => setQuizForm({ ...quizForm, description: event.target.value })}
                            />
                          </div>
                          <div className="mt-3 flex flex-wrap gap-3">
                            <label className="flex h-9 items-center gap-2 rounded-md border px-3 text-sm">
                              <Checkbox
                                checked={quizForm.allow_retake}
                                disabled={!selectedChapter || selectedCourse.is_published}
                                onCheckedChange={(checked) => setQuizForm({ ...quizForm, allow_retake: Boolean(checked) })}
                              />
                              {page.allowRetake}
                            </label>
                            <label className="flex h-9 items-center gap-2 rounded-md border px-3 text-sm">
                              <Checkbox
                                checked={quizForm.randomize_questions}
                                disabled={!selectedChapter || selectedCourse.is_published}
                                onCheckedChange={(checked) => setQuizForm({ ...quizForm, randomize_questions: Boolean(checked) })}
                              />
                              {page.randomizeQuestions}
                            </label>
                            <ProtectedButton
                              size="sm"
                              onClick={createQuiz}
                              disabled={!selectedChapter || quizSaving || selectedCourse.is_published}
                              isPublished={selectedCourse.is_published}
                            >
                              <ClipboardList className="mr-2 h-4 w-4" />
                              {selectedQuiz ? (page.updateQuiz || page.createQuiz) : page.createQuiz}
                            </ProtectedButton>
                            {selectedQuiz && (
                              <Button variant="outline" size="sm" onClick={resetQuizForm} disabled={!selectedChapter}>
                                {t.common.cancel}
                              </Button>
                            )}
                          </div>

                          <div className="mt-3 overflow-hidden rounded-md border">
                            {quizzes.length === 0 ? (
                              <div className="px-3 py-6 text-center text-sm text-muted-foreground">{page.noQuizzes}</div>
                            ) : (
                              quizzes.map((quiz) => (
                                <button
                                  key={quiz.quiz_id}
                                  type="button"
                                  onClick={() => selectQuiz(quiz)}
                                  className={cn(
                                    "block w-full border-b px-3 py-3 text-left text-sm last:border-b-0 hover:bg-muted/60",
                                    selectedQuizId === quiz.quiz_id && "bg-muted"
                                  )}
                                >
                                  <div className="flex items-center justify-between gap-3">
                                    <span className="truncate font-medium">{quiz.title || quiz.quiz_id}</span>
                                    <Badge variant={quiz.is_active ? "default" : "outline"}>{quiz.is_active ? page.active : page.inactive}</Badge>
                                  </div>
                                  <div className="mt-1 truncate text-xs text-muted-foreground">{quiz.quiz_id}</div>
                                </button>
                              ))
                            )}
                          </div>

                          <div className="mt-4 rounded-md border p-3">
                            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                              <div>
                                <h4 className="text-sm font-semibold">{page.quizQuestions}</h4>
                                <p className="mt-1 text-xs text-muted-foreground">{selectedQuiz ? selectedQuiz.title || selectedQuiz.quiz_id : page.selectQuizHint}</p>
                              </div>
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => loadQuestions()}
                                disabled={!selectedQuiz || questionsLoading}
                              >
                                <RefreshCw className={cn("mr-2 h-4 w-4", questionsLoading && "animate-spin")} />
                                {page.loadQuestions}
                              </Button>
                            </div>
                            <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_160px_100px]">
                              <div className="space-y-2">
                                <Label htmlFor="questionText">{page.questionText}</Label>
                                <Input
                                  id="questionText"
                                  value={questionForm.question_text}
                                  disabled={!selectedQuiz || selectedCourse.is_published}
                                  onChange={(event) => setQuestionForm({ ...questionForm, question_text: event.target.value })}
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="questionType">{page.questionType}</Label>
                                <Select
                                  value={questionForm.question_type}
                                  disabled={!selectedQuiz || selectedCourse.is_published}
                                  onValueChange={(value) => setQuestionForm({ ...questionForm, question_type: value })}
                                >
                                  <SelectTrigger id="questionType">
                                    <SelectValue />
                                  </SelectTrigger>
                                  <SelectContent>
                                    <SelectItem value="1">{page.singleChoice}</SelectItem>
                                    <SelectItem value="2">{page.multipleChoice}</SelectItem>
                                    <SelectItem value="3">{page.trueFalse}</SelectItem>
                                  </SelectContent>
                                </Select>
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="questionPoints">{page.points}</Label>
                                <Input
                                  id="questionPoints"
                                  type="number"
                                  min="1"
                                  value={questionForm.points}
                                  disabled={!selectedQuiz || selectedCourse.is_published}
                                  onChange={(event) => setQuestionForm({ ...questionForm, points: event.target.value })}
                                />
                              </div>
                            </div>
                            <div className="mt-3 flex flex-wrap gap-3">
                              <Input
                                type="number"
                                min="0"
                                className="w-32"
                                placeholder={page.questionSortOrder}
                                value={questionForm.sort_order}
                                disabled={!selectedQuiz || selectedCourse.is_published}
                                onChange={(event) => setQuestionForm({ ...questionForm, sort_order: event.target.value })}
                              />
                              <label className="flex h-9 items-center gap-2 rounded-md border px-3 text-sm">
                                <Checkbox
                                  checked={questionForm.is_required}
                                  disabled={!selectedQuiz || selectedCourse.is_published}
                                  onCheckedChange={(checked) => setQuestionForm({ ...questionForm, is_required: Boolean(checked) })}
                                />
                                {page.requiredQuestion}
                              </label>
                              <ProtectedButton size="sm" onClick={createQuestion} disabled={!selectedQuiz || questionSaving || selectedCourse.is_published} isPublished={selectedCourse.is_published}>
                                <Plus className="mr-2 h-4 w-4" />
                                {page.createQuestion}
                              </ProtectedButton>
                            </div>

                            <div className="mt-3 overflow-hidden rounded-md border">
                              {questions.length === 0 ? (
                                <div className="px-3 py-6 text-center text-sm text-muted-foreground">{page.noQuestions}</div>
                              ) : (
                                questions.map((question) => (
                                  <button
                                    key={question.question_id}
                                    type="button"
                                    onClick={() => selectQuestion(question)}
                                    className={cn(
                                      "block w-full border-b px-3 py-3 text-left text-sm last:border-b-0 hover:bg-muted/60",
                                      selectedQuestionId === question.question_id && "bg-muted"
                                    )}
                                  >
                                    <div className="flex items-center justify-between gap-3">
                                      <span className="truncate font-medium">{question.question_text || question.question_id}</span>
                                      <Badge variant="outline">{question.points || 0}</Badge>
                                    </div>
                                    <div className="mt-1 truncate text-xs text-muted-foreground">{question.question_id}</div>
                                  </button>
                                ))
                              )}
                            </div>
                          </div>

                          <div className="mt-4 rounded-md border p-3">
                            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                              <div>
                                <h4 className="text-sm font-semibold">{page.questionOptions}</h4>
                                <p className="mt-1 text-xs text-muted-foreground">{selectedQuestion ? selectedQuestion.question_text || selectedQuestion.question_id : page.selectQuestionHint}</p>
                              </div>
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => loadOptions()}
                                disabled={!selectedQuestion || optionsLoading}
                              >
                                <RefreshCw className={cn("mr-2 h-4 w-4", optionsLoading && "animate-spin")} />
                                {page.loadOptions}
                              </Button>
                            </div>
                            <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_120px]">
                              <div className="space-y-2">
                                <Label htmlFor="optionText">{page.optionText}</Label>
                                <Input
                                  id="optionText"
                                  value={optionForm.option_text}
                                  disabled={!selectedQuestion || selectedCourse.is_published}
                                  onChange={(event) => setOptionForm({ ...optionForm, option_text: event.target.value })}
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="optionSortOrder">{page.optionSortOrder}</Label>
                                <Input
                                  id="optionSortOrder"
                                  type="number"
                                  min="0"
                                  value={optionForm.sort_order}
                                  disabled={!selectedQuestion || selectedCourse.is_published}
                                  onChange={(event) => setOptionForm({ ...optionForm, sort_order: event.target.value })}
                                />
                              </div>
                            </div>
                            <div className="mt-3 flex flex-wrap gap-3">
                              <label className="flex h-9 items-center gap-2 rounded-md border px-3 text-sm">
                                <Checkbox
                                  checked={optionForm.is_correct}
                                  disabled={!selectedQuestion || selectedCourse.is_published}
                                  onCheckedChange={(checked) => setOptionForm({ ...optionForm, is_correct: Boolean(checked) })}
                                />
                                {page.correctOption}
                              </label>
                              <ProtectedButton size="sm" onClick={createOption} disabled={!selectedQuestion || optionSaving || selectedCourse.is_published} isPublished={selectedCourse.is_published}>
                                <Plus className="mr-2 h-4 w-4" />
                                {page.createOption}
                              </ProtectedButton>
                            </div>

                            <div className="mt-3 overflow-hidden rounded-md border">
                              {options.length === 0 ? (
                                <div className="px-3 py-6 text-center text-sm text-muted-foreground">{page.noOptions}</div>
                              ) : (
                                options.map((option) => (
                                  <div key={option.option_id} className="flex items-center justify-between gap-3 border-b px-3 py-3 text-sm last:border-b-0">
                                    <div className="min-w-0">
                                      <div className="truncate font-medium">{option.option_text || option.option_id}</div>
                                      <div className="truncate text-xs text-muted-foreground">{option.option_id}</div>
                                    </div>
                                    <Badge variant={option.is_correct ? "default" : "outline"}>{option.is_correct ? page.correct : page.incorrect}</Badge>
                                  </div>
                                ))
                              )}
                            </div>
                          </div>
                          </div>
                          </div>
                        </DialogContent>
                      </Dialog>
                    </>
                  )}
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex items-center justify-between border-b px-4 py-3">
                  <h2 className="font-semibold">{page.completePreview}</h2>
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm" onClick={loadCourseDetail} disabled={!selectedCourse || courseDetailLoading}>
                      <RefreshCw className={cn("mr-2 h-4 w-4", courseDetailLoading && "animate-spin")} />
                      {page.loadCourseDetail}
                    </Button>
                    <Button variant="outline" size="sm" onClick={loadCompleteCourse} disabled={!selectedCourse || previewLoading}>
                      {previewLoading ? <RefreshCw className="mr-2 h-4 w-4 animate-spin" /> : <Eye className="mr-2 h-4 w-4" />}
                      {page.loadComplete}
                    </Button>
                  </div>
                </div>
                <div className="p-4">
                  {!selectedCourse ? (
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <BookOpen className="h-4 w-4" />
                      {page.selectCourseHint}
                    </div>
                  ) : preview ? (
                    <pre className="max-h-96 overflow-auto rounded-md bg-muted p-3 text-xs text-muted-foreground">
                      {JSON.stringify(preview, null, 2)}
                    </pre>
                  ) : (
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <CheckCircle2 className="h-4 w-4" />
                      {page.noPreview}
                    </div>
                  )}
                  {courseDetail && (
                    <div className="mt-4 grid gap-3 text-sm md:grid-cols-3">
                      <div className="rounded-md border p-3">
                        <div className="text-xs text-muted-foreground">{page.chapters}</div>
                        <div className="mt-1 text-lg font-semibold">{courseDetail.chapter_count || 0}</div>
                      </div>
                      <div className="rounded-md border p-3">
                        <div className="text-xs text-muted-foreground">{page.lessons}</div>
                        <div className="mt-1 text-lg font-semibold">{courseDetail.lesson_count || 0}</div>
                      </div>
                      <div className="rounded-md border p-3">
                        <div className="text-xs text-muted-foreground">{page.quizzes}</div>
                        <div className="mt-1 text-lg font-semibold">{courseDetail.quiz_count || 0}</div>
                      </div>
                    </div>
                  )}
                </div>
              </div>

              {/* Supplementary Material block */}
              <div className="rounded-lg border bg-card">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
                  <div>
                    <h2 className="font-semibold">{'补充资料 (Supplementary Material)'}</h2>
                    <p className="mt-1 text-xs text-muted-foreground">{'管理当前课程的全局补充资料及结构化 JSON 数据'}</p>
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <Button variant="outline" size="sm" onClick={() => loadSupplementaryMaterial()} disabled={!selectedCourse || supplementaryMaterialLoading}>
                      <RefreshCw className={cn("mr-2 h-4 w-4", supplementaryMaterialLoading && "animate-spin")} />
                      {page.loadMaterials || '加载'}
                    </Button>
                  </div>
                </div>
                <div className="p-4">
                  <div className="mb-4">
                    <Button onClick={() => setSuppMaterialModalOpen(true)} disabled={!selectedCourse}>
                      {page.materialEditMode || "编辑补充资料"}
                    </Button>
                  </div>
                  
                  <Dialog open={suppMaterialModalOpen} onOpenChange={setSuppMaterialModalOpen}>
                    <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
                      <DialogHeader>
                        <DialogTitle>{page.materialEditMode || "编辑补充资料"}</DialogTitle>
                      </DialogHeader>
                      <div className="grid gap-4 md:grid-cols-[340px_minmax(0,1fr)]">
                        <div className="space-y-3">
                          <div className="grid gap-3">
                      <div className="space-y-2">
                        <Label htmlFor="suppKind">{'类型 (Kind)'}</Label>
                        <Input
                          id="suppKind"
                          placeholder="例如 1"
                          value={supplementaryMaterialForm.kind}
                          onChange={(e) => setSupplementaryMaterialForm({ ...supplementaryMaterialForm, kind: e.target.value })}
                          disabled={supplementaryMaterialSaving || !selectedCourse}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="suppDataJson">{'JSON 数据 (Data JSON)'}</Label>
                        <Textarea
                          id="suppDataJson"
                          rows={6}
                          placeholder='{"key": "value"}'
                          value={supplementaryMaterialForm.data_json}
                          onChange={(e) => setSupplementaryMaterialForm({ ...supplementaryMaterialForm, data_json: e.target.value })}
                          disabled={supplementaryMaterialSaving || !selectedCourse}
                        />
                      </div>
                    </div>
                    <div className="flex items-center gap-2 pt-2">
                      <Button onClick={saveSupplementaryMaterial} disabled={supplementaryMaterialSaving || !selectedCourse}>
                        <Save className="mr-2 h-4 w-4" />
                        {t.common.save || '保存'}
                      </Button>
                      {supplementaryMaterial?.material_id && (
                        <Button variant="destructive" onClick={deleteSupplementaryMaterial} disabled={supplementaryMaterialSaving}>
                          <Trash2 className="mr-2 h-4 w-4" />
                          {page.delete || '删除'}
                        </Button>
                      )}
                    </div>
                  </div>
                  <div className="flex flex-col gap-3 rounded-md border p-3">
                     <h3 className="text-sm font-medium">{'当前补充资料数据'}</h3>
                     <pre className="flex-1 overflow-auto rounded bg-muted/50 p-2 text-xs">
                       {supplementaryMaterial ? JSON.stringify(supplementaryMaterial, null, 2) : '暂无数据'}
                     </pre>
                  </div>
                </div>
              </DialogContent>
            </Dialog>

            <div className="mt-4 flex flex-col gap-3 rounded-md border bg-muted/20 p-4">
              <h3 className="text-sm font-medium">{'当前补充资料数据预览'}</h3>
              <pre className="max-h-60 overflow-auto rounded bg-muted/50 p-2 text-xs text-muted-foreground">
                {supplementaryMaterial ? JSON.stringify(supplementaryMaterial, null, 2) : '暂无数据'}
              </pre>
            </div>
          </div>
        </div>

        <div className="rounded-lg border bg-card">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
                  <div>
                    <h2 className="font-semibold">{page.materialsTitle}</h2>
                    <p className="mt-1 text-xs text-muted-foreground">{page.materialsHint}</p>
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <Button variant="outline" size="sm" onClick={() => loadMaterials()} disabled={!selectedCourse || materialsLoading}>
                      <RefreshCw className={cn("mr-2 h-4 w-4", materialsLoading && "animate-spin")} />
                      {page.loadMaterials}
                    </Button>
                  </div>
                </div>
                <div className="p-4">
                  <div className="mb-4">
                    <Button onClick={() => { resetMaterialForm(); setMaterialModalOpen(true); }} disabled={!selectedCourse || selectedCoursePublished}>
                      <Plus className="mr-2 h-4 w-4" />
                      {page.materialCreate || "新增资料"}
                    </Button>
                  </div>
                  
                  <Dialog open={materialModalOpen} onOpenChange={setMaterialModalOpen}>
                    <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
                      <DialogHeader>
                        <DialogTitle>{selectedMaterial ? page.materialEditMode : page.materialCreateMode}</DialogTitle>
                      </DialogHeader>
                      <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_300px]">
                        <div className="space-y-3">
                          <div className="flex flex-wrap items-center justify-between gap-2 rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
                      <span>{selectedMaterial ? page.materialEditMode : page.materialCreateMode}</span>
                      {selectedMaterial && (
                        <span className="truncate">{selectedMaterial.material_id}</span>
                      )}
                    </div>
                    <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_160px_120px]">
                      <div className="space-y-2">
                        <Label htmlFor="materialTitle">{page.materialTitle}</Label>
                        <Input
                          id="materialTitle"
                          value={materialForm.title}
                          disabled={!selectedCourse || selectedCoursePublished}
                          onChange={(event) => setMaterialForm({ ...materialForm, title: event.target.value })}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="materialType">{page.materialType}</Label>
                        <Select
                          value={materialForm.material_type}
                          disabled={!selectedCourse || selectedCoursePublished}
                          onValueChange={(value) => setMaterialForm({ ...materialForm, material_type: value })}
                        >
                          <SelectTrigger id="materialType">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="1">{page.materialTypeTextbook || "教材"}</SelectItem>
                            <SelectItem value="2">{page.materialTypeSlides || "课件"}</SelectItem>
                            <SelectItem value="3">{page.materialTypeReference || "参考资料"}</SelectItem>
                            <SelectItem value="4">{page.materialTypeOther || "其他"}</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="materialSortOrder">{page.materialSortOrder}</Label>
                        <Input
                          id="materialSortOrder"
                          type="number"
                          min="0"
                          value={materialForm.sort_order}
                          disabled={!selectedCourse || selectedCoursePublished}
                          onChange={(event) => setMaterialForm({ ...materialForm, sort_order: event.target.value })}
                        />
                      </div>
                    </div>

                    <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_160px_120px]">
                      <div className="space-y-2">
                        <Label htmlFor="materialFileKey">{page.materialFileKey}</Label>
                        <Input
                          id="materialFileKey"
                          value={materialForm.file_object_key}
                          disabled={!selectedCourse || selectedCoursePublished}
                          onChange={(event) => setMaterialForm({ ...materialForm, file_object_key: event.target.value })}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="materialFileHash">{page.materialFileHash}</Label>
                        <Input
                          id="materialFileHash"
                          value={materialForm.file_hash}
                          disabled={!selectedCourse || selectedCoursePublished}
                          onChange={(event) => setMaterialForm({ ...materialForm, file_hash: event.target.value })}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="materialFileSize">{page.materialFileSize}</Label>
                        <Input
                          id="materialFileSize"
                          type="number"
                          min="0"
                          value={materialForm.file_size}
                          disabled={!selectedCourse || selectedCoursePublished}
                          onChange={(event) => setMaterialForm({ ...materialForm, file_size: event.target.value })}
                        />
                      </div>
                    </div>

                    <div className="flex flex-wrap gap-2">
                      <label className="inline-flex cursor-pointer items-center gap-2 rounded-md border px-3 py-2 text-sm">
                        <UploadCloud className="h-4 w-4" />
                        <span>{page.materialOpenFile}</span>
                        <input
                          className="hidden"
                          type="file"
                          accept={materialUploadAccept}
                          disabled={!selectedCourse || selectedCoursePublished || materialUploading}
                          onChange={(event) => {
                            void uploadMaterialFile(event.target.files?.[0] || null)
                            event.currentTarget.value = ""
                          }}
                        />
                      </label>
                      <ProtectedButton size="sm" onClick={createMaterial} disabled={!selectedCourse || materialSaving || selectedCoursePublished} isPublished={selectedCoursePublished}>
                        <FileText className="mr-2 h-4 w-4" />
                        {selectedMaterial ? page.materialUpdate : page.materialCreate}
                      </ProtectedButton>
                      <Button variant="outline" size="sm" onClick={resetMaterialForm} disabled={!selectedCourse}>
                        {t.common.cancel}
                      </Button>
                      {selectedMaterial && (
                        <ProtectedButton
                          variant="destructive"
                          size="sm"
                          onClick={deleteMaterial}
                          disabled={!selectedCourse || materialSaving || selectedCoursePublished}
                          isPublished={selectedCoursePublished}
                        >
                          <Trash2 className="mr-2 h-4 w-4" />
                          {page.materialDelete}
                        </ProtectedButton>
                      )}
                    </div>

                        </div>
                        <div className="rounded-md border bg-muted/20 p-4">
                          {selectedMaterial ? (
                            <div className="space-y-3">
                              <div className="flex items-center justify-between gap-3">
                                <div>
                                  <h3 className="text-sm font-semibold">{page.materialPreview}</h3>
                                  <p className="mt-1 text-xs text-muted-foreground">{selectedMaterial.title || selectedMaterial.material_id}</p>
                                </div>
                                <Badge variant="outline">{materialTypeLabel(selectedMaterial.material_type)}</Badge>
                              </div>
                              <div className="grid gap-2 text-sm text-muted-foreground">
                                <div>{page.materialFileKey}: {selectedMaterial.file_object_key || t.common.na}</div>
                                <div>{page.materialFileHash}: {selectedMaterial.file_hash || t.common.na}</div>
                                <div>{page.materialFileSize}: {selectedMaterial.file_size || 0}</div>
                                <div>{page.materialSortOrder}: {selectedMaterial.sort_order || 0}</div>
                                <div>{page.version}: {selectedMaterial.version || 0}</div>
                              </div>
                            </div>
                          ) : (
                            <div className="flex min-h-[220px] items-center justify-center text-sm text-muted-foreground">
                              {page.noMaterials}
                            </div>
                          )}
                        </div>
                      </div>
                    </DialogContent>
                  </Dialog>

                  <div className="overflow-hidden rounded-md border">
                    {materialsLoading ? (
                        <div className="px-3 py-6 text-center text-sm text-muted-foreground">{t.common.loading}</div>
                      ) : materials.length === 0 ? (
                        <div className="px-3 py-6 text-center text-sm text-muted-foreground">{page.noMaterials}</div>
                      ) : (
                        materials.map((material) => (
                          <button
                            key={material.material_id}
                            type="button"
                            onClick={() => void loadMaterialForEdit(material)}
                            className={cn(
                              "block w-full border-b px-3 py-3 text-left text-sm last:border-b-0 hover:bg-muted/60",
                              selectedMaterialId === material.material_id && "bg-muted"
                            )}
                          >
                            <div className="flex items-center justify-between gap-3">
                              <span className="truncate font-medium">{material.title || material.material_id}</span>
                              <Badge variant="outline">{materialTypeLabel(material.material_type)}</Badge>
                            </div>
                            <div className="mt-1 truncate text-xs text-muted-foreground">{material.file_object_key || t.common.na}</div>
                            <div className="mt-1 flex items-center justify-between gap-3 text-xs text-muted-foreground">
                              <span className="truncate">{material.material_id}</span>
                              <span>{formatBackendDate(material.updated_at || material.created_at)}</span>
                            </div>
                          </button>
                        ))
                      )}
                    </div>

                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
                  <h2 className="font-semibold">{page.enrollments}</h2>
                  <div className="flex items-center gap-2">
                    <Input
                      className="h-9 w-52"
                      placeholder={page.candidateIdPlaceholder}
                      value={batchEnrollCandidateId}
                      onChange={(event) => setBatchEnrollCandidateId(event.target.value)}
                    />
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={batchEnrollCandidate}
                      disabled={!selectedCourse || batchEnrolling}
                    >
                      {batchEnrolling ? (
                        <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                      ) : (
                        <Users className="mr-2 h-4 w-4" />
                      )}
                      {page.batchEnroll}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={grantCourseAccess}
                      disabled={!selectedCourse || batchEnrolling}
                      className="text-green-600 hover:text-green-700"
                    >
                      <CheckCircle2 className="mr-2 h-4 w-4" />
                      {page.grantPermission || '授权'}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={revokeCourseAccess}
                      disabled={!selectedCourse || batchEnrolling}
                      className="text-orange-600 hover:text-orange-700"
                    >
                      <AlertTriangle className="mr-2 h-4 w-4" />
                      {page.revokePermission || '撤销授权'}
                    </Button>
                    <Select value={enrollmentStatus} onValueChange={setEnrollmentStatus}>
                      <SelectTrigger className="h-9 w-36">
                        <SelectValue placeholder={page.enrollmentStatus} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all">{page.statusAll}</SelectItem>
                        <SelectItem value="learning">{page.statusLearning}</SelectItem>
                        <SelectItem value="completed">{page.statusCompleted}</SelectItem>
                      </SelectContent>
                    </Select>
                    <Button variant="outline" size="sm" onClick={loadCourseEnrollments} disabled={!selectedCourse || enrollmentsLoading}>
                      <RefreshCw className={cn("mr-2 h-4 w-4", enrollmentsLoading && "animate-spin")} />
                      {page.loadEnrollments}
                    </Button>
                  </div>
                </div>
                <div className="p-4">
                  {!selectedCourse ? (
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <Users className="h-4 w-4" />
                      {page.selectCourseForEnrollments}
                    </div>
                  ) : enrollments.length === 0 ? (
                    <div className="text-sm text-muted-foreground">{page.noEnrollments}</div>
                  ) : (
                    <div className="overflow-hidden rounded-md border">
                      {enrollments.map((enrollment) => (
                        <div
                          key={enrollment.enrollment_id}
                          className="grid gap-3 border-b px-3 py-3 text-sm last:border-b-0 xl:grid-cols-[minmax(0,1.4fr)_90px_90px_120px_minmax(0,1fr)]"
                        >
                          <div className="min-w-0">
                            <div className="truncate font-medium">{enrollment.candidate_id}</div>
                            <div className="truncate text-xs text-muted-foreground">{enrollment.enrollment_id}</div>
                          </div>
                          <Badge variant="outline" className={`w-fit ${statusBadgeClassForStatusValue(enrollment.status)}`}>
                            {statusLabel(t, LMS_ENROLLMENT_STATUS_LABELS, enrollment.status)}
                          </Badge>
                          <div className="text-muted-foreground">
                            {enrollment.progress_percentage || 0}%
                          </div>
                          <div className="text-xs text-muted-foreground">{formatBackendDate(enrollment.joined_at)}</div>
                          <div className="flex flex-wrap justify-end gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => loadEnrollmentDetail(enrollment)}
                              disabled={enrollmentDetailLoadingFor === enrollment.enrollment_id}
                            >
                              <Eye className="mr-1 h-4 w-4" />
                              {page.detail}
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => loadCandidateProgress(enrollment.candidate_id)}
                              disabled={progressLoadingFor === enrollment.candidate_id}
                            >
                              {progressLoadingFor === enrollment.candidate_id ? (
                                <RefreshCw className="mr-1 h-4 w-4 animate-spin" />
                              ) : (
                                <Eye className="mr-1 h-4 w-4" />
                              )}
                              {page.viewProgress}
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => syncCandidateProgress(enrollment.candidate_id)}
                              disabled={syncProgressLoadingFor === enrollment.candidate_id}
                            >
                              <RefreshCw className={cn("mr-1 h-4 w-4", syncProgressLoadingFor === enrollment.candidate_id && "animate-spin")} />
                              {page.syncProgress}
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => loadLessonProgressForCandidate(enrollment.candidate_id)}
                              disabled={lessonProgressLoadingFor === enrollment.candidate_id}
                            >
                              <BookOpen className="mr-1 h-4 w-4" />
                              {page.lessonProgress}
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => loadChapterProgressForCandidate(enrollment.candidate_id)}
                              disabled={chapterProgressLoadingFor === enrollment.candidate_id}
                            >
                              <ClipboardList className="mr-1 h-4 w-4" />
                              {page.chapterProgress}
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => loadQuizAttemptsForCandidate(enrollment.candidate_id)}
                              disabled={quizAttemptsLoadingFor === enrollment.candidate_id}
                            >
                              <CheckCircle2 className="mr-1 h-4 w-4" />
                              {page.quizAttempts}
                            </Button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}

                  {enrollmentDetail && (
                    <div className="mt-4 rounded-md bg-muted p-3 text-sm">
                      <div className="mb-2 font-medium">{page.enrollmentDetail}</div>
                      <div className="grid gap-2 text-muted-foreground md:grid-cols-3">
                        <div>{page.candidate}: {enrollmentDetail.candidate_id}</div>
                        <div>{page.status}: {enrollmentDetail.status || t.common.unknown}</div>
                        <div>{page.progress}: {enrollmentDetail.progress_percentage || 0}%</div>
                        <div>{page.completedLessons}: {enrollmentDetail.completed_lessons_count || 0}/{enrollmentDetail.total_lessons || 0}</div>
                        <div>{page.passedQuizzes}: {enrollmentDetail.passed_quizzes_count || 0}/{enrollmentDetail.total_quizzes || 0}</div>
                        <div>{page.completedAt}: {formatBackendDate(enrollmentDetail.completed_at)}</div>
                      </div>
                    </div>
                  )}

                  {progressDetail && (
                    <div className="mt-4 rounded-md bg-muted p-3 text-sm">
                      <div className="mb-2 font-medium">{page.progressDetail}</div>
                      <div className="grid gap-2 text-muted-foreground md:grid-cols-2">
                        <div>{page.candidate}: {progressDetail.candidate_id}</div>
                        <div>{page.status}: {statusLabel(t, LMS_ENROLLMENT_STATUS_LABELS, progressDetail.status)}</div>
                        <div>{page.progress}: {progressDetail.progress_percentage || 0}%</div>
                        <div>{page.completedAt}: {formatBackendDate(progressDetail.completed_at)}</div>
                        <div className="md:col-span-2">
                          {page.completedLessons}: {(progressDetail.completed_lesson_ids || []).join(", ") || t.common.na}
                        </div>
                        <div className="md:col-span-2">
                          {page.passedQuizzes}: {(progressDetail.passed_quiz_ids || []).join(", ") || t.common.na}
                        </div>
                      </div>
                    </div>
                  )}

                  {lessonProgress.length > 0 && (
                    <div className="mt-4 overflow-hidden rounded-md border">
                      <div className="border-b bg-muted px-3 py-2 text-sm font-medium">{page.lessonProgress}</div>
                      {lessonProgress.map((item) => (
                        <div key={`${item.user_id || item.candidate_id}-${item.lesson_id}`} className="grid gap-2 border-b px-3 py-2 text-sm last:border-b-0 md:grid-cols-[minmax(0,1fr)_100px_150px_80px]">
                          <div className="min-w-0">
                            <div className="truncate font-medium">{item.lesson_title || item.lesson_id}</div>
                            <div className="truncate text-xs text-muted-foreground">{item.lesson_id}</div>
                          </div>
                          <Badge variant="outline" className={`w-fit ${statusBadgeClassForStatusValue(item.status)}`}>{statusLabel(t, LMS_LESSON_PROGRESS_STATUS_LABELS, item.status)}</Badge>
                          <div className="text-xs text-muted-foreground">{formatBackendDate(item.completed_at || item.updated_at)}</div>
                          <Button variant="ghost" size="sm" onClick={() => loadLessonProgressDetail(item)}>{page.detail}</Button>
                        </div>
                      ))}
                    </div>
                  )}

                  {lessonProgressDetail && (
                    <pre className="mt-3 max-h-72 overflow-auto rounded-md bg-muted p-3 text-xs text-muted-foreground">
                      {JSON.stringify(lessonProgressDetail, null, 2)}
                    </pre>
                  )}

                  {chapterProgress && (
                    <div className="mt-4 rounded-md bg-muted p-3 text-sm">
                      <div className="mb-2 font-medium">{page.chapterProgress}</div>
                      <div className="grid gap-2 text-muted-foreground md:grid-cols-3">
                        <div>{page.currentChapter}: {chapterProgress.chapter_title || chapterProgress.chapter_id}</div>
                        <div>{page.status}: {statusLabel(t, LMS_CHAPTER_PROGRESS_STATUS_LABELS, chapterProgress.status)}</div>
                        <div>{page.completedLessons}: {chapterProgress.completed_lessons_count || 0}/{chapterProgress.total_lessons || 0}</div>
                        <div>{page.passedQuizzes}: {chapterProgress.passed_quizzes_count || 0}/{chapterProgress.total_quizzes || 0}</div>
                      </div>
                    </div>
                  )}

                  {quizAttempts.length > 0 && (
                    <div className="mt-4 overflow-hidden rounded-md border">
                      <div className="border-b bg-muted px-3 py-2 text-sm font-medium">{page.quizAttempts}</div>
                      {quizAttempts.map((attempt) => (
                        <div key={attempt.attempt_id} className="grid gap-2 border-b px-3 py-2 text-sm last:border-b-0 md:grid-cols-[minmax(0,1fr)_100px_120px_150px_80px]">
                          <div className="min-w-0">
                            <div className="truncate font-medium">{attempt.quiz_title || attempt.quiz_id}</div>
                            <div className="truncate text-xs text-muted-foreground">{attempt.attempt_id}</div>
                          </div>
                          <Badge variant="outline" className={`w-fit ${statusBadgeClassForStatusValue(attempt.status)}`}>{statusLabel(t, LMS_QUIZ_ATTEMPT_STATUS_LABELS, attempt.status)}</Badge>
                          <div>{attempt.score || 0}/{attempt.max_score || 0}</div>
                          <div className="text-xs text-muted-foreground">{formatBackendDate(attempt.completed_at || attempt.started_at)}</div>
                          <Button variant="ghost" size="sm" onClick={() => loadQuizAttemptDetail(attempt)}>{page.detail}</Button>
                        </div>
                      ))}
                    </div>
                  )}

                  {quizAttemptDetail && (
                    <pre className="mt-3 max-h-72 overflow-auto rounded-md bg-muted p-3 text-xs text-muted-foreground">
                      {JSON.stringify(quizAttemptDetail, null, 2)}
                    </pre>
                  )}
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
                  <h2 className="font-semibold">{page.courseAssets}</h2>
                  <div className="flex items-center gap-2">
                    <Select
                      value={assetStatus}
                      onValueChange={(value) => {
                        setAssetStatus(value)
                        setBrokenAssets([])
                        setBrokenAssetsNextPageToken("")
                      }}
                    >
                      <SelectTrigger className="h-9 w-36">
                        <SelectValue placeholder={page.assetStatus} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all">{page.statusAll}</SelectItem>
                        <SelectItem value="active">{page.assetStatusActive}</SelectItem>
                        <SelectItem value="broken">{page.assetStatusBroken}</SelectItem>
                        <SelectItem value="missing">{page.assetStatusMissing}</SelectItem>
                      </SelectContent>
                    </Select>
                    <Select
                      value={brokenAssetType}
                      onValueChange={(value) => {
                        setBrokenAssetType(value)
                        setBrokenAssets([])
                        setBrokenAssetsNextPageToken("")
                      }}
                    >
                      <SelectTrigger className="h-9 w-36">
                        <SelectValue placeholder={page.assetType} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all">{page.assetTypeAll}</SelectItem>
                        <SelectItem value="thumbnail">{page.assetTypeThumbnail}</SelectItem>
                        <SelectItem value="material">{page.assetTypeMaterial}</SelectItem>
                        <SelectItem value="lesson">{page.assetTypeLesson}</SelectItem>
                      </SelectContent>
                    </Select>
                    <Button variant="outline" size="sm" onClick={() => loadBrokenAssets()} disabled={brokenAssetsLoading}>
                      <RefreshCw className={cn("mr-2 h-4 w-4", brokenAssetsLoading && "animate-spin")} />
                      {page.loadCourseAssets}
                    </Button>
                  </div>
                </div>
                <div className="p-4">
                  {brokenAssets.length === 0 ? (
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <AlertTriangle className="h-4 w-4" />
                      {page.noCourseAssets}
                    </div>
                  ) : (
                    <div className="overflow-hidden rounded-md border">
                      {brokenAssets.map((asset) => (
                        <div
                          key={`${asset.object_key}-${asset.asset_type}-${asset.created_at}`}
                          className="grid gap-3 border-b px-3 py-3 text-sm last:border-b-0 xl:grid-cols-[120px_minmax(0,1.3fr)_minmax(0,1fr)_120px_90px]"
                        >
                          <Badge variant="outline" className="w-fit">
                            {asset.asset_type || t.common.unknown}
                          </Badge>
                          <div className="min-w-0">
                            <div className="truncate font-medium">{asset.object_key}</div>
                            <div className="truncate text-xs text-muted-foreground">{asset.error_message || page.noErrorMessage}</div>
                          </div>
                          <div className="min-w-0 text-muted-foreground">
                            <div className="truncate">{describeBrokenAssetOwner(asset)}</div>
                            <div className="truncate text-xs">{asset.associated_id || asset.course_id || asset.chapter_id || asset.lesson_id || asset.material_id || t.common.na}</div>
                          </div>
                          <div className="text-xs text-muted-foreground">{formatBackendDate(asset.updated_at || asset.created_at)}</div>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => loadAssetDetail(asset)}
                            disabled={assetDetailLoadingFor === `${asset.object_key}:${asset.associated_id}`}
                          >
                            <Eye className="mr-1 h-4 w-4" />
                            {page.detail}
                          </Button>
                        </div>
                      ))}
                    </div>
                  )}

                  {assetDetail && (
                    <div className="mt-4 rounded-md bg-muted p-3 text-sm">
                      <div className="mb-2 font-medium">{page.assetDetail}</div>
                      <div className="grid gap-2 text-muted-foreground md:grid-cols-2">
                        <div className="truncate">Object Key: {assetDetail.object_key}</div>
                        <div>{page.assetType}: {assetDetail.asset_type || t.common.unknown}</div>
                        <div>{page.status}: {statusLabel(t, LMS_ASSET_STATUS_LABELS, assetDetail.status)}</div>
                        <div>Associated ID: {assetDetail.associated_id || t.common.unknown}</div>
                        <div className="truncate">SHA-256: {assetDetail.file_hash || t.common.unknown}</div>
                        <div>{formatBackendDate(assetDetail.reconciled_at || assetDetail.updated_at)}</div>
                        <div className="md:col-span-2">{assetDetail.error_message || page.noErrorMessage}</div>
                      </div>
                    </div>
                  )}

                  {brokenAssetsNextPageToken && (
                    <div className="mt-3 flex justify-end">
                      <Button variant="outline" size="sm" onClick={() => loadBrokenAssets(brokenAssetsNextPageToken)} disabled={brokenAssetsLoading}>
                        {page.loadMore}
                      </Button>
                    </div>
                  )}
                </div>
              </div>
            </section>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
