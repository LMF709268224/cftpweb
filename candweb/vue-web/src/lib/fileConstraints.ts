export interface FileConstraintInfo {
  exts: string[]
  extLabel: string
  acceptStr: string
  maxSize: number
  maxLabel: string
}

export function getFileConstraintInfo(type: number): FileConstraintInfo {
  switch (type) {
    case 1: // Image
      return {
        exts: [".jpg", ".jpeg", ".png", ".gif"],
        extLabel: "JPG, PNG, GIF",
        acceptStr: "image/jpeg,image/png,image/gif",
        maxSize: 10 * 1024 * 1024,
        maxLabel: "10MB",
      }
    case 2: // PDF
      return {
        exts: [".pdf"],
        extLabel: "PDF",
        acceptStr: "application/pdf",
        maxSize: 20 * 1024 * 1024,
        maxLabel: "20MB",
      }
    case 4: // Video
      return {
        exts: [".mp4", ".mov", ".webm"],
        extLabel: "MP4, MOV, WEBM",
        acceptStr: "video/mp4,video/quicktime,video/webm",
        maxSize: 200 * 1024 * 1024,
        maxLabel: "200MB",
      }
    case 8: // Text
      return {
        exts: [".txt", ".doc", ".docx"],
        extLabel: "TXT, DOC, DOCX",
        acceptStr: "text/plain,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document",
        maxSize: 10 * 1024 * 1024,
        maxLabel: "10MB",
      }
    default: // Unspecified
      return {
        exts: [],
        extLabel: "Any",
        acceptStr: "*/*",
        maxSize: 50 * 1024 * 1024,
        maxLabel: "50MB",
      }
  }
}
