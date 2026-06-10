"use client"

import React, { Suspense, useState } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import Link from "next/link"
import { ArrowLeft, Send } from "lucide-react"
import { toast } from "sonner"

import { useTranslation } from "@/lib/useLanguage"
import { Sidebar } from "@/components/sidebar"
import { apiClient } from "@/lib/apiClient"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

function ExamSignupContent() {
  const { t } = useTranslation()
  const router = useRouter()
  const searchParams = useSearchParams()
  const unitId = searchParams.get("unitId") || ""
  const pipelineId = searchParams.get("pipelineId") || ""
  const courseId = searchParams.get("courseId") || ""

  const [loading, setLoading] = useState(false)
  const [formData, setFormData] = useState({
    first_name: "",
    middle_name: "",
    last_name: "",
    email: "",
    gender: "",
    birthdate: "",
    country: "",
    province: "",
    city: "",
    address: "",
    postal_code: "",
    home_phone: "",
    work_phone: "",
  })

  React.useEffect(() => {
    const fetchProfile = async () => {
      try {
        const res = await apiClient("/api/user/me")
        if (res) {
          setFormData(prev => ({
            ...prev,
            email: res.email || prev.email,
            gender: res.gender || prev.gender,
            birthdate: res.birthday ? res.birthday.split('T')[0] : prev.birthdate,
            first_name: res.first_name || (res.real_name ? res.real_name.split(' ')[0] : prev.first_name),
            last_name: res.last_name || (res.real_name && res.real_name.includes(' ') 
              ? res.real_name.split(' ').slice(1).join(' ') 
              : prev.last_name),
            home_phone: res.phone || prev.home_phone,
            country: res.region || prev.country,
            city: res.location || prev.city,
            address: (res.address && res.address.length > 0) ? res.address.join(', ') : prev.address,
          }))
        }
      } catch (err) {
        console.error("Failed to load user profile", err)
      }
    }
    fetchProfile()
  }, [])

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData((prev) => ({ ...prev, [e.target.name]: e.target.value }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!unitId) {
      toast.error(t.common.error)
      return
    }

    // Local required validation
    const requiredFields = [
      { key: "first_name", label: t.examSignup.formFirstName },
      { key: "last_name", label: t.examSignup.formLastName },
      { key: "email", label: t.examSignup.formEmail },
      { key: "gender", label: t.examSignup.formGender },
      { key: "birthdate", label: t.examSignup.formBirthdate },
      { key: "country", label: t.examSignup.formCountry },
      { key: "province", label: t.examSignup.formProvince },
      { key: "city", label: t.examSignup.formCity },
      { key: "address", label: t.examSignup.formAddress },
      { key: "postal_code", label: t.examSignup.formPostalCode },
      { key: "home_phone", label: t.examSignup.formHomePhone },
      { key: "work_phone", label: t.examSignup.formWorkPhone },
    ]

    for (const field of requiredFields) {
      if (!formData[field.key as keyof typeof formData].trim()) {
        toast.error(t.examSignup.validationRequired.replace("{{field}}", field.label))
        return
      }
    }

    setLoading(true)
    try {
      await apiClient(`/api/exams/units/${encodeURIComponent(unitId)}/signup`, {
        method: "POST",
        body: JSON.stringify(formData),
      })
      toast.success(t.examSignup.success)
      router.push("/exams")
    } catch (err: any) {
      // apiClient handles standard errors
    } finally {
      setLoading(false)
    }
  }

  const backLink = pipelineId
    ? `/courses/detail?id=${encodeURIComponent(pipelineId)}`
    : courseId
      ? `/courses/learn?courseId=${encodeURIComponent(courseId)}`
      : "/courses"

  return (
    <div className="min-h-screen bg-background flex">
      <Sidebar />
      <main className="flex-1 ml-64 p-8">
        <Link
          href={backLink}
          className="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          {t.examSignup.backToCourse}
        </Link>

        <div className="mb-8 max-w-2xl">
          <h1 className="text-3xl font-bold tracking-tight text-foreground">{t.examSignup.title}</h1>
          <p className="mt-2 text-muted-foreground">{t.examSignup.subtitle}</p>
        </div>

        <div className="rounded-2xl border border-border bg-card p-6 shadow-sm max-w-2xl">
          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="first_name">{t.examSignup.formFirstName} <span className="text-destructive">*</span></Label>
                <Input id="first_name" name="first_name" value={formData.first_name} onChange={handleChange} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="last_name">{t.examSignup.formLastName} <span className="text-destructive">*</span></Label>
                <Input id="last_name" name="last_name" value={formData.last_name} onChange={handleChange} required />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="middle_name">{t.examSignup.formMiddleName}</Label>
              <Input id="middle_name" name="middle_name" value={formData.middle_name} onChange={handleChange} />
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="email">{t.examSignup.formEmail} <span className="text-destructive">*</span></Label>
                <Input id="email" name="email" type="email" value={formData.email} onChange={handleChange} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="gender">{t.examSignup.formGender} <span className="text-destructive">*</span></Label>
                <Input id="gender" name="gender" placeholder="Male / Female" value={formData.gender} onChange={handleChange} required />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="birthdate">{t.examSignup.formBirthdate} <span className="text-destructive">*</span></Label>
              <Input id="birthdate" name="birthdate" type="date" value={formData.birthdate} onChange={handleChange} required />
            </div>

            <div className="grid gap-4 sm:grid-cols-3">
              <div className="space-y-2">
                <Label htmlFor="country">{t.examSignup.formCountry} <span className="text-destructive">*</span></Label>
                <Input id="country" name="country" value={formData.country} onChange={handleChange} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="province">{t.examSignup.formProvince} <span className="text-destructive">*</span></Label>
                <Input id="province" name="province" value={formData.province} onChange={handleChange} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="city">{t.examSignup.formCity} <span className="text-destructive">*</span></Label>
                <Input id="city" name="city" value={formData.city} onChange={handleChange} required />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="address">{t.examSignup.formAddress} <span className="text-destructive">*</span></Label>
              <Input id="address" name="address" value={formData.address} onChange={handleChange} required />
            </div>

            <div className="space-y-2">
              <Label htmlFor="postal_code">{t.examSignup.formPostalCode} <span className="text-destructive">*</span></Label>
              <Input id="postal_code" name="postal_code" value={formData.postal_code} onChange={handleChange} required />
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="work_phone">{t.examSignup.formWorkPhone} <span className="text-destructive">*</span></Label>
                <Input id="work_phone" name="work_phone" type="tel" value={formData.work_phone} onChange={handleChange} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="home_phone">{t.examSignup.formHomePhone} <span className="text-destructive">*</span></Label>
                <Input id="home_phone" name="home_phone" type="tel" value={formData.home_phone} onChange={handleChange} required />
              </div>
            </div>

            <div className="flex justify-end pt-4">
              <Button type="submit" disabled={loading} className="w-full sm:w-auto">
                {loading ? t.examSignup.submitting : (
                  <>
                    <Send className="mr-2 h-4 w-4" />
                    {t.examSignup.submit}
                  </>
                )}
              </Button>
            </div>
          </form>
        </div>
      </main>
    </div>
  )
}

export default function ExamSignupPage() {
  return (
    <Suspense fallback={null}>
      <ExamSignupContent />
    </Suspense>
  )
}
