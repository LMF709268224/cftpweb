"use client"

import { useState, useEffect } from "react"
import { useSearchParams } from "next/navigation"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { apiClient } from "@/lib/apiClient"
import { getErrorMessage } from "@/lib/errorCodes"
import { toast } from "sonner"
import { Loader2 } from "lucide-react"
import { useTranslation } from "@/lib/useLanguage"
import { getMessage } from "@/lib/messages"
import { Sidebar } from "@/components/sidebar"

export default function SettingsPage() {
  const searchParams = useSearchParams()
  const defaultTab = searchParams.get("tab") || "profile"
  const { t, lang } = useTranslation()

  // Profile State
  const [name, setName] = useState("")
  const [displayName, setDisplayName] = useState("")
  const [email, setEmail] = useState("")
  const [affiliation, setAffiliation] = useState("")
  const [title, setTitle] = useState("")
  const [realName, setRealName] = useState("")
  const [bio, setBio] = useState("")
  const [gender, setGender] = useState("")
  const [birthday, setBirthday] = useState("")
  const [education, setEducation] = useState("")

  const [isProfileLoading, setIsProfileLoading] = useState(false)

  // Password State
  const [oldPassword, setOldPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [isPasswordLoading, setIsPasswordLoading] = useState(false)

  useEffect(() => {
    const fetchProfile = async () => {
      try {
        const payload = await apiClient("/api/user/me")
        if (payload) {
          setName(payload.name || "")
          setDisplayName(payload.display_name || "")
          setEmail(payload.email || "")
          setAffiliation(payload.affiliation || "")
          setTitle(payload.title || "")
          setRealName(payload.real_name || "")
          setBio(payload.bio || "")
          setGender(payload.gender || "")
          setBirthday(payload.birthday || "")
          setEducation(payload.education || "")
        }
      } catch (error) {
        // apiClient 自动抛错，这里不需要做什么
      }
    }
    fetchProfile()
  }, [])

  const handleUpdateProfile = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsProfileLoading(true)
    try {
      await apiClient("/api/user/profile", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          display_name: displayName,
          email: email,
          affiliation: affiliation,
          title: title,
          real_name: realName,
          bio: bio,
          gender: gender,
          birthday: birthday,
          education: education,
        })
      })
      // 走到这里说明 apiClient 没抛错
      toast.success(getMessage("PROFILE_UPDATE_SUCCESS", lang))
      if (displayName) {
        localStorage.setItem("user_name", displayName)
        window.dispatchEvent(new Event("storage"))
      }
    } catch (error) {
      // 已经在 apiClient 里 toast 过了，直接 catch 住别让整个页面崩溃即可
    } finally {
      setIsProfileLoading(false)
    }
  }

  const handleUpdatePassword = async (e: React.FormEvent) => {
    e.preventDefault()
    if (newPassword !== confirmPassword) {
      toast.error(getMessage("PASSWORD_MISMATCH", lang))
      return
    }
    setIsPasswordLoading(true)
    try {
      await apiClient("/api/user/password", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ old_password: oldPassword, new_password: newPassword })
      })

      toast.success(getMessage("PASSWORD_UPDATE_SUCCESS", lang))
      localStorage.removeItem("is_authenticated")
      localStorage.removeItem("user_name")
      setTimeout(() => {
        window.location.href = "/login"
      }, 1500)
    } catch (error) {
      // 已经在 apiClient 报过错了
    } finally {
      setIsPasswordLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-background flex">
      <Sidebar />
      <main className="flex-1 ml-64 p-8 pt-6 space-y-4">
      <div className="flex items-center justify-between space-y-2">
        <h2 className="text-3xl font-bold tracking-tight">{t.settings.title}</h2>
      </div>

      <Tabs defaultValue={defaultTab} className="space-y-4">
        <TabsList>
          <TabsTrigger value="profile">{t.settings.profileTab}</TabsTrigger>
          <TabsTrigger value="account">{t.settings.accountTab}</TabsTrigger>
        </TabsList>

        <TabsContent value="profile" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>{t.settings.profileTab}</CardTitle>
              <CardDescription>
                {t.settings.profileDesc}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleUpdateProfile} className="space-y-4 max-w-2xl">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="username">{t.settings.loginId}</Label>
                    <Input id="username" value={name} disabled className="bg-muted" />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="email">{t.settings.email}</Label>
                    <Input id="email" value={email} disabled className="bg-muted" />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="displayName">{t.settings.displayName}</Label>
                    <Input
                      id="displayName"
                      value={displayName}
                      onChange={(e) => setDisplayName(e.target.value)}
                      placeholder={t.settings.displayNamePlaceholder}
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="realName">{t.settings.realName}</Label>
                    <Input
                      id="realName"
                      value={realName}
                      onChange={(e) => setRealName(e.target.value)}
                      placeholder={t.settings.realNamePlaceholder}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="gender">{t.settings.gender}</Label>
                    <Input
                      id="gender"
                      value={gender}
                      onChange={(e) => setGender(e.target.value)}
                      placeholder={t.settings.genderPlaceholder}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="birthday">{t.settings.birthday}</Label>
                    <Input
                      id="birthday"
                      type="date"
                      value={birthday}
                      onChange={(e) => setBirthday(e.target.value)}
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="affiliation">{t.settings.affiliation}</Label>
                    <Input
                      id="affiliation"
                      value={affiliation}
                      onChange={(e) => setAffiliation(e.target.value)}
                      placeholder={t.settings.affiliationPlaceholder}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="title">{t.settings.jobTitle}</Label>
                    <Input
                      id="title"
                      value={title}
                      onChange={(e) => setTitle(e.target.value)}
                      placeholder={t.settings.jobTitlePlaceholder}
                    />
                  </div>

                  <div className="space-y-2 md:col-span-2">
                    <Label htmlFor="education">{t.settings.education}</Label>
                    <Input
                      id="education"
                      value={education}
                      onChange={(e) => setEducation(e.target.value)}
                      placeholder={t.settings.educationPlaceholder}
                    />
                  </div>

                  <div className="space-y-2 md:col-span-2">
                    <Label htmlFor="bio">{t.settings.bio}</Label>
                    <Textarea
                      id="bio"
                      value={bio}
                      onChange={(e) => setBio(e.target.value)}
                      placeholder={t.settings.bioPlaceholder}
                      className="resize-none"
                      rows={3}
                    />
                  </div>
                </div>

                <div className="pt-4">
                  <Button type="submit" disabled={isProfileLoading}>
                    {isProfileLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                    {t.common.save}
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="account" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>{t.settings.updatePassword}</CardTitle>
              <CardDescription>
                {t.settings.updatePasswordDesc}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleUpdatePassword} className="space-y-4 max-w-xl">
                <div className="space-y-2">
                  <Label htmlFor="oldPassword">{t.settings.currentPassword}</Label>
                  <Input
                    id="oldPassword"
                    type="password"
                    value={oldPassword}
                    onChange={(e) => setOldPassword(e.target.value)}
                    required
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="newPassword">{t.settings.newPassword}</Label>
                  <Input
                    id="newPassword"
                    type="password"
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    required
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="confirmPassword">{t.settings.confirmNewPassword}</Label>
                  <Input
                    id="confirmPassword"
                    type="password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    required
                  />
                </div>
                <Button type="submit" variant="default" disabled={isPasswordLoading}>
                  {isPasswordLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  {t.settings.updatePasswordBtn}
                </Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
      </main>
    </div>
  )
}
