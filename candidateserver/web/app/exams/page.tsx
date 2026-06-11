"use client"

import React, { useEffect, useMemo, useState } from "react"
import Link from "next/link"
import { useSearchParams } from "next/navigation"
import { toast } from "sonner"
import { useTranslation } from "@/lib/useLanguage"
import { Sidebar } from "@/components/sidebar"
import { apiClient } from "@/lib/apiClient"
import { cn, formatBackendDate } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { EXAM_STATUS_LABELS, normalizeEnumValueUpper, statusBadgeClassForStatusValue, statusLabel } from "@cftpweb/shared"
import {
  AlertCircle,
  CalendarClock,
  CheckCircle2,
  ClipboardList,
  ExternalLink,
  Filter,
  History,
  Loader2,
  Search,
  ShieldCheck,
} from "lucide-react"

type ExamItem = {
  exam_id?: string
  program_code?: string
  exam_code?: string
  exam_status?: string
  result_status?: string
  total_score?: number
  is_passed?: boolean
  candidate_first_name?: string
  candidate_last_name?: string
  candidate_email?: string
  confirmation_number?: string
  appointment_start_time?: string
  appointment_end_time?: string
  site_name?: string
}

type TabId = "current" | "history" | "exemption" | "records"

const examStatusLabel = (t: any, status?: string | number | null) =>
  statusLabel(t, EXAM_STATUS_LABELS, normalizeEnumValueUpper(status))

const normalizedExamStatus = (status?: string | number | null) => normalizeEnumValueUpper(status)

const shouldShowExamStatus = (status?: string | number | null) => {
  const normalized = normalizedExamStatus(status)
  return Boolean(normalized && !["NONE", "UNKNOWN", "UNSPECIFIED"].some((item) => normalized.includes(item)))
}

const hasExamResult = (exam: ExamItem) => {
  const normalized = normalizedExamStatus(exam.result_status)
  return (
    typeof exam.total_score === "number" ||
    exam.is_passed === true ||
    normalized === "DONE" ||
    normalized === "PASSED" ||
    normalized === "FAILED" ||
    normalized === "RESULT_STATUS_PASSED" ||
    normalized === "RESULT_STATUS_FAILED"
  )
}

const hasText = (value?: string | null) => Boolean(value?.trim())

const hasAppointmentDetails = (exam: ExamItem) =>
  hasText(exam.confirmation_number) ||
  hasText(exam.site_name) ||
  hasText(exam.appointment_start_time) ||
  hasText(exam.appointment_end_time)

const canScheduleExam = (exam: ExamItem) => {
  const status = normalizedExamStatus(exam.exam_status)
  return Boolean(exam.exam_id && status && status.includes("OPEN"))
}

const tabs: Array<{ id: TabId; icon: React.ComponentType<{ className?: string }>; labelKey: keyof any }> = [
  { id: "current", icon: CalendarClock, labelKey: "currentTab" },
  { id: "history", icon: History, labelKey: "historyTab" },
  { id: "exemption", icon: ShieldCheck, labelKey: "exemptionTab" },
  { id: "records", icon: ClipboardList, labelKey: "recordsTab" },
]

const emptyCopy: Record<TabId, { titleKey: string; descriptionKey: string; icon: React.ComponentType<{ className?: string }> }> = {
  current: { titleKey: "noExams", descriptionKey: "noExamsDesc", icon: AlertCircle },
  history: { titleKey: "noHistory", descriptionKey: "noHistoryDesc", icon: History },
  exemption: { titleKey: "noExemption", descriptionKey: "noExemptionDesc", icon: ShieldCheck },
  records: { titleKey: "noRecords", descriptionKey: "noRecordsDesc", icon: ClipboardList },
}

export default function ExamsPage() {
  const { t } = useTranslation()
  const searchParams = useSearchParams()
  const [activeTab, setActiveTab] = useState<TabId>("current")
  const [loading, setLoading] = useState(false)
  const [scheduleLoadingExamId, setScheduleLoadingExamId] = useState<string | null>(null)
  const [search, setSearch] = useState("")
  const [exams, setExams] = useState<ExamItem[]>([])
  const [total, setTotal] = useState(0)

  useEffect(() => {
    if (searchParams.get("schedule_return") === "1") {
      toast.success(t.examsPage.scheduleReturnToast)
    }
  }, [searchParams, t.examsPage.scheduleReturnToast])

  const loadExams = async (tab = activeTab, keyword = search) => {
    if (tab === "exemption" || tab === "records") {
      setExams([])
      setTotal(0)
      return
    }

    setLoading(true)
    try {
      const params = new URLSearchParams()
      params.set("page", "1")
      params.set("page_size", "50")
      if (tab === "history") {
        params.set("result_status", "DONE")
      }
      if (keyword.trim()) {
        params.set("confirmation_number", keyword.trim())
      }
      const res = await apiClient(`/api/exams?${params.toString()}`)
      setExams(res?.exams || [])
      setTotal(res?.total || 0)
    } catch {
      setExams([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }

  const handleScheduleExam = async (exam: ExamItem) => {
    if (!exam.exam_id || scheduleLoadingExamId) return
    setScheduleLoadingExamId(exam.exam_id)
    try {
      const termUrlBase = window.location.origin + `/api/exams/${encodeURIComponent(exam.exam_id)}/schedule-callback`
      const params = new URLSearchParams({
        url_type: "schd",
        term_url_base: termUrlBase,
      })
      const res = await apiClient(`/api/exams/${encodeURIComponent(exam.exam_id)}/schedule-url?${params.toString()}`)
      if (res?.url) {
        toast.info(t.examsPage.scheduleRedirecting)
        window.open(res.url, "_blank", "noopener,noreferrer")
      } else {
        toast.error(t.examsPage.scheduleURLMissing)
      }
    } catch {
      toast.error(t.examsPage.scheduleFailed)
    } finally {
      setScheduleLoadingExamId(null)
    }
  }

  useEffect(() => {
    void loadExams()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab])

  const filtered = useMemo(
    () =>
      exams.filter((exam) => {
        const text = [
          exam.exam_id,
          exam.program_code,
          exam.exam_code,
          exam.exam_status,
          exam.result_status,
          exam.confirmation_number,
          exam.site_name,
        ]
          .filter(Boolean)
          .join(" ")
          .toLowerCase()
        return text.includes(search.toLowerCase())
      }),
    [exams, search]
  )

  const state = emptyCopy[activeTab]
  const Icon = state.icon

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          <div className="mb-8 flex items-start justify-between gap-4">
            <div>
              <h1 className="text-3xl font-bold tracking-tight text-foreground">{t.examsPage.title}</h1>
              <p className="mt-1 text-muted-foreground">{t.examsPage.subtitle}</p>
            </div>
            <Button variant="outline" className="gap-2" asChild>
              <Link href="/courses">
                {t.courses.browseCoursesBtn}
                <ExternalLink className="h-4 w-4" />
              </Link>
            </Button>
          </div>

          <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="relative max-w-md flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder={t.examsPage.searchPlaceholder}
                className="pl-10"
              />
            </div>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" className="gap-2" onClick={() => void loadExams()}>
                <Filter className="h-4 w-4" />
                {t.examsPage.refresh}
              </Button>
            </div>
          </div>

          <div className="mb-8 flex w-fit gap-1 overflow-x-auto rounded-xl bg-muted p-1">
            {tabs.map((tab) => {
              const TabIcon = tab.icon
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={cn(
                    "inline-flex items-center gap-2 whitespace-nowrap rounded-lg px-4 py-2 text-sm font-medium transition-all duration-200",
                    activeTab === tab.id ? "bg-card text-card-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"
                  )}
                >
                  <TabIcon className="h-4 w-4" />
                  {t.examsPage[tab.labelKey as keyof typeof t.examsPage] as string}
                </button>
              )
            })}
          </div>

          <div className="rounded-2xl border border-border bg-card p-6 shadow-sm">
            {loading ? (
              <div className="py-16 text-center text-muted-foreground">{t.common.loading}</div>
            ) : filtered.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 text-center">
                <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-muted">
                  <Icon className="h-8 w-8 text-muted-foreground" />
                </div>
                <h3 className="mb-2 text-lg font-semibold text-foreground">
                  {t.examsPage[state.titleKey as keyof typeof t.examsPage] as string}
                </h3>
                <p className="max-w-md text-muted-foreground">
                  {t.examsPage[state.descriptionKey as keyof typeof t.examsPage] as string}
                </p>
              </div>
            ) : (
              <div className="space-y-4">
                <div className="flex items-center justify-between text-sm text-muted-foreground">
                  <span>{t.examsPage.countPrefix} {total > 0 ? total : filtered.length} {t.examsPage.countSuffix}</span>
                  <span>{activeTab === "history" ? t.examsPage.historyFilterHint : t.examsPage.visibleRecordsHint}</span>
                </div>
                <div className="grid gap-4">
                  {filtered.map((exam) => (
                    <div key={exam.exam_id} className="rounded-xl border bg-background p-5 transition-shadow hover:shadow-md">
                      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                        <div className="space-y-2">
                          <div className="flex flex-wrap items-center gap-2">
                            {shouldShowExamStatus(exam.exam_status) && (
                              <Badge variant="outline" className={statusBadgeClassForStatusValue(exam.exam_status)}>{examStatusLabel(t, exam.exam_status)}</Badge>
                            )}
                            {shouldShowExamStatus(exam.result_status) && (
                              <Badge variant="outline" className={statusBadgeClassForStatusValue(exam.result_status)}>{examStatusLabel(t, exam.result_status)}</Badge>
                            )}
                            {hasExamResult(exam) && exam.is_passed ? (
                              <Badge className={`gap-1 ${statusBadgeClassForStatusValue("SUCCESS")}`}>
                                <CheckCircle2 className="h-3 w-3" />
                                {t.examsPage.statusPassed}
                              </Badge>
                            ) : !hasExamResult(exam) ? (
                              <Badge variant="outline" className={statusBadgeClassForStatusValue("PENDING")}>
                                {(t.examsPage as any).statusNoResult || t.examsPage.statusPending}
                              </Badge>
                            ) : (
                              <Badge variant="outline" className={statusBadgeClassForStatusValue("FAILED")}>{t.examsPage.statusFailed}</Badge>
                            )}
                          </div>
                          <h3 className="text-lg font-semibold text-foreground">{exam.exam_code || exam.program_code || exam.exam_id || t.common.unknown}</h3>
                          <div className="grid gap-2 text-sm text-muted-foreground sm:grid-cols-2">
                            {hasText(exam.confirmation_number) && (
                              <div>
                                <span className="font-medium text-foreground">{t.examsPage.confirmationNumber}:</span> {exam.confirmation_number}
                              </div>
                            )}
                            {hasText(exam.site_name) && (
                              <div>
                                <span className="font-medium text-foreground">{t.examsPage.site}:</span> {exam.site_name}
                              </div>
                            )}
                            {hasText(exam.appointment_start_time) && (
                              <div>
                                <span className="font-medium text-foreground">{t.examsPage.appointmentStart}:</span> {formatBackendDate(exam.appointment_start_time)}
                              </div>
                            )}
                            {hasText(exam.appointment_end_time) && (
                              <div>
                                <span className="font-medium text-foreground">{t.examsPage.appointmentEnd}:</span> {formatBackendDate(exam.appointment_end_time)}
                              </div>
                            )}
                            {!hasAppointmentDetails(exam) && !hasExamResult(exam) && (
                              <div className="rounded-lg border border-blue-200 bg-blue-50 px-3 py-2 text-blue-700 sm:col-span-2">
                                <div className="flex items-start gap-2">
                                  <CalendarClock className="mt-0.5 h-4 w-4 shrink-0" />
                                  <div>
                                    <div className="font-medium text-blue-800">{t.examsPage.notScheduledTitle}</div>
                                    <div className="mt-1 text-xs">{t.examsPage.notScheduledDesc}</div>
                                  </div>
                                </div>
                              </div>
                            )}
                            <div>
                              <span className="font-medium text-foreground">{t.examsPage.candidate}:</span>{" "}
                              {[exam.candidate_first_name, exam.candidate_last_name].filter(Boolean).join(" ") || exam.candidate_email || t.common.unknown}
                            </div>
                            {hasExamResult(exam) && (
                              <div>
                                <span className="font-medium text-foreground">{t.examsPage.score}:</span>{" "}
                                {typeof exam.total_score === "number" ? exam.total_score.toFixed(2) : t.common.unknown}
                              </div>
                            )}
                          </div>
                        </div>
                        <div className="flex flex-wrap gap-2">
                          {canScheduleExam(exam) && exam.exam_id && (
                            <Button
                              className="gap-2"
                              disabled={scheduleLoadingExamId === exam.exam_id}
                              onClick={() => void handleScheduleExam(exam)}
                            >
                              {scheduleLoadingExamId === exam.exam_id ? (
                                <Loader2 className="h-4 w-4 animate-spin" />
                              ) : (
                                <ExternalLink className="h-4 w-4" />
                              )}
                              {t.learning.actionScheduleExam}
                            </Button>
                          )}
                          {hasExamResult(exam) && exam.exam_id && (
                            <Button variant="outline" asChild>
                              <Link href={`/exams/result?examId=${encodeURIComponent(exam.exam_id)}`}>
                                {t.examsPage.viewResult}
                              </Link>
                            </Button>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
