"use client"

import { useEffect, useState } from "react"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { Sidebar } from "@/components/sidebar"

export default function PipelinesPage() {
  const { t } = useTranslation()
  const [pipelines, setPipelines] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchPipelines = async () => {
      try {
        const res = await apiClient("/api/pipelines")
        if (res && res.pipelines) {
          setPipelines(res.pipelines)
        }
      } catch (err) {
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    fetchPipelines()
  }, [])

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          <div className="mb-8">
            <h1 className="text-3xl font-bold text-foreground mb-2">{t.sidebar.pipelines}</h1>
            <p className="text-muted-foreground">Manage course pipelines and outlines</p>
          </div>

          {loading ? (
            <div className="text-muted-foreground">{t.common.loading}</div>
          ) : (
            <div className="bg-card rounded-xl border p-4">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b text-muted-foreground">
                    <th className="pb-3 font-medium">ID</th>
                    <th className="pb-3 font-medium">Name</th>
                    <th className="pb-3 font-medium">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {pipelines.length > 0 ? (
                    pipelines.map((p) => (
                      <tr key={p.pipeline_id} className="border-b last:border-0">
                        <td className="py-3 text-foreground">{p.pipeline_id}</td>
                        <td className="py-3 text-foreground">{p.name || t.common.unknownCourse}</td>
                        <td className="py-3 text-foreground">{p.status}</td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={3} className="py-8 text-center text-muted-foreground">
                        No pipelines found
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
