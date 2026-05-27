'use client'

import React from "react"

import { useState } from 'react'
import { LogIn, ArrowRight, ShieldCheck, Zap, Globe } from 'lucide-react'
import { getErrorMessage } from "@/lib/errorCodes"

export default function LoginPage() {
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleLogin = async () => {
    try {
      setIsLoading(true)
      setError('')

      // 获取 Casdoor 的登录跳转链接
      // 传递 callback 参数，让 Casdoor 登录完成后跳转到我们的 /callback 页面
      const callbackUrl = encodeURIComponent(window.location.origin + '/callback')
      const response = await fetch(`/api/auth/login-url?callback=${callbackUrl}`)

      if (!response.ok) {
        throw new Error('AUTH_FAILED')
      }

      const resData = await response.json()

      if (resData.data && resData.data.url) {
        // 直接跳转到 Casdoor 统一登录中心
        window.location.href = resData.data.url
      } else {
        throw new Error('AUTH_FAILED')
      }
    } catch (err: any) {
      const currentLang = (localStorage.getItem("app_lang") || "zh") as "zh" | "en"
      setError(getErrorMessage(err.message, currentLang))
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen w-full flex bg-slate-950 text-slate-50 selection:bg-indigo-500/30">
      {/* 左侧装饰区域 (Glassmorphism + 动态光效) */}
      <div className="hidden lg:flex flex-1 relative overflow-hidden bg-slate-900 items-center justify-center">
        {/* 背景光效 */}
        <div className="absolute top-[-20%] left-[-10%] w-[70%] h-[70%] rounded-full bg-indigo-600/20 blur-[120px]" />
        <div className="absolute bottom-[-20%] right-[-10%] w-[70%] h-[70%] rounded-full bg-blue-600/20 blur-[120px]" />

        {/* 悬浮卡片 */}
        <div className="relative z-10 p-12 max-w-2xl">
          <div className="inline-flex items-center rounded-full border border-indigo-500/30 bg-indigo-500/10 px-3 py-1 text-sm font-medium text-indigo-300 backdrop-blur-sm mb-8">
            <Zap className="mr-2 h-4 w-4" />
            企业级统一认证
          </div>
          <h1 className="text-5xl font-extrabold tracking-tight mb-6 bg-gradient-to-br from-white to-slate-400 bg-clip-text text-transparent">
            安全、高效的<br />下一代通行证
          </h1>
          <p className="text-lg text-slate-400 mb-12 max-w-xl leading-relaxed">
            基于 Casdoor 强力驱动，为您提供金融级别的安全防护、无缝单点登录体验与全球边缘加速接入。
          </p>

          <div className="grid grid-cols-2 gap-6">
            <div className="rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-md">
              <ShieldCheck className="h-8 w-8 text-indigo-400 mb-4" />
              <h3 className="font-semibold text-white mb-2">多端同调</h3>
              <p className="text-sm text-slate-400">一次登录，畅享全生态微服务矩阵，告别重复认证。</p>
            </div>
            <div className="rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur-md">
              <Globe className="h-8 w-8 text-blue-400 mb-4" />
              <h3 className="font-semibold text-white mb-2">全球网络</h3>
              <p className="text-sm text-slate-400">智能感知网络链路，为您分配最近的认证节点。</p>
            </div>
          </div>
        </div>
      </div>

      {/* 右侧登录交互区域 */}
      <div className="flex-1 flex flex-col items-center justify-center p-8 sm:p-12 lg:p-24 relative z-10 bg-slate-950 shadow-2xl">
        <div className="w-full max-w-md space-y-10">
          <div className="text-center">
            <h2 className="text-3xl font-bold tracking-tight text-white mb-2">欢迎回来</h2>
            <p className="text-slate-400">请使用统一身份认证系统 (SSO) 登录您的账号</p>
          </div>

          {error && (
            <div className="rounded-xl border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-400 backdrop-blur-sm flex items-start">
              <span className="block sm:inline">{error}</span>
            </div>
          )}

          <div className="space-y-6">
            <button
              onClick={handleLogin}
              disabled={isLoading}
              className="group relative w-full flex justify-center py-4 px-4 border border-transparent text-sm font-semibold rounded-2xl text-white bg-indigo-600 hover:bg-indigo-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-600 focus:ring-offset-slate-950 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed shadow-[0_0_40px_-10px_rgba(79,70,229,0.5)] hover:shadow-[0_0_60px_-15px_rgba(79,70,229,0.7)] overflow-hidden"
            >
              <div className="absolute inset-0 w-full h-full bg-gradient-to-r from-transparent via-white/10 to-transparent -translate-x-full group-hover:animate-[shimmer_1.5s_infinite]" />
              <span className="relative flex items-center gap-2">
                {isLoading ? (
                  <>
                    <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    正在安全连接...
                  </>
                ) : (
                  <>
                    <LogIn className="h-5 w-5" />
                    登录
                    <ArrowRight className="h-4 w-4 ml-1 opacity-70 group-hover:translate-x-1 transition-transform" />
                  </>
                )}
              </span>
            </button>
          </div>

          <p className="text-center text-xs text-slate-500">
            点击登录即表示您同意我们的{' '}
            <a href="#" className="font-medium text-indigo-400 hover:text-indigo-300 transition-colors">服务条款</a>
            {' '}和{' '}
            <a href="#" className="font-medium text-indigo-400 hover:text-indigo-300 transition-colors">隐私政策</a>
          </p>
        </div>
      </div>
    </div>
  )
}
