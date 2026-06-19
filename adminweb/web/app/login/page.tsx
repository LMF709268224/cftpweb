'use client'

import React, { useEffect } from "react"
import { apiClient } from "@/lib/apiClient"

export default function LoginPage() {
  useEffect(() => {
    const handleLogin = async () => {
      try {
        const callbackUrl = encodeURIComponent(window.location.origin + "/callback")
        const resData = await apiClient(`/api/auth/login-url?callback=${callbackUrl}`)

        if (resData?.url) {
          window.location.href = resData.url
          return
        }

        throw new Error("AUTH_FAILED")
      } catch (err: any) {
        console.error(err)
      }
    }

    handleLogin()
  }, [])

  return (
    <div className="min-h-screen w-full flex items-center justify-center bg-slate-950 text-slate-50">
      <div className="flex flex-col items-center gap-4">
        <svg className="animate-spin h-8 w-8 text-indigo-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        <p className="text-slate-400">正在前往安全认证中心...</p>
      </div>
    </div>
  )
}
