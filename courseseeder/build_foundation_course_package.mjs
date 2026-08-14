import fs from "node:fs"
import path from "node:path"

const root = path.dirname(new URL(import.meta.url).pathname).replace(/^\//, "").replaceAll("%20", " ")
const contentPath = path.join(root, "course_content.txt")
const quizPath = path.join(root, "course_quiz.txt")
const outputPath = path.join(root, "foundation_crypto_regulation.course-package.json")

const courseDescription =
  "The Foundation in Crypto Regulation and Compliance Course is catered for professionals in regulatory bodies, financial institutions, and compliance roles, especially those with a passion and curiosity for crypto. Designed by leading industry experts and academics, this 16-hour online program provides regulators and compliance professionals with a comprehensive understanding of blockchain, cryptocurrency, and financial regulations. Participants will also gain hands-on insights into numerous important areas such as AML/KYC compliance and financial crime prevention, blockchain forensics and cross-border regulatory frameworks, RegTech solutions, and more. By the end of the course, attendees will be well-equipped to tackle compliance challenges, mitigate risks, and adapt to the evolving crypto landscape."

function fields(line) {
  return line.split("|").map((value) => value.trim())
}

function dataLines(filePath, expectedFields) {
  return fs.readFileSync(filePath, "utf8")
    .split(/\r?\n/)
    .map(fields)
    .filter((values) => values.length === expectedFields && /^\d+$/.test(values[0]))
}

const contentRows = dataLines(contentPath, 9)
const chapterMap = new Map()
for (const row of contentRows) {
  const sourcePosition = Number(row[0])
  const chapter = chapterMap.get(sourcePosition) || {
    sourcePosition,
    title: row[1],
    lessons: [],
  }
  if (row[3].toLowerCase() === "video") {
    chapter.lessons.push({
      title: row[4],
      lesson_type: "video",
      video_provider: "cloudflare",
      video_stream_uid: row[6],
    })
  }
  chapterMap.set(sourcePosition, chapter)
}

const chapters = [...chapterMap.values()]
  .sort((left, right) => left.sourcePosition - right.sourcePosition)
  .filter(({ lessons }) => lessons.length > 0)
  .map(({ title, lessons }) => ({ title, lessons }))

const quizDescriptions = {
  "Module 1 Quiz": "Test your knowledge of blockchain technology and digital assets",
  "Module 2 Quiz": "Test your knowledge of crypto regulations and compliance",
  "Module 3 Quiz": "Test your knowledge of RegTech and SupTech",
  "Module 4 Quiz": "Test your knowledge of stablecoins, CBDCs, and regulatory projects",
}

const quizChapterTitles = {
  "Module 1 Quiz": "Module 1: Case Studies",
  "Module 2 Quiz": "Module 2: Cross-Border Issues",
  "Module 3 Quiz": "Module 3: Regulatory Technology Use Cases",
  "Module 4 Quiz": "Module 4 Case Study: Project Mandala",
}

const quizMap = new Map()
for (const row of dataLines(quizPath, 10)) {
  const sourcePosition = Number(row[0])
  const title = row[1]
  const questionNumber = Number(row[5])
  const optionNumber = Number(row[7])
  if (!quizChapterTitles[title]) {
    throw new Error(`missing chapter mapping for quiz: ${title}`)
  }
  const quiz = quizMap.get(sourcePosition) || {
    sourcePosition,
    chapter_title: quizChapterTitles[title],
    title,
    description: quizDescriptions[title] || "",
    passing_score: Number(row[2]),
    time_limit: Number(row[3]),
    randomize_questions: row[4].toLowerCase() === "t",
    quiz_type: 1,
    questions: new Map(),
  }
  const question = quiz.questions.get(questionNumber) || {
    question_text: row[6],
    question_type: 1,
    points: 1,
    sort_order: questionNumber,
    is_required: true,
    explanation: "",
    media_items_json: "[]",
    options: [],
  }
  question.options.push({
    option_text: row[8],
    is_correct: row[9].toLowerCase() === "t",
    sort_order: optionNumber,
  })
  quiz.questions.set(questionNumber, question)
  quizMap.set(sourcePosition, quiz)
}

const quizzes = [...quizMap.values()]
  .sort((left, right) => left.sourcePosition - right.sourcePosition)
  .map(({ sourcePosition, questions, ...quiz }) => ({
    ...quiz,
    questions: [...questions.values()].sort((left, right) => left.sort_order - right.sort_order),
  }))

const packageDocument = {
  package_type: "cftp-lms-course-package",
  course_gpath: "/gcc/pipeline/core/foundation_in_crypto_regulation_and_compliance",
  package_version: 1,
  category_tips: "Crypto Regulation & Compliance",
  title: "Foundation in Crypto Regulation and Compliance",
  description: courseDescription,
  duration_min: 960,
  certification_enabled: false,
  chapters,
  quizzes,
}

const lessonCount = chapters.reduce((count, chapter) => count + chapter.lessons.length, 0)
if (chapters.length !== 26 || lessonCount !== 109 || quizzes.length !== 4) {
  throw new Error(`unexpected source shape: chapters=${chapters.length}, lessons=${lessonCount}, quizzes=${quizzes.length}`)
}

fs.writeFileSync(outputPath, `${JSON.stringify(packageDocument, null, 2)}\n`, "utf8")
console.log(`wrote ${outputPath}: ${chapters.length} chapters, ${lessonCount} videos, ${quizzes.length} quizzes`)
