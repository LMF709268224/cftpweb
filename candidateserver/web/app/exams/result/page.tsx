"use client"

import React, { useEffect, useState } from "react"
import Link from "next/link"
import { useSearchParams } from "next/navigation"
import { ArrowLeft, Award, CheckCircle2, ClipboardList, ExternalLink, Loader2 } from "lucide-react"

import { Sidebar } from "@/components/sidebar"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { statusBadgeClassForStatusValue } from "@cftpweb/shared"

type ExamResultRsp = {
  exam_id?: string
  total_score?: number
  is_passed?: boolean
  has_result?: boolean
  score_details_raw?: string
}

export default function ExamResultPage() {
  const { t } = useTranslation()
  const searchParams = useSearchParams()
  const examId = decodeURIComponent(searchParams.get("examId") || "")
  const [loading, setLoading] = useState(true)
  const [result, setResult] = useState<ExamResultRsp | null>(null)

  useEffect(() => {
    const load = async () => {
      if (!examId) {
        setLoading(false)
        return
      }
      setLoading(true)
      try {
        const res = await apiClient(`/api/exams/${encodeURIComponent(examId)}/result`)
        setResult(res)
      } finally {
        setLoading(false)
      }
    }
    void load()
  }, [examId])

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />

      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          <Link
            href="/exams"
            className="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="h-4 w-4" />
            {t.examsPage.backToExams}
          </Link>

          {loading ? (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              {t.common.loading}
            </div>
          ) : !examId ? (
            <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">
              {t.examsPage.selectExamFirst}
            </div>
          ) : !result || result.has_result === false ? (
            <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">
              {t.examsPage.noScoreDetails}
            </div>
          ) : (
            <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
              <section className="rounded-2xl border bg-card p-6 shadow-sm">
                <div className="mb-4 flex flex-wrap items-center gap-2">
                  <Badge className={cn(statusBadgeClassForStatusValue(result.is_passed ? "SUCCESS" : "FAILED"), "border")}>
                    <Award className="mr-1 h-3 w-3" />
                    {result.is_passed ? t.examsPage.statusPassed : t.examsPage.statusFailed}
                  </Badge>
                  <Badge variant="outline">{result.exam_id || examId}</Badge>
                </div>

                <h1 className="text-3xl font-bold text-foreground">{t.examsPage.resultTitle}</h1>
                <p className="mt-2 text-muted-foreground">{t.examsPage.resultDesc}</p>

                <div className="mt-6 grid gap-4 sm:grid-cols-2">
                  <div className="rounded-xl border bg-background p-4">
                    <div className="text-xs text-muted-foreground">{t.examsPage.score}</div>
                    <div className="mt-1 text-2xl font-bold text-foreground">
                      {typeof result.total_score === "number" ? result.total_score.toFixed(2) : t.common.unknown}
                    </div>
                  </div>
                  <div className="rounded-xl border bg-background p-4">
                    <div className="text-xs text-muted-foreground">{t.examsPage.passStatus}</div>
                    <div className="mt-1 inline-flex items-center gap-2 text-lg font-semibold">
                      <CheckCircle2 className={cn("h-5 w-5", result.is_passed ? "text-blue-600" : "text-yellow-600")} />
                      {result.is_passed ? t.examsPage.statusPassed : t.examsPage.statusFailed}
                    </div>
                  </div>
                </div>

                <div className="mt-6 flex flex-wrap gap-2">
                  <Button asChild>
                    <Link href="/exams">
                      <ClipboardList className="h-4 w-4" />
                      {t.examsPage.backToExams}
                    </Link>
                  </Button>
                  <Button variant="outline" asChild>
                    <Link href="/certificates">
                      {t.examsPage.viewCertificate}
                      <ExternalLink className="h-4 w-4" />
                    </Link>
                  </Button>
                </div>
              </section>

              <section className="rounded-2xl border bg-card p-6 shadow-sm">
                <h2 className="mb-4 text-lg font-semibold text-foreground">{t.examsPage.scoreDetails}</h2>
                {result.score_details_raw ? (
                  <pre className="max-h-[560px] overflow-auto rounded-xl border bg-muted/30 p-4 text-xs leading-6 text-muted-foreground whitespace-pre-wrap">
                    {result.score_details_raw}
                  </pre>
                ) : (
                  <div className="rounded-xl border bg-muted/30 p-4 text-sm text-muted-foreground">
                    {t.examsPage.noScoreDetails}
                  </div>
                )}
              </section>
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
