'use client'

import React from "react"

import { useEffect, useState, useRef, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { Loader2, ShieldAlert, CheckCircle2 } from 'lucide-react'
import { getErrorMessage } from "@/lib/errorCodes"

function CallbackContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading')
  const [errorMsg, setErrorMsg] = useState('')
  const hasAttemptedLogin = useRef(false)

  useEffect(() => {
    // 确保在严格模式下只执行一次
    if (hasAttemptedLogin.current) return
    hasAttemptedLogin.current = true

    const code = searchParams.get('code')
    const state = searchParams.get('state')
    const currentLang = (localStorage.getItem("app_lang") || "zh") as "zh" | "en"

    if (!code || !state) {
      setStatus('error')
      setErrorMsg(getErrorMessage("INVALID_REQUEST", currentLang))
      setTimeout(() => router.push('/login'), 3000)
      return
    }

    const performLogin = async () => {
      try {
        const res = await fetch('/api/auth/login', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ code, state }),
        })

        if (!res.ok) {
          const errData = await res.json().catch(() => ({}))
          throw new Error(errData.error_code || 'AUTH_FAILED')
        }

        const resData = await res.json()
        const payload = resData.data || {}
        
        console.log("Login callback payload:", payload)
        
        setStatus('success')
        
        // 可以在这里把用户的基本信息存入 localStorage 或全局状态
        if (payload.user) {
          localStorage.setItem('user_name', payload.user.name)
        }
        if (payload.token) {
          localStorage.setItem('access_token', payload.token)
        }
        localStorage.setItem('is_authenticated', 'true')

        // 认证成功，跳转到首页
        setTimeout(() => {
          router.push('/')
        }, 1000)

      } catch (err: any) {
        setStatus('error')
        setErrorMsg(getErrorMessage(err.message, currentLang))
        setTimeout(() => router.push('/login'), 3000)
      }
    }

    performLogin()
  }, [searchParams, router])

  return (
    <div className="min-h-screen w-full flex flex-col items-center justify-center bg-slate-950 text-slate-50 relative overflow-hidden">
      {/* 动态背景光效 */}
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] rounded-full bg-indigo-600/10 blur-[150px] pointer-events-none" />

      <div className="relative z-10 flex flex-col items-center p-8 rounded-3xl bg-white/5 border border-white/10 backdrop-blur-xl shadow-2xl max-w-sm w-full mx-4">
        
        {status === 'loading' && (
          <>
            <div className="relative">
              <div className="absolute inset-0 rounded-full blur-xl bg-indigo-500/50 animate-pulse" />
              <Loader2 className="w-16 h-16 text-indigo-400 animate-spin relative z-10" />
            </div>
            <h2 className="mt-8 text-xl font-semibold tracking-tight text-white">正在验证身份</h2>
            <p className="mt-2 text-sm text-slate-400 text-center">
              我们正在与 Casdoor 建立安全会话<br />请不要关闭此页面...
            </p>
          </>
        )}

        {status === 'success' && (
          <>
            <div className="relative">
              <div className="absolute inset-0 rounded-full blur-xl bg-emerald-500/50 animate-pulse" />
              <CheckCircle2 className="w-16 h-16 text-emerald-400 relative z-10 animate-in zoom-in duration-300" />
            </div>
            <h2 className="mt-8 text-xl font-semibold tracking-tight text-white">认证成功</h2>
            <p className="mt-2 text-sm text-slate-400">正在为您跳转到控制台...</p>
          </>
        )}

        {status === 'error' && (
          <>
            <div className="relative">
              <div className="absolute inset-0 rounded-full blur-xl bg-red-500/50 animate-pulse" />
              <ShieldAlert className="w-16 h-16 text-red-400 relative z-10 animate-in zoom-in duration-300" />
            </div>
            <h2 className="mt-8 text-xl font-semibold tracking-tight text-white">认证遇到问题</h2>
            <p className="mt-2 text-sm text-red-300 text-center bg-red-500/10 border border-red-500/20 p-3 rounded-lg w-full">
              {errorMsg}
            </p>
            <p className="mt-4 text-xs text-slate-500">
              3 秒后将返回登录页面
            </p>
          </>
        )}
      </div>
    </div>
  )
}

export default function CallbackPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen w-full flex flex-col items-center justify-center bg-slate-950 text-slate-50 relative overflow-hidden">
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] rounded-full bg-indigo-600/10 blur-[150px] pointer-events-none" />
        <div className="relative z-10 flex flex-col items-center p-8 rounded-3xl bg-white/5 border border-white/10 backdrop-blur-xl shadow-2xl max-w-sm w-full mx-4">
          <div className="relative">
            <div className="absolute inset-0 rounded-full blur-xl bg-indigo-500/50 animate-pulse" />
            <Loader2 className="w-16 h-16 text-indigo-400 animate-spin relative z-10" />
          </div>
          <h2 className="mt-8 text-xl font-semibold tracking-tight text-white">加载中</h2>
        </div>
      </div>
    }>
      <CallbackContent />
    </Suspense>
  )
}
