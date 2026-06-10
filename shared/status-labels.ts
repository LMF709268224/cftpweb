export type StatusValue = string | number | null | undefined
export type TranslationTree = Record<string, any>
export type StatusLabelMap = Record<string, string>
export type StatusEnumNameMap = Record<string, string>
export type StatusTone = "running" | "success" | "warning" | "danger" | "neutral"

export const normalizeEnumValue = (value?: StatusValue) => {
  if (value === null || value === undefined) return ""
  return String(value).trim()
}

export const normalizeEnumValueLower = (value?: StatusValue) => normalizeEnumValue(value).toLowerCase()

export const normalizeEnumValueUpper = (value?: StatusValue) => normalizeEnumValue(value).toUpperCase()

const resolvePath = (object: TranslationTree, path: string) =>
  path.split(".").reduce<any>((current, key) => current?.[key], object)

const unknownLabel = (t: TranslationTree, fallbackPath = "common.unknown") =>
  resolvePath(t, fallbackPath) || resolvePath(t, "common.unknown") || ""

const resolveStatusLabelPath = (map: StatusLabelMap, status?: StatusValue) => {
  const normalized = normalizeEnumValue(status)
  return map[normalized] || ""
}

export const statusEnumNameForStatus = (
  enumNameMap: StatusEnumNameMap,
  status?: StatusValue,
) => {
  const normalized = normalizeEnumValue(status)
  return enumNameMap[normalized] || ""
}

export const statusLabel = (t: TranslationTree, map: StatusLabelMap, status?: StatusValue, fallbackPath = "common.unknown") => {
  const labelPath = resolveStatusLabelPath(map, status)
  return labelPath ? resolvePath(t, labelPath) || unknownLabel(t, fallbackPath) : unknownLabel(t, fallbackPath)
}

// TODO: Remove diagnostic suffix once white-box pipeline troubleshooting is no longer needed.
export const statusLabelWithDiagnostics = (
  t: TranslationTree,
  map: StatusLabelMap,
  _enumNameMap: StatusEnumNameMap,
  status?: StatusValue,
  fallbackPath = "common.unknown",
) => {
  return statusLabel(t, map, status, fallbackPath)
}

export const statusToneFromStatusValue = (status?: StatusValue): StatusTone => {
  const normalized = normalizeEnumValueUpper(status)
  if (!normalized) return "neutral"

  if (normalized.includes("UNSPECIFIED") || normalized.includes("UNKNOWN")) {
    return "neutral"
  }

  if (
    normalized.includes("FAILED") ||
    normalized.includes("ERROR") ||
    normalized.includes("BROKEN") ||
    normalized.includes("DELETED") ||
    normalized.includes("MISSING") ||
    normalized.includes("REJECTED") ||
    normalized.includes("REVOKED") ||
    normalized.includes("TERMINATED")
  ) {
    return "danger"
  }

  if (
    normalized.includes("RUNNING") ||
    normalized.includes("IN_PROGRESS") ||
    normalized.includes("PROCESSING") ||
    normalized.includes("ONGOING") ||
    normalized.includes("ACTIVE") ||
    normalized.includes("GENERATING") ||
    normalized.includes("LEARNING") ||
    normalized.includes("OPEN")
  ) {
    return "running"
  }

  if (
    normalized.includes("COMPLETED") ||
    normalized.includes("PASSED") ||
    normalized.includes("APPROVED") ||
    normalized.includes("FINISHED") ||
    normalized.includes("DONE") ||
    normalized.includes("SUCCESS") ||
    normalized.includes("READ") ||
    normalized.includes("SENT")
  ) {
    return "success"
  }

  if (
    normalized.includes("UNREAD") ||
    normalized.includes("WAIT") ||
    normalized.includes("PENDING") ||
    normalized.includes("SCHEDULING") ||
    normalized.includes("SCHEDULED") ||
    normalized.includes("APPROVAL") ||
    normalized.includes("REVIEW") ||
    normalized.includes("RESUBMIT") ||
    normalized.includes("REUPLOAD") ||
    normalized.includes("CANCELLED") ||
    normalized.includes("TESTING")
  ) {
    return "warning"
  }

  return "neutral"
}

export const statusToneFromEnumName = (enumName?: StatusValue): StatusTone => {
  return statusToneFromStatusValue(enumName)
}

export const statusBadgeClassFromTone = (tone: StatusTone) => {
  switch (tone) {
    case "running":
      return "border-green-400 bg-green-100 text-black"
    case "success":
      return "border-blue-400 bg-blue-100 text-black"
    case "warning":
      return "border-yellow-400 bg-yellow-100 text-black"
    case "danger":
      return "border-red-400 bg-red-100 text-black"
    case "neutral":
      return "border-gray-400 bg-gray-100 text-black"
    default:
      return "border-gray-400 bg-gray-100 text-black"
  }
}

export const statusBadgeClassForStatus = (
  enumNameMap: StatusEnumNameMap,
  status?: StatusValue,
) => {
  const enumName = statusEnumNameForStatus(enumNameMap, status)
  return statusBadgeClassFromTone(enumName ? statusToneFromEnumName(enumName) : "neutral")
}

export const statusBadgeClassForStatusValue = (status?: StatusValue) => {
  return statusBadgeClassFromTone(statusToneFromStatusValue(status))
}

export const stageStatusHintLabel = (t: TranslationTree, status?: StatusValue) => {
  switch (normalizeEnumValue(status)) {
    case "1":
      return resolvePath(t, "learning.stageWaitCandidateHint") || unknownLabel(t)
    case "2":
      return resolvePath(t, "learning.stageRunningHint") || unknownLabel(t)
    case "3":
      return resolvePath(t, "learning.stageCompletedHint") || unknownLabel(t)
    default:
      return resolvePath(t, "learning.nextStepDesc") || unknownLabel(t)
  }
}

export const timelineStatusLabel = (t: TranslationTree, entityType?: StatusValue, status?: StatusValue) => {
  switch (normalizeEnumValueUpper(entityType)) {
    case "PIPELINE":
      return statusLabel(t, CANDIDATE_PIPELINE_STATUS_LABELS, status)
    case "STAGE":
      return statusLabel(t, STAGE_STATUS_LABELS, status)
    case "COURSE_UNIT":
      return statusLabel(t, COURSE_UNIT_STATUS_LABELS, status)
    default:
      return unknownLabel(t)
  }
}

const timelineStatusMapsForEntityType = (entityType?: StatusValue) => {
  switch (normalizeEnumValueUpper(entityType)) {
    case "PIPELINE":
      return {
        labelMap: CANDIDATE_PIPELINE_STATUS_LABELS,
        enumNameMap: CANDIDATE_PIPELINE_STATUS_ENUM_NAMES,
      }
    case "STAGE":
      return {
        labelMap: STAGE_STATUS_LABELS,
        enumNameMap: STAGE_STATUS_ENUM_NAMES,
      }
    case "COURSE_UNIT":
      return {
        labelMap: COURSE_UNIT_STATUS_LABELS,
        enumNameMap: COURSE_UNIT_STATUS_ENUM_NAMES,
      }
    default:
      return null
  }
}

export const timelineStatusLabelWithDiagnostics = (
  t: TranslationTree,
  entityType?: StatusValue,
  status?: StatusValue,
  fallbackPath = "common.unknown",
) => {
  const maps = timelineStatusMapsForEntityType(entityType)
  if (!maps) {
    return unknownLabel(t, fallbackPath)
  }
  return statusLabelWithDiagnostics(t, maps.labelMap, maps.enumNameMap, status, fallbackPath)
}

export const timelineStatusBadgeClassForStatus = (entityType?: StatusValue, status?: StatusValue) => {
  const maps = timelineStatusMapsForEntityType(entityType)
  if (!maps) {
    return statusBadgeClassFromTone("neutral")
  }
  return statusBadgeClassForStatus(maps.enumNameMap, status)
}

export const CANDIDATE_PIPELINE_STATUS_LABELS: StatusLabelMap = {
  "1": "learning.statusRunning",
  "2": "learning.statusWaitFinalElig",
  "3": "learning.statusCompleted",
  "4": "learning.statusIssuingCert",
}

export const CANDIDATE_PIPELINE_STATUS_ENUM_NAMES: StatusEnumNameMap = {
  "0": "PIPELINE_STATUS_UNSPECIFIED",
  "1": "PIPELINE_STATUS_RUNNING",
  "2": "PIPELINE_STATUS_WAIT_FINAL_ELIG",
  "3": "PIPELINE_STATUS_COMPLETED",
  "4": "PIPELINE_STATUS_ISSUING_CERT",
}

export const ADMIN_PIPELINE_STATUS_LABELS: StatusLabelMap = {
  "1": "progPage.pipelineStatusRunning",
  "2": "progPage.pipelineStatusWaitFinalElig",
  "3": "progPage.pipelineStatusCompleted",
  "4": "progPage.pipelineStatusIssuingCert",
}

export const ADMIN_PIPELINE_STATUS_ENUM_NAMES: StatusEnumNameMap = {
  "0": "PIPELINE_STATUS_UNSPECIFIED",
  "1": "PIPELINE_STATUS_RUNNING",
  "2": "PIPELINE_STATUS_WAIT_FINAL_ELIG",
  "3": "PIPELINE_STATUS_COMPLETED",
  "4": "PIPELINE_STATUS_ISSUING_CERT",
}

export const STAGE_STATUS_LABELS: StatusLabelMap = {
  "1": "learning.statusWaitCandidate",
  "2": "learning.statusRunning",
  "3": "learning.statusCompleted",
}

export const STAGE_STATUS_ENUM_NAMES: StatusEnumNameMap = {
  "0": "STAGE_STATUS_UNSPECIFIED",
  "1": "STAGE_STATUS_WAIT_CANDIDATE",
  "2": "STAGE_STATUS_RUNNING",
  "3": "STAGE_STATUS_COMPLETED",
}

export const COURSE_UNIT_STATUS_LABELS: StatusLabelMap = {
  "1": "learning.statusWaitingStudy",
  "2": "learning.statusWaitingSignupExam",
  "3": "learning.statusExamOpen",
  "4": "learning.statusExamScheduled",
  "5": "learning.statusExamFailed",
  "6": "learning.statusCompleted",
}

export const COURSE_UNIT_STATUS_ENUM_NAMES: StatusEnumNameMap = {
  "0": "COURSE_UNIT_STATUS_UNSPECIFIED",
  "1": "COURSE_UNIT_STATUS_WAITING_STUDY",
  "2": "COURSE_UNIT_STATUS_WAITING_SIGNUP_EXAM",
  "3": "COURSE_UNIT_STATUS_EXAM_OPEN",
  "4": "COURSE_UNIT_STATUS_EXAM_SCHEDULED",
  "5": "COURSE_UNIT_STATUS_EXAM_FAILED",
  "6": "COURSE_UNIT_STATUS_COMPLETED",
}

export type CourseUnitNextStepAction =
  | "continue_learning"
  | "signup_exam"
  | "schedule_exam"
  | "view_exam_schedule"
  | "apply_retake"
  | "view_exam_result"
  | "view_certificate"
  | ""

export const courseUnitNextStepActionFromStatus = (status?: StatusValue, allowRetake = false): CourseUnitNextStepAction => {
  switch (normalizeEnumValue(status)) {
    case "1":
      return "continue_learning"
    case "2":
      return "signup_exam"
    case "3":
      return "schedule_exam"
    case "4":
      return "view_exam_schedule"
    case "5":
      return allowRetake ? "apply_retake" : "view_exam_result"
    case "6":
      return "view_certificate"
    default:
      return ""
  }
}

export const MESSAGE_STATUS_LABELS: StatusLabelMap = {
  "0": "messagesPage.statusUnread",
  "1": "messagesPage.statusRead",
  "2": "messagesPage.statusDeleted",
  "3": "messagesPage.statusRevoked",
}

export const MESSAGE_STATUS_ENUM_NAMES: StatusEnumNameMap = {
  "0": "MESSAGE_STATUS_UNREAD",
  "1": "MESSAGE_STATUS_READ",
  "2": "MESSAGE_STATUS_DELETED",
  "3": "MESSAGE_STATUS_REVOKED",
}

export const ADMIN_APPLICATION_STATUS_LABELS: StatusLabelMap = {
  "1": "applicationsPage.statusPending",
  "2": "applicationsPage.statusApproved",
  "3": "applicationsPage.statusRejected",
  "4": "applicationsPage.statusResubmit",
}

export const ADMIN_APPLICATION_STATUS_ENUM_NAMES: StatusEnumNameMap = {
  "0": "APPLICATION_STATUS_UNSPECIFIED",
  "1": "APPLICATION_STATUS_PENDING",
  "2": "APPLICATION_STATUS_APPROVED",
  "3": "APPLICATION_STATUS_REJECTED",
  "4": "APPLICATION_STATUS_RESUBMIT",
}

export const CANDIDATE_APPLICATION_STATUS_LABELS: StatusLabelMap = {
  "1": "credentialsPage.appStatusPending",
  "2": "credentialsPage.appStatusApproved",
  "3": "credentialsPage.appStatusRejected",
  "4": "credentialsPage.appStatusResubmit",
}

export const CANDIDATE_APPLICATION_STATUS_ENUM_NAMES: StatusEnumNameMap = {
  "0": "APPLICATION_STATUS_UNSPECIFIED",
  "1": "APPLICATION_STATUS_PENDING",
  "2": "APPLICATION_STATUS_APPROVED",
  "3": "APPLICATION_STATUS_REJECTED",
  "4": "APPLICATION_STATUS_RESUBMIT",
}

export const CANDIDATE_COURSE_STATUS_LABELS: StatusLabelMap = {
  learning: "learning.statusLearning",
  completed: "learning.statusCompleted",
}

export const EXAM_STATUS_LABELS: StatusLabelMap = {
  OPEN: "examsPage.statusOpen",
  CREATED: "examsPage.statusOpen",
  DONE: "examsPage.statusPassed",
  PASSED: "examsPage.statusPassed",
  EXAM_STATUS_PASSED: "examsPage.statusPassed",
  RESULT_STATUS_PASSED: "examsPage.statusPassed",
  FAILED: "examsPage.statusFailed",
  EXAM_STATUS_FAILED: "examsPage.statusFailed",
  RESULT_STATUS_FAILED: "examsPage.statusFailed",
  SCHEDULED: "examsPage.statusScheduled",
  EXAM_STATUS_SCHEDULED: "examsPage.statusScheduled",
  PENDING: "examsPage.statusPending",
  EXAM_STATUS_PENDING: "examsPage.statusPending",
  RESULT_STATUS_PENDING: "examsPage.statusPending",
}

export const LMS_COURSE_STATUS_LABELS: StatusLabelMap = {
  draft: "lmsCoursesPage.statusDraft",
  active: "lmsCoursesPage.statusActive",
  deprecated: "lmsCoursesPage.statusDeprecated",
}

export const LMS_ENROLLMENT_STATUS_LABELS: StatusLabelMap = {
  learning: "lmsCoursesPage.statusLearning",
  testing: "lmsCoursesPage.statusTesting",
  completed: "lmsCoursesPage.statusCompleted",
}

export const LMS_LESSON_PROGRESS_STATUS_LABELS: StatusLabelMap = {
  learning: "lmsCoursesPage.statusLearning",
  testing: "lmsCoursesPage.statusTesting",
  completed: "lmsCoursesPage.statusCompleted",
}

export const LMS_CHAPTER_PROGRESS_STATUS_LABELS: StatusLabelMap = {
  learning: "lmsCoursesPage.statusLearning",
  completed: "lmsCoursesPage.statusCompleted",
}

export const LMS_QUIZ_ATTEMPT_STATUS_LABELS: StatusLabelMap = {
  completed: "lmsCoursesPage.statusCompleted",
}

export const LMS_ASSET_STATUS_LABELS: StatusLabelMap = {
  active: "lmsCoursesPage.assetStatusActive",
  broken: "lmsCoursesPage.assetStatusBroken",
  missing: "lmsCoursesPage.assetStatusMissing",
}
