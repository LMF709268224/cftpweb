"use client"

import { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { formatBackendDate } from "@/lib/utils"
import { FileCode2, Edit, Plus } from "lucide-react"

export default function PdfTemplatesPage() {
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
      alert("Save failed")
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
              PDF Templates
            </h1>
            <p className="text-muted-foreground mt-2">Manage HTML-to-PDF templates for certificate generation.</p>
          </div>
          <Button onClick={handleOpenCreate} className="gap-2">
            <Plus className="h-4 w-4" />
            New Template
          </Button>
        </div>

        {loading ? (
          <div>Loading...</div>
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
                  <p>ID: {tmpl.template_id}</p>
                  <p>Created: {formatBackendDate(tmpl.created_at)}</p>
                </div>
                <Button variant="outline" className="w-full gap-2" onClick={() => handleOpenEdit(tmpl)}>
                  <Edit className="h-4 w-4" /> Edit Template
                </Button>
              </div>
            ))}
          </div>
        )}

        <Dialog open={isOpen} onOpenChange={setIsOpen}>
          <DialogContent className="max-w-2xl h-[80vh] flex flex-col">
            <DialogHeader>
              <DialogTitle>{isEditing ? "Edit Template" : "New Template"}</DialogTitle>
            </DialogHeader>
            <div className="py-4 flex-1 flex flex-col gap-4 overflow-y-auto">
              <div className="space-y-2">
                <Label>Template Name</Label>
                <Input 
                  value={formData.name}
                  onChange={e => setFormData({...formData, name: e.target.value})}
                  placeholder="e.g. CFTA Certificate Template v1"
                />
              </div>
              <div className="space-y-2">
                <Label>Description</Label>
                <Input 
                  value={formData.description}
                  onChange={e => setFormData({...formData, description: e.target.value})}
                  placeholder="Description of this template"
                />
              </div>
              <div className="space-y-2 flex-1 flex flex-col">
                <Label>HTML Source (Go html/template syntax)</Label>
                <Textarea 
                  className="flex-1 font-mono text-sm"
                  value={formData.html_template}
                  onChange={e => setFormData({...formData, html_template: e.target.value})}
                  placeholder="<html><body><h1>{{.CandidateName}}</h1>...</body></html>"
                />
              </div>
            </div>
            <div className="flex justify-end gap-3 border-t pt-4">
              <Button variant="outline" onClick={() => setIsOpen(false)}>Cancel</Button>
              <Button onClick={handleSubmit}>Save Template</Button>
            </div>
          </DialogContent>
        </Dialog>
      </main>
    </div>
  )
}
