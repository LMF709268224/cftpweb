"use client"

import React, { Suspense, useEffect, useState } from "react"
import Link from "next/link"
import { useParams, useSearchParams } from "next/navigation"
import { CheckCircle2, Loader2 } from "lucide-react"

import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { Sidebar } from "@/components/sidebar"
import { Button } from "@/components/ui/button"

function ExamTermUrlContent() {
  const { t } = useTranslation()
  const params = useParams<{ urlType: string; examId: string }>()
  const searchParams = useSearchParams()
  const [callbackDone, setCallbackDone] = useState(false)

  useEffect(() => {
    const examId = params?.examId
    const urlType = params?.urlType
    if (!examId || !urlType) {
      setCallbackDone(true)
      return
    }

    const callbackBody = JSON.stringify({
      url_type: urlType,
      query: Object.fromEntries(searchParams.entries()),
    })

    apiClient(`/api/exams/${encodeURIComponent(examId)}/schedule-callback`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        url_type: urlType,
        callback_body: callbackBody,
      }),
    })
      .catch(() => undefined)
      .finally(() => setCallbackDone(true))
  }, [params?.examId, params?.urlType, searchParams])

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <main className="pl-64 transition-all duration-300">
        <div className="flex min-h-screen items-center justify-center px-8 py-8">
          <div className="w-full max-w-xl rounded-2xl border border-border bg-card p-8 text-center shadow-sm">
            <div className="mx-auto mb-5 flex h-16 w-16 items-center justify-center rounded-full bg-blue-50">
              {callbackDone ? (
                <CheckCircle2 className="h-8 w-8 text-blue-600" />
              ) : (
                <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
              )}
            </div>
            <h1 className="text-2xl font-bold text-foreground">{t.examsPage.scheduleReturnTitle}</h1>
            <p className="mt-3 text-muted-foreground">{t.examsPage.scheduleReturnDesc}</p>
            <div className="mt-6 flex justify-center">
              <Button asChild>
                <Link href="/exams">{t.examsPage.backToExams}</Link>
              </Button>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}

export default function ExamTermUrlPage() {
  return (
    <Suspense fallback={null}>
      <ExamTermUrlContent />
    </Suspense>
  )
}
