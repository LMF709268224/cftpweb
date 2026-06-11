"use client"

import React from "react"

import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  GraduationCap,
  FileText,
  CheckCircle2,
  Clock,
  AlertCircle,
  ChevronRight,
  Plus,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { statusBadgeClassFromTone } from "@cftpweb/shared"
import { useTranslation } from "@/lib/useLanguage"

type RecordItem = {
  id: string
  title: string
  institution: string
  date: string
  status: keyof ReturnType<typeof getStatusConfig>
}

const records: RecordItem[] = []

const getStatusConfig = (t: any) => ({
  verified: {
    label: t.recordsPage.verified,
    icon: CheckCircle2,
    tone: "success" as const,
  },
  pending: {
    label: t.recordsPage.pending,
    icon: Clock,
    tone: "warning" as const,
  },
  rejected: {
    label: t.recordsPage.rejected,
    icon: AlertCircle,
    tone: "danger" as const,
  },
})

export default function RecordsPage() {
  const { t } = useTranslation()
  const statusConfig = getStatusConfig(t)
  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          {/* Header */}
          <div className="mb-8 flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold tracking-tight text-foreground">{t.recordsPage.title}</h1>
              <p className="mt-1 text-muted-foreground">{t.recordsPage.subtitle}</p>
            </div>
            <Button className="gap-2" disabled>
              <Plus className="h-4 w-4" />
              {t.recordsPage.uploadNew}
            </Button>
          </div>

          {/* Stats */}
          <div className="mb-8 grid gap-4 sm:grid-cols-3">
            {Object.entries(statusConfig).map(([status, config]) => {
              const count = records.filter((r) => r.status === status).length
              return (
                <div
                  key={status}
                  className="flex items-center gap-4 rounded-xl border border-border bg-card p-4"
                >
                  <div className={cn(
                    "flex h-10 w-10 items-center justify-center rounded-lg",
                    status === "verified" && "bg-blue-100",
                    status === "pending" && "bg-yellow-100",
                    status === "rejected" && "bg-red-100"
                  )}>
                    <config.icon className={cn(
                      "h-5 w-5",
                      config.tone === "success" && "text-blue-600",
                      config.tone === "warning" && "text-yellow-600",
                      config.tone === "danger" && "text-red-600"
                    )} />
                  </div>
                  <div>
                    <p className="text-2xl font-bold text-card-foreground">{count}</p>
                    <p className="text-sm text-muted-foreground">{config.label}</p>
                  </div>
                </div>
              )
            })}
          </div>

          {/* Records List */}
          <div className="rounded-2xl border border-border bg-card shadow-sm overflow-hidden">
            <div className="flex items-center gap-3 border-b border-border px-6 py-4">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10">
                <GraduationCap className="h-4 w-4 text-primary" />
              </div>
              <h2 className="font-semibold text-card-foreground">{t.recordsPage.myRecords}</h2>
            </div>
            
            {records.length === 0 ? (
              <div className="flex flex-col items-center justify-center px-6 py-14 text-center">
                <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-muted">
                  <FileText className="h-7 w-7 text-muted-foreground" />
                </div>
                <h3 className="mb-2 text-lg font-semibold text-foreground">{t.recordsPage.noRecords}</h3>
                <p className="max-w-md text-sm text-muted-foreground">
                  {t.recordsPage.noRecordsDesc}
                </p>
              </div>
            ) : (
              <div className="divide-y divide-border">
                {records.map((record) => {
                const config = statusConfig[record.status as keyof typeof statusConfig]
                return (
                  <div
                    key={record.id}
                    className="group flex items-center justify-between p-6 transition-colors hover:bg-muted/50"
                  >
                    <div className="flex items-center gap-4">
                      <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-muted">
                        <FileText className="h-6 w-6 text-muted-foreground" />
                      </div>
                      <div>
                        <h3 className="font-medium text-card-foreground">{record.title}</h3>
                        <p className="text-sm text-muted-foreground">
                          {record.institution} · {record.date}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <Badge className={statusBadgeClassFromTone(config.tone)}>
                        {config.label}
                      </Badge>
                      <ChevronRight className="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1" />
                    </div>
                  </div>
                )
                })}
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
