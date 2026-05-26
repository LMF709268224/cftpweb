"use client"

import { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/useLanguage"
import { FileCode2, Edit, Plus } from "lucide-react"

export default function PdfTemplatesPage() {
  const { t } = useTranslation()
  const [templates, setTemplates] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  const [isOpen, setIsOpen] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  
  const [formData, setFormData] = useState({
    template_id: "",
    name: "",
    description: "",
    html_template: ""
  })

  const fetchTemplates = async () => {
    setLoading(true)
    try {
      const res = await apiClient("/api/pdf-templates")
      if (res?.templates) {
        setTemplates(res.templates)
      } else {
        setTemplates([])
      }
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchTemplates()
  }, [])

  const handleOpenCreate = () => {
    setFormData({ template_id: "", name: "", description: "", html_template: "" })
    setIsEditing(false)
    setIsOpen(true)
  }

  const handleOpenEdit = (template: any) => {
    setFormData({
      template_id: template.template_id,
      name: template.name,
      description: template.description,
      html_template: template.html_template
    })
    setIsEditing(true)
    setIsOpen(true)
  }

  const handleSubmit = async () => {
    try {
      if (isEditing) {
        await apiClient("/api/pdf-templates", {
          method: "PUT",
          body: JSON.stringify(formData)
        })
      } else {
        await apiClient("/api/pdf-templates", {
          method: "POST",
          body: JSON.stringify({
            name: formData.name,
            description: formData.description,
            html_template: formData.html_template
          })
        })
      }
      setIsOpen(false)
      fetchTemplates()
    } catch (e) {
      alert(t.common.error)
    }
  }

  return (
    <div className="min-h-screen bg-background flex">
      <Sidebar />
      <main className="flex-1 ml-64 p-8">
        <div className="flex justify-between items-center mb-8">
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-foreground flex items-center gap-2">
              <FileCode2 className="h-8 w-8 text-primary" />
              {t.pdfTemplatesPage.title}
            </h1>
            <p className="text-muted-foreground mt-2">{t.pdfTemplatesPage.subtitle}</p>
          </div>
          <Button onClick={handleOpenCreate} className="gap-2">
            <Plus className="h-4 w-4" />
            {t.pdfTemplatesPage.newTemplate}
          </Button>
        </div>

        {loading ? (
          <div>{t.common.loading}</div>
        ) : (
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {templates.map(tmpl => (
              <div key={tmpl.template_id} className="rounded-xl border bg-card p-6 shadow-sm">
                <div className="flex justify-between items-start mb-4">
                  <h3 className="text-xl font-bold">{tmpl.name}</h3>
                  <Badge variant="secondary">v{tmpl.version}</Badge>
                </div>
                <p className="text-sm text-muted-foreground mb-4 line-clamp-2 min-h-[40px]">
                  {tmpl.description}
                </p>
                <div className="text-xs text-muted-foreground mb-4">
                  <p>{t.pdfTemplatesPage.id}: {tmpl.template_id}</p>
                  <p>{t.pdfTemplatesPage.created}: {formatBackendDate(tmpl.created_at)}</p>
                </div>
                <Button variant="outline" className="w-full gap-2" onClick={() => handleOpenEdit(tmpl)}>
                  <Edit className="h-4 w-4" /> {t.pdfTemplatesPage.editTemplate}
                </Button>
              </div>
            ))}
          </div>
        )}

        <Dialog open={isOpen} onOpenChange={setIsOpen}>
          <DialogContent className="max-w-2xl h-[80vh] flex flex-col">
            <DialogHeader>
              <DialogTitle>{isEditing ? t.pdfTemplatesPage.editTemplate : t.pdfTemplatesPage.newTemplate}</DialogTitle>
            </DialogHeader>
            <div className="py-4 flex-1 flex flex-col gap-4 overflow-y-auto">
              <div className="space-y-2">
                <Label>{t.pdfTemplatesPage.templateName}</Label>
                <Input 
                  value={formData.name}
                  onChange={e => setFormData({...formData, name: e.target.value})}
                  placeholder={t.pdfTemplatesPage.namePlaceholder}
                />
              </div>
              <div className="space-y-2">
                <Label>{t.pdfTemplatesPage.description}</Label>
                <Input 
                  value={formData.description}
                  onChange={e => setFormData({...formData, description: e.target.value})}
                  placeholder={t.pdfTemplatesPage.descPlaceholder}
                />
              </div>
              <div className="space-y-2 flex-1 flex flex-col">
                <Label>{t.pdfTemplatesPage.htmlSource}</Label>
                <Textarea 
                  className="flex-1 font-mono text-sm"
                  value={formData.html_template}
                  onChange={e => setFormData({...formData, html_template: e.target.value})}
                  placeholder={t.pdfTemplatesPage.htmlPlaceholder}
                />
              </div>
            </div>
            <div className="flex justify-end gap-3 border-t pt-4">
              <Button variant="outline" onClick={() => setIsOpen(false)}>{t.common.cancel}</Button>
              <Button onClick={handleSubmit}>{t.pdfTemplatesPage.saveTemplate}</Button>
            </div>
          </DialogContent>
        </Dialog>
      </main>
    </div>
  )
}
