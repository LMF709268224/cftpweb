"use client";

import React, { Suspense, useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { useSearchParams, useRouter } from "next/navigation";
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
} from "lucide-react";
import { toast } from "sonner";

import { apiClient } from "@/lib/apiClient";
import { useTranslation } from "@/lib/useLanguage";
import { Sidebar } from "@/components/sidebar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import {
  CANDIDATE_COURSE_STATUS_LABELS,
  CANDIDATE_PIPELINE_STATUS_LABELS,
  COURSE_UNIT_STATUS_LABELS,
  courseUnitNextStepActionFromStatus,
  STAGE_STATUS_LABELS,
  LMS_LESSON_PROGRESS_STATUS_LABELS,
  LMS_CHAPTER_PROGRESS_STATUS_LABELS,
  LMS_QUIZ_ATTEMPT_STATUS_LABELS,
  stageStatusHintLabel,
  statusLabel,
  timelineStatusBadgeClassForStatus,
  timelineStatusLabelWithDiagnostics,
} from "@cftpweb/shared";

type CourseCompleteResponse = {
  complete_course?: CompleteCourse;
  quiz_progress?: Record<string, QuizProgressItem>;
};

type CompleteCourse = {
  course?: Course;
  chapters?: ChapterDetail[];
  materials?: CourseMaterialSummary[];
  quizzes?: any[];
};

type Course = {
  course_id?: string;
  title?: string;
  description?: string;
  category_tips?: string;
  duration_min?: number;
};

type ChapterDetail = {
  chapter?: {
    chapter_id?: string;
    title?: string;
    sort_order?: number;
  };
  lessons?: LessonDetail[];
  quizzes?: any[];
};

type LessonDetail = {
  lesson?: Lesson;
  quizzes?: any[];
};

type Lesson = {
  lesson_id?: string;
  title?: string;
  lesson_type?: number;
  body?: string;
  external_url?: string;
  media_object_key?: string;
  video_provider?: string;
  video_embed_code?: string;
  video_stream_uid?: string;
};

type CourseMaterialSummary = {
  material_id?: string;
  course_id?: string;
  title?: string;
  material_type?: number;
  file_object_key?: string;
  file_size?: number;
  sort_order?: number;
  file_hash?: string;
};

type LessonView = LessonDetail & {
  chapterTitle: string;
  chapterId: string;
};

type QuizTask = {
  key: string;
  quiz: any;
  quizId: string;
  title: string;
  scope: "course" | "chapter" | "lesson";
  scopeLabel: string;
  ownerTitle: string;
  chapterId?: string;
  chapterTitle?: string;
  lessonId?: string;
  lessonTitle?: string;
  completed?: boolean;
};

type QuizProgressItem = {
  quiz_id?: string;
  is_passed?: boolean;
  status?: string;
  attempt_id?: string;
};

type SyncProgressRsp = {
  success?: boolean;
  course_status?: string;
  progress_percentage?: number;
  completed_lessons_count?: number;
  passed_quizzes_count?: number;
};

type ProgressRecord = {
  candidate_id?: string;
  material_id?: string;
  course_package_id?: string;
  progress_type?: string;
  progress_value?: number;
  recorded_at?: string;
};

type MaterialGroupKey = "all" | "textbook" | "slides" | "reference" | "other";

type PipelineRuntime = {
  instance?: {
    pipeline_ulid?: string;
  };
  pipeline_status?: string | number;
  current_stage_ulid?: string;
  current_stage_name?: string;
  current_stage_status?: string | number;
  current_unit_status?: string | number;
  next_step?: {
    action?: string;
    message?: string;
    stage_name?: string;
    course_id?: string;
    pipeline_status?: string | number;
    status?: string | number;
    allow_retake?: boolean;
  };
};

const courseUnitStatusLabel = (t: any, status?: string | number | null) =>
  timelineStatusLabelWithDiagnostics(t, "COURSE_UNIT", status);

const pipelineIsTerminal = (status?: string | number | null) => {
  const normalized = String(status ?? "").trim();
  return normalized === "3" || normalized === "4";
};

const nextStepDisplayFromAction = (t: any, action?: string) => {
  switch (action) {
    case "continue_learning":
      return {
        action,
        label: t.learning.actionContinueLearning,
        desc: t.learning.nextStepContinueLearningDesc,
      };
    case "wait_candidate":
      return {
        action,
        label: t.learning.actionWaitCandidate,
        desc: t.learning.nextStepWaitCandidateDesc,
      };
    case "signup_exam":
      return {
        action,
        label: t.learning.actionSignupExam,
        desc: t.learning.nextStepGoToExamsDesc,
      };
    case "schedule_exam":
      return {
        action,
        label: t.learning.actionScheduleExam,
        desc: t.learning.nextStepGoToExamsDesc,
      };
    case "view_exam_schedule":
      return {
        action,
        label: t.learning.actionViewExamSchedule,
        desc: t.learning.nextStepGoToExamsDesc,
      };
    case "apply_retake":
      return {
        action,
        label: t.learning.actionApplyRetake,
        desc: t.learning.nextStepGoToExamsDesc,
      };
    case "view_exam_result":
      return {
        action,
        label: t.learning.actionViewExamResult,
        desc: t.learning.nextStepGoToExamsDesc,
      };
    case "view_certificate":
      return {
        action,
        label: t.learning.actionViewCertificate,
        desc: t.learning.nextStepViewCertificateDesc,
      };
    default:
      return {
        action: "",
        label: t.common.unknown,
        desc: t.learning.nextStepDesc,
      };
  }
};

const nextStepDisplay = (
  t: any,
  status?: string | number | null,
  hasNextLesson = false,
  allowRetake = false,
  hasPendingQuizzes = false,
) => {
  const action = courseUnitNextStepActionFromStatus(status, allowRetake);
  switch (action) {
    case "continue_learning":
      if (!hasNextLesson && hasPendingQuizzes) {
        return {
          action: "take_quiz",
          label: t.learning.takeQuiz,
          desc: t.learning.nextStepTakeQuizDesc,
        };
      } else if (!hasNextLesson) {
        return {
          action: "wait_sync",
          label: t.learning.timelineRefresh,
          desc: t.learning.nextStepWaitSyncDesc,
        };
      }
      return {
        action,
        label: t.learning.actionContinueLearning,
        desc: t.learning.nextStepContinueLearningDesc,
      };
    case "signup_exam":
    case "schedule_exam":
    case "view_exam_schedule":
    case "apply_retake":
    case "view_exam_result":
    case "view_certificate":
      return nextStepDisplayFromAction(t, action);
    default:
      return {
        action: "",
        label: t.common.unknown,
        desc: t.learning.nextStepDesc,
      };
  }
};

const formatFileSize = (size?: number) => {
  if (!size || size <= 0) return "";
  if (size < 1024) return `${size} B`;
  const kb = size / 1024;
  if (kb < 1024) return `${kb.toFixed(kb >= 100 ? 0 : 1)} KB`;
  const mb = kb / 1024;
  return `${mb.toFixed(mb >= 100 ? 0 : 1)} MB`;
};

const courseStatusLabel = (t: any, status?: string | number | null) =>
  statusLabel(t, CANDIDATE_COURSE_STATUS_LABELS, status);

const stageStatusLabel = (t: any, status?: string | number | null) =>
  timelineStatusLabelWithDiagnostics(t, "STAGE", status);

const materialGroupKey = (materialType?: number): MaterialGroupKey => {
  switch (materialType) {
    case 1:
      return "textbook";
    case 2:
      return "slides";
    case 3:
      return "reference";
    case 4:
      return "other";
    default:
      return "other";
  }
};

const lessonTypeLabel = (t: any, lessonType?: number) => {
  switch (lessonType) {
    case 1:
      return t.learning.lessonTypeVideo;
    case 2:
      return t.learning.lessonTypeText;
    case 3:
      return t.learning.lessonTypePdf;
    case 4:
      return t.learning.lessonTypeImage;
    case 5:
      return t.learning.lessonTypeAudio;
    case 6:
      return t.learning.lessonTypeFile;
    case 7:
      return t.learning.lessonTypeLink;
    default:
      return t.learning.lessonTypeUnknown;
  }
};

const pipelineStatusLabel = (t: any, status?: string | number | null) =>
  timelineStatusLabelWithDiagnostics(t, "PIPELINE", status);

const materialTypeLabel = (t: any, materialType?: number) => {
  switch (materialType) {
    case 1:
      return t.learning.materialTypeTextbook;
    case 2:
      return t.learning.materialTypeSlides;
    case 3:
      return t.learning.materialTypeReference;
    case 4:
      return t.learning.materialTypeOther;
    default:
      return t.learning.materialTypeUnknown;
  }
};

function QuizTaskCard({
  task,
  index,
  t,
  startQuiz,
  showChapter = false,
}: {
  task: QuizTask;
  index: number;
  t: any;
  startQuiz: (quizId: string) => void;
  showChapter?: boolean;
}) {
  return (
    <div className="rounded-lg border bg-background p-3">
      <div className="mb-2 flex flex-wrap items-center gap-2">
        <Badge variant="outline">{task.scopeLabel}</Badge>
        {showChapter && task.chapterTitle && (
          <Badge variant="outline">
            {t.learning.chapters}: {task.chapterTitle}
          </Badge>
        )}
        {task.lessonTitle && (
          <Badge variant="outline">{task.lessonTitle}</Badge>
        )}
        {task.completed && (
          <Badge
            variant="outline"
            className="border-emerald-200 bg-emerald-50 text-emerald-700"
          >
            <CheckCircle2 className="mr-1 h-3.5 w-3.5" />
            {t.learning.completedTag}
          </Badge>
        )}
      </div>
      <div className="text-sm font-medium text-foreground">
        {index + 1}. {task.title}
      </div>
      <div className="mt-3">
        <Button
          size="sm"
          onClick={() => startQuiz(task.quizId)}
          disabled={!task.quizId || task.completed}
        >
          {task.completed ? t.learning.completedTag : t.learning.takeQuiz}
        </Button>
      </div>
    </div>
  );
}

function LearningContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const courseId = searchParams.get("courseId") || "";
  const pipelineId = searchParams.get("pipelineId") || "";
  const lessonId = searchParams.get("lessonId") || "";
  const { t, lang } = useTranslation();
  const [payload, setPayload] = useState<CourseCompleteResponse | null>(null);
  const [loading, setLoading] = useState(Boolean(courseId));
  const [syncing, setSyncing] = useState(false);
  const [activeLessonId, setActiveLessonId] = useState(lessonId);
  const [activeChapterId, setActiveChapterId] = useState("");
  const [syncState, setSyncState] = useState<SyncProgressRsp | null>(null);
  const [progressRecords, setProgressRecords] = useState<ProgressRecord[]>([]);
  const [selectedMaterialId, setSelectedMaterialId] = useState<string>("");
  const [activeMaterialGroup, setActiveMaterialGroup] =
    useState<MaterialGroupKey>("all");
  const [runtime, setRuntime] = useState<PipelineRuntime | null>(null);
  const [scheduleLoading, setScheduleLoading] = useState(false);
  const [lessonContentExpanded, setLessonContentExpanded] = useState(true);
  const materialsSectionRef = useRef<HTMLDivElement | null>(null);

  const loadCourse = async () => {
    if (!courseId) {
      setPayload(null);
      setLoading(false);
      return;
    }

    setLoading(true);
    try {
      const res = await apiClient(`/api/pipeline/courses/${courseId}/complete`);
      setPayload(res);
      if (!activeLessonId) {
        const firstLesson = res?.complete_course?.chapters
          ?.flatMap((chapter: ChapterDetail) => chapter.lessons || [])
          .find((item: LessonDetail) => item.lesson?.lesson_id);
        setActiveLessonId(firstLesson?.lesson?.lesson_id || "");
      }
      const firstMaterial = res?.complete_course?.materials?.find(
        (item: CourseMaterialSummary) => item.material_id,
      );
      if (!selectedMaterialId && firstMaterial?.material_id) {
        setSelectedMaterialId(firstMaterial.material_id);
      }
    } finally {
      setLoading(false);
    }
  };

  const loadProgress = async () => {
    if (!courseId) {
      setProgressRecords([]);
      return;
    }
    try {
      const res = await apiClient("/api/progress");
      setProgressRecords(res?.records || []);
    } catch {
      setProgressRecords([]);
    }
  };

  const loadRuntime = async () => {
    if (!pipelineId) {
      setRuntime(null);
      return;
    }
    try {
      const res = await apiClient(`/api/mall/pipelines/${pipelineId}/runtime`);
      setRuntime(res);
    } catch {
      setRuntime(null);
    }
  };

  useEffect(() => {
    void loadCourse();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [courseId]);

  useEffect(() => {
    if (courseId) {
      void loadProgress();
      void syncProgress(courseId, false);
    }
    void loadRuntime();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [courseId, pipelineId]);

  const completeCourse = payload?.complete_course;
  const course = completeCourse?.course;
  const chapters = completeCourse?.chapters || [];
  const materials = completeCourse?.materials || [];
  const courseQuizzes = completeCourse?.quizzes || [];

  const lessons = useMemo<LessonView[]>(
    () =>
      chapters.flatMap((chapter, chapterIndex) =>
        (chapter.lessons || []).map((lessonDetail) => ({
          chapterTitle: chapter.chapter?.title || t.learning.chapters,
          chapterId: chapter.chapter?.chapter_id || `chapter-${chapterIndex}`,
          ...lessonDetail,
        })),
      ),
    [chapters, t.learning.chapters],
  );

  useEffect(() => {
    if (!activeLessonId && lessons.length > 0) {
      setActiveLessonId(lessons[0].lesson?.lesson_id || "");
    }
  }, [lessons, activeLessonId]);

  useEffect(() => {
    if (activeChapterId) return;
    if (activeLessonId) {
      const chapterFromLesson = lessons.find(
        (item) => item.lesson?.lesson_id === activeLessonId,
      )?.chapterId;
      if (chapterFromLesson) {
        setActiveChapterId(chapterFromLesson);
        return;
      }
    }
    const firstChapterId =
      chapters[0]?.chapter?.chapter_id ||
      (chapters.length > 0 ? "chapter-0" : "");
    if (firstChapterId) {
      setActiveChapterId(firstChapterId);
    }
  }, [activeChapterId, activeLessonId, chapters, lessons]);

  useEffect(() => {
    if (!selectedMaterialId && materials.length > 0) {
      setSelectedMaterialId(materials[0].material_id || "");
    }
  }, [materials, selectedMaterialId]);

  const activeLesson =
    lessons.find((item) => item.lesson?.lesson_id === activeLessonId) ||
    lessons[0];
  const activeChapter =
    chapters.find(
      (chapter) => chapter.chapter?.chapter_id === activeChapterId,
    ) ||
    chapters.find(
      (chapter) => chapter.chapter?.chapter_id === activeLesson?.chapterId,
    ) ||
    chapters[0];
  const lesson = activeLesson?.lesson;
  const quizProgress = payload?.quiz_progress || {};
  const quizCompleted = (quizId?: string) =>
    Boolean(quizId && quizProgress[quizId]?.is_passed);

  const completedLessonIds = useMemo(
    () =>
      new Set(
        progressRecords
          .map((record) => record.material_id)
          .filter((value): value is string => Boolean(value)),
      ),
    [progressRecords],
  );

  const progressPercentage = syncState?.progress_percentage ?? 0;
  const completedLessonsCount =
    syncState?.completed_lessons_count ?? completedLessonIds.size;
  const passedQuizzesCount = syncState?.passed_quizzes_count ?? 0;
  const nextStep = runtime?.next_step;
  const pipelineStatus = runtime?.pipeline_status;
  const isPipelineTerminal = pipelineIsTerminal(pipelineStatus);
  const currentStageName = runtime?.current_stage_name;
  const currentStageStatus = runtime?.current_stage_status;
  const currentUnitStatus = runtime?.current_unit_status;
  const currentLessonRawCompleted = Boolean(
    lesson?.lesson_id && completedLessonIds.has(lesson.lesson_id),
  );
  const nextUnitStatus = nextStep?.status || currentUnitStatus;

  const totalQuizzesCount = useMemo(() => {
    let count = courseQuizzes.length;
    for (const chapter of chapters) {
      if (chapter.quizzes) count += chapter.quizzes.length;
      for (const lessonDetail of chapter.lessons || []) {
        if (lessonDetail.quizzes) count += lessonDetail.quizzes.length;
      }
    }
    return count;
  }, [chapters, courseQuizzes.length]);

  const quizTasks = useMemo<QuizTask[]>(() => {
    const tasks: QuizTask[] = [];
    courseQuizzes.forEach((quizDetail: any, index: number) => {
      const quiz = quizDetail.quiz || quizDetail || {};
      const quizId = quiz.quiz_id || "";
      tasks.push({
        key: quizId || `course-quiz-${index}`,
        quiz,
        quizId,
        title: quiz.title || `${t.learning.quizPrefix} ${index + 1}`,
        scope: "course",
        scopeLabel: t.learning.quizScopeCourse,
        ownerTitle: course?.title || t.common.unknownCourse,
        completed: quizCompleted(quizId),
      });
    });
    chapters.forEach((chapter, chapterIndex) => {
      const chapterId =
        chapter.chapter?.chapter_id || `chapter-${chapterIndex}`;
      const chapterTitle =
        chapter.chapter?.title ||
        `${t.learning.chapterPrefix} ${chapterIndex + 1}`;
      (chapter.quizzes || []).forEach((quizDetail: any, index: number) => {
        const quiz = quizDetail.quiz || quizDetail || {};
        const quizId = quiz.quiz_id || "";
        tasks.push({
          key: quizId || `chapter-${chapterIndex}-quiz-${index}`,
          quiz,
          quizId,
          title:
            quiz.title ||
            `${chapterTitle} ${t.learning.quizPrefix} ${index + 1}`,
          scope: "chapter",
          scopeLabel: t.learning.quizScopeChapter,
          ownerTitle: chapterTitle,
          chapterId,
          chapterTitle,
          completed: quizCompleted(quizId),
        });
      });
      (chapter.lessons || []).forEach((lessonDetail, lessonIndex) => {
        const lessonTitle =
          lessonDetail.lesson?.title ||
          `${t.learning.unknownLesson} ${lessonIndex + 1}`;
        (lessonDetail.quizzes || []).forEach(
          (quizDetail: any, index: number) => {
            const quiz = quizDetail.quiz || quizDetail || {};
            const quizId = quiz.quiz_id || "";
            tasks.push({
              key:
                quizId || `lesson-${chapterIndex}-${lessonIndex}-quiz-${index}`,
              quiz,
              quizId,
              title:
                quiz.title ||
                `${lessonTitle} ${t.learning.quizPrefix} ${index + 1}`,
              scope: "lesson",
              scopeLabel: t.learning.quizScopeLesson,
              ownerTitle: lessonTitle,
              chapterId,
              chapterTitle,
              lessonId: lessonDetail.lesson?.lesson_id,
              lessonTitle,
              completed: quizCompleted(quizId),
            });
          },
        );
      });
    });
    return tasks;
  }, [
    chapters,
    course?.title,
    courseQuizzes,
    quizProgress,
    t.common.unknownCourse,
    t.learning.chapterPrefix,
    t.learning.quizPrefix,
    t.learning.unknownLesson,
  ]);

  const courseQuizTasks = useMemo(
    () => quizTasks.filter((task) => task.scope === "course"),
    [quizTasks],
  );

  const lessonQuizTasksByLessonId = useMemo(() => {
    const map = new Map<string, QuizTask[]>();
    quizTasks.forEach((task) => {
      if (task.scope !== "lesson" || !task.lessonId) return;
      map.set(task.lessonId, [...(map.get(task.lessonId) || []), task]);
    });
    return map;
  }, [quizTasks]);

  const lessonFullyCompleted = (lessonId?: string) => {
    if (!lessonId || !completedLessonIds.has(lessonId)) return false;
    return (lessonQuizTasksByLessonId.get(lessonId) || []).every(
      (task) => task.completed,
    );
  };

  const lessonHasPendingQuizzes = (lessonId?: string) =>
    Boolean(
      lessonId &&
      (lessonQuizTasksByLessonId.get(lessonId) || []).some(
        (task) => !task.completed,
      ),
    );

  const lessonTasks = useMemo(
    () =>
      lessons.map((item, index) => ({
        key: item.lesson?.lesson_id || `lesson-${index}`,
        lesson: item.lesson,
        chapterTitle: item.chapterTitle,
        completed: lessonFullyCompleted(item.lesson?.lesson_id),
      })),
    [completedLessonIds, lessonQuizTasksByLessonId, lessons],
  );

  const activeChapterLessonTasks = useMemo(() => {
    const chapterId = activeChapter?.chapter?.chapter_id || activeChapterId;
    return lessons
      .filter((item) => item.chapterId === chapterId)
      .map((item, index) => ({
        key: item.lesson?.lesson_id || `chapter-lesson-${index}`,
        lesson: item.lesson,
        chapterTitle: item.chapterTitle,
        completed: lessonFullyCompleted(item.lesson?.lesson_id),
      }));
  }, [
    activeChapter?.chapter?.chapter_id,
    activeChapterId,
    completedLessonIds,
    lessonQuizTasksByLessonId,
    lessons,
  ]);

  const activeChapterQuizTasks = useMemo(() => {
    const chapterId = activeChapter?.chapter?.chapter_id || activeChapterId;
    return quizTasks.filter(
      (task) => task.scope === "chapter" && task.chapterId === chapterId,
    );
  }, [activeChapter?.chapter?.chapter_id, activeChapterId, quizTasks]);

  const activeLessonQuizTasks = useMemo(
    () =>
      activeLessonId ? lessonQuizTasksByLessonId.get(activeLessonId) || [] : [],
    [activeLessonId, lessonQuizTasksByLessonId],
  );

  const currentLessonCompleted =
    currentLessonRawCompleted &&
    activeLessonQuizTasks.every((task) => task.completed);
  const visibleChapterAndLessonQuizTasks = useMemo(
    () => [...activeChapterQuizTasks, ...activeLessonQuizTasks],
    [activeChapterQuizTasks, activeLessonQuizTasks],
  );

  const nextLearningLessonId = useMemo(() => {
    if (!lessons.length) return "";
    for (let index = 0; index < lessons.length; index += 1) {
      const candidate = lessons[index]?.lesson?.lesson_id;
      if (candidate && !lessonFullyCompleted(candidate)) {
        return candidate;
      }
    }
    return "";
  }, [completedLessonIds, lessonQuizTasksByLessonId, lessons]);

  const activeChapterContentCount =
    activeChapterLessonTasks.length + activeChapterQuizTasks.length;
  const activeChapterCompleted =
    activeChapterContentCount > 0 &&
    activeChapterLessonTasks.every((item) => item.completed) &&
    activeChapterQuizTasks.every((task) => task.completed);

  const hasPendingQuizzes = passedQuizzesCount < totalQuizzesCount;

  const nextStepState = useMemo(
    () =>
      nextStep?.action
        ? nextStepDisplayFromAction(t, nextStep.action)
        : nextStepDisplay(
            t,
            nextUnitStatus,
            Boolean(nextLearningLessonId),
            Boolean(nextStep?.allow_retake),
            hasPendingQuizzes,
          ),
    [
      t,
      nextStep?.action,
      nextStep?.allow_retake,
      nextUnitStatus,
      nextLearningLessonId,
      hasPendingQuizzes,
    ],
  );

  const syncProgress = async (targetCourseId = courseId, showToast = false) => {
    if (!targetCourseId) return;
    setSyncing(true);
    try {
      const res = await apiClient(
        `/api/progress/courses/${targetCourseId}/sync`,
        { method: "POST" },
      );
      setSyncState(res);
      if (showToast) {
        toast.success(t.common.success);
      }
    } catch {
      // apiClient already handles localized errors
    } finally {
      setSyncing(false);
    }
  };

  const refreshProgress = async (showToast = false) => {
    await syncProgress(courseId, showToast);
    await loadProgress();
    await loadCourse();
    await loadRuntime();
  };

  const startQuiz = async (quizId: string) => {
    try {
      const res = await apiClient(`/api/quizzes/${quizId}/take`, {
        method: "POST",
      });
      if (res.attempt_id) {
        router.push(`/quizzes?attemptId=${res.attempt_id}`);
      } else {
        toast.error(t.common.error);
      }
    } catch (e) {
      toast.error(t.common.error);
    }
  };

  const handleScheduleExam = async () => {
    const targetPipelineUlid = runtime?.instance?.pipeline_ulid;
    if (!(nextStep as any)?.exam_id || !targetPipelineUlid) return;
    setScheduleLoading(true);
    try {
      const res = await apiClient(
        `/api/exams/${encodeURIComponent((nextStep as any).exam_id)}/schedule-url?pipeline_ulid=${encodeURIComponent(targetPipelineUlid)}&course_ulid=${encodeURIComponent((nextStep as any).course_unit_ulid || "")}&url_type=1`,
      );
      if (res?.url) {
        window.open(res.url, "_blank", "noopener,noreferrer");
      } else {
        toast.error(t.common.error);
      }
    } finally {
      setScheduleLoading(false);
    }
  };

  const openExternalLesson = () => {
    const url = lesson?.external_url?.trim();
    if (!url) {
      toast.error(t.common.error);
      return;
    }
    window.open(url, "_blank", "noopener,noreferrer");
  };

  const markCompleted = async () => {
    if (!lesson?.lesson_id) return;
    if (currentLessonCompleted) {
      toast.success(t.learning.completedTag);
      return;
    }
    if (lessonHasPendingQuizzes(lesson.lesson_id)) {
      toast.warning(t.learning.nextStepTakeQuizDesc);
      return;
    }
    try {
      await apiClient(`/api/pipeline/lessons/${lesson.lesson_id}/complete`, {
        method: "POST",
      });
      toast.success(t.common.success);
      await refreshProgress(false);
      await loadCourse();
    } catch {
      // apiClient already reports a localized error
    }
  };

  const markLessonCompleted = async (lessonId?: string) => {
    if (!lessonId) return;
    if (lessonFullyCompleted(lessonId)) {
      toast.success(t.learning.completedTag);
      return;
    }
    if (lessonHasPendingQuizzes(lessonId)) {
      toast.warning(t.learning.nextStepTakeQuizDesc);
      return;
    }
    try {
      await apiClient(`/api/pipeline/lessons/${lessonId}/complete`, {
        method: "POST",
      });
      toast.success(t.common.success);
      await refreshProgress(false);
      await loadCourse();
    } catch {
      // apiClient already reports a localized error
    }
  };

  const openMaterial = async (material: CourseMaterialSummary) => {
    if (!material.material_id) return;
    try {
      const res = await apiClient(
        `/api/pipeline/materials/${material.material_id}/url`,
      );
      if (res?.url) {
        window.open(res.url, "_blank", "noopener,noreferrer");
      } else {
        toast.error(t.common.error);
      }
    } catch {
      // apiClient already reports a localized error
    }
  };

  const downloadMaterial = async (material: CourseMaterialSummary) => {
    if (!material.material_id) return;
    try {
      const res = await apiClient(
        `/api/pipeline/materials/${material.material_id}/url`,
      );
      if (!res?.url) {
        toast.error(t.common.error);
        return;
      }
      const link = document.createElement("a");
      link.href = res.url;
      link.download = material.title || "material";
      link.rel = "noopener noreferrer";
      document.body.appendChild(link);
      link.click();
      link.remove();
    } catch {
      // apiClient already reports a localized error
    }
  };

  const selectLesson = (lessonId?: string, chapterId?: string) => {
    if (lessonId) {
      setActiveLessonId(lessonId);
    }
    if (chapterId) {
      setActiveMaterialGroup("all");
      if (materials.length > 0) {
        setSelectedMaterialId(
          (current) => current || materials[0].material_id || "",
        );
      }
      window.requestAnimationFrame(() => {
        materialsSectionRef.current?.scrollIntoView({
          behavior: "smooth",
          block: "start",
        });
      });
    }
    void refreshProgress(false);
  };

  const filteredMaterials = useMemo(() => {
    if (activeMaterialGroup === "all") return materials;
    return materials.filter(
      (material) =>
        materialGroupKey(material.material_type) === activeMaterialGroup,
    );
  }, [activeMaterialGroup, materials]);

  const groupedMaterials = useMemo(() => {
    const groups: Array<{
      key: MaterialGroupKey;
      label: string;
      items: CourseMaterialSummary[];
    }> = [
      { key: "textbook", label: t.learning.materialTypeTextbook, items: [] },
      { key: "slides", label: t.learning.materialTypeSlides, items: [] },
      { key: "reference", label: t.learning.materialTypeReference, items: [] },
      { key: "other", label: t.learning.materialTypeOther, items: [] },
    ];

    for (const material of materials) {
      const key = materialGroupKey(material.material_type);
      const target = groups.find((item) => item.key === key);
      target?.items.push(material);
    }

    return groups.filter((item) => item.items.length > 0);
  }, [
    materials,
    t.learning.materialTypeOther,
    t.learning.materialTypeReference,
    t.learning.materialTypeSlides,
    t.learning.materialTypeTextbook,
  ]);

  const selectedMaterial =
    filteredMaterials.find((item) => item.material_id === selectedMaterialId) ||
    materials.find((item) => item.material_id === selectedMaterialId) ||
    filteredMaterials[0] ||
    materials[0];

  useEffect(() => {
    if (!selectedMaterial?.material_id) return;
    setSelectedMaterialId(selectedMaterial.material_id);
  }, [selectedMaterial?.material_id]);

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />

      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          <Link
            href={
              pipelineId
                ? `/courses/detail?id=${encodeURIComponent(pipelineId)}`
                : "/courses"
            }
            className="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="h-4 w-4" />
            {t.learning.backToCourse}
          </Link>

          {loading ? (
            <div className="text-muted-foreground">{t.common.loading}</div>
          ) : !course ? (
            <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">
              {t.common.na}
            </div>
          ) : (
            <div className="grid gap-6 lg:grid-cols-[340px_1fr]">
              <aside className="space-y-4">
                <div className="rounded-2xl border bg-card p-6 shadow-sm">
                  <div className="mb-3 flex flex-wrap gap-2">
                    <Badge className="border-0 bg-primary/10 text-primary">
                      {t.learning.title}
                    </Badge>
                    {course.category_tips && (
                      <Badge variant="outline">{course.category_tips}</Badge>
                    )}
                  </div>
                  <h1 className="text-2xl font-bold text-foreground">
                    {course.title || t.common.unknownCourse}
                  </h1>
                  <p className="mt-2 text-sm text-muted-foreground">
                    {course.description || t.common.na}
                  </p>
                  <div className="mt-4 flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
                    <span className="inline-flex items-center gap-1.5">
                      <BookOpen className="h-4 w-4" />
                      {chapters.length} {t.learning.chapters}
                    </span>
                    <span className="inline-flex items-center gap-1.5">
                      <Clock className="h-4 w-4" />
                      {lessons.length} {t.learning.lessons}
                    </span>
                    <span className="inline-flex items-center gap-1.5 text-primary">
                      <CheckCircle2 className="h-4 w-4" />
                      {progressPercentage}%
                    </span>
                  </div>
                  <div className="mt-4 space-y-3">
                    <div className="flex items-center justify-between text-xs text-muted-foreground">
                      <span>{t.learning.progressLabel}</span>
                      <span>
                        {completedLessonsCount}/{lessons.length}{" "}
                        {t.learning.lessons}
                      </span>
                    </div>
                    <Progress value={progressPercentage} className="h-2.5" />
                    <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                      <Badge variant="outline">
                        {t.learning.completedLessonsBadge}{" "}
                        {completedLessonsCount}
                      </Badge>
                      <Badge variant="outline">
                        {t.learning.passedQuizBadge} {passedQuizzesCount}
                      </Badge>
                      {syncState?.course_status && (
                        <Badge variant="outline">
                          {t.learning.courseStatusLabel}:{" "}
                          {courseStatusLabel(t, syncState.course_status)}
                        </Badge>
                      )}
                    </div>
                  </div>
                  <div className="mt-4 flex flex-wrap gap-2">
                    <Button variant="outline" size="sm" asChild>
                      <Link href="/exams">{t.learning.goToExams}</Link>
                    </Button>
                  </div>

                  <div className="mt-4 rounded-xl border bg-muted/20 p-4">
                    <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-foreground">
                      <Sparkles className="h-4 w-4 text-primary" />
                      {t.learning.statusSummaryTitle}
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {isPipelineTerminal
                        ? nextStepState.desc
                        : stageStatusHintLabel(t, currentStageStatus)}
                    </p>
                    <div className="mt-3 flex flex-wrap gap-2 text-xs">
                      <Badge
                        variant="outline"
                        className={timelineStatusBadgeClassForStatus(
                          "PIPELINE",
                          pipelineStatus,
                        )}
                      >
                        {t.learning.pipelineStatusLabel}:{" "}
                        {pipelineStatusLabel(t, pipelineStatus)}
                      </Badge>
                      {!isPipelineTerminal && currentStageName && (
                        <Badge variant="outline">
                          {t.learning.currentStageNameLabel}: {currentStageName}
                        </Badge>
                      )}
                      {!isPipelineTerminal &&
                        currentStageStatus !== undefined &&
                        currentStageStatus !== "" && (
                          <Badge
                            variant="outline"
                            className={timelineStatusBadgeClassForStatus(
                              "STAGE",
                              currentStageStatus,
                            )}
                          >
                            {t.learning.currentStageStatusLabel}:{" "}
                            {stageStatusLabel(t, currentStageStatus)}
                          </Badge>
                        )}
                      {!isPipelineTerminal &&
                        currentUnitStatus !== undefined &&
                        currentUnitStatus !== "" && (
                          <Badge
                            variant="outline"
                            className={timelineStatusBadgeClassForStatus(
                              "COURSE_UNIT",
                              currentUnitStatus,
                            )}
                          >
                            {t.learning.unitStatusLabel}:{" "}
                            {courseUnitStatusLabel(t, currentUnitStatus)}
                          </Badge>
                        )}
                      {/* <Badge variant="outline">
                        {t.learning.nextStepUnitStatusLabel}: {courseUnitStatusLabel(t, nextUnitStatus)}
                      </Badge> */}
                      <Badge variant="outline">
                        {t.learning.nextStepActionLabel}: {nextStepState.label}
                      </Badge>
                    </div>
                  </div>

                  {(nextStepState.action || nextUnitStatus) && (
                    <div className="mt-4 rounded-xl border border-primary/20 bg-primary/5 p-4">
                      <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-primary">
                        <Sparkles className="h-4 w-4" />
                        {t.learning.nextStepTitle}
                      </div>
                      <div className="text-sm text-muted-foreground">
                        {nextStepState.desc}
                      </div>
                      <div className="mt-3 flex items-center justify-between gap-3">
                        {nextStepState.action === "schedule_exam" ? (
                          <Button
                            size="sm"
                            onClick={handleScheduleExam}
                            disabled={scheduleLoading}
                          >
                            {nextStepState.label}
                            <ArrowRight className="h-4 w-4 ml-1" />
                          </Button>
                        ) : nextStepState.action === "take_quiz" ? (
                          <Button
                            size="sm"
                            onClick={() =>
                              window.scrollTo({
                                top: document.body.scrollHeight,
                                behavior: "smooth",
                              })
                            }
                          >
                            {nextStepState.label}
                            <ArrowRight className="h-4 w-4 ml-1" />
                          </Button>
                        ) : nextStepState.action === "wait_sync" ? (
                          <Button
                            size="sm"
                            onClick={() => void refreshProgress(true)}
                            disabled={syncing}
                          >
                            {syncing ? (
                              <Loader2 className="h-4 w-4 animate-spin mr-1" />
                            ) : null}
                            {nextStepState.label}
                          </Button>
                        ) : (
                          <Button asChild size="sm">
                            <Link
                              href={
                                nextStepState.action === "continue_learning"
                                  ? nextLearningLessonId
                                    ? `/courses/learn?courseId=${encodeURIComponent(nextStep?.course_id || courseId)}&pipelineId=${encodeURIComponent(pipelineId)}&lessonId=${encodeURIComponent(nextLearningLessonId)}`
                                    : `/courses/learn?courseId=${encodeURIComponent(courseId)}&pipelineId=${encodeURIComponent(pipelineId)}`
                                  : nextStepState.action === "view_certificate"
                                    ? "/certificates"
                                    : nextStepState.action === "signup_exam"
                                      ? `/exams/signup?unitId=${encodeURIComponent((nextStep as any)?.course_unit_ulid || "")}&pipelineId=${encodeURIComponent(pipelineId)}&courseId=${encodeURIComponent(courseId)}`
                                      : "/exams"
                              }
                            >
                              {nextStepState.label}
                              <ArrowRight className="h-4 w-4 ml-1" />
                            </Link>
                          </Button>
                        )}
                      </div>
                    </div>
                  )}
                </div>

                <div className="rounded-2xl border bg-card shadow-sm">
                  <div className="border-b px-5 py-4">
                    <h2 className="text-sm font-semibold text-foreground">
                      {t.learning.chapters}
                    </h2>
                  </div>
                  <div className="divide-y">
                    {chapters.map((chapter, chapterIndex) => {
                      const chapterId =
                        chapter.chapter?.chapter_id ||
                        `chapter-${chapterIndex}`;
                      const chapterLessons = lessons.filter(
                        (item) => item.chapterId === chapterId,
                      );
                      const chapterLessonTasks = chapterLessons.map((item) => ({
                        completed: lessonFullyCompleted(item.lesson?.lesson_id),
                      }));
                      const chapterQuizTasks = quizTasks.filter(
                        (task) => task.chapterId === chapterId,
                      );
                      const chapterContentCount =
                        chapterLessonTasks.length + chapterQuizTasks.length;
                      const chapterCompleted =
                        chapterContentCount > 0 &&
                        chapterLessonTasks.every((item) => item.completed) &&
                        chapterQuizTasks.every((task) => task.completed);
                      const currentChapter = chapterId === activeChapterId;
                      return (
                        <div
                          key={chapterId}
                          className={`px-5 py-4 ${currentChapter ? "bg-primary/5" : ""}`}
                        >
                          <button
                            type="button"
                            onClick={() => {
                              setActiveChapterId(chapterId);
                              selectLesson(
                                chapter.lessons?.[0]?.lesson?.lesson_id,
                                chapterId,
                              );
                            }}
                            className="mb-3 flex w-full items-center gap-3 text-left"
                          >
                            <div
                              className={`flex h-8 w-8 items-center justify-center rounded-md text-sm font-semibold ${chapterCompleted ? "bg-emerald-100 text-emerald-700" : "bg-primary/10 text-primary"}`}
                            >
                              {chapterCompleted ? (
                                <CheckCircle2 className="h-4 w-4" />
                              ) : (
                                chapterIndex + 1
                              )}
                            </div>
                            <div className="min-w-0 flex-1">
                              <div className="truncate font-medium text-foreground">
                                {chapter.chapter?.title ||
                                  `${t.learning.chapterPrefix} ${chapterIndex + 1}`}
                              </div>
                              <div className="text-xs text-muted-foreground">
                                {chapter.lessons?.length || 0}{" "}
                                {t.learning.lessons}
                                {chapterQuizTasks.length > 0
                                  ? ` / ${chapterQuizTasks.length} ${t.learning.quizPrefix}`
                                  : ""}
                              </div>
                            </div>
                            <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
                          </button>
                          <div className="space-y-1 pl-11">
                            {(chapter.lessons || []).map((lessonDetail) => {
                              const current =
                                lessonDetail.lesson?.lesson_id ===
                                activeLessonId;
                              const completed = lessonFullyCompleted(
                                lessonDetail.lesson?.lesson_id,
                              );
                              return (
                                <button
                                  key={
                                    lessonDetail.lesson?.lesson_id ||
                                    `${chapterId}-${lessonDetail.lesson?.title}`
                                  }
                                  type="button"
                                  onClick={() => {
                                    setActiveChapterId(chapterId);
                                    setActiveLessonId(
                                      lessonDetail.lesson?.lesson_id || "",
                                    );
                                    setActiveMaterialGroup("all");
                                    if (
                                      materials.length > 0 &&
                                      !selectedMaterialId
                                    ) {
                                      setSelectedMaterialId(
                                        materials[0].material_id || "",
                                      );
                                    }
                                    void refreshProgress(false);
                                  }}
                                  className={`flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm transition-colors ${
                                    current
                                      ? "bg-primary/10 text-primary"
                                      : "hover:bg-muted"
                                  }`}
                                >
                                  <span className="flex min-w-0 items-center gap-2 truncate">
                                    {completed ? (
                                      <CheckCircle2 className="h-3.5 w-3.5 shrink-0 text-emerald-500" />
                                    ) : (
                                      <span className="h-3.5 w-3.5 shrink-0 rounded-full border border-muted-foreground/30" />
                                    )}
                                    <span className="truncate">
                                      {lessonDetail.lesson?.title ||
                                        t.learning.unknownLesson}
                                    </span>
                                  </span>
                                  <span className="flex items-center gap-2">
                                    {current ? (
                                      <ChevronDown className="h-4 w-4" />
                                    ) : (
                                      <ChevronRight className="h-4 w-4" />
                                    )}
                                  </span>
                                </button>
                              );
                            })}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              </aside>

              <section className="space-y-4">
                <div className="flex justify-end">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => void refreshProgress(true)}
                    disabled={syncing}
                  >
                    {syncing ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <RefreshCw className="h-4 w-4" />
                    )}
                    {t.learning.syncProgress}
                  </Button>
                </div>

                {(courseQuizTasks.length > 0 ||
                  visibleChapterAndLessonQuizTasks.length > 0) && (
                  <div className="rounded-2xl border bg-card p-6 shadow-sm">
                    <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                      <div>
                        <div className="mb-2 flex items-center gap-2">
                          <Target className="h-5 w-5 text-primary" />
                          <h2 className="text-xl font-semibold text-foreground">
                            {t.learning.allQuizzesTitle}
                          </h2>
                        </div>
                        {/* <p className="text-sm text-muted-foreground">{t.learning.allQuizzesDesc}</p> */}
                      </div>
                      <Badge variant="outline">
                        {courseQuizTasks.filter((task) => task.completed)
                          .length +
                          visibleChapterAndLessonQuizTasks.filter(
                            (task) => task.completed,
                          ).length}
                        /
                        {courseQuizTasks.length +
                          visibleChapterAndLessonQuizTasks.length}
                      </Badge>
                    </div>

                    <div className="grid gap-4 xl:grid-cols-2">
                      <div className="rounded-xl border bg-primary/5 p-4">
                        <div className="mb-3 flex items-center justify-between gap-3">
                          <h3 className="font-semibold text-foreground">
                            {t.learning.courseQuizzesTitle}
                          </h3>
                          <Badge variant="outline">
                            {
                              courseQuizTasks.filter((task) => task.completed)
                                .length
                            }
                            /{courseQuizTasks.length}
                          </Badge>
                        </div>
                        {courseQuizTasks.length === 0 ? (
                          <div className="rounded-lg border border-dashed bg-background p-4 text-sm text-muted-foreground">
                            {t.learning.noCourseQuizzes}
                          </div>
                        ) : (
                          <div className="space-y-2">
                            {courseQuizTasks.map((task, index) => (
                              <QuizTaskCard
                                key={task.key}
                                task={task}
                                index={index}
                                t={t}
                                startQuiz={startQuiz}
                              />
                            ))}
                          </div>
                        )}
                      </div>

                      <div className="rounded-xl border bg-muted/20 p-4">
                        <div className="mb-3 flex items-center justify-between gap-3">
                          <h3 className="font-semibold text-foreground">
                            {t.learning.chapterQuizzesTitle}
                          </h3>
                          <Badge variant="outline">
                            {
                              visibleChapterAndLessonQuizTasks.filter(
                                (task) => task.completed,
                              ).length
                            }
                            /{visibleChapterAndLessonQuizTasks.length}
                          </Badge>
                        </div>
                        {visibleChapterAndLessonQuizTasks.length === 0 ? (
                          <div className="rounded-lg border border-dashed bg-background p-4 text-sm text-muted-foreground">
                            {t.learning.noChapterQuizzes}
                          </div>
                        ) : (
                          <div className="space-y-2">
                            {visibleChapterAndLessonQuizTasks.map(
                              (task, index) => (
                                <QuizTaskCard
                                  key={task.key}
                                  task={task}
                                  index={index}
                                  t={t}
                                  startQuiz={startQuiz}
                                  showChapter
                                />
                              ),
                            )}
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                )}

                <div className="hidden rounded-2xl border bg-card p-6 shadow-sm">
                  <div className="mb-5 flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div>
                      <div className="mb-2 flex items-center gap-2">
                        <Target className="h-5 w-5 text-primary" />
                        <h2 className="text-xl font-semibold text-foreground">
                          {t.learning.courseTodoTitle}
                        </h2>
                      </div>
                      <p className="text-sm text-muted-foreground">
                        {t.learning.courseTodoDesc}
                      </p>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Badge variant="outline">
                        {materials.length} {t.learning.materialsCountSuffix}
                      </Badge>
                      <Badge variant="outline">
                        {completedLessonsCount}/{lessons.length}{" "}
                        {t.learning.lessons}
                      </Badge>
                      <Badge variant="outline">
                        {passedQuizzesCount}/{totalQuizzesCount}{" "}
                        {t.learning.quizPrefix}
                      </Badge>
                    </div>
                  </div>

                  <div className="grid gap-4 xl:grid-cols-3">
                    <div className="rounded-xl border bg-muted/20 p-4">
                      <div className="mb-3 flex items-center justify-between gap-3">
                        <h3 className="font-semibold text-foreground">
                          {t.learning.todoMaterialsTitle}
                        </h3>
                        <Badge variant="outline">{materials.length}</Badge>
                      </div>
                      {materials.length === 0 ? (
                        <div className="rounded-lg border border-dashed bg-background p-4 text-sm text-muted-foreground">
                          {t.learning.materialsEmpty}
                        </div>
                      ) : (
                        <div className="space-y-2">
                          {materials.map((material) => (
                            <div
                              key={material.material_id || material.title}
                              className="rounded-lg border bg-background p-3"
                            >
                              <div className="mb-2 flex flex-wrap items-center gap-2">
                                <Badge variant="outline">
                                  {materialTypeLabel(t, material.material_type)}
                                </Badge>
                                <span className="text-xs text-muted-foreground">
                                  {formatFileSize(material.file_size) ||
                                    t.learning.materialSizeUnknown}
                                </span>
                              </div>
                              <div className="line-clamp-2 text-sm font-medium text-foreground">
                                {material.title || t.learning.unknownMaterial}
                              </div>
                              <div className="mt-3 flex flex-wrap gap-2">
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => void openMaterial(material)}
                                >
                                  <Play className="h-3.5 w-3.5" />
                                  {t.learning.openMaterial}
                                </Button>
                                <Button
                                  size="sm"
                                  variant="ghost"
                                  onClick={() =>
                                    void downloadMaterial(material)
                                  }
                                >
                                  <Download className="h-3.5 w-3.5" />
                                  {t.learning.downloadMaterial}
                                </Button>
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>

                    <div className="rounded-xl border bg-muted/20 p-4">
                      <div className="mb-3 flex items-center justify-between gap-3">
                        <h3 className="font-semibold text-foreground">
                          {t.learning.todoLessonsTitle}
                        </h3>
                        <Badge variant="outline">
                          {completedLessonsCount}/{lessons.length}
                        </Badge>
                      </div>
                      {lessonTasks.length === 0 ? (
                        <div className="rounded-lg border border-dashed bg-background p-4 text-sm text-muted-foreground">
                          {t.learning.unknownLesson}
                        </div>
                      ) : (
                        <div className="space-y-2">
                          {lessonTasks.map((item, index) => (
                            <div
                              key={item.key}
                              className={`rounded-lg border bg-background p-3 ${item.lesson?.lesson_id === activeLessonId ? "border-primary/40 ring-1 ring-primary/10" : ""}`}
                            >
                              <div className="mb-2 flex flex-wrap items-center gap-2">
                                <Badge variant="outline">
                                  {item.chapterTitle}
                                </Badge>
                                <Badge
                                  variant="outline"
                                  className={
                                    item.completed
                                      ? "border-emerald-200 bg-emerald-50 text-emerald-700"
                                      : ""
                                  }
                                >
                                  {item.completed
                                    ? t.learning.completedTag
                                    : t.learning.toStudy}
                                </Badge>
                              </div>
                              <div className="text-sm font-medium text-foreground">
                                {index + 1}.{" "}
                                {item.lesson?.title || t.learning.unknownLesson}
                              </div>
                              <div className="mt-3 flex flex-wrap gap-2">
                                <Button
                                  size="sm"
                                  variant={
                                    item.lesson?.lesson_id === activeLessonId
                                      ? "default"
                                      : "outline"
                                  }
                                  onClick={() => {
                                    setActiveLessonId(
                                      item.lesson?.lesson_id || "",
                                    );
                                    window.requestAnimationFrame(() => {
                                      document
                                        .getElementById("lesson-detail")
                                        ?.scrollIntoView({
                                          behavior: "smooth",
                                          block: "start",
                                        });
                                    });
                                  }}
                                >
                                  {t.learning.viewLesson}
                                </Button>
                                <Button
                                  size="sm"
                                  variant={
                                    item.completed ? "outline" : "default"
                                  }
                                  className={
                                    item.completed
                                      ? "border-emerald-200 bg-emerald-50 text-emerald-700 disabled:opacity-100"
                                      : "shadow-sm shadow-primary/20"
                                  }
                                  disabled={item.completed}
                                  onClick={() =>
                                    void markLessonCompleted(
                                      item.lesson?.lesson_id,
                                    )
                                  }
                                >
                                  <CheckCircle2 className="h-3.5 w-3.5" />
                                  {item.completed
                                    ? t.learning.completedTag
                                    : t.learning.completeLesson}
                                </Button>
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>

                    <div className="rounded-xl border bg-primary/5 p-4">
                      <div className="mb-3 flex items-center justify-between gap-3">
                        <h3 className="font-semibold text-foreground">
                          {t.learning.todoQuizzesTitle}
                        </h3>
                        <Badge variant="outline">
                          {passedQuizzesCount}/{quizTasks.length}
                        </Badge>
                      </div>
                      {quizTasks.length === 0 ? (
                        <div className="rounded-lg border border-dashed bg-background p-4 text-sm text-muted-foreground">
                          {t.learning.noQuizzes}
                        </div>
                      ) : (
                        <div className="space-y-2">
                          {quizTasks.map((task, index) => (
                            <div
                              key={task.key}
                              className="rounded-lg border bg-background p-3"
                            >
                              <div className="mb-2 flex flex-wrap items-center gap-2">
                                <Badge variant="outline">
                                  {task.scopeLabel}
                                </Badge>
                                <Badge variant="outline">
                                  {t.learning.quizBelongsTo}: {task.ownerTitle}
                                </Badge>
                              </div>
                              <div className="text-sm font-medium text-foreground">
                                {index + 1}. {task.title}
                              </div>
                              {(task.chapterTitle || task.lessonTitle) && (
                                <div className="mt-1 text-xs text-muted-foreground">
                                  {task.chapterTitle
                                    ? `${t.learning.chapters}: ${task.chapterTitle}`
                                    : ""}
                                  {task.lessonTitle
                                    ? ` · ${t.learning.lessons}: ${task.lessonTitle}`
                                    : ""}
                                </div>
                              )}
                              <div className="mt-3">
                                <Button
                                  size="sm"
                                  onClick={() => startQuiz(task.quizId)}
                                  disabled={!task.quizId}
                                >
                                  {t.learning.takeQuiz}
                                </Button>
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                </div>

                <div
                  id="lesson-detail"
                  className="rounded-2xl border bg-card p-6 shadow-sm"
                >
                  <div className="grid gap-4 lg:grid-cols-[1fr_auto_1fr] lg:items-start">
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge className="border-0 bg-primary/10 text-primary">
                        {lessonTypeLabel(t, lesson?.lesson_type)}
                      </Badge>
                      {activeLesson?.chapterTitle && (
                        <Badge variant="outline">
                          {activeLesson.chapterTitle}
                        </Badge>
                      )}
                    </div>
                    <h2 className="text-center text-2xl font-bold text-foreground">
                      {lesson?.title || t.common.unknownCourse}
                    </h2>
                    <div className="flex justify-start gap-2 lg:justify-end">
                      <Button
                        variant={currentLessonCompleted ? "outline" : "default"}
                        className={
                          currentLessonCompleted
                            ? "border-emerald-200 bg-emerald-50 text-emerald-700 disabled:opacity-100"
                            : "shadow-md shadow-primary/20"
                        }
                        onClick={markCompleted}
                        disabled={currentLessonCompleted}
                      >
                        <CheckCircle2 className="h-4 w-4" />
                        {currentLessonCompleted
                          ? t.learning.completedTag
                          : t.learning.completeLesson}
                      </Button>
                    </div>
                  </div>

                  <div className="mt-5 border-t pt-4">
                    <button
                      type="button"
                      onClick={() =>
                        setLessonContentExpanded((expanded) => !expanded)
                      }
                      className="flex w-full items-center justify-between rounded-lg px-2 py-2 text-left text-sm font-semibold text-foreground transition-colors hover:bg-muted/60"
                    >
                      <span>{t.learning.lessonContentTitle}</span>
                      {lessonContentExpanded ? (
                        <ChevronDown className="h-4 w-4 text-muted-foreground" />
                      ) : (
                        <ChevronRight className="h-4 w-4 text-muted-foreground" />
                      )}
                    </button>

                    {lessonContentExpanded && (
                      <div className="mt-3">
                        {lesson?.video_embed_code ? (
                          <div
                            className="overflow-hidden rounded-xl border bg-muted"
                            dangerouslySetInnerHTML={{
                              __html: lesson.video_embed_code,
                            }}
                          />
                        ) : lesson?.external_url ? (
                          <div className="space-y-4">
                            <div className="rounded-xl border bg-muted/30 p-4 text-sm text-muted-foreground">
                              {t.learning.noLessonBody}
                            </div>
                            <Button onClick={openExternalLesson}>
                              <ExternalLink className="h-4 w-4" />
                              {t.learning.openExternalLesson}
                            </Button>
                          </div>
                        ) : (
                          <div className="prose max-w-none text-sm text-foreground">
                            {lesson?.body ? (
                              <div
                                dangerouslySetInnerHTML={{
                                  __html: lesson.body,
                                }}
                              />
                            ) : (
                              <div className="rounded-xl border bg-muted/30 p-4 text-muted-foreground">
                                {t.learning.noLessonBody}
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                </div>

                {/* Lesson Quizzes */}
                {/*activeLesson?.lesson?.lesson_id && activeLesson.quizzes && activeLesson.quizzes.length > 0 && (
                  <div className="rounded-2xl border bg-card p-6 shadow-sm">
                    <h3 className="mb-4 text-lg font-semibold text-foreground">{t.learning.lessonQuizzes}</h3>
                    <div className="space-y-3">
                      {activeLesson.quizzes.map((quizDetail: any, index: number) => {
                        const quiz = quizDetail.quiz || {}
                        const quizId = quiz.quiz_id || ""
                        return (
                          <div key={quizId || index} className="flex flex-wrap items-center justify-between gap-4 rounded-xl border px-4 py-3 text-sm transition-colors hover:bg-muted/50">
                            <div className="font-medium text-foreground">
                              {quiz.title || `${t.learning.quizPrefix} ${index + 1}`}
                            </div>
                            <Button size="sm" onClick={() => startQuiz(quizId)} disabled={!quizId}>
                              {t.learning.takeQuiz}
                            </Button>
                          </div>
                        )
                      })}
                    </div>
                  </div>
                )*/}

                {/* Chapter Quizzes */}
                {/*activeChapter?.chapter?.chapter_id && activeChapter.quizzes && activeChapter.quizzes.length > 0 && (
                  <div className="rounded-2xl border border-primary/20 bg-primary/5 p-6 shadow-sm mt-6">
                    <div className="mb-4 flex items-center gap-2">
                      <Sparkles className="h-5 w-5 text-primary" />
                      <h3 className="text-lg font-semibold text-primary">{t.learning.lessonQuizzes} (章节课检)</h3>
                    </div>
                    <div className="space-y-3">
                      {activeChapter.quizzes.map((quizDetail: any, index: number) => {
                        const quiz = quizDetail.quiz || {}
                        const quizId = quiz.quiz_id || ""
                        return (
                          <div key={quizId || index} className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-primary/10 bg-background px-4 py-3 text-sm transition-colors hover:border-primary/30 shadow-sm">
                            <div className="font-medium text-foreground">
                              {quiz.title || `Chapter Quiz ${index + 1}`}
                            </div>
                            <Button size="sm" onClick={() => startQuiz(quizId)} disabled={!quizId}>
                              {t.learning.takeQuiz}
                            </Button>
                          </div>
                        )
                      })}
                    </div>
                  </div>
                )*/}

                <div
                  ref={materialsSectionRef}
                  className="rounded-2xl border bg-card p-6 shadow-sm"
                >
                  <div className="mb-4 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                    <div>
                      <div className="mb-2 flex items-center gap-2">
                        <Sparkles className="h-4 w-4 text-primary" />
                        <h3 className="text-lg font-semibold text-foreground">
                          {t.learning.materialsTitle}
                        </h3>
                      </div>
                      <p className="text-sm text-muted-foreground">
                        {t.learning.materialsDesc}
                      </p>
                    </div>
                    <Badge variant="outline">
                      {materials.length} {t.learning.materialsCountSuffix}
                    </Badge>
                  </div>

                  {materials.length === 0 ? (
                    <div className="rounded-xl border bg-muted/30 p-6 text-center text-sm text-muted-foreground">
                      {t.learning.materialsEmpty}
                      <div className="mt-2 text-xs text-muted-foreground">
                        {t.learning.materialsEmptyHint}
                      </div>
                    </div>
                  ) : (
                    <div className="grid gap-4 xl:grid-cols-[240px_1fr]">
                      <div className="rounded-xl border bg-muted/20 p-3">
                        <div className="mb-3 flex flex-wrap gap-2">
                          <Button
                            size="sm"
                            variant={
                              activeMaterialGroup === "all"
                                ? "default"
                                : "outline"
                            }
                            onClick={() => setActiveMaterialGroup("all")}
                          >
                            {t.learning.materialGroupAll}
                          </Button>
                          {groupedMaterials.map((group) => (
                            <Button
                              key={group.key}
                              size="sm"
                              variant={
                                activeMaterialGroup === group.key
                                  ? "default"
                                  : "outline"
                              }
                              onClick={() => setActiveMaterialGroup(group.key)}
                            >
                              {group.label}
                            </Button>
                          ))}
                        </div>

                        <div className="space-y-2">
                          {filteredMaterials.map((material) => {
                            const active =
                              material.material_id === selectedMaterialId;
                            return (
                              <button
                                key={material.material_id || material.title}
                                type="button"
                                onClick={() =>
                                  setSelectedMaterialId(
                                    material.material_id || "",
                                  )
                                }
                                className={`w-full rounded-lg border px-3 py-3 text-left transition-colors ${
                                  active
                                    ? "border-primary bg-primary/5"
                                    : "bg-background hover:bg-muted/60"
                                }`}
                              >
                                <div className="mb-1 flex items-center gap-2">
                                  <Badge variant="outline">
                                    {materialTypeLabel(
                                      t,
                                      material.material_type,
                                    )}
                                  </Badge>
                                </div>
                                <div className="line-clamp-2 text-sm font-medium text-foreground">
                                  {material.title || t.learning.unknownMaterial}
                                </div>
                                <div className="mt-1 text-xs text-muted-foreground">
                                  {formatFileSize(material.file_size) ||
                                    t.learning.materialSizeUnknown}
                                </div>
                              </button>
                            );
                          })}
                        </div>
                      </div>

                      <div className="rounded-xl border bg-background p-5">
                        {selectedMaterial ? (
                          <div className="space-y-5">
                            <div className="flex flex-wrap items-center gap-2">
                              <Badge variant="outline">
                                {materialTypeLabel(
                                  t,
                                  selectedMaterial.material_type,
                                )}
                              </Badge>
                              {selectedMaterial.sort_order !== undefined && (
                                <Badge variant="outline">
                                  {t.learning.materialSortOrder}{" "}
                                  {selectedMaterial.sort_order}
                                </Badge>
                              )}
                            </div>
                            <div>
                              <h4 className="text-xl font-semibold text-foreground">
                                {selectedMaterial.title ||
                                  t.learning.unknownMaterial}
                              </h4>
                              <p className="mt-2 text-sm text-muted-foreground">
                                {selectedMaterial.file_object_key ||
                                  t.learning.materialFileKeyUnknown}
                              </p>
                            </div>

                            <div className="grid gap-3 sm:grid-cols-2">
                              <div className="rounded-lg border bg-muted/20 p-4">
                                <div className="text-xs text-muted-foreground">
                                  {t.learning.materialSizeLabel}
                                </div>
                                <div className="mt-1 text-sm font-medium text-foreground">
                                  {formatFileSize(selectedMaterial.file_size) ||
                                    t.learning.materialSizeUnknown}
                                </div>
                              </div>
                              <div className="rounded-lg border bg-muted/20 p-4">
                                <div className="text-xs text-muted-foreground">
                                  {t.learning.materialHashLabel}
                                </div>
                                <div className="mt-1 break-all text-sm font-medium text-foreground">
                                  {selectedMaterial.file_hash ||
                                    t.learning.materialHashUnknown}
                                </div>
                              </div>
                            </div>

                            <div className="rounded-xl border bg-muted/20 p-4">
                              <div className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                                {t.learning.materialPreviewLabel}
                              </div>
                              <div className="flex min-h-[240px] items-center justify-center rounded-lg border border-dashed bg-background p-6 text-center text-sm text-muted-foreground">
                                <div className="space-y-3">
                                  {selectedMaterial.material_type === 1 ||
                                  selectedMaterial.material_type === 2 ? (
                                    <FileText className="mx-auto h-10 w-10 text-primary" />
                                  ) : selectedMaterial.material_type === 3 ? (
                                    <BookOpen className="mx-auto h-10 w-10 text-primary" />
                                  ) : (
                                    <Video className="mx-auto h-10 w-10 text-primary" />
                                  )}
                                  <div>{t.learning.materialPreviewHint}</div>
                                </div>
                              </div>
                            </div>

                            <div className="flex flex-wrap gap-2">
                              <Button
                                onClick={() =>
                                  void openMaterial(selectedMaterial)
                                }
                              >
                                <Play className="h-4 w-4" />
                                {t.learning.openMaterial}
                              </Button>
                              <Button
                                variant="outline"
                                onClick={() =>
                                  void downloadMaterial(selectedMaterial)
                                }
                              >
                                <Download className="h-4 w-4" />
                                {t.learning.downloadMaterial}
                              </Button>
                            </div>
                          </div>
                        ) : (
                          <div className="flex min-h-[320px] items-center justify-center rounded-xl border border-dashed bg-muted/20 p-8 text-sm text-muted-foreground">
                            {t.learning.materialPreviewEmpty}
                          </div>
                        )}
                      </div>
                    </div>
                  )}
                </div>
              </section>
            </div>
          )}
        </div>
      </main>
    </div>
  );
}

export default function CourseLearnPage() {
  return (
    <Suspense fallback={null}>
      <LearningContent />
    </Suspense>
  );
}
