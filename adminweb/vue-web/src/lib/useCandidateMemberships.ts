import { onScopeDispose, ref, watch } from "vue"
import { apiClient } from "./apiClient"
import { useAdminLanguage } from "./language"

type Candidate = { user_group: string; candidate_ulid?: string }
export type MembershipRecord = {
  candidate_ulid: string
  membership_ulid: string
  started_at?: string
  expires_at?: string
}
type MembershipState = {
  state: "loading" | "none" | "active" | "error"
  name?: string
  record?: MembershipRecord
}

export function useCandidateMemberships(users: () => Candidate[]) {
  const { lang } = useAdminLanguage()
  const memberships = ref<Record<string, MembershipState>>({})
  let controller: AbortController | undefined

  watch([users, lang], async ([rows, language]) => {
    controller?.abort()
    const request = new AbortController()
    controller = request
    const init = { signal: request.signal }
    const ids = [...new Set(rows.filter((user) => user.user_group === "candidate").map((user) => user.candidate_ulid).filter((id): id is string => !!id))]
    memberships.value = Object.fromEntries(ids.map((id) => [id, { state: "loading" }]))
    const names = new Map<string, Promise<string>>()
    const locale = language === "zh" ? "zh-CN" : "en-US"

    function planName(id: string) {
      if (!names.has(id)) {
        names.set(id, (async () => {
          const translations = await apiClient<{ translations?: Record<string, { name?: string }> }>(`/api/memberships/${encodeURIComponent(id)}/translations?locale=${locale}`, init)
          const name = translations.translations?.[locale]?.name
          if (typeof name !== "string" || !name.trim()) throw new Error("Membership translation unavailable")
          return name
        })())
      }
      return names.get(id)!
    }

    // Only hydrate the displayed page; bound concurrency and reuse plan lookups.
    const queue = [...ids]
    await Promise.all(Array.from({ length: Math.min(4, queue.length) }, async () => {
      while (queue.length && !request.signal.aborted) {
        const id = queue.shift()!
        try {
          const result = await apiClient<{ membership?: MembershipRecord | null }>(`/api/memberships/active?candidate_ulid=${encodeURIComponent(id)}`, init)
          if (!result || !Object.hasOwn(result, "membership")) throw new Error("Membership result missing")
          const record = result.membership
          if (record !== null && (typeof record !== "object" || Array.isArray(record))) throw new Error("Invalid membership result")
          if (record && (record.candidate_ulid !== id || !record.membership_ulid)) throw new Error("Invalid candidate membership response")
          const expires = record ? Date.parse(record.expires_at || "") : 0
          const starts = record ? Date.parse(record.started_at || "") : 0
          if (record && (!Number.isFinite(expires) || !Number.isFinite(starts))) throw new Error("Invalid membership validity dates")
          const current = record && starts <= Date.now() && expires > Date.now()
          const value: MembershipState = current ? { state: "active", record, name: await planName(record.membership_ulid) } : { state: "none" }
          if (!request.signal.aborted) memberships.value[id] = value
        } catch (err) {
          if (request.signal.aborted) return
          console.error(err)
          memberships.value[id] = { state: "error" }
        }
      }
    }))
  }, { immediate: true })

  onScopeDispose(() => controller?.abort())
  return memberships
}
