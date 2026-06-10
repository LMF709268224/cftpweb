"use client"

import React, { useEffect, useState } from "react"
import { useSearchParams, useRouter } from "next/navigation"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { CheckCircle2, ChevronLeft, Clock, FileText, AlertCircle } from "lucide-react"
import { Sidebar } from "@/components/sidebar"

export default function QuizPage() {
  const searchParams = useSearchParams()
  const attemptId = searchParams.get("attemptId") || ""
  const router = useRouter()
  const { t } = useTranslation()

  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [paper, setPaper] = useState<any>(null)
  const [answers, setAnswers] = useState<Record<string, string[]>>({})
  const [result, setResult] = useState<any>(null)

  useEffect(() => {
    if (!attemptId) return
    loadPaper()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [attemptId])

  const loadPaper = async () => {
    setLoading(true)
    try {
      const res = await apiClient(`/api/quizzes/attempts/${attemptId}/paper`)
      setPaper(res)
    } catch {
      // apiClient will show error toast automatically
    } finally {
      setLoading(false)
    }
  }

  const handleSelectOption = (questionId: string, optionId: string, isMultipleChoice: boolean) => {
    setAnswers((prev) => {
      const current = prev[questionId] || []
      if (!isMultipleChoice) {
        return { ...prev, [questionId]: [optionId] }
      }
      if (current.includes(optionId)) {
        return { ...prev, [questionId]: current.filter((id) => id !== optionId) }
      }
      return { ...prev, [questionId]: [...current, optionId] }
    })
  }

  const submitQuiz = async () => {
    if (!paper?.questions) return
    
    const submissions = Object.entries(answers).map(([questionId, selectedOptionIds]) => ({
      question_id: questionId,
      selected_option_ids: selectedOptionIds,
    }))

    setSubmitting(true)
    try {
      const res = await apiClient(`/api/quizzes/attempts/${attemptId}/submit`, {
        method: "POST",
        body: JSON.stringify({ submissions }),
      })
      setResult(res)
      toast.success(t.learning?.quizSubmittedDesc || t.common.success)
    } catch {
      // apiClient handles toast
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-background flex">
        <Sidebar />
        <main className="flex-1 ml-64 p-8 flex items-center justify-center">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-r-transparent"></div>
        </main>
      </div>
    )
  }

  if (!paper) {
    return (
      <div className="min-h-screen bg-background flex">
        <Sidebar />
        <main className="flex-1 ml-64 p-8 flex flex-col items-center justify-center gap-4">
          <AlertCircle className="h-12 w-12 text-destructive" />
          <h2 className="text-lg font-semibold text-foreground">{t.learning?.quizNotFound}</h2>
          <Button onClick={() => router.back()}>
            <ChevronLeft className="mr-2 h-4 w-4" /> {t.common.back}
          </Button>
        </main>
      </div>
    )
  }

  if (result) {
    return (
      <div className="min-h-screen bg-background flex">
        <Sidebar />
        <main className="flex-1 ml-64 p-8">
          <div className="mx-auto max-w-2xl py-12">
            <Card className="border-t-4 border-t-primary shadow-lg">
              <CardHeader className="text-center">
                <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
                  <CheckCircle2 className="h-8 w-8 text-primary" />
                </div>
                <CardTitle className="text-2xl font-bold">{t.learning?.quizCompleted}</CardTitle>
                <CardDescription>{t.learning?.quizSubmittedDesc}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="rounded-xl border bg-card p-6 shadow-sm">
                  <div className="flex flex-col items-center justify-center gap-2">
                    <span className="text-sm font-medium text-muted-foreground">{t.learning?.quizScore}</span>
                    <div className="flex items-baseline gap-2">
                      <span className="text-4xl font-bold text-foreground">{result.score || 0}</span>
                      <span className="text-xl text-muted-foreground">/ {result.max_score || 0}</span>
                    </div>
                    <div className={`mt-2 rounded-full px-4 py-1 text-sm font-semibold ${result.is_passed ? 'bg-emerald-100 text-emerald-700' : 'bg-rose-100 text-rose-700'}`}>
                      {result.is_passed ? t.learning?.quizPassed : t.learning?.quizFailed}
                    </div>
                  </div>
                </div>
              </CardContent>
              <CardFooter className="flex justify-center pb-8">
                <Button size="lg" onClick={() => router.back()} className="px-8">
                  <ChevronLeft className="mr-2 h-4 w-4" /> {t.learning?.quizReturn}
                </Button>
              </CardFooter>
            </Card>
          </div>
        </main>
      </div>
    )
  }

  const questions = paper.questions || []
  const allAnswered = questions.every((q: any) => (answers[q.question_id]?.length || 0) > 0)

  return (
    <div className="min-h-screen bg-background flex">
      <Sidebar />
      <main className="flex-1 ml-64 p-8 bg-muted/10">
        <div className="mx-auto max-w-3xl space-y-8">
          
          {/* Header */}
          <div className="rounded-2xl border bg-card p-6 sm:p-8 shadow-sm">
            <Button variant="ghost" size="sm" onClick={() => router.back()} className="-ml-2 mb-4 text-muted-foreground">
              <ChevronLeft className="mr-2 h-4 w-4" /> {t.learning?.quizReturn}
            </Button>
            <div className="flex items-start justify-between gap-4">
              <div>
                <h1 className="text-2xl sm:text-3xl font-bold text-foreground">{paper.title || t.learning?.quizPrefix}</h1>
                {paper.description && (
                  <p className="mt-2 text-muted-foreground">{paper.description}</p>
                )}
              </div>
              {paper.time_limit > 0 && (
                <div className="flex shrink-0 items-center gap-2 rounded-full border bg-muted/30 px-3 py-1.5 text-sm font-medium text-foreground">
                  <Clock className="h-4 w-4 text-primary" />
                  {paper.time_limit} {t.learning?.quizMin}
                </div>
              )}
            </div>
          </div>

          {/* Questions */}
          <div className="space-y-6">
            {questions.map((question: any, index: number) => {
              const isMultipleChoice = question.question_type === 2 // QUIZ_QUESTION_TYPE_MULTIPLE_CHOICE
              const selectedOptions = answers[question.question_id] || []
              
              const questionCountLabel = (t.learning?.quizQuestionCount || "").replace("{{current}}", String(index + 1)).replace("{{total}}", String(questions.length))
              
              return (
                <Card key={question.question_id} className="overflow-hidden border-border/50 shadow-sm transition-all hover:border-primary/20 hover:shadow-md">
                  <div className="bg-muted/30 px-6 py-3 border-b text-sm font-medium text-muted-foreground flex items-center justify-between">
                    <span>{questionCountLabel}</span>
                    <span className="rounded bg-background px-2 py-0.5 border text-xs">{question.points || 0} {t.learning?.quizPts}</span>
                  </div>
                  <CardContent className="p-6">
                    <h3 className="mb-6 text-lg font-medium text-foreground leading-relaxed">
                      {question.question_text}
                    </h3>
                    <div className="space-y-3">
                      {(question.options || []).map((option: any) => {
                        const isSelected = selectedOptions.includes(option.option_id)
                        return (
                          <button
                            key={option.option_id}
                            onClick={() => handleSelectOption(question.question_id, option.option_id, isMultipleChoice)}
                            className={`flex w-full items-start gap-3 rounded-xl border p-4 text-left transition-all ${
                              isSelected 
                                ? "border-primary bg-primary/5 ring-1 ring-primary/20" 
                                : "border-border hover:bg-muted/50 hover:border-border/80"
                            }`}
                          >
                            <div className={`mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center border ${isMultipleChoice ? 'rounded-md' : 'rounded-full'} ${isSelected ? 'border-primary bg-primary text-primary-foreground' : 'border-muted-foreground/30 bg-background'}`}>
                              {isSelected && <CheckCircle2 className="h-3.5 w-3.5" />}
                            </div>
                            <span className={`text-sm ${isSelected ? 'font-medium text-foreground' : 'text-muted-foreground'}`}>
                              {option.option_text}
                            </span>
                          </button>
                        )
                      })}
                    </div>
                  </CardContent>
                </Card>
              )
            })}
          </div>

          {/* Footer Actions */}
          <div className="sticky bottom-4 rounded-2xl border bg-card/80 p-4 backdrop-blur-md shadow-lg sm:p-6 flex flex-col sm:flex-row items-center justify-between gap-4">
            <div className="text-sm text-muted-foreground">
              {(t.learning?.quizAnsweredCount || "").replace("{{current}}", String(Object.keys(answers).length)).replace("{{total}}", String(questions.length))}
            </div>
            <Button 
              size="lg" 
              className="w-full sm:w-auto px-8" 
              onClick={submitQuiz} 
              disabled={submitting || !allAnswered}
            >
              {submitting ? (
                <div className="flex items-center gap-2">
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground border-r-transparent"></div>
                  {t.common.loading}
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <FileText className="h-4 w-4" />
                  {t.learning?.quizSubmit}
                </div>
              )}
            </Button>
          </div>

        </div>
      </main>
    </div>
  )
}
