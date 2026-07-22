import { ref } from "vue"
import { apiClient } from "./apiClient"
import { getAccessToken } from "./authStorage"

export interface UserProfile {
  id: string
  name: string
  display_name: string
  email: string
  phone: string
  work_phone_country_code?: string
  work_phone?: string
  avatar: string
  [key: string]: any
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

    if (!getAccessToken()) {
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
