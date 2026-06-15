<script setup lang="ts">
import { Toaster } from "vue-sonner"
import { useUser } from "@/lib/user"
import { onMounted } from "vue"

const { fetchUser } = useUser()

onMounted(() => {
  fetchUser()
})
</script>

<template>
  <RouterView />
  <Toaster rich-colors position="top-center" />
</template>
