"use client"

import React from "react"

import { useState } from "react"
import { Sidebar } from "@/components/sidebar"
import { cn } from "@/lib/utils"
import {
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

const emptyState = {
  current: {
    title: "暂无可预约考试",
    description: "考试服务接入后，这里会展示可预约和进行中的考试。",
    icon: AlertCircle,
  },
  history: {
    title: "暂无历史成绩",
    description: "完成考试后，这里会展示真实成绩记录。",
    icon: History,
  },
  exemption: {
    title: "暂无免考申请",
    description: "免考申请服务接入后，这里会展示可提交的申请入口。",
    icon: FileCheck,
  },
  records: {
    title: "暂无申请记录",
    description: "提交考试或免考申请后，这里会展示真实审批记录。",
    icon: ClipboardList,
  },
} as const

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

          <div className="rounded-2xl border border-border bg-card p-6 shadow-sm">
            <div className="flex flex-col items-center justify-center py-12 text-center">
              {(() => {
                const state = emptyState[activeTab as keyof typeof emptyState]
                const Icon = state.icon
                return (
                  <>
                    <div className="mb-4 h-16 w-16 rounded-full bg-muted flex items-center justify-center">
                      <Icon className="h-8 w-8 text-muted-foreground" />
                    </div>
                    <h3 className="text-lg font-semibold text-foreground mb-2">{state.title}</h3>
                    <p className="max-w-md text-muted-foreground">{state.description}</p>
                  </>
                )
              })()}
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}
