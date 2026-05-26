"use client"

import { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Plus, Trash2 } from "lucide-react"

interface FileConstraint {
  name: string
  type: number
  is_required: boolean
}

interface CredentialDefinition {
  cred_def_id: string
  name: string
  description: string
  category: string
  file_constraints: FileConstraint[]
}

const getFileTypes = (t: any) => [
  { value: 0, label: t.credentialsDefPage.fileTypes.unspecified },
  { value: 1, label: t.credentialsDefPage.fileTypes.image },
  { value: 2, label: t.credentialsDefPage.fileTypes.pdf },
  { value: 4, label: t.credentialsDefPage.fileTypes.video },
  { value: 8, label: t.credentialsDefPage.fileTypes.text },
]

export default function CredentialsPage() {
  const { t } = useTranslation()
  const [definitions, setDefinitions] = useState<CredentialDefinition[]>([])
  const [loading, setLoading] = useState(true)
  const [isDialogOpen, setIsDialogOpen] = useState(false)

  // Form State
  const [name, setName] = useState("")
  const [description, setDescription] = useState("")
  const [category, setCategory] = useState("")
  const [constraints, setConstraints] = useState<FileConstraint[]>([])

  const fileTypes = getFileTypes(t)

  const fetchDefinitions = async () => {
    try {
      setLoading(true)
      const res = await apiClient("/api/credentials/definitions")
      if (res && res.definitions) {
        setDefinitions(res.definitions)
      } else {
        setDefinitions([])
      }
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchDefinitions()
  }, [])

  const handleAddConstraint = () => {
    setConstraints([...constraints, { name: "", type: 2, is_required: true }])
  }

  const handleUpdateConstraint = (index: number, field: keyof FileConstraint, value: any) => {
    const newConstraints = [...constraints]
    newConstraints[index] = { ...newConstraints[index], [field]: value }
    setConstraints(newConstraints)
  }

  const handleRemoveConstraint = (index: number) => {
    setConstraints(constraints.filter((_, i) => i !== index))
  }

  const handleSubmit = async () => {
    if (!name || !category) {
      alert(t.credentialsDefPage.alertNameCategory)
      return
    }

    try {
      await apiClient("/api/credentials/definitions", {
        method: "POST",
        body: JSON.stringify({
          name,
          description,
          category,
          file_constraints: constraints,
        }),
      })
      setIsDialogOpen(false)
      setName("")
      setDescription("")
      setCategory("")
      setConstraints([])
      fetchDefinitions()
    } catch (err) {
      console.error("Failed to create definition", err)
    }
  }

  const getFileTypeName = (type: number) => {
    return fileTypes.find(ft => ft.value === type)?.label || t.common.unknown
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-foreground mb-2">{t.credentialsDefPage.title}</h1>
          <p className="text-muted-foreground">{t.credentialsDefPage.subtitle}</p>
        </div>
        <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              {t.credentialsDefPage.newDefinition}
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>{t.credentialsDefPage.createTitle}</DialogTitle>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label>{t.credentialsDefPage.name}</Label>
                <Input value={name} onChange={e => setName(e.target.value)} placeholder={t.credentialsDefPage.namePlaceholder} />
              </div>
              <div className="grid gap-2">
                <Label>{t.credentialsDefPage.category}</Label>
                <Input value={category} onChange={e => setCategory(e.target.value)} placeholder={t.credentialsDefPage.categoryPlaceholder} />
              </div>
              <div className="grid gap-2">
                <Label>{t.credentialsDefPage.description}</Label>
                <Input value={description} onChange={e => setDescription(e.target.value)} placeholder={t.credentialsDefPage.descPlaceholder} />
              </div>

              <div className="mt-4">
                <div className="flex items-center justify-between mb-2">
                  <Label>{t.credentialsDefPage.requiredFiles}</Label>
                  <Button variant="outline" size="sm" onClick={handleAddConstraint}>
                    <Plus className="h-4 w-4 mr-1" /> {t.credentialsDefPage.addFile}
                  </Button>
                </div>
                {constraints.map((c, i) => (
                  <div key={i} className="flex items-center gap-2 mb-2 p-2 border rounded-md">
                    <Input 
                      placeholder={t.credentialsDefPage.fileNamePlaceholder} 
                      value={c.name} 
                      onChange={e => handleUpdateConstraint(i, 'name', e.target.value)} 
                    />
                    <select 
                      className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm"
                      value={c.type}
                      onChange={e => handleUpdateConstraint(i, 'type', parseInt(e.target.value))}
                    >
                      {fileTypes.map(ft => (
                        <option key={ft.value} value={ft.value}>{ft.label}</option>
                      ))}
                    </select>
                    <label className="flex items-center gap-1 text-sm whitespace-nowrap">
                      <input 
                        type="checkbox" 
                        checked={c.is_required} 
                        onChange={e => handleUpdateConstraint(i, 'is_required', e.target.checked)} 
                      />
                      {t.credentialsDefPage.isRequired}
                    </label>
                    <Button variant="ghost" size="icon" onClick={() => handleRemoveConstraint(i)}>
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                ))}
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setIsDialogOpen(false)}>{t.common.cancel}</Button>
              <Button onClick={handleSubmit}>{t.credentialsDefPage.createBtn}</Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      <div className="rounded-md border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t.credentialsDefPage.name}</TableHead>
              <TableHead>{t.credentialsDefPage.category}</TableHead>
              <TableHead>{t.credentialsDefPage.description}</TableHead>
              <TableHead>{t.credentialsDefPage.constraints}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={4} className="text-center py-8">{t.common.loading}</TableCell>
              </TableRow>
            ) : definitions.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="text-center py-8 text-muted-foreground">
                  {t.credentialsDefPage.noDefinitions}
                </TableCell>
              </TableRow>
            ) : (
              definitions.map((def) => (
                <TableRow key={def.cred_def_id}>
                  <TableCell className="font-medium">{def.name}</TableCell>
                  <TableCell>{def.category}</TableCell>
                  <TableCell className="text-muted-foreground">{def.description || '-'}</TableCell>
                  <TableCell>
                    {def.file_constraints?.length > 0 ? (
                      <ul className="text-xs space-y-1">
                        {def.file_constraints.map((fc, i) => (
                          <li key={i} className="flex gap-1 items-center">
                            <span className="font-semibold">{fc.name}</span>
                            <span className="text-muted-foreground">({getFileTypeName(fc.type)})</span>
                            {fc.is_required && <span className="text-red-500">*</span>}
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <span className="text-muted-foreground text-xs">{t.credentialsDefPage.noFiles}</span>
                    )}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
