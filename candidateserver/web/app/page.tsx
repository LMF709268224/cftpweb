"use client"

import React from "react"

import { useEffect, useState } from "react"
import { apiClient } from "@/lib/apiClient"
import { getCachedDashboard } from "@/lib/dashboardCache"
import { Sidebar } from "@/components/sidebar"
import { StatsCard } from "@/components/stats-card"
import { TodoList } from "@/components/todo-list"
import { BookOpen, CheckCircle2, Crown, MessageSquare } from "lucide-react"
import { useTranslation } from "@/lib/useLanguage"

type DashboardStats = {
  courses_in_progress?: number
  certifications_earned?: number
  membership_level?: string
  unread_messages?: number
}

export default function HomePage() {
  const { t, lang } = useTranslation()
  const [userName, setUserName] = useState("...")
  const [unreadCount, setUnreadCount] = useState(0)
  const [stats, setStats] = useState<DashboardStats>({})

  const todoItems = unreadCount > 0
    ? [
        {
          id: "message-unread",
          icon: "message" as const,
          title: lang === "zh" ? `你有 ${unreadCount} 条未读消息` : `You have ${unreadCount} unread messages`,
          action: { label: t.home.view, href: "/messages" },
        },
      ]
    : []

  useEffect(() => {
    // 尝试从 /api/user/me 获取最新用户信息
    const fetchUser = async () => {
      try {
        // 尝试从 /api/user/me 获取最新用户信息
        const payload = await apiClient("/api/user/me")
        if (payload) {
          const nameToSet = payload.display_name || payload.name
          if (nameToSet) {
            setUserName(nameToSet)
            localStorage.setItem("user_name", nameToSet) // 同步更新本地存储给侧边栏用
          }
        }
      } catch (err) {
        const localName = localStorage.getItem("user_name")
        if (localName) setUserName(localName)
      }
    }
    const fetchDashboard = async () => {
      try {
        const dashboard = await getCachedDashboard()
        if (dashboard && dashboard.unread_messages_count !== undefined) {
          setUnreadCount(dashboard.unread_messages_count)
        }
        if (dashboard?.stats) {
          setStats(dashboard.stats)
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
              value={String(stats.courses_in_progress || 0)}
              icon={BookOpen}
              variant="primary"
              description={t.courses.tabs.my}
              href="/courses"
            />
            <StatsCard
              title={t.home.certified}
              value={String(stats.certifications_earned || 0)}
              icon={CheckCircle2}
              variant="success"
              description={t.sidebar.certificates}
              href="/certificates"
            />
            <StatsCard
              title={t.home.memberLevel}
              value={stats.membership_level || t.common.na}
              icon={Crown}
              variant="warning"
              description={t.membership.title}
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
          <div>
            <TodoList items={todoItems} />
          </div>
        </div>
      </main>
    </div>
  )
}
