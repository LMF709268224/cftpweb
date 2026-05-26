import React from "react"
import Link from "next/link"
import { cn } from "@/lib/utils"
import type { LucideIcon } from "lucide-react"

interface StatsCardProps {
  title: string
  value: string | number
  icon: LucideIcon
  description?: string
  href?: string
  trend?: {
    value: number
    isPositive: boolean
  }
  variant?: "default" | "primary" | "success" | "warning" | "info"
}

const variantStyles = {
  default: {
    bg: "bg-card",
    iconBg: "bg-muted",
    iconColor: "text-muted-foreground",
  },
  primary: {
    bg: "bg-card",
    iconBg: "bg-primary/10",
    iconColor: "text-primary",
  },
  success: {
    bg: "bg-card",
    iconBg: "bg-emerald-500/10",
    iconColor: "text-emerald-600",
  },
  warning: {
    bg: "bg-card",
    iconBg: "bg-amber-500/10",
    iconColor: "text-amber-600",
  },
  info: {
    bg: "bg-card",
    iconBg: "bg-blue-500/10",
    iconColor: "text-blue-600",
  },
}

export function StatsCard({
  title,
  value,
  icon: Icon,
  description,
  href,
  trend,
  variant = "default",
}: StatsCardProps) {
  const styles = variantStyles[variant]

  const content = (
    <>
      {/* Background decoration */}
      <div className="absolute -right-8 -top-8 h-32 w-32 rounded-full bg-gradient-to-br from-primary/5 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
      
      <div className="relative flex items-start justify-between">
        <div className="space-y-2">
          <p className="text-sm font-medium text-muted-foreground">{title}</p>
          <p className="text-3xl font-bold tracking-tight text-card-foreground">{value}</p>
          {description && (
            <p className="text-xs text-muted-foreground">{description}</p>
          )}
          {trend && (
            <div className="flex items-center gap-1 text-xs">
              <span
                className={cn(
                  "font-medium",
                  trend.isPositive ? "text-emerald-600" : "text-red-500"
                )}
              >
                {trend.isPositive ? "+" : "-"}{trend.value}%
              </span>
              <span className="text-muted-foreground">较上�?/span>
            </div>
          )}
        </div>
        <div
          className={cn(
            "flex h-12 w-12 shrink-0 items-center justify-center rounded-xl transition-transform duration-300 group-hover:scale-110",
            styles.iconBg
          )}
        >
          <Icon className={cn("h-6 w-6", styles.iconColor)} />
        </div>
      </div>
    </>
  )

  const className = cn(
    "group relative overflow-hidden rounded-2xl border border-border p-6 shadow-sm transition-all duration-300 hover:shadow-md hover:border-primary/20",
    styles.bg,
    href && "cursor-pointer"
  )

  if (href) {
    return (
      <Link href={href} className={cn(className, "block")}>
        {content}
      </Link>
    )
  }

  return <div className={className}>{content}</div>
}
