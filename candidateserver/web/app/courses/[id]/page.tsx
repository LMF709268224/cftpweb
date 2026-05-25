import Link from "next/link"
import Image from "next/image"
import { notFound } from "next/navigation"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import {
  ArrowLeft,
  Play,
  Clock,
  Users,
  CheckCircle2,
  Lock,
  BookOpen,
  Award,
  ChevronRight,
} from "lucide-react"

// 课程数据
const coursesData: Record<string, {
  id: string
  title: string
  description: string
  image: string
  duration: string
  students: number
  progress: number
  modules: Array<{
    id: string
    title: string
    description: string
    duration: string
    status: "completed" | "current" | "locked"
    lessons: Array<{
      id: string
      title: string
      duration: string
      completed: boolean
    }>
  }>
}> = {
  cftp: {
    id: "cftp",
    title: "CFtP (Chartered Fintech Practitioner)",
    description: "金融科技专业从业者认证，涵盖区块链、人工智能、大数据等核心领域的系统化学习路径",
    image: "https://images.unsplash.com/photo-1531482615713-2afd69097998?w=800&auto=format&fit=crop&q=60",
    duration: "120小时",
    students: 2850,
    progress: 38,
    modules: [
      {
        id: "l1a",
        title: "L1A Foundation",
        description: "金融科技基础概念与行业概览",
        duration: "20小时",
        status: "completed",
        lessons: [
          { id: "1-1", title: "金融科技导论", duration: "45分钟", completed: true },
          { id: "1-2", title: "传统金融与科技金融", duration: "60分钟", completed: true },
          { id: "1-3", title: "金融科技生态系统", duration: "50分钟", completed: true },
          { id: "1-4", title: "监管科技概述", duration: "45分钟", completed: true },
        ],
      },
      {
        id: "l1b",
        title: "L1B Fintech",
        description: "核心金融科技技术与应用",
        duration: "30小时",
        status: "current",
        lessons: [
          { id: "2-1", title: "区块链基础", duration: "60分钟", completed: true },
          { id: "2-2", title: "智能合约入门", duration: "75分钟", completed: true },
          { id: "2-3", title: "数字货币与支付", duration: "60分钟", completed: false },
          { id: "2-4", title: "去中心化金融 (DeFi)", duration: "90分钟", completed: false },
        ],
      },
      {
        id: "l2a",
        title: "L2A Advanced Analytics",
        description: "高级数据分析与人工智能应用",
        duration: "25小时",
        status: "locked",
        lessons: [
          { id: "3-1", title: "机器学习在金融中的应用", duration: "90分钟", completed: false },
          { id: "3-2", title: "风险建模与评估", duration: "75分钟", completed: false },
          { id: "3-3", title: "量化投资策略", duration: "80分钟", completed: false },
        ],
      },
      {
        id: "l2b",
        title: "L2B Digital Banking",
        description: "数字银行与新兴支付技术",
        duration: "25小时",
        status: "locked",
        lessons: [
          { id: "4-1", title: "数字银行架构", duration: "60分钟", completed: false },
          { id: "4-2", title: "开放银行与API经济", duration: "75分钟", completed: false },
          { id: "4-3", title: "跨境支付创新", duration: "60分钟", completed: false },
        ],
      },
      {
        id: "l3",
        title: "L3 Capstone",
        description: "综合项目与认证考试",
        duration: "20小时",
        status: "locked",
        lessons: [
          { id: "5-1", title: "案例研究", duration: "120分钟", completed: false },
          { id: "5-2", title: "项目实践", duration: "180分钟", completed: false },
          { id: "5-3", title: "认证考试准备", duration: "60分钟", completed: false },
        ],
      },
    ],
  },
  cftx: {
    id: "cftx",
    title: "CFtX 金融科技入门",
    description: "零基础入门金融科技，快速了解行业概况、核心技术和发展趋势",
    image: "https://images.unsplash.com/photo-1551434678-e076c223a692?w=800&auto=format&fit=crop&q=60",
    duration: "20小时",
    students: 5420,
    progress: 100,
    modules: [
      {
        id: "intro",
        title: "入门概述",
        description: "金融科技行业入门",
        duration: "5小时",
        status: "completed",
        lessons: [
          { id: "1-1", title: "什么是金融科技", duration: "30分钟", completed: true },
          { id: "1-2", title: "行业发展历程", duration: "45分钟", completed: true },
        ],
      },
      {
        id: "tech",
        title: "核心技术",
        description: "关键技术介绍",
        duration: "8小时",
        status: "completed",
        lessons: [
          { id: "2-1", title: "区块链简介", duration: "40分钟", completed: true },
          { id: "2-2", title: "人工智能简介", duration: "40分钟", completed: true },
          { id: "2-3", title: "大数据简介", duration: "40分钟", completed: true },
        ],
      },
      {
        id: "future",
        title: "未来趋势",
        description: "行业展望与职业发展",
        duration: "7小时",
        status: "completed",
        lessons: [
          { id: "3-1", title: "行业趋势分析", duration: "50分钟", completed: true },
          { id: "3-2", title: "职业发展路径", duration: "40分钟", completed: true },
        ],
      },
    ],
  },
}

export function generateStaticParams() {
  return Object.keys(coursesData).map((id) => ({
    id: id,
  }))
}

export default async function CourseDetailPage({
  params
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const course = coursesData[id]

  if (!course) {
    notFound()
  }

  const completedLessons = course.modules.reduce(
    (acc, m) => acc + m.lessons.filter((l) => l.completed).length,
    0
  )
  const totalLessons = course.modules.reduce((acc, m) => acc + m.lessons.length, 0)

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />

      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          {/* Back Button */}
          <Link
            href="/courses"
            className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground mb-6 transition-colors"
          >
            <ArrowLeft className="h-4 w-4" />
            返回课程列表
          </Link>

          {/* Course Header */}
          <div className="mb-8 flex flex-col lg:flex-row gap-8">
            <div className="relative aspect-video w-full lg:w-96 shrink-0 overflow-hidden rounded-2xl bg-muted">
              <Image
                src={course.image}
                alt={course.title}
                fill
                className="object-cover"
              />
              <div className="absolute inset-0 flex items-center justify-center bg-black/30">
                <Button size="lg" className="gap-2 rounded-full">
                  <Play className="h-5 w-5 fill-current" />
                  继续学习
                </Button>
              </div>
            </div>

            <div className="flex-1">
              <Badge className="mb-3 bg-emerald-500/10 text-emerald-700 border-0">
                <CheckCircle2 className="mr-1 h-3 w-3" />
                已购买
              </Badge>
              <h1 className="text-2xl font-bold text-foreground mb-2">{course.title}</h1>
              <p className="text-muted-foreground mb-6">{course.description}</p>

              <div className="flex flex-wrap gap-6 text-sm text-muted-foreground mb-6">
                <div className="flex items-center gap-1.5">
                  <Clock className="h-4 w-4" />
                  <span>{course.duration}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <Users className="h-4 w-4" />
                  <span>{course.students.toLocaleString()} 名学员</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <BookOpen className="h-4 w-4" />
                  <span>{course.modules.length} 个模块</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <Award className="h-4 w-4" />
                  <span>{totalLessons} 节课程</span>
                </div>
              </div>

              {/* Progress */}
              <div className="rounded-xl bg-muted/50 p-4">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm text-muted-foreground">学习进度</span>
                  <span className="font-semibold text-primary">{course.progress}%</span>
                </div>
                <div className="h-2 w-full rounded-full bg-muted overflow-hidden mb-2">
                  <div
                    className="h-full rounded-full bg-primary transition-all"
                    style={{ width: `${course.progress}%` }}
                  />
                </div>
                <p className="text-xs text-muted-foreground">
                  已完成 {completedLessons}/{totalLessons} 节课程
                </p>
              </div>
            </div>
          </div>

          {/* Course Modules */}
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-foreground">课程大纲</h2>
            {course.modules.map((module, index) => (
              <div
                key={module.id}
                className={cn(
                  "rounded-2xl border bg-card overflow-hidden transition-all",
                  module.status === "current" && "border-primary shadow-sm",
                  module.status === "locked" && "opacity-60"
                )}
              >
                <div className="flex items-center justify-between p-5 border-b border-border">
                  <div className="flex items-center gap-4">
                    <div
                      className={cn(
                        "flex h-10 w-10 items-center justify-center rounded-xl font-semibold",
                        module.status === "completed" && "bg-emerald-500/10 text-emerald-600",
                        module.status === "current" && "bg-primary/10 text-primary",
                        module.status === "locked" && "bg-muted text-muted-foreground"
                      )}
                    >
                      {module.status === "completed" ? (
                        <CheckCircle2 className="h-5 w-5" />
                      ) : module.status === "locked" ? (
                        <Lock className="h-5 w-5" />
                      ) : (
                        index + 1
                      )}
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <h3 className="font-semibold text-card-foreground">{module.title}</h3>
                        {module.status === "current" && (
                          <Badge className="bg-primary/10 text-primary border-0 text-xs">
                            进行中
                          </Badge>
                        )}
                      </div>
                      <p className="text-sm text-muted-foreground">{module.description}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="text-sm text-muted-foreground">{module.duration}</span>
                    <ChevronRight className="h-5 w-5 text-muted-foreground" />
                  </div>
                </div>

                {/* Lessons */}
                <div className="divide-y divide-border">
                  {module.lessons.map((lesson) => (
                    <div
                      key={lesson.id}
                      className={cn(
                        "flex items-center justify-between px-5 py-3 transition-colors",
                        module.status !== "locked" && "hover:bg-muted/50 cursor-pointer"
                      )}
                    >
                      <div className="flex items-center gap-3">
                        {lesson.completed ? (
                          <CheckCircle2 className="h-4 w-4 text-emerald-500" />
                        ) : module.status === "locked" ? (
                          <Lock className="h-4 w-4 text-muted-foreground" />
                        ) : (
                          <Play className="h-4 w-4 text-primary" />
                        )}
                        <span
                          className={cn(
                            "text-sm",
                            lesson.completed
                              ? "text-muted-foreground"
                              : "text-card-foreground"
                          )}
                        >
                          {lesson.title}
                        </span>
                      </div>
                      <span className="text-xs text-muted-foreground">{lesson.duration}</span>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      </main>
    </div>
  )
}
