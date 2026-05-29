"use client"

import { useState } from "react"
import Link from "next/link"
import Image from "next/image"
import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Clock, Users, ChevronRight, CheckCircle2, Play, ShoppingCart } from "lucide-react"
import { PurchaseDialog } from "./purchase-dialog"
import { useTranslation } from "@/lib/useLanguage"

type CourseCardStat = {
  label: string
  value: string | number
}

interface CourseCardProps {
  id: string
  title: string
  description: string
  image: string
  category: "course" | "column" | "short"
  provider: string
  duration?: string
  students?: number
  isPurchased?: boolean
  progress?: number
  price?: number
  priceLabel?: string
  paymentConfigured?: boolean
  statusLabel?: string
  versionLabel?: string
  stats?: CourseCardStat[]
}

const categoryStyles = {
  course: "bg-primary/10 text-primary",
  column: "bg-emerald-500/10 text-emerald-700",
  short: "bg-amber-500/10 text-amber-700",
}

export function CourseCard({
  id,
  title,
  description,
  image,
  category,
  provider,
  duration,
  students = 1200,
  isPurchased = false,
  progress,
  price = 500,
  priceLabel,
  paymentConfigured = false,
  statusLabel,
  versionLabel,
  stats = [],
}: CourseCardProps) {
  const { t } = useTranslation()
  const [showPurchaseDialog, setShowPurchaseDialog] = useState(false)
  const categoryLabels = {
    course: t.courses.categoryCourse,
    column: t.courses.categoryColumn,
    short: t.courses.categoryShort,
  }

  const handleClick = (e: React.MouseEvent) => {
    if (!isPurchased) {
      e.preventDefault()
      setShowPurchaseDialog(true)
    }
  }

  const cardContent = (
    <>
      {/* Image Section */}
      <div className="relative aspect-[16/9] overflow-hidden bg-muted">
        <Image
          src={image}
          alt={title}
          fill
          className="object-cover transition-transform duration-500 group-hover:scale-105"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent" />
        
        {/* Category Badge */}
        <Badge
          className={cn(
            "absolute left-4 top-4 border-0",
            categoryStyles[category]
          )}
        >
          {categoryLabels[category]}
        </Badge>

        {/* Purchased Badge or Price */}
        {isPurchased ? (
          <Badge className="absolute right-4 top-4 bg-emerald-500 text-white border-0 gap-1">
            <CheckCircle2 className="h-3 w-3" />
            {t.courses.purchased}
          </Badge>
        ) : (
          <Badge className="absolute right-4 top-4 bg-primary text-white border-0 gap-1">
            <ShoppingCart className="h-3 w-3" />
            {priceLabel || (paymentConfigured ? t.courses.configuredPayment : `$${price}`)}
          </Badge>
        )}

        {/* Play button on hover */}
        <div className="absolute inset-0 flex items-center justify-center opacity-0 transition-opacity duration-300 group-hover:opacity-100">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-white/90 text-primary shadow-lg backdrop-blur-sm transition-transform duration-300 group-hover:scale-110">
            {isPurchased ? (
              <Play className="h-6 w-6 fill-current" />
            ) : (
              <ShoppingCart className="h-6 w-6" />
            )}
          </div>
        </div>
      </div>

      {/* Content Section */}
      <div className="p-5">
        <h3 className="mb-2 text-lg font-semibold text-card-foreground line-clamp-1 group-hover:text-primary transition-colors">
          {title}
        </h3>
        <p className="mb-4 text-sm text-muted-foreground line-clamp-2">{description}</p>

        {(statusLabel || versionLabel) && (
          <div className="mb-4 flex flex-wrap gap-2">
            {statusLabel && <Badge variant="outline">{statusLabel}</Badge>}
            {versionLabel && <Badge variant="outline">{versionLabel}</Badge>}
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
        <div className="flex items-center justify-between text-sm text-muted-foreground">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-1.5">
              <Clock className="h-4 w-4" />
              <span>{duration || `40 ${t.courses.hours}`}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <Users className="h-4 w-4" />
              <span>{students.toLocaleString()}</span>
            </div>
          </div>
        </div>

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
        price={price}
        pipelineId={id}
      />
    </>
  )
}
