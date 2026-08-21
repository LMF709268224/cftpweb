export function findRuntimeCourseUnit(runtime: any, courseID: string) {
  const expectedCourseID = String(courseID || "").trim()
  if (!expectedCourseID) return null

  for (const stage of runtime?.config?.stages || []) {
    for (const unit of stage?.units || []) {
      const unitCourseID = String(
        unit?.glms_course_id || unit?.course_id || unit?.course_ulid || unit?.courseUlid || "",
      ).trim()
      if (unitCourseID === expectedCourseID) return unit
    }
  }
  return null
}

export function runtimeCourseHasFormalExam(runtime: any, courseID: string) {
  const unit = findRuntimeCourseUnit(runtime, courseID)
  return Boolean(unit?.exam_id || unit?.program)
}
