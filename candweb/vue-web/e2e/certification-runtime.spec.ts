import { expect, test } from "@playwright/test"
import {
  findRuntimeCourseUnit,
  runtimeCourseHasFormalExam,
} from "../e2e-live/support/certification-runtime"

const runtime = {
  config: {
    stages: [
      {
        units: [
          {
            glms_course_id: "course-without-exam",
            course_unit_ulid: "course-unit-1",
          },
          {
            glms_course_id: "course-with-exam",
            course_unit_ulid: "course-unit-2",
            exam_id: "exam-1",
            program: "PROGRAM-1",
          },
        ],
      },
    ],
  },
}

test("runtime course lookup returns the matching course unit", () => {
  expect(findRuntimeCourseUnit(runtime, "course-without-exam")?.course_unit_ulid).toBe("course-unit-1")
  expect(findRuntimeCourseUnit(runtime, "missing-course")).toBeNull()
})

test("a runtime course unit ID alone does not imply a formal exam", () => {
  expect(runtimeCourseHasFormalExam(runtime, "course-without-exam")).toBe(false)
  expect(runtimeCourseHasFormalExam(runtime, "course-with-exam")).toBe(true)
})
