"use client"

import React from "react"
import { useState, useEffect } from "react"
import Link from "next/link"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { formatBackendDate } from "@/lib/utils"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Award,
  Download,
  Share2,
  Eye,
  Calendar,
  CheckCircle2,
  ExternalLink,
} from "lucide-react"

export default function CertificatesPage() {
  const { t } = useTranslation()
  const [certificates, setCertificates] = useState<any[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    const fetchCertificates = async () => {
      setLoading(true)
      try {
        const res = await apiClient("/api/certificates")
        if (res?.certificates) {
          setCertificates(res.certificates.map((cert: any) => ({
            id: cert.cred_id || cert.catalog_id,
            name: cert.name,
            description: cert.description || "",
            issueDate: cert.created_at ? formatBackendDate(cert.created_at).split(" ")[0] : t.common.na,
            expiryDate: cert.valid_until ? formatBackendDate(cert.valid_until).split(" ")[0] : t.common.permanent,
            status: cert.status === 2 ? "active" : "inactive", // Example mapping
            credentialId: cert.cred_guid || cert.cred_id || t.common.na,
          })))
        }
      } catch (e) {
        console.error(e)
      } finally {
        setLoading(false)
      }
    }
    fetchCertificates()
  }, [])
  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          {/* Header */}
          <div className="mb-8">
            <h1 className="text-3xl font-bold tracking-tight text-foreground">证书中心</h1>
            <p className="mt-1 text-muted-foreground">查看和管理您获得的所有专业认证证�?/p>
          </div>

          {/* Certificate Cards */}
          <div className="grid gap-6 lg:grid-cols-2">
            {certificates.map((cert) => (
              <div
                key={cert.id}
                className="group relative overflow-hidden rounded-2xl border border-border bg-card shadow-sm"
              >
                {/* Certificate Header with gradient */}
                <div className="relative bg-gradient-to-br from-primary via-primary/90 to-primary p-6 text-white">
                  <div className="absolute -right-8 -top-8 h-32 w-32 rounded-full bg-white/10" />
                  <div className="absolute -bottom-4 -left-4 h-24 w-24 rounded-full bg-white/5" />
                  
                  <div className="relative flex items-start justify-between">
                    <div>
                      <Badge className="mb-3 bg-white/20 text-white border-0">
                        <CheckCircle2 className="mr-1 h-3 w-3" />
                        有效
                      </Badge>
                      <h3 className="text-xl font-bold mb-1">{cert.name}</h3>
                      <p className="text-sm text-white/80">{cert.description}</p>
                    </div>
                    <div className="flex h-14 w-14 items-center justify-center rounded-full bg-white/20 backdrop-blur-sm">
                      <Award className="h-7 w-7" />
                    </div>
                  </div>
                </div>

                {/* Certificate Details */}
                <div className="p-6">
                  <div className="mb-6 grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-xs text-muted-foreground mb-1">颁发日期</p>
                      <p className="font-medium text-card-foreground flex items-center gap-1.5">
                        <Calendar className="h-4 w-4 text-muted-foreground" />
                        {cert.issueDate}
                      </p>
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground mb-1">有效期至</p>
                      <p className="font-medium text-card-foreground flex items-center gap-1.5">
                        <Calendar className="h-4 w-4 text-muted-foreground" />
                        {cert.expiryDate}
                      </p>
                    </div>
                  </div>

                  <div className="mb-6 rounded-lg bg-muted/50 p-3">
                    <p className="text-xs text-muted-foreground mb-1">证书编号</p>
                    <p className="font-mono text-sm text-card-foreground">{cert.credentialId}</p>
                  </div>

                  {/* Actions */}
                  <div className="flex gap-3">
                    <Button className="flex-1 gap-2">
                      <Download className="h-4 w-4" />
                      下载证书
                    </Button>
                    <Button variant="outline" size="icon">
                      <Share2 className="h-4 w-4" />
                    </Button>
                    <Button variant="outline" size="icon">
                      <Eye className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </div>
            ))}

            {/* Empty State for more certificates */}
            <div className="flex flex-col items-center justify-center rounded-2xl border-2 border-dashed border-border p-12 text-center">
              <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-muted">
                <Award className="h-8 w-8 text-muted-foreground" />
              </div>
              <h3 className="text-lg font-semibold text-foreground mb-2">继续学习，获得更多认�?/h3>
              <p className="text-sm text-muted-foreground mb-4">
                完成更多课程模块以获得额外的专业认证
              </p>
              <Button variant="outline" className="gap-2" asChild>
                <Link href="/courses">
                  浏览课程
                  <ExternalLink className="h-4 w-4" />
                </Link>
              </Button>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}
