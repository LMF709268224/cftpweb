"use client"

import { useEffect, useState } from "react"
import { Edit3, Plus, RefreshCw, Save, X } from "lucide-react"
import { toast } from "sonner"

import { Sidebar } from "@/components/sidebar"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { cn, formatBackendDate } from "@/lib/utils"

type CourseCategory = {
  catalog_id: string
  name?: string
  description?: string
  created_at?: string
}

type CategoryForm = {
  name: string
  description: string
}

const emptyForm: CategoryForm = {
  name: "",
  description: "",
}

export default function CatalogsPage() {
  const { t } = useTranslation()
  const page = t.catalogsPage
  const [categories, setCategories] = useState<CourseCategory[]>([])
  const [selectedId, setSelectedId] = useState("")
  const [form, setForm] = useState<CategoryForm>(emptyForm)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  const selectedCategory = categories.find((category) => category.catalog_id === selectedId) || null

  const loadCategories = async () => {
    setLoading(true)
    try {
      const res = await apiClient("/api/catalogs")
      setCategories(res?.catalogs || [])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadCategories().catch((error) => {
      console.error(error)
      toast.error(page.loadFailed)
      setLoading(false)
    })
  }, [page.loadFailed])

  const startNew = () => {
    setSelectedId("")
    setForm(emptyForm)
  }

  const selectCategory = (category: CourseCategory) => {
    setSelectedId(category.catalog_id)
    setForm({
      name: category.name || "",
      description: category.description || "",
    })
  }

  const saveCategory = async () => {
    if (!form.name.trim()) {
      toast.error(page.fillName)
      return
    }

    setSaving(true)
    try {
      const payload = {
        name: form.name.trim(),
        description: form.description.trim(),
      }

      if (selectedCategory) {
        await apiClient(`/api/catalogs/${selectedCategory.catalog_id}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        })
        toast.success(page.updateSuccess)
      } else {
        const res = await apiClient("/api/catalogs", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        })
        setSelectedId(res?.catalog_id || "")
        toast.success(page.createSuccess)
      }

      await loadCategories()
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
            <div>
              <h1 className="text-3xl font-bold text-foreground">{page.title}</h1>
              <p className="mt-2 text-muted-foreground">{page.subtitle}</p>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" onClick={loadCategories} disabled={loading}>
                <RefreshCw className={cn("mr-2 h-4 w-4", loading && "animate-spin")} />
                {page.refresh}
              </Button>
              <Button onClick={startNew}>
                <Plus className="mr-2 h-4 w-4" />
                {page.newCategory}
              </Button>
            </div>
          </div>

          <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_420px]">
            <section className="rounded-lg border bg-card">
              <div className="flex items-center justify-between border-b px-4 py-3">
                <h2 className="font-semibold">{page.categoryList}</h2>
                <span className="text-sm text-muted-foreground">{categories.length}</span>
              </div>
              <div className="overflow-hidden">
                <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)_160px_90px] gap-3 border-b px-4 py-3 text-sm font-medium text-muted-foreground">
                  <div>{page.name}</div>
                  <div>{page.description}</div>
                  <div>{page.createdAt}</div>
                  <div>{page.actions}</div>
                </div>
                {loading ? (
                  <div className="p-4 text-sm text-muted-foreground">{t.common.loading}</div>
                ) : categories.length === 0 ? (
                  <div className="p-10 text-center text-sm text-muted-foreground">{page.noCategories}</div>
                ) : (
                  categories.map((category) => (
                    <button
                      key={category.catalog_id}
                      onClick={() => selectCategory(category)}
                      className={cn(
                        "grid w-full grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)_160px_90px] gap-3 border-b px-4 py-3 text-left text-sm transition-colors last:border-b-0 hover:bg-accent",
                        selectedId === category.catalog_id && "bg-accent"
                      )}
                    >
                      <div className="min-w-0">
                        <div className="truncate font-medium text-foreground">{category.name || t.common.unknown}</div>
                        <div className="truncate text-xs text-muted-foreground">{category.catalog_id}</div>
                      </div>
                      <div className="truncate text-muted-foreground">{category.description || t.common.na}</div>
                      <div className="text-xs text-muted-foreground">{formatBackendDate(category.created_at)}</div>
                      <div className="flex items-center text-primary">
                        <Edit3 className="mr-1 h-4 w-4" />
                        {page.edit}
                      </div>
                    </button>
                  ))
                )}
              </div>
            </section>

            <section className="rounded-lg border bg-card">
              <div className="flex items-center justify-between border-b px-4 py-3">
                <h2 className="font-semibold">{selectedCategory ? page.editCategory : page.createCategory}</h2>
                {selectedCategory && (
                  <Button variant="ghost" size="sm" onClick={startNew}>
                    <X className="mr-2 h-4 w-4" />
                    {page.clearSelection}
                  </Button>
                )}
              </div>
              <div className="space-y-4 p-4">
                <div className="space-y-2">
                  <Label htmlFor="categoryName">{page.name}</Label>
                  <Input
                    id="categoryName"
                    value={form.name}
                    onChange={(event) => setForm({ ...form, name: event.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="categoryDescription">{page.description}</Label>
                  <Textarea
                    id="categoryDescription"
                    value={form.description}
                    onChange={(event) => setForm({ ...form, description: event.target.value })}
                  />
                </div>
                {selectedCategory && (
                  <div className="rounded-md bg-muted px-3 py-2 text-xs text-muted-foreground">
                    ID: {selectedCategory.catalog_id}
                  </div>
                )}
              </div>
              <div className="flex justify-end border-t px-4 py-3">
                <Button onClick={saveCategory} disabled={saving}>
                  {selectedCategory ? <Save className="mr-2 h-4 w-4" /> : <Plus className="mr-2 h-4 w-4" />}
                  {selectedCategory ? page.saveCategory : page.createCategory}
                </Button>
              </div>
            </section>
          </div>
        </div>
      </main>
    </div>
  )
}
