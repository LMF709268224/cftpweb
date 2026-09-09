import { ref } from "vue"
import { apiClient } from "./apiClient"
import { isAuthenticated } from "./authStorage"

export interface UserProfile {
  id: string
  name: string
  display_name: string
  email: string
  phone: string
  work_phone_country_code?: string
  work_phone?: string
  avatar: string
  account_status?: UserAccountStatus
  [key: string]: any
}

export interface UserAccountStatus {
  membership?: {
    available: boolean
    is_member: boolean
    membership_record_ulid?: string
    membership_ulid?: string
    membership_gpath?: string
    plan_name?: string
    tier_level?: number
    status?: string
    expires_at?: string
  }
  certification?: {
    available: boolean
    is_candidate: boolean
    purchase_count: number
    programs: Array<{
      pipeline_ulid: string
      pipeline_config_ulid: string
      pipeline_gpath?: string
      name?: string
      status: string
    }>
  }
  qualification?: {
    available: boolean
    has_qualification: boolean
    credential_count: number
  }
}

const currentUser = ref<UserProfile | null>(null)
const isLoading = ref(false)
const hasLoaded = ref(false)
let pendingUserRequest: Promise<UserProfile | null> | null = null

export function useUser() {
  const fetchUser = async (force = false) => {
    if (hasLoaded.value && !force) {
      return currentUser.value
    }

    if (pendingUserRequest && !force) {
      return pendingUserRequest
    }

    if (!isAuthenticated()) {
      currentUser.value = null
      hasLoaded.value = false
      return null
    }
    
    isLoading.value = true
    pendingUserRequest = (async () => {
      try {
        // /api/user/me returns the candidate info from casdoor
        const res = await apiClient("/api/user/me")
        currentUser.value = res as UserProfile
        hasLoaded.value = true
        return currentUser.value
      } catch (err) {
        console.error("Failed to fetch user info globally", err)
        return null
      } finally {
        isLoading.value = false
        pendingUserRequest = null
      }
    })()

    return pendingUserRequest
  }

  const clearUser = () => {
    currentUser.value = null
    hasLoaded.value = false
    pendingUserRequest = null
  }

  return {
    currentUser,
    isLoading,
    hasLoaded,
    fetchUser,
    clearUser
  }
}
