"use client"

import React, { Suspense, useEffect, useState } from "react"
import Link from "next/link"
import { useSearchParams } from "next/navigation"
import { ArrowLeft, Clock, RefreshCw } from "lucide-react"

import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { timelineStatusLabel } from "@cftpweb/shared"

type TimelineLog = {
  transition_ulid?: string
  entity_type?: string
  entity_ulid?: string
  from_status?: string
  to_status?: string
  reason_code?: string
  reason_message?: string
  trigger_source?: string
  event_type?: string
  created_at?: string
}

export default function CourseTimelinePage() {
  return (
    <Suspense fallback={null}>
      <CourseTimelineContent />
    </Suspense>
  )
}

function CourseTimelineContent() {
  const searchParams = useSearchParams()
  const pipelineId = searchParams.get("id") || ""
  const { t } = useTranslation()
  const [logs, setLogs] = useState<TimelineLog[]>([])
  const [loading, setLoading] = useState(false)

  const loadTimeline = async () => {
    if (!pipelineId) {
      setLogs([])
      return
    }
    setLoading(true)
    try {
      const res = await apiClient(`/api/mall/pipelines/${pipelineId}/timeline`)
      setLogs(res?.logs || [])
    } catch {
      setLogs([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadTimeline()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pipelineId])

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          <Link
            href={pipelineId ? `/courses/detail?id=${encodeURIComponent(pipelineId)}` : "/courses"}
            className="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="h-4 w-4" />
            {t.learning.timelineBackToCourseDetails}
          </Link>

          {/* <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
            <div>
              <h1 className="text-3xl font-bold tracking-tight text-foreground">{t.learning.pipelineTimelineTitle}</h1>
              <p className="mt-1 text-muted-foreground">{t.learning.pipelineTimelineDesc}</p>
            </div>
            <Button variant="outline" className="gap-2" onClick={() => void loadTimeline()} disabled={loading}>
              {loading ? <RefreshCw className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
              {t.learning.timelineRefresh}
            </Button>
          </div> */}

          <div className="rounded-2xl border bg-card p-6 shadow-sm">
            {loading ? (
              <div className="py-16 text-center text-muted-foreground">{t.common.loading}</div>
            ) : logs.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-14 text-center">
                <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-muted">
                  <Clock className="h-8 w-8 text-muted-foreground" />
                </div>
                <h3 className="mb-2 text-lg font-semibold text-foreground">{t.learning.pipelineTimelineEmpty}</h3>
                <p className="max-w-md text-muted-foreground">{t.learning.pipelineTimelineEmptyHint}</p>
              </div>
            ) : (
              <div className="space-y-4">
                {logs.map((log) => (
                  <div key={log.transition_ulid} className="rounded-xl border bg-background p-4">
                    <div className="mb-2 flex flex-wrap items-center gap-2">
                      <Badge variant="outline">{log.entity_type || t.common.unknown}</Badge>
                      <Badge className="border-0 bg-primary/10 text-primary">{log.event_type || t.common.unknown}</Badge>
                      <span className="text-sm text-muted-foreground">{log.created_at || t.common.unknown}</span>
                    </div>
                    <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
                      <span>{timelineStatusLabel(t, log.entity_type, log.from_status)}</span>
                      <ArrowLeft className="h-3.5 w-3.5 rotate-180" />
                      <span>{timelineStatusLabel(t, log.entity_type, log.to_status)}</span>
                    </div>
                    <div className="mt-2 text-sm text-muted-foreground">
                      {log.reason_message || log.reason_code || t.common.unknown}
                    </div>
                    <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                      <span>{log.trigger_source || t.common.unknown}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
