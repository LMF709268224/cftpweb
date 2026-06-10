import React from "react"
import type { Metadata } from 'next'
import { Geist, Geist_Mono } from 'next/font/google'
import { AuthProvider } from '@/components/auth-provider'
import { PaymentReturnHandler } from '@/components/payment-return-handler'
import './globals.css'

const _geist = Geist({ subsets: ["latin"] });
const _geistMono = Geist_Mono({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: 'CFtP 培训系统 - 金融科技专业认证',
  description: 'CFtP (Chartered Fintech Practitioner) 专业金融科技认证培训平台',
  generator: 'v0.app',
  icons: {
    icon: [
      {
        url: '/icon-light-32x32.png',
        media: '(prefers-color-scheme: light)',
      },
      {
        url: '/icon-dark-32x32.png',
        media: '(prefers-color-scheme: dark)',
      },
      {
        url: '/icon.svg',
        type: 'image/svg+xml',
      },
    ],
    apple: '/apple-icon.png',
  },
}

import { Toaster } from "@/components/ui/sonner"

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="zh-CN" className="bg-background">
      <body className="font-sans antialiased">
        <AuthProvider>
          <PaymentReturnHandler />
          {children}
        </AuthProvider>
        <Toaster />
        <div className="hidden border-green-400 bg-green-100 text-black border-blue-400 bg-blue-100 border-yellow-400 bg-yellow-100 border-red-400 bg-red-100 border-gray-400 bg-gray-100" />
      </body>
    </html>
  )
}
