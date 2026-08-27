<script setup lang="ts">
import { computed } from "vue"
import { Download, Eye, FileText } from "lucide-vue-next"
import { useTranslation } from "@/lib/language"

const props = defineProps<{ attachments?: any[] }>()
const { t } = useTranslation()

function attachmentURL(value: unknown) {
  const text = String(value || "").trim()
  if (!text) return ""
  try {
    const parsed = new URL(text)
    return parsed.protocol === "https:" || parsed.protocol === "http:" ? parsed.toString() : ""
  } catch {
    return ""
  }
}

const validAttachments = computed(() =>
  (Array.isArray(props.attachments) ? props.attachments : []).filter(
    (attachment) => attachment && typeof attachment === "object" && attachmentURL(attachment.download_url),
  ),
)

function formatAttachmentSize(value: unknown) {
  const bytes = Number(value)
  if (!Number.isFinite(bytes) || bytes <= 0) return ""
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}
</script>

<template>
  <section v-if="validAttachments.length" class="space-y-3">
    <div>
      <h4 class="text-sm font-semibold text-foreground">{{ t.credentialsPage.referenceAttachments }}</h4>
      <p class="mt-1 text-xs leading-5 text-muted-foreground">{{ t.credentialsPage.referenceAttachmentsHint }}</p>
    </div>
    <div class="divide-y divide-border overflow-hidden rounded-lg border border-border bg-white/70">
      <div
        v-for="(attachment, index) in validAttachments"
        :key="attachment.attachment_id || attachment.file_key || `${attachment.file_name}-${index}`"
        class="flex flex-col gap-3 p-3 sm:flex-row sm:items-center sm:justify-between"
      >
        <div class="flex min-w-0 items-start gap-3">
          <FileText class="mt-0.5 h-5 w-5 shrink-0 text-primary" />
          <div class="min-w-0">
            <div class="break-words text-sm font-semibold text-foreground">{{ attachment.name || attachment.file_name }}</div>
            <p v-if="attachment.description" class="mt-1 whitespace-pre-wrap break-words text-xs leading-5 text-muted-foreground">{{ attachment.description }}</p>
            <div class="mt-1 flex flex-wrap gap-x-2 text-xs text-muted-foreground">
              <span>{{ attachment.file_name }}</span>
              <span v-if="formatAttachmentSize(attachment.file_size)">{{ formatAttachmentSize(attachment.file_size) }}</span>
            </div>
          </div>
        </div>
        <div class="flex shrink-0 flex-wrap gap-2">
          <a
            class="btn btn-outline inline-flex h-9 cursor-pointer items-center justify-center gap-2 rounded-lg px-3 text-xs hover:border-primary/25 hover:bg-primary/10 hover:text-primary"
            :href="attachmentURL(attachment.download_url)"
            target="_blank"
            rel="noopener noreferrer"
          >
            <Eye class="h-4 w-4" /> {{ t.credentialsPage.previewAttachment }}
          </a>
          <a
            class="btn btn-outline inline-flex h-9 cursor-pointer items-center justify-center gap-2 rounded-lg px-3 text-xs hover:border-primary/25 hover:bg-primary/10 hover:text-primary"
            :href="attachmentURL(attachment.download_url)"
            :download="attachment.file_name || true"
            target="_blank"
            rel="noopener noreferrer"
          >
            <Download class="h-4 w-4" /> {{ t.credentialsPage.downloadAttachment }}
          </a>
        </div>
      </div>
    </div>
  </section>
</template>
