"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { Badge } from "@/components/ui/badge"
import { AlertCircle, BookOpen, Clock, Users, ChevronRight, CheckCircle2, Lock, Play, ShoppingCart } from "lucide-react"
import { PurchaseDialog } from "./purchase-dialog"
import { useTranslation } from "@/lib/useLanguage"
import { apiClient } from "@/lib/apiClient"
import { cn } from "@/lib/utils"
import { CANDIDATE_PIPELINE_STATUS_LABELS, statusLabel as getStatusLabel } from "@cftpweb/shared"

type CourseCardStat = {
  label: string
  value: string | number
}

type EligibilityBlocker = {
  blocker_type?: string
  description?: string
  details?: unknown[]
}

type EligibilityPreview = {
  can_purchase?: boolean
  can_unlock?: boolean
  blockers?: EligibilityBlocker[]
}

interface CourseCardProps {
  id: string
  title: string
  description: string
  image?: string
  category: "course" | "column" | "short"
  provider: string
  duration?: string
  students?: number
  isPurchased?: boolean
  progress?: number
  statusLabel?: string
  statusValue?: string | number
  versionLabel?: string
  stats?: CourseCardStat[]
}

export function CourseCard({
  id,
  title,
  description,
  image,
  category,
  provider,
  duration,
  students,
  isPurchased = false,
  progress,
  statusLabel,
  statusValue,
  versionLabel,
  stats = [],
}: CourseCardProps) {
  const { t, lang } = useTranslation()
  const [showPurchaseDialog, setShowPurchaseDialog] = useState(false)
  const [eligibility, setEligibility] = useState<EligibilityPreview | null>(null)
  const [eligibilityLoading, setEligibilityLoading] = useState(false)
  const resolvedStatusLabel = statusValue !== undefined ? getStatusLabel(t, CANDIDATE_PIPELINE_STATUS_LABELS, statusValue) : statusLabel
  const blockers = eligibility?.blockers || []
  const cardCopy = {
    ready: lang === "zh" ? "可购买认证" : "Ready to buy",
    unlock: lang === "zh" ? "需要先解锁" : "Unlock required",
    blocked: lang === "zh" ? "暂不可购买" : "Unavailable",
    checking: lang === "zh" ? "检查中" : "Checking",
    missingQualification: lang === "zh" ? "缺少解锁资格" : "Missing unlock qualification",
    alreadyPurchased: lang === "zh" ? "已购买" : "Already purchased",
    inProgressPurchase: lang === "zh" ? "有未完成订单" : "Order in progress",
    pipelineNotFound: lang === "zh" ? "认证已不可用" : "No longer available",
  }

  useEffect(() => {
    if (isPurchased || !id) return
    let cancelled = false
    const loadEligibility = async () => {
      setEligibilityLoading(true)
      try {
        const res = await apiClient(`/api/mall/pipelines/${id}/eligibility`)
        if (!cancelled) setEligibility(res)
      } catch {
        if (!cancelled) setEligibility(null)
      } finally {
        if (!cancelled) setEligibilityLoading(false)
      }
    }
    void loadEligibility()
    return () => {
      cancelled = true
    }
  }, [id, isPurchased])

  const blockerText = (blocker?: EligibilityBlocker) => {
    if (!blocker) return ""
    if (blocker.blocker_type === "MISSING_UNLOCK_QUALIFICATION") return cardCopy.missingQualification
    if (blocker.blocker_type === "ALREADY_PURCHASED") return cardCopy.alreadyPurchased
    if (blocker.blocker_type === "IN_PROGRESS_PURCHASE") return cardCopy.inProgressPurchase
    if (blocker.blocker_type === "PIPELINE_NOT_FOUND") return cardCopy.pipelineNotFound
    return blocker.description || blocker.blocker_type || ""
  }

  const accessState = (() => {
    if (isPurchased) return null
    if (eligibilityLoading && !eligibility) {
      return {
        label: cardCopy.checking,
        icon: Clock,
        className: "border-slate-200 bg-slate-50 text-slate-700",
        hint: "",
      }
    }
    if (eligibility?.can_purchase) {
      return {
        label: cardCopy.ready,
        icon: ShoppingCart,
        className: "border-emerald-200 bg-emerald-50 text-emerald-700",
        hint: "",
      }
    }
    if (eligibility?.can_unlock) {
      return {
        label: cardCopy.unlock,
        icon: Lock,
        className: "border-blue-200 bg-blue-50 text-blue-700",
        hint: "",
      }
    }
    if (eligibility) {
      return {
        label: cardCopy.blocked,
        icon: AlertCircle,
        className: "border-amber-200 bg-amber-50 text-amber-800",
        hint: blockerText(blockers[0]),
      }
    }
    return null
  })()

  const handleClick = (e: React.MouseEvent) => {
    if (!isPurchased) {
      e.preventDefault()
      setShowPurchaseDialog(true)
    }
  }

  const cardContent = (
    <>
      <div className="relative h-36 overflow-hidden bg-muted sm:h-40 xl:h-44">
        {image ? (
          <>
            <img
              src={image}
              alt={title}
              className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105"
            />
            <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent" />
          </>
        ) : (
          <div className="flex h-full items-center justify-center bg-muted">
            <BookOpen className="h-14 w-14 text-muted-foreground/60" />
          </div>
        )}
        
        {/* Purchased Badge */}
        {isPurchased && (
          <Badge className="absolute right-3 top-3 bg-emerald-500 text-white border-0 gap-1">
            <CheckCircle2 className="h-3 w-3" />
            {t.courses.purchased}
          </Badge>
        )}

        {/* Play button on hover */}
        <div className="absolute inset-0 flex items-center justify-center opacity-0 transition-opacity duration-300 group-hover:opacity-100">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-white/90 text-primary shadow-lg backdrop-blur-sm transition-transform duration-300 group-hover:scale-110">
            {isPurchased ? (
              <Play className="h-4 w-4 fill-current" />
            ) : (
              <ShoppingCart className="h-4 w-4" />
            )}
          </div>
        </div>
      </div>

      {/* Content Section */}
      <div className="p-4">
        <h3 className="mb-2 text-lg font-semibold text-card-foreground line-clamp-1 group-hover:text-primary transition-colors">
          {title}
        </h3>
        <p className="mb-4 text-sm text-muted-foreground line-clamp-2">{description}</p>

        {(resolvedStatusLabel || versionLabel) && (
          <div className="mb-4 flex flex-wrap gap-2">
            {resolvedStatusLabel && <Badge variant="outline">{resolvedStatusLabel}</Badge>}
            {versionLabel && <Badge variant="outline">{versionLabel}</Badge>}
          </div>
        )}

        {accessState && (
          <div className={cn("mb-4 rounded-lg border px-3 py-2 text-xs", accessState.className)}>
            <div className="flex items-center gap-1.5 font-medium">
              <accessState.icon className="h-3.5 w-3.5" />
              {accessState.label}
            </div>
            {accessState.hint && <div className="mt-1 text-[11px] opacity-80">{accessState.hint}</div>}
          </div>
        )}

        {/* Progress Bar (if purchased and has progress) */}
        {isPurchased && progress !== undefined && (
          <div className="mb-4">
            <div className="flex items-center justify-between text-xs mb-1.5">
              <span className="text-muted-foreground">{t.courses.courseProgress}</span>
              <span className="font-medium text-primary">{progress}%</span>
            </div>
            <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
              <div
                className="h-full rounded-full bg-primary transition-all"
                style={{ width: `${progress}%` }}
              />
            </div>
          </div>
        )}

        {/* Meta Info */}
        {(duration || students !== undefined) && (
          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <div className="flex items-center gap-4">
              {duration && (
                <div className="flex items-center gap-1.5">
                  <Clock className="h-4 w-4" />
                  <span>{duration}</span>
                </div>
              )}
              {students !== undefined && (
                <div className="flex items-center gap-1.5">
                  <Users className="h-4 w-4" />
                  <span>{students.toLocaleString()}</span>
                </div>
              )}
            </div>
          </div>
        )}

        {stats.length > 0 && (
          <div className="mt-4 grid grid-cols-3 gap-2 rounded-md bg-muted p-2 text-center">
            {stats.map((stat) => (
              <div key={stat.label}>
                <div className="text-sm font-semibold text-foreground">{stat.value}</div>
                <div className="truncate text-[11px] text-muted-foreground">{stat.label}</div>
              </div>
            ))}
          </div>
        )}

        {/* Provider */}
        <div className="mt-4 flex items-center justify-between pt-4 border-t border-border">
          <div className="flex items-center gap-2">
            <div className="h-6 w-6 rounded-full bg-primary/10 flex items-center justify-center text-[10px] font-bold text-primary">
              GF
            </div>
            <span className="text-sm text-muted-foreground">{provider}</span>
          </div>
          <ChevronRight className="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1 group-hover:text-primary" />
        </div>
      </div>
    </>
  )

  return (
    <>
      {isPurchased ? (
        <Link
          href={`/courses/detail?id=${encodeURIComponent(id)}`}
          className="group block overflow-hidden rounded-2xl border border-border bg-card shadow-sm transition-all duration-300 hover:shadow-lg hover:border-primary/20 hover:-translate-y-1"
        >
          {cardContent}
        </Link>
      ) : (
        <div
          onClick={handleClick}
          className="group block overflow-hidden rounded-2xl border border-border bg-card shadow-sm transition-all duration-300 hover:shadow-lg hover:border-primary/20 hover:-translate-y-1 cursor-pointer"
        >
          {cardContent}
        </div>
      )}

      <PurchaseDialog
        open={showPurchaseDialog}
        onOpenChange={setShowPurchaseDialog}
        courseName={title}
        pipelineId={id}
      />
    </>
  )
}
