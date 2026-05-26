import React from "react"
import { cn } from "@/lib/utils"
import { Crown, Check, RefreshCw, Calendar } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { useTranslation } from "@/lib/useLanguage"

interface MembershipCardProps {
  level: "basic" | "certified" | "premium"
  levelName: string
  expiryDate: string
  subscribedDate: string
  isAutoRenew?: boolean
  className?: string
}

const levelStyles = {
  basic: {
    gradient: "from-slate-600 via-slate-500 to-slate-600",
    accent: "bg-slate-400/20",
    icon: "text-slate-300",
  },
  certified: {
    gradient: "from-primary via-primary/90 to-primary",
    accent: "bg-white/10",
    icon: "text-white/90",
  },
  premium: {
    gradient: "from-amber-500 via-yellow-500 to-amber-500",
    accent: "bg-white/15",
    icon: "text-white",
  },
}

export function MembershipCard({
  level,
  levelName,
  expiryDate,
  subscribedDate,
  isAutoRenew = true,
  className,
}: MembershipCardProps) {
  const { t } = useTranslation()
  const styles = levelStyles[level]

  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-2xl p-6 text-white shadow-xl",
        className
      )}
    >
      {/* Background Gradient */}
      <div className={cn("absolute inset-0 bg-gradient-to-br", styles.gradient)} />
      
      {/* Decorative Elements */}
      <div className="absolute -right-12 -top-12 h-48 w-48 rounded-full bg-white/5" />
      <div className="absolute -bottom-8 -left-8 h-32 w-32 rounded-full bg-white/5" />
      <div className="absolute right-8 top-1/2 h-24 w-24 -translate-y-1/2 rounded-full bg-white/5" />

      {/* Content */}
      <div className="relative">
        {/* Header */}
        <div className="mb-6 flex items-start justify-between">
          <div>
            <p className="text-sm font-medium text-white/70">{t.membership.currentMember}</p>
            <h2 className="mt-1 text-2xl font-bold">{levelName}</h2>
          </div>
          <div className={cn("rounded-full p-3", styles.accent)}>
            <Crown className={cn("h-8 w-8", styles.icon)} />
          </div>
        </div>

        {/* Dates */}
        <div className="mb-4 grid grid-cols-2 gap-4">
          <div className="rounded-xl bg-white/10 p-3 backdrop-blur-sm">
            <div className="flex items-center gap-2 text-xs text-white/70 mb-1">
              <Calendar className="h-3.5 w-3.5" />
              <span>{t.membership.expiryDate}</span>
            </div>
            <p className="font-semibold">{expiryDate}</p>
          </div>
          <div className="rounded-xl bg-white/10 p-3 backdrop-blur-sm">
            <div className="flex items-center gap-2 text-xs text-white/70 mb-1">
              <Calendar className="h-3.5 w-3.5" />
              <span>{t.membership.subscribedOn}</span>
            </div>
            <p className="font-semibold">{subscribedDate}</p>
          </div>
        </div>

        {/* Auto Renew Badge */}
        {isAutoRenew && (
          <Badge className="bg-white/20 text-white border-0 hover:bg-white/30 gap-1.5">
            <RefreshCw className="h-3 w-3" />
            {t.membership.autoRenew}
            <Check className="h-3 w-3" />
          </Badge>
        )}
      </div>
    </div>
  )
}
