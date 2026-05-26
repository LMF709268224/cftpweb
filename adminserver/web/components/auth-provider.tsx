"use client"

import { useEffect, useState } from "react"
import { useRouter, usePathname } from "next/navigation"

const publicPaths = ['/login', '/callback']

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const [isAuthenticated, setIsAuthenticated] = useState<boolean | null>(null)

  useEffect(() => {
    // 允许未登录访问公共路径
    if (publicPaths.includes(pathname)) {
      setIsAuthenticated(true)
      return
    }

    // 检查是否登录过
    const isAuthenticated = localStorage.getItem('is_authenticated') === 'true'
    if (!isAuthenticated) {
      setIsAuthenticated(false)
      router.push('/login')
    } else {
      setIsAuthenticated(true)
    }
  }, [pathname, router])

  // 在确定认证状态之前，不渲染内容以避免页面闪烁
  if (isAuthenticated === null) {
    return null
  }

  // 如果未认证且不在公共路径，不渲染内容（useEffect正在处理跳转）
  if (!isAuthenticated && !publicPaths.includes(pathname)) {
    return null
  }

  return <>{children}</>
}
