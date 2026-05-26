import React from "react"
import Link from "next/link"
import { cn } from "@/lib/utils"
import { BookOpen, ArrowRight, Play, CheckCircle2 } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useTranslation } from "@/lib/useLanguage"

interface LearningProgressProps {
  courseName: string
  courseDescription?: string
  currentModule: string
  progress: number
  totalModules?: number
  completedModules?: number
  className?: string
}

export function LearningProgress({
  courseName,
  courseDescription,
  currentModule,
  progress,
  totalModules = 5,
  completedModules = 2,
  className,
}: LearningProgressProps) {
  const { t } = useTranslation()
  return (
    <div
      className={cn(
        "rounded-2xl border border-border bg-card shadow-sm overflow-hidden",
        className
      )}
    >
      <div className="flex items-center justify-between border-b border-border px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10">
            <BookOpen className="h-4 w-4 text-primary" />
          </div>
          <h3 className="font-semibold text-card-foreground">{t.home.currentLearning}</h3>
        </div>
        <Badge className="bg-emerald-500/10 text-emerald-700 hover:bg-emerald-500/20 border-0">
          {t.home.learningInProgress}
        </Badge>
      </div>

      <div className="p-6">
        <div className="mb-6">
          <h4 className="text-lg font-semibold text-card-foreground mb-1">{courseName}</h4>
          {courseDescription && (
            <p className="text-sm text-muted-foreground">{courseDescription}</p>
          )}
        </div>

        {/* Progress Section */}
        <div className="mb-6 space-y-3">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">{t.home.totalProgress}</span>
            <span className="font-semibold text-primary">{progress}%</span>
          </div>
          <div className="relative h-2.5 w-full overflow-hidden rounded-full bg-muted">
            <div
              className="absolute left-0 top-0 h-full rounded-full bg-gradient-to-r from-primary to-primary/80 transition-all duration-500"
              style={{ width: `${progress}%` }}
            />
          </div>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500" />
            <span>{t.home.completedModulesText.replace("{{completed}}", String(completedModules)).replace("{{total}}", String(totalModules))}</span>
          </div>
        </div>

        {/* Current Module */}
        <div className="rounded-xl bg-muted/50 p-4 mb-6">
          <div className="flex items-center gap-2 text-xs text-muted-foreground mb-2">
            <Play className="h-3.5 w-3.5" />
            <span>{t.home.currentStage}</span>
          </div>
          <p className="font-medium text-card-foreground">{currentModule}</p>
        </div>

        {/* Action Button */}
        <Button asChild className="w-full group">
          <Link href="/courses/cftp">
            {t.home.continueLearning}
            <ArrowRight className="ml-2 h-4 w-4 transition-transform group-hover:translate-x-1" />
          </Link>
        </Button>
      </div>
    </div>
  )
}
