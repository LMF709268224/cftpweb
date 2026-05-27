"use client"

import React from "react"

import { useState } from "react"
import { Sidebar } from "@/components/sidebar"
import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  FileText,
  Clock,
  Target,
  Calendar,
  MapPin,
  ArrowRight,
  CheckCircle2,
  XCircle,
  AlertCircle,
  History,
  FileCheck,
  ClipboardList,
} from "lucide-react"

const tabs = [
  { id: "current", label: "预约与进行中" },
  { id: "history", label: "历史成绩" },
  { id: "exemption", label: "免考申请" },
  { id: "records", label: "申请记录" },
]

const currentExams = [
  {
    id: "l1b",
    name: "L1B Fintech Exam",
    platform: "Prometric",
    duration: "120 分钟",
    passingScore: 65,
    status: "available",
    description: "金融科技基础模块考试",
  },
]

const historyExams = [
  {
    id: "l1a",
    name: "L1A Foundation Exam",
    date: "2025-12-15",
    score: 78,
    passingScore: 65,
    status: "passed",
  },
  {
    id: "mock1",
    name: "模拟考试 - L1B",
    date: "2026-01-10",
    score: 62,
    passingScore: 65,
    status: "failed",
  },
]

export default function ExamsPage() {
  const [activeTab, setActiveTab] = useState("current")

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          {/* Header */}
          <div className="mb-8">
            <h1 className="text-3xl font-bold tracking-tight text-foreground">考试中心</h1>
            <p className="mt-1 text-muted-foreground">查看最新考试进度并回顾您的历史考试成绩</p>
          </div>

          {/* Tabs */}
          <div className="mb-8 flex gap-1 rounded-xl bg-muted p-1 w-fit overflow-x-auto">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={cn(
                  "px-4 py-2 text-sm font-medium rounded-lg transition-all duration-200 whitespace-nowrap",
                  activeTab === tab.id
                    ? "bg-card text-card-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Current Exams */}
          {activeTab === "current" && (
            <div className="space-y-6">
              <div className="rounded-2xl border border-border bg-card shadow-sm overflow-hidden">
                <div className="flex items-center gap-3 border-b border-border px-6 py-4">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-red-500/10">
                    <AlertCircle className="h-4 w-4 text-red-500" />
                  </div>
                  <h2 className="font-semibold text-card-foreground">当前考试</h2>
                </div>
                
                <div className="divide-y divide-border">
                  {currentExams.map((exam) => (
                    <div key={exam.id} className="p-6">
                      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
                        <div className="flex items-start gap-4">
                          <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
                            <FileText className="h-6 w-6" />
                          </div>
                          <div>
                            <h3 className="text-lg font-semibold text-card-foreground mb-1">
                              {exam.name}
                            </h3>
                            <p className="text-sm text-muted-foreground mb-3">{exam.description}</p>
                            <div className="flex flex-wrap gap-4 text-sm text-muted-foreground">
                              <div className="flex items-center gap-1.5">
                                <MapPin className="h-4 w-4" />
                                <span>平台：{exam.platform}</span>
                              </div>
                              <div className="flex items-center gap-1.5">
                                <Clock className="h-4 w-4" />
                                <span>时长：{exam.duration}</span>
                              </div>
                              <div className="flex items-center gap-1.5">
                                <Target className="h-4 w-4" />
                                <span>合格分：{exam.passingScore}</span>
                              </div>
                            </div>
                          </div>
                        </div>
                        <Button className="shrink-0 group">
                          立即预约考试
                          <ArrowRight className="ml-2 h-4 w-4 transition-transform group-hover:translate-x-1" />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Tips Section */}
              <div className="rounded-2xl border border-border bg-card p-6 shadow-sm">
                <h3 className="font-semibold text-card-foreground mb-4">考试须知</h3>
                <ul className="space-y-3 text-sm text-muted-foreground">
                  <li className="flex items-start gap-2">
                    <CheckCircle2 className="h-4 w-4 text-emerald-500 mt-0.5 shrink-0" />
                    <span>请在考试前至少 30 分钟到达考试中心</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <CheckCircle2 className="h-4 w-4 text-emerald-500 mt-0.5 shrink-0" />
                    <span>携带有效身份证件（身份证或护照）</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <CheckCircle2 className="h-4 w-4 text-emerald-500 mt-0.5 shrink-0" />
                    <span>考试期间不允许使用任何电子设备</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <CheckCircle2 className="h-4 w-4 text-emerald-500 mt-0.5 shrink-0" />
                    <span>考试成绩将在 3-5 个工作日内公布</span>
                  </li>
                </ul>
              </div>
            </div>
          )}

          {/* History */}
          {activeTab === "history" && (
            <div className="rounded-2xl border border-border bg-card shadow-sm overflow-hidden">
              <div className="flex items-center gap-3 border-b border-border px-6 py-4">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10">
                  <History className="h-4 w-4 text-primary" />
                </div>
                <h2 className="font-semibold text-card-foreground">历史成绩</h2>
              </div>
              
              <div className="divide-y divide-border">
                {historyExams.map((exam) => (
                  <div key={exam.id} className="flex items-center justify-between p-6">
                    <div className="flex items-center gap-4">
                      <div className={cn(
                        "flex h-10 w-10 items-center justify-center rounded-xl",
                        exam.status === "passed" ? "bg-emerald-500/10" : "bg-red-500/10"
                      )}>
                        {exam.status === "passed" ? (
                          <CheckCircle2 className="h-5 w-5 text-emerald-600" />
                        ) : (
                          <XCircle className="h-5 w-5 text-red-500" />
                        )}
                      </div>
                      <div>
                        <h3 className="font-medium text-card-foreground">{exam.name}</h3>
                        <div className="flex items-center gap-2 text-sm text-muted-foreground mt-1">
                          <Calendar className="h-3.5 w-3.5" />
                          <span>{exam.date}</span>
                        </div>
                      </div>
                    </div>
                    <div className="text-right">
                      <div className={cn(
                        "text-2xl font-bold",
                        exam.status === "passed" ? "text-emerald-600" : "text-red-500"
                      )}>
                        {exam.score}
                      </div>
                      <Badge variant={exam.status === "passed" ? "default" : "destructive"}>
                        {exam.status === "passed" ? "通过" : "未通过"}
                      </Badge>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Exemption & Records */}
          {(activeTab === "exemption" || activeTab === "records") && (
            <div className="rounded-2xl border border-border bg-card p-6 shadow-sm">
              <div className="flex flex-col items-center justify-center py-12 text-center">
                <div className="mb-4 h-16 w-16 rounded-full bg-muted flex items-center justify-center">
                  {activeTab === "exemption" ? (
                    <FileCheck className="h-8 w-8 text-muted-foreground" />
                  ) : (
                    <ClipboardList className="h-8 w-8 text-muted-foreground" />
                  )}
                </div>
                <h3 className="text-lg font-semibold text-foreground mb-2">
                  {activeTab === "exemption" ? "免考申请" : "申请记录"}
                </h3>
                <p className="text-muted-foreground mb-4">
                  {activeTab === "exemption" 
                    ? "如果您拥有相关专业资格证书，可申请免考相应模块" 
                    : "查看您的所有申请记录和审批状态"
                  }
                </p>
                {activeTab === "exemption" && (
                  <Button>
                    提交免考申请
                    <ArrowRight className="ml-2 h-4 w-4" />
                  </Button>
                )}
              </div>
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
