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

type RecordItem = {
  id: string
  title: string
  institution: string
  date: string
  status: keyof typeof statusConfig
}

const records: RecordItem[] = []

const statusConfig = {
  verified: {
    label: "已认证",
    icon: CheckCircle2,
    color: "bg-emerald-500/10 text-emerald-700 border-emerald-200",
    iconColor: "text-emerald-500",
  },
  pending: {
    label: "审核中",
    icon: Clock,
    color: "bg-amber-500/10 text-amber-700 border-amber-200",
    iconColor: "text-amber-500",
  },
  rejected: {
    label: "已驳回",
    icon: AlertCircle,
    color: "bg-red-500/10 text-red-600 border-red-200",
    iconColor: "text-red-500",
  },
}

export default function RecordsPage() {
  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          {/* Header */}
          <div className="mb-8 flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold tracking-tight text-foreground">档案中心</h1>
              <p className="mt-1 text-muted-foreground">管理您的学历、证书和工作经历档案</p>
            </div>
            <Button className="gap-2" disabled>
              <Plus className="h-4 w-4" />
              上传新档案
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
                    status === "verified" && "bg-emerald-500/10",
                    status === "pending" && "bg-amber-500/10",
                    status === "rejected" && "bg-red-500/10"
                  )}>
                    <config.icon className={cn("h-5 w-5", config.iconColor)} />
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
              <h2 className="font-semibold text-card-foreground">我的档案</h2>
            </div>
            
            {records.length === 0 ? (
              <div className="flex flex-col items-center justify-center px-6 py-14 text-center">
                <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-muted">
                  <FileText className="h-7 w-7 text-muted-foreground" />
                </div>
                <h3 className="mb-2 text-lg font-semibold text-foreground">暂无档案记录</h3>
                <p className="max-w-md text-sm text-muted-foreground">
                  档案上传和审核接口接入后，这里会展示真实的学历、证书和工作经历记录。
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
                      <Badge className={config.color}>
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
