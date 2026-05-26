"use client"

import React from "react"
import { useEffect, useState } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { StatsCard } from "@/components/stats-card"

import { BookOpen } from "lucide-react"
import { useTranslation } from "@/lib/useLanguage"

export default function HomePage() {
  const { t } = useTranslation()
  const [userName, setUserName] = useState("...")


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

    fetchUser()
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

          <div className="mb-8 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            <StatsCard
              title={t.sidebar.pipelines}
              value="..."
              icon={BookOpen}
              variant="primary"
              description="Manage Course Outlines"
              href="/pipelines"
            />
            <StatsCard
              title={t.sidebar.catalogs}
              value="..."
              icon={BookOpen}
              variant="success"
              description="Manage Categories"
              href="/catalogs"
            />
          </div>
        </div>
      </main>
    </div>
  )
}
