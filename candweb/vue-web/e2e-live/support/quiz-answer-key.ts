import { readFileSync } from "node:fs"

export type QuizAnswerKey = Map<string, string[]>
export type QuizAnswerSelection = Map<string, string[]>

const answerKeyFiles = {
  foundation_crypto_regulation: new URL(
    "../../../../courseseeder/foundation_crypto_regulation.course-package.json",
    import.meta.url,
  ),
} as const

function normalizedQuizText(value: unknown) {
  return String(value || "")
    .normalize("NFKC")
    .trim()
    .replace(/\s+/g, " ")
    .toLowerCase()
}

function questionText(question: any) {
  return String(question?.question_text || question?.questionText || "").trim()
}

function questionID(question: any) {
  return String(
    question?.question_id || question?.question_ulid || question?.questionUlid || "",
  ).trim()
}

function optionText(option: any) {
  return String(option?.option_text || option?.optionText || "").trim()
}

function optionID(option: any) {
  return String(
    option?.option_id || option?.option_ulid || option?.optionUlid || "",
  ).trim()
}

export function loadQuizAnswerKey(name: string): QuizAnswerKey | null {
  const keyName = String(name || "").trim()
  if (!keyName || keyName === "none") return null

  const fixtureURL = answerKeyFiles[keyName as keyof typeof answerKeyFiles]
  if (!fixtureURL) {
    throw new Error(
      `Unsupported live journey quiz answer key ${JSON.stringify(keyName)}. Supported keys: ${Object.keys(answerKeyFiles).join(", ")}`,
    )
  }

  const coursePackage = JSON.parse(readFileSync(fixtureURL, "utf8")) as {
    quizzes?: Array<{
      questions?: Array<{
        question_text?: string
        options?: Array<{ option_text?: string; is_correct?: boolean }>
      }>
    }>
  }
  const answerKey: QuizAnswerKey = new Map()

  for (const quiz of coursePackage.quizzes || []) {
    for (const question of quiz.questions || []) {
      const rawQuestionText = String(question.question_text || "").trim()
      const normalizedQuestion = normalizedQuizText(rawQuestionText)
      if (!normalizedQuestion) throw new Error("Quiz answer fixture contains a question without text")

      const correctOptions = (question.options || [])
        .filter(option => option.is_correct === true)
        .map(option => normalizedQuizText(option.option_text))
        .filter(Boolean)
        .sort()
      if (correctOptions.length === 0) {
        throw new Error(`Quiz answer fixture has no correct option for question ${JSON.stringify(rawQuestionText)}`)
      }

      const existing = answerKey.get(normalizedQuestion)
      if (existing && existing.join("\n") !== correctOptions.join("\n")) {
        throw new Error(`Quiz answer fixture has conflicting answers for question ${JSON.stringify(rawQuestionText)}`)
      }
      answerKey.set(normalizedQuestion, correctOptions)
    }
  }

  if (answerKey.size === 0) throw new Error(`Quiz answer fixture ${keyName} contains no questions`)
  return answerKey
}

export function answersFromQuizAnswerKey(
  paper: any,
  answerKey: QuizAnswerKey,
): QuizAnswerSelection {
  const answers: QuizAnswerSelection = new Map()

  for (const question of paper?.questions || []) {
    const currentQuestionID = questionID(question)
    const rawQuestionText = questionText(question)
    if (!currentQuestionID) throw new Error("Quiz paper question does not expose an ID")
    if (!rawQuestionText) throw new Error(`Quiz paper question ${currentQuestionID} does not expose text`)

    const correctOptionTexts = answerKey.get(normalizedQuizText(rawQuestionText))
    if (!correctOptionTexts) {
      throw new Error(`Quiz answer fixture does not contain question ${JSON.stringify(rawQuestionText)}`)
    }

    const optionsByText = new Map<string, string>()
    for (const option of question?.options || []) {
      const currentOptionID = optionID(option)
      const currentOptionText = normalizedQuizText(optionText(option))
      if (!currentOptionID || !currentOptionText) continue
      if (optionsByText.has(currentOptionText)) {
        throw new Error(`Quiz paper contains duplicate option text for question ${JSON.stringify(rawQuestionText)}`)
      }
      optionsByText.set(currentOptionText, currentOptionID)
    }

    const selectedOptionIDs = correctOptionTexts.map((correctOptionText) => {
      const currentOptionID = optionsByText.get(correctOptionText)
      if (!currentOptionID) {
        throw new Error(
          `Quiz paper is missing a configured correct option for question ${JSON.stringify(rawQuestionText)}`,
        )
      }
      return currentOptionID
    })
    answers.set(currentQuestionID, selectedOptionIDs)
  }

  if (answers.size === 0) throw new Error("Quiz paper contains no questions")
  return answers
}
