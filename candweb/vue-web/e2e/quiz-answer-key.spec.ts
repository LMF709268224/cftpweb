import { expect, test } from "@playwright/test"
import {
  answersFromQuizAnswerKey,
  loadQuizAnswerKey,
} from "../e2e-live/support/quiz-answer-key"

test("CourseSeeder answer key maps imported question text to runtime option IDs", () => {
  const answerKey = loadQuizAnswerKey("foundation_crypto_regulation")
  expect(answerKey).not.toBeNull()

  const answers = answersFromQuizAnswerKey(
    {
      questions: [
        {
          question_ulid: "question-runtime-1",
          question_text: "Which NFT project was created in 2015 and has since become a collector's item for being part of Ethereum's history?",
          options: [
            { option_ulid: "option-runtime-1", option_text: "CryptoPunks" },
            { option_ulid: "option-runtime-2", option_text: "CryptoKitties" },
            { option_ulid: "option-runtime-3", option_text: "Etheria" },
          ],
        },
      ],
    },
    answerKey!,
  )

  expect(answers.get("question-runtime-1")).toEqual(["option-runtime-3"])
})

test("unknown answer key fails before mutating live test data", () => {
  expect(() => loadQuizAnswerKey("unknown-course")).toThrow(/Unsupported live journey quiz answer key/)
})
