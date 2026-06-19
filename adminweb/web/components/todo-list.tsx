import Link from "next/link"
import { cn } from "@/lib/utils"
import { ChevronRight, MessageSquare, FileCheck, XCircle, Clock } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { useTranslation } from "@/lib/useLanguage"

interface TodoItem {
  id: string
  icon: "message" | "file" | "rejected" | "pending"
  title: string
  description?: string
  action: {
    label: string
    href: string
  }
  priority?: "high" | "medium" | "low"
}

const iconMap = {
  message: MessageSquare,
  file: FileCheck,
  rejected: XCircle,
  pending: Clock,
}

const iconStyles = {
  message: "bg-blue-500/10 text-blue-600",
  file: "bg-amber-500/10 text-amber-600",
  rejected: "bg-red-500/10 text-red-500",
  pending: "bg-slate-500/10 text-slate-600",
}

const priorityStyles = {
  high: "bg-red-500/10 text-red-600 border-red-200",
  medium: "bg-amber-500/10 text-amber-600 border-amber-200",
  low: "bg-slate-500/10 text-slate-600 border-slate-200",
}

interface TodoListProps {
  items: TodoItem[]
  className?: string
}

export function TodoList({ items, className }: TodoListProps) {
  const { t, lang } = useTranslation()
  return (
    <div className={cn("rounded-2xl border border-border bg-card shadow-sm", className)}>
      <div className="flex items-center justify-between border-b border-border px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-amber-500/10">
            <Clock className="h-4 w-4 text-amber-600" />
          </div>
          <h3 className="font-semibold text-card-foreground">{t.home.pendingTasks}</h3>
        </div>
        <Badge variant="secondary" className="bg-amber-500/10 text-amber-700 hover:bg-amber-500/20">
          {items.length} {lang === "zh" ? "项" : "Items"}
        </Badge>
      </div>
      <div className="divide-y divide-border">
        {items.map((item) => {
          const Icon = iconMap[item.icon]
          return (
            <div
              key={item.id}
              className="group flex items-center justify-between px-6 py-4 transition-colors hover:bg-muted/50"
            >
              <div className="flex items-center gap-4">
                <div
                  className={cn(
                    "flex h-10 w-10 items-center justify-center rounded-xl transition-transform group-hover:scale-105",
                    iconStyles[item.icon]
                  )}
                >
                  <Icon className="h-5 w-5" />
                </div>
                <div>
                  <p className="font-medium text-card-foreground">{item.title}</p>
                  {item.description && (
                    <p className="text-sm text-muted-foreground">{item.description}</p>
                  )}
                </div>
              </div>
              <Link
                href={item.action.href}
                className="flex items-center gap-1 text-sm font-medium text-primary transition-colors hover:text-primary/80"
              >
                {item.action.label}
                <ChevronRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
              </Link>
            </div>
          )
        })}
      </div>
    </div>
  )
}
