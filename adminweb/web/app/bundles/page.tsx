"use client"

import React, { useCallback, useEffect, useMemo, useState } from "react"
import { toast } from "sonner"
import { ArrowLeft, FileJson, Package, Plus, RefreshCw, Save, Send, Trash2 } from "lucide-react"

import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { apiClient } from "@/lib/apiClient"
import { statusBadgeClassForStatusValue } from "@/lib/status-labels"
import { useTranslation } from "@/lib/useLanguage"
import { formatBackendDate } from "@/lib/utils"

type BundleInfo = {
  bundle_ulid?: string
  bundle_gpath?: string
  version?: number
  name?: string
  description?: string
  items_json?: string
  pricing_json?: string
  thumbnail_object_key?: string
  thumbnail_file_hash?: string
  status?: string
  is_current?: boolean
  created_at?: string
  updated_at?: string
  display_amount_min?: number
  display_amount_max?: number
  display_currency?: string
}

type BundleForm = {
  bundle_ulid: string
  bundle_gpath: string
  name: string
  description: string
  items_json: string
  pricing_json: string
  thumbnail_object_key: string
  thumbnail_file_hash: string
}

const emptyForm: BundleForm = {
  bundle_ulid: "",
  bundle_gpath: "",
  name: "",
  description: "",
  items_json: "[]",
  pricing_json: "{}",
  thumbnail_object_key: "",
  thumbnail_file_hash: "",
}

function bundleIdOf(bundle: BundleInfo | null | undefined) {
  return bundle?.bundle_ulid || ""
}

function formFromBundle(bundle: BundleInfo | null): BundleForm {
  if (!bundle) return emptyForm
  return {
    bundle_ulid: bundle.bundle_ulid || "",
    bundle_gpath: bundle.bundle_gpath || "",
    name: bundle.name || "",
    description: bundle.description || "",
    items_json: bundle.items_json || "[]",
    pricing_json: bundle.pricing_json || "{}",
    thumbnail_object_key: bundle.thumbnail_object_key || "",
    thumbnail_file_hash: bundle.thumbnail_file_hash || "",
  }
}

function validateJson(value: string, field: string, isZh: boolean) {
  try {
    JSON.parse(value || "")
    return true
  } catch {
    toast.error(isZh ? `${field} 不是合法 JSON` : `${field} must be valid JSON`)
    return false
  }
}

function formatDisplayPrice(bundle: BundleInfo) {
  const currency = bundle.display_currency || ""
  const min = Number(bundle.display_amount_min || 0) / 100
  const max = Number(bundle.display_amount_max || 0) / 100
  if (!currency || (!min && !max)) return "-"
  if (min === max) return `${currency} ${min.toFixed(2)}`
  return `${currency} ${min.toFixed(2)} - ${max.toFixed(2)}`
}

export default function BundlesPage() {
  const { lang } = useTranslation()
  const isZh = lang === "zh"
  const [bundles, setBundles] = useState<BundleInfo[]>([])
  const [selectedId, setSelectedId] = useState("")
  const [creatingDraft, setCreatingDraft] = useState(false)
  const [form, setForm] = useState<BundleForm>(emptyForm)
  const [statusFilter, setStatusFilter] = useState("")
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [offset, setOffset] = useState(0)
  const [schemas, setSchemas] = useState<{ items_schema?: string; pricing_schema?: string } | null>(null)
  const limit = 20

  const selectedBundle = useMemo(
    () => bundles.find((bundle) => bundleIdOf(bundle) === selectedId) || null,
    [bundles, selectedId],
  )
  const selectedIsActive = String(selectedBundle?.status || "").toLowerCase() === "active"
  const showingEditor = Boolean(selectedId || creatingDraft)

  const loadBundles = useCallback(async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      params.set("limit", String(limit))
      params.set("offset", String(offset))
      if (statusFilter) params.set("status", statusFilter)
      const res = await apiClient(`/api/mall/bundles?${params.toString()}`)
      const nextBundles = Array.isArray(res?.bundles) ? res.bundles : []
      setBundles(nextBundles)
      if (selectedId && !nextBundles.some((bundle: BundleInfo) => bundleIdOf(bundle) === selectedId)) {
        setSelectedId("")
        setForm(emptyForm)
        setCreatingDraft(false)
      }
    } finally {
      setLoading(false)
    }
  }, [offset, selectedId, statusFilter])

  useEffect(() => {
    loadBundles().catch(() => toast.error(isZh ? "加载商品配置失败" : "Failed to load bundles"))
  }, [isZh, loadBundles])

  const loadSchemas = async () => {
    const res = await apiClient("/api/mall/bundles/schemas")
    setSchemas(res || null)
  }

  const selectBundle = async (bundle: BundleInfo) => {
    const bundleId = bundleIdOf(bundle)
    if (!bundleId) return
    setSelectedId(bundleId)
    setCreatingDraft(false)
    try {
      const res = await apiClient(`/api/mall/bundles/${encodeURIComponent(bundleId)}`)
      setForm(formFromBundle(res?.bundle || res))
    } catch {
      setForm(formFromBundle(bundle))
    }
  }

  const createBundle = async () => {
    if (!form.bundle_ulid.trim() || !form.bundle_gpath.trim() || !form.name.trim()) {
      toast.error(isZh ? "请填写 bundle_ulid、bundle_gpath 和名称" : "Please fill bundle_ulid, bundle_gpath, and name")
      return
    }
    if (!validateJson(form.items_json, "items_json", isZh) || !validateJson(form.pricing_json, "pricing_json", isZh)) {
      return
    }
    setSaving(true)
    try {
      const res = await apiClient("/api/mall/bundles", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          bundle_ulid: form.bundle_ulid.trim(),
          bundle_gpath: form.bundle_gpath.trim(),
          name: form.name.trim(),
          description: form.description.trim(),
          items_json: form.items_json.trim(),
          pricing_json: form.pricing_json.trim(),
          thumbnail_object_key: form.thumbnail_object_key.trim(),
          thumbnail_file_hash: form.thumbnail_file_hash.trim(),
        }),
      })
      toast.success(isZh ? "商品草稿已创建" : "Bundle draft created")
      setSelectedId(res?.bundle_ulid || form.bundle_ulid.trim())
      setCreatingDraft(false)
      await loadBundles()
    } finally {
      setSaving(false)
    }
  }

  const saveMetadata = async () => {
    if (!selectedId) return
    setSaving(true)
    try {
      await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId)}/meta`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: form.name.trim(),
          description: form.description.trim(),
          thumbnail_object_key: form.thumbnail_object_key.trim(),
          thumbnail_file_hash: form.thumbnail_file_hash.trim(),
        }),
      })
      toast.success(isZh ? "基础信息已保存" : "Metadata saved")
      await loadBundles()
    } finally {
      setSaving(false)
    }
  }

  const saveStructure = async () => {
    if (!selectedId) return
    if (!validateJson(form.items_json, "items_json", isZh) || !validateJson(form.pricing_json, "pricing_json", isZh)) {
      return
    }
    setSaving(true)
    try {
      const res = await apiClient("/api/mall/bundles/pricing", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          bundle_ulid: selectedId,
          items_json: form.items_json.trim(),
          pricing_json: form.pricing_json.trim(),
        }),
      })
      toast.success(isZh ? "商品结构与价格已保存" : "Bundle structure and pricing saved")
      setForm(formFromBundle(res || selectedBundle))
      await loadBundles()
    } finally {
      setSaving(false)
    }
  }

  const publishBundle = async () => {
    if (!selectedId) return
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId)}/publish`, { method: "POST" })
    toast.success(isZh ? "商品已发布" : "Bundle published")
    await loadBundles()
  }

  const deprecateBundle = async () => {
    if (!selectedId) return
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId)}/deprecate`, { method: "POST" })
    toast.success(isZh ? "商品已下架" : "Bundle deprecated")
    await loadBundles()
  }

  const deleteBundle = async () => {
    if (!selectedId) return
    if (!window.confirm(isZh ? "确认删除这个商品草稿？已发布商品通常不能删除。" : "Delete this bundle draft? Published bundles usually cannot be deleted.")) return
    await apiClient(`/api/mall/bundles/${encodeURIComponent(selectedId)}`, { method: "DELETE" })
    toast.success(isZh ? "商品已删除" : "Bundle deleted")
    setSelectedId("")
    setForm(emptyForm)
    setCreatingDraft(false)
    await loadBundles()
  }

  const syncDisplayPricing = async () => {
    await apiClient("/api/mall/bundles/sync-display-pricing", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ bundle_ulid: selectedId || undefined }),
    })
    toast.success(isZh ? "展示价格已同步" : "Display pricing synced")
    await loadBundles()
  }

  const resetForCreate = () => {
    setSelectedId("")
    setForm(emptyForm)
    setCreatingDraft(true)
  }

  const backToList = () => {
    setSelectedId("")
    setForm(emptyForm)
    setCreatingDraft(false)
  }

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          <div className="mb-6 flex items-start justify-between gap-4">
            <div>
              <h1 className="flex items-center gap-2 text-3xl font-bold text-foreground">
                <Package className="h-8 w-8 text-primary" />
                {isZh ? "商品配置" : "Bundle Config"}
              </h1>
              <p className="mt-1 text-muted-foreground">
                {isZh ? "配置 mall Bundle：认证商品包含哪些管线、价格 JSON、封面和发布状态。" : "Configure mall bundles: items, pricing JSON, thumbnails, and publish status."}
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              {showingEditor && (
                <Button variant="outline" onClick={backToList}>
                  <ArrowLeft className="h-4 w-4" />
                  {isZh ? "返回列表" : "Back to list"}
                </Button>
              )}
              {!showingEditor && (
                <Button onClick={resetForCreate}>
                  <Plus className="h-4 w-4" />
                  {isZh ? "新建" : "New"}
                </Button>
              )}
              <Button variant="outline" onClick={syncDisplayPricing}>
                <RefreshCw className="h-4 w-4" />
                {isZh ? "同步展示价格" : "Sync Display Price"}
              </Button>
              <Button variant="outline" onClick={loadBundles} disabled={loading}>
                <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
                {isZh ? "刷新" : "Refresh"}
              </Button>
            </div>
          </div>

          <div className="space-y-4">
            {!showingEditor && (
            <section className="rounded-lg border bg-card">
              <div className="flex items-center justify-between border-b px-4 py-3">
                <div>
                  <h2 className="font-semibold">{isZh ? "商品列表" : "Bundle List"}</h2>
                  <p className="text-xs text-muted-foreground">{isZh ? "来自 gmall 的 Bundle 配置。" : "Bundle configs from gmall."}</p>
                </div>
              </div>
              <div className="border-b p-3">
                <Select value={statusFilter || "all"} onValueChange={(value) => {
                  setStatusFilter(value === "all" ? "" : value)
                  setOffset(0)
                }}>
                  <SelectTrigger>
                    <SelectValue placeholder={isZh ? "全部状态" : "All status"} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">{isZh ? "全部状态" : "All status"}</SelectItem>
                    {["Draft", "Active", "Deprecated"].map((status) => (
                      <SelectItem key={status} value={status}>{status}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="max-h-[720px] divide-y overflow-auto">
                {loading ? (
                  <div className="p-8 text-center text-sm text-muted-foreground">{isZh ? "加载中..." : "Loading..."}</div>
                ) : bundles.length === 0 ? (
                  <div className="p-8 text-center text-sm text-muted-foreground">{isZh ? "暂无商品" : "No bundles"}</div>
                ) : (
                  bundles.map((bundle) => {
                    const active = bundleIdOf(bundle) === selectedId
                    return (
                      <button
                        key={bundle.bundle_ulid}
                        type="button"
                        onClick={() => selectBundle(bundle)}
                        className={`block w-full px-4 py-3 text-left transition ${active ? "bg-muted" : "hover:bg-muted/60"}`}
                      >
                        <div className="flex items-start justify-between gap-2">
                          <div className="min-w-0">
                            <div className="truncate font-semibold">{bundle.name || bundle.bundle_gpath || (isZh ? "未命名商品" : "Untitled bundle")}</div>
                            <div className="mt-1 truncate text-xs text-muted-foreground">{bundle.description || bundle.bundle_gpath || "-"}</div>
                          </div>
                          <Badge variant="outline" className={statusBadgeClassForStatusValue(bundle.status)}>
                            {bundle.status || "-"}
                          </Badge>
                        </div>
                        <div className="mt-2 flex flex-wrap gap-2 text-xs text-muted-foreground">
                          <span>{isZh ? "版本" : "Version"} {bundle.version || 0}</span>
                          <span>{formatDisplayPrice(bundle)}</span>
                        </div>
                      </button>
                    )
                  })
                )}
              </div>
              <div className="flex items-center justify-between border-t p-3">
                <Button variant="outline" size="sm" disabled={offset === 0 || loading} onClick={() => setOffset(Math.max(0, offset - limit))}>
                  {isZh ? "上一页" : "Previous"}
                </Button>
                <span className="text-xs text-muted-foreground">{isZh ? "偏移" : "Offset"}: {offset}</span>
                <Button variant="outline" size="sm" disabled={bundles.length < limit || loading} onClick={() => setOffset(offset + limit)}>
                  {isZh ? "下一页" : "Next"}
                </Button>
              </div>
            </section>
            )}

            {showingEditor && (
            <section className="space-y-4">
              <div className="rounded-lg border bg-card">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
                  <div>
                    <h2 className="font-semibold">{selectedId ? (isZh ? "编辑商品" : "Edit Bundle") : (isZh ? "新建商品草稿" : "Create Bundle Draft")}</h2>
                    <p className="text-xs text-muted-foreground">
                      {isZh ? "Bundle 是现在购买/展示价格的主配置，管线配置只负责课程结构。" : "Bundle is now the main purchase and display pricing config; pipeline config owns course structure."}
                    </p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {selectedId ? (
                      <>
                        <Button variant="outline" size="sm" onClick={saveMetadata} disabled={saving}>
                          <Save className="h-4 w-4" />
                          {isZh ? "保存基础信息" : "Save Metadata"}
                        </Button>
                        <Button size="sm" onClick={saveStructure} disabled={saving || selectedIsActive}>
                          <FileJson className="h-4 w-4" />
                          {isZh ? "保存结构/价格" : "Save JSON"}
                        </Button>
                      </>
                    ) : (
                      <Button size="sm" onClick={createBundle} disabled={saving}>
                        <Plus className="h-4 w-4" />
                        {isZh ? "创建草稿" : "Create Draft"}
                      </Button>
                    )}
                  </div>
                </div>
                <div className="grid gap-4 p-4 md:grid-cols-2">
                  <div>
                    <Label>Bundle ULID</Label>
                    <Input value={form.bundle_ulid} onChange={(event) => setForm({ ...form, bundle_ulid: event.target.value })} disabled={Boolean(selectedId)} />
                  </div>
                  <div>
                    <Label>Bundle GPath</Label>
                    <Input value={form.bundle_gpath} onChange={(event) => setForm({ ...form, bundle_gpath: event.target.value })} disabled={Boolean(selectedId)} />
                  </div>
                  <div>
                    <Label>{isZh ? "名称" : "Name"}</Label>
                    <Input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
                  </div>
                  <div>
                    <Label>{isZh ? "封面 Object Key" : "Thumbnail Object Key"}</Label>
                    <Input value={form.thumbnail_object_key} onChange={(event) => setForm({ ...form, thumbnail_object_key: event.target.value })} />
                  </div>
                  <div className="md:col-span-2">
                    <Label>{isZh ? "封面文件 Hash" : "Thumbnail File Hash"}</Label>
                    <Input value={form.thumbnail_file_hash} onChange={(event) => setForm({ ...form, thumbnail_file_hash: event.target.value })} />
                  </div>
                  <div className="md:col-span-2">
                    <Label>{isZh ? "描述" : "Description"}</Label>
                    <Textarea value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} rows={3} />
                  </div>
                </div>
              </div>

              <div className="rounded-lg border bg-card">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
                  <div>
                    <h2 className="font-semibold">Items / Pricing JSON</h2>
                    <p className="text-xs text-muted-foreground">
                      {isZh ? "items_json 配 Bundle 包含哪些 pipeline；pricing_json 配 unlock、课程单元、重考、免考等 Stripe 价格。" : "items_json defines included pipelines; pricing_json defines Stripe prices for unlocks, units, retakes, exemptions, etc."}
                    </p>
                  </div>
                  <Button variant="outline" size="sm" onClick={loadSchemas}>
                    <FileJson className="h-4 w-4" />
                    {isZh ? "查看 Schema" : "View Schemas"}
                  </Button>
                </div>
                <div className="grid gap-4 p-4 xl:grid-cols-2">
                  <div>
                    <Label>items_json</Label>
                    <Textarea className="font-mono text-xs" rows={18} value={form.items_json} onChange={(event) => setForm({ ...form, items_json: event.target.value })} disabled={selectedIsActive} />
                  </div>
                  <div>
                    <Label>pricing_json</Label>
                    <Textarea className="font-mono text-xs" rows={18} value={form.pricing_json} onChange={(event) => setForm({ ...form, pricing_json: event.target.value })} disabled={selectedIsActive} />
                  </div>
                </div>
                {schemas && (
                  <div className="grid gap-4 border-t p-4 xl:grid-cols-2">
                    <pre className="max-h-72 overflow-auto rounded-md bg-muted p-3 text-xs">{schemas.items_schema || "{}"}</pre>
                    <pre className="max-h-72 overflow-auto rounded-md bg-muted p-3 text-xs">{schemas.pricing_schema || "{}"}</pre>
                  </div>
                )}
              </div>

              {selectedId && (
                <div className="rounded-lg border bg-card">
                  <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
                    <div>
                      <h2 className="font-semibold">{isZh ? "发布操作" : "Publish Actions"}</h2>
                      <p className="text-xs text-muted-foreground">{isZh ? "发布后商品会被候选人端购买流程使用。" : "Published bundles are used by candidate purchase flows."}</p>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button onClick={publishBundle}>
                        <Send className="h-4 w-4" />
                        {isZh ? "发布" : "Publish"}
                      </Button>
                      <Button variant="outline" onClick={deprecateBundle}>
                        {isZh ? "下架" : "Deprecate"}
                      </Button>
                      <Button variant="destructive" onClick={deleteBundle}>
                        <Trash2 className="h-4 w-4" />
                        {isZh ? "删除草稿" : "Delete Draft"}
                      </Button>
                    </div>
                  </div>
                  <div className="grid gap-3 p-4 text-sm text-muted-foreground md:grid-cols-3">
                    <div>{isZh ? "״̬" : "Status"}: {selectedBundle?.status || "-"}</div>
                    <div>{isZh ? "展示价" : "Display Price"}: {selectedBundle ? formatDisplayPrice(selectedBundle) : "-"}</div>
                    <div>{isZh ? "更新时间" : "Updated At"}: {formatBackendDate(selectedBundle?.updated_at)}</div>
                  </div>
                </div>
              )}
            </section>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
