"use client"

import { useState, useEffect } from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import { cn } from "@/lib/utils"
import { apiClient } from "@/lib/apiClient"
import { getCachedDashboard } from "@/lib/dashboardCache"
import {
  Home,
  BookOpen,
  Crown,
  FileText,
  Award,
  ShoppingCart,
  MessageSquare,
  GraduationCap,
  ChevronLeft,
  ChevronRight,
  User,
  Settings,
  LogOut,
  Globe,
} from "lucide-react"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { useTranslation } from "@/lib/useLanguage"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Badge } from "@/components/ui/badge"

export function Sidebar() {
  const { t, lang, changeLanguage } = useTranslation()
  const [collapsed, setCollapsed] = useState(false)
  const [userName, setUserName] = useState<string>(t.common.user)
  const [unreadCount, setUnreadCount] = useState<number>(0)
  const pathname = usePathname()

  const navItems = [
    { href: "/", icon: Home, label: t.sidebar.home },
    { href: "/courses", icon: BookOpen, label: t.sidebar.courses },
    { href: "/membership", icon: Crown, label: t.sidebar.membership },
    { href: "/exams", icon: FileText, label: t.sidebar.exams },
    { href: "/records", icon: GraduationCap, label: t.sidebar.records },
    { href: "/credentials", icon: Award, label: t.sidebar.credentials },
    { href: "/certificates", icon: Crown, label: t.sidebar.certificates },
    { href: "/orders", icon: ShoppingCart, label: t.sidebar.orders },
    { href: "/messages", icon: MessageSquare, label: t.sidebar.messages, badge: unreadCount > 0 ? unreadCount : undefined },
  ]

  useEffect(() => {
    const updateName = () => {
      const name = localStorage.getItem('user_name')
      if (name) {
        setUserName(name)
      }
    }
    const fetchUnreadCount = async () => {
      try {
        const dashboard = await getCachedDashboard()
        if (dashboard && dashboard.unread_messages_count !== undefined) {
          setUnreadCount(dashboard.unread_messages_count)
        }
      } catch (err) {}
    }

    updateName()
    fetchUnreadCount()
    window.addEventListener('storage', updateName)
    return () => window.removeEventListener('storage', updateName)
  }, [])

  const handleLogout = async () => {
    try {
      await apiClient('/api/auth/logout', { method: 'POST' })
    } catch (err) {
      // apiClient 已经静默处理错误弹窗
    } finally {
      localStorage.removeItem('access_token')
      localStorage.removeItem('user_name')
      window.location.href = '/login'
    }
  }

  return (
    <aside
      className={cn(
        "fixed left-0 top-0 z-40 flex h-screen flex-col border-r border-sidebar-border bg-sidebar transition-all duration-300",
        collapsed ? "w-[72px]" : "w-64"
      )}
    >
      {/* Logo */}
      <div className="flex h-16 items-center gap-3 border-b border-sidebar-border px-4">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary text-primary-foreground">
          <GraduationCap className="h-5 w-5" />
        </div>
        {!collapsed && (
          <div className="flex flex-col">
            <span className="text-sm font-semibold text-sidebar-foreground">{t.sidebar.systemBrand}</span>
            <span className="text-xs text-muted-foreground">{t.sidebar.systemName}</span>
          </div>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-4">
        {navItems.map((item) => {
          const isActive = pathname === item.href
          return (
            <Link
              key={item.href}
              href={item.href}
              prefetch={false}
              className={cn(
                "group relative flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all duration-200",
                isActive
                  ? "bg-primary text-primary-foreground shadow-sm"
                  : "text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
              )}
            >
              <item.icon className={cn("h-5 w-5 shrink-0", collapsed && "mx-auto")} />
              {!collapsed && (
                <>
                  <span>{item.label}</span>
                  {item.badge && (
                    <Badge
                      variant={isActive ? "secondary" : "default"}
                      className={cn(
                        "ml-auto h-5 min-w-[20px] px-1.5 text-xs",
                        isActive ? "bg-primary-foreground/20 text-primary-foreground" : ""
                      )}
                    >
                      {item.badge}
                    </Badge>
                  )}
                </>
              )}
              {collapsed && item.badge && (
                <span className="absolute -right-1 -top-1 flex h-4 w-4 items-center justify-center rounded-full bg-destructive text-[10px] text-destructive-foreground">
                  {item.badge}
                </span>
              )}
            </Link>
          )
        })}
      </nav>

      {/* User Profile */}
      <div className="border-t border-sidebar-border p-3">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              className={cn(
                "flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-colors hover:bg-sidebar-accent",
                collapsed && "justify-center px-0"
              )}
            >
              <Avatar className="h-9 w-9 border-2 border-primary/20">
                <AvatarFallback className="bg-primary/10 text-primary font-medium">
                  {userName.charAt(0)}
                </AvatarFallback>
              </Avatar>
              {!collapsed && (
                <div className="flex-1 overflow-hidden">
                  <p className="truncate text-sm font-medium text-sidebar-foreground">{userName}</p>
                  <p className="truncate text-xs text-muted-foreground">{t.common.certifiedMember}</p>
                </div>
              )}
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuItem asChild>
              <Link href="/settings?tab=profile" className="flex items-center w-full cursor-pointer">
                <User className="mr-2 h-4 w-4" />
                {t.sidebar.profile}
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href="/settings?tab=account" className="flex items-center w-full cursor-pointer">
                <Settings className="mr-2 h-4 w-4" />
                {t.sidebar.settings}
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => changeLanguage(lang === "zh" ? "en" : "zh")}>
              <Globe className="mr-2 h-4 w-4" />
              {t.sidebar.switchLang}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-destructive" onClick={handleLogout}>
              <LogOut className="mr-2 h-4 w-4" />
              {t.sidebar.logout}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {/* Collapse Button */}
      <button
        onClick={() => setCollapsed(!collapsed)}
        className="absolute -right-3 top-20 flex h-6 w-6 items-center justify-center rounded-full border bg-card text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
      >
        {collapsed ? <ChevronRight className="h-3 w-3" /> : <ChevronLeft className="h-3 w-3" />}
      </button>
    </aside>
  )
}
