"use client"

import React from "react"

import { useEffect, useState } from "react"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { Sidebar } from "@/components/sidebar"

export default function CatalogsPage() {
  const { t } = useTranslation()
  const [catalogs, setCatalogs] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchCatalogs = async () => {
      try {
        const res = await apiClient("/api/catalogs")
        if (res && res.catalogs) {
          setCatalogs(res.catalogs)
        }
      } catch (err) {
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    fetchCatalogs()
  }, [])

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          <div className="mb-8">
            <h1 className="text-3xl font-bold text-foreground mb-2">{t.sidebar.catalogs}</h1>
            <p className="text-muted-foreground">Manage category catalogs</p>
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
                    <th className="pb-3 font-medium">Description</th>
                  </tr>
                </thead>
                <tbody>
                  {catalogs.length > 0 ? (
                    catalogs.map((c) => (
                      <tr key={c.catalog_id} className="border-b last:border-0">
                        <td className="py-3 text-foreground">{c.catalog_id}</td>
                        <td className="py-3 text-foreground">{c.name || t.common.unknown}</td>
                        <td className="py-3 text-foreground">{c.description}</td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={3} className="py-8 text-center text-muted-foreground">
                        No catalogs found
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
