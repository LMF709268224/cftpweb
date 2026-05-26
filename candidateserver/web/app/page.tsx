"use client"

import React from "react"
import { useEffect, useState } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { StatsCard } from "@/components/stats-card"
import { TodoList } from "@/components/todo-list"
import { LearningProgress } from "@/components/learning-progress"
import { BookOpen, CheckCircle2, Crown, MessageSquare } from "lucide-react"
import { useTranslation } from "@/lib/useLanguage"

export default function HomePage() {
  const { t, lang } = useTranslation()
  const [userName, setUserName] = useState("...")
  const [unreadCount, setUnreadCount] = useState(0)

  const todoItems = [
    {
      id: "1",
      icon: "message" as const,
      title: lang === "zh" ? `你有 ${unreadCount} 条未读消息` : `You have ${unreadCount} unread messages`,
      action: { label: t.home.view, href: "/messages" },
    },
    {
      id: "2",
      icon: "file" as const,
      title: t.home.todo2Title,
      description: t.home.todo2Desc,
      action: { label: t.home.view, href: "/records" },
    },
    {
      id: "3",
      icon: "rejected" as const,
      title: t.home.todo3Title,
      description: t.home.todo3Desc,
      action: { label: t.home.reapply, href: "/exams" },
    },
  ]

  useEffect(() => {
    // 尝试�?/api/user/me 获取最新用户信�?    const fetchUser = async () => {
      try {
        // 尝试�?/api/user/me 获取最新用户信�?        const payload = await apiClient("/api/user/me")
        if (payload) {
          const nameToSet = payload.display_name || payload.name
          if (nameToSet) {
            setUserName(nameToSet)
            localStorage.setItem("user_name", nameToSet) // 同步更新本地存储给侧边栏�?          }
        }
      } catch (err) {
        const localName = localStorage.getItem("user_name")
        if (localName) setUserName(localName)
      }
    }
    const fetchDashboard = async () => {
      try {
        const dashboard = await apiClient("/api/dashboard")
        if (dashboard && dashboard.unread_messages_count !== undefined) {
          setUnreadCount(dashboard.unread_messages_count)
        }
      } catch (err) {
        console.error(err)
      }
    }

    fetchUser()
    fetchDashboard()
  }, [])

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          {/* Header */}
          <div className="mb-8">
            <h1 className="text-3xl font-bold tracking-tight text-foreground">{t.sidebar.home}</h1>
            <p className="mt-1 text-muted-foreground">{t.home.welcomeBack}，{userName}</p>
          </div>

          {/* Stats Grid */}
          <div className="mb-8 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            <StatsCard
              title={t.home.courseInProgress}
              value="1"
              icon={BookOpen}
              variant="primary"
              description="CFtP 认证课程"
              href="/courses"
            />
            <StatsCard
              title={t.home.certified}
              value="1"
              icon={CheckCircle2}
              variant="success"
              description="L1A 基础模块"
              href="/certificates"
            />
            <StatsCard
              title={t.home.memberLevel}
              value={t.common.certifiedMember}
              icon={Crown}
              variant="warning"
              description="Charterholder"
              href="/membership"
            />
            <StatsCard
              title={t.home.unreadMessages}
              value={String(unreadCount)}
              icon={MessageSquare}
              variant="info"
              description={t.home.unreadMessagesCount}
              href="/messages"
            />
          </div>

          {/* Main Content Grid */}
          <div className="grid gap-6 lg:grid-cols-5">
            {/* Todo List - Takes 3 columns */}
            <div className="lg:col-span-3">
              <TodoList items={todoItems} />
            </div>

            {/* Learning Progress - Takes 2 columns */}
            <div className="lg:col-span-2">
              <LearningProgress
                courseName="CFtP (Chartered Fintech Practitioner)"
                courseDescription="金融科技专业认证"
                currentModule="L1B Fintech"
                progress={38}
                totalModules={5}
                completedModules={2}
              />
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}
