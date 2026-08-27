package handler

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	gexampb "github.com/afnandelfin620-star/cftptest/cftp/gexam"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
	"github.com/xuri/excelize/v2"
)

const (
	essayGradingSheet       = "Essay Grading"
	essayGradingFormat      = "1"
	essayGradingMaxFileSize = 10 << 20
	essayGradingHeaderRow   = 16
)

type essayGradingWorkbookItem struct {
	QuestionSeq       int32   `json:"question_seq"`
	QuestionName      string  `json:"question_name"`
	SectionName       string  `json:"section_name"`
	CandidateResponse string  `json:"candidate_response"`
	MaxScore          float64 `json:"max_score"`
	Score             float64 `json:"score"`
	Comment           string  `json:"comment"`
}

type essayGradingWorkbook struct {
	ExamULID       string                     `json:"exam_ulid"`
	CandidateName  string                     `json:"candidate_name"`
	CandidateEmail string                     `json:"candidate_email"`
	ProgramCode    string                     `json:"program_code"`
	ExamCode       string                     `json:"exam_code"`
	ExamForm       string                     `json:"exam_form"`
	ObjectiveScore float64                    `json:"objective_score"`
	GraderName     string                     `json:"grader_name"`
	GraderID       string                     `json:"grader_id,omitempty"`
	IsPassed       bool                       `json:"is_passed"`
	OverallComment string                     `json:"overall_comment"`
	EssayScore     float64                    `json:"essay_score"`
	FinalScore     float64                    `json:"final_score"`
	Items          []essayGradingWorkbookItem `json:"items"`
}

func (h *Handler) ExportExamEssayGradeWorkbook(w http.ResponseWriter, r *http.Request) {
	examULID := strings.TrimSpace(chi.URLParam(r, "exam_ulid"))
	if !requireRequestField(w, examULID, "exam_ulid") {
		return
	}
	detail, err := h.Gexam.GetExamEssayDetails(r.Context(), &gexampb.GetExamEssayDetailsRequest{ExamUlid: examULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if detail.GetResultStatus() != "PENDING_GRADING" {
		WriteError(w, http.StatusConflict, ErrPrecondition, "exam is no longer pending grading")
		return
	}
	if len(detail.GetEssays()) == 0 {
		WriteError(w, http.StatusConflict, ErrPrecondition, "exam has no essay questions to grade")
		return
	}

	book, err := buildEssayGradingWorkbook(detail)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to build grading workbook")
		return
	}
	defer func() { _ = book.Close() }()

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "essay-grading-"+examULID+".xlsx"))
	w.Header().Set("Cache-Control", "no-store")
	if err := book.Write(w); err != nil {
		return
	}
}

func (h *Handler) PreviewExamEssayGradeWorkbook(w http.ResponseWriter, r *http.Request) {
	parsed, ok := h.readAndValidateEssayGradingWorkbook(w, r)
	if !ok {
		return
	}
	WriteJSON(w, http.StatusOK, parsed)
}

func (h *Handler) ImportExamEssayGradeWorkbook(w http.ResponseWriter, r *http.Request) {
	parsed, ok := h.readAndValidateEssayGradingWorkbook(w, r)
	if !ok {
		return
	}
	items := make([]*gexampb.ExamEssayGradeItem, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		items = append(items, &gexampb.ExamEssayGradeItem{
			QuestionSeq: item.QuestionSeq,
			Score:       item.Score,
			Comment:     item.Comment,
		})
	}
	resp, err := h.Gexam.SubmitExamEssayGrade(r.Context(), &gexampb.SubmitExamEssayGradeRequest{
		ExamUlid:       parsed.ExamULID,
		GraderId:       parsed.GraderID,
		GraderName:     parsed.GraderName,
		IsPassed:       parsed.IsPassed,
		Items:          items,
		OverallComment: parsed.OverallComment,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) readAndValidateEssayGradingWorkbook(w http.ResponseWriter, r *http.Request) (*essayGradingWorkbook, bool) {
	examULID := strings.TrimSpace(chi.URLParam(r, "exam_ulid"))
	if !requireRequestField(w, examULID, "exam_ulid") {
		return nil, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, essayGradingMaxFileSize)
	if err := r.ParseMultipartForm(essayGradingMaxFileSize); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "grading workbook must be a file smaller than 10 MB")
		return nil, false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "grading workbook file is required")
		return nil, false
	}
	defer func() { _ = file.Close() }()
	if !strings.EqualFold(filepath.Ext(header.Filename), ".xlsx") {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "grading workbook must be an .xlsx file")
		return nil, false
	}
	book, err := excelize.OpenReader(file)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "grading workbook is invalid or corrupted")
		return nil, false
	}
	defer func() { _ = book.Close() }()

	detail, err := h.Gexam.GetExamEssayDetails(r.Context(), &gexampb.GetExamEssayDetailsRequest{ExamUlid: examULID})
	if err != nil {
		HandleGrpcError(w, err)
		return nil, false
	}
	if detail.GetResultStatus() != "PENDING_GRADING" {
		WriteError(w, http.StatusConflict, ErrPrecondition, "exam is no longer pending grading")
		return nil, false
	}
	if len(detail.GetEssays()) == 0 {
		WriteError(w, http.StatusConflict, ErrPrecondition, "exam has no essay questions to grade")
		return nil, false
	}
	parsed, err := parseEssayGradingWorkbook(book, examULID, detail)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, err.Error())
		return nil, false
	}
	return parsed, true
}

func buildEssayGradingWorkbook(detail *gexampb.GetExamEssayDetailsResponse) (*excelize.File, error) {
	book := excelize.NewFile()
	defaultSheet := book.GetSheetName(0)
	if _, err := book.NewSheet(essayGradingSheet); err != nil {
		return nil, err
	}
	if defaultSheet != "" && defaultSheet != essayGradingSheet {
		_ = book.DeleteSheet(defaultSheet)
	}
	sheetIndex, err := book.GetSheetIndex(essayGradingSheet)
	if err != nil {
		return nil, err
	}
	book.SetActiveSheet(sheetIndex)

	_ = book.MergeCell(essayGradingSheet, "A1", "G1")
	_ = book.SetCellValue(essayGradingSheet, "A1", "Prometric Essay Grading Workbook")
	_ = book.MergeCell(essayGradingSheet, "A2", "G2")
	_ = book.SetCellValue(essayGradingSheet, "A2", "Professor: fill only the yellow cells. Use TRUE or FALSE for is_passed. Return this .xlsx file to the administrator.")

	metadata := [][3]any{
		{"format_version", essayGradingFormat, "Do not modify"},
		{"exam_ulid", detail.GetExamUlid(), "Do not modify"},
		{"candidate_name", strings.TrimSpace(detail.GetCandidateFirstName() + " " + detail.GetCandidateLastName()), "Do not modify"},
		{"candidate_email", detail.GetCandidateEmail(), "Do not modify"},
		{"program_code", detail.GetProgramCode(), "Do not modify"},
		{"exam_code", detail.GetExamCode(), "Do not modify"},
		{"exam_form", detail.GetExamForm(), "Do not modify"},
		{"objective_score", detail.GetObjectiveScore(), "Do not modify"},
		{"grader_name", "", "Required: professor name, maximum 100 characters"},
		{"grader_id", "", "Optional: professor ULID"},
		{"is_passed", "", "Required: TRUE or FALSE"},
		{"overall_comment", "", "Optional: maximum 1000 characters"},
	}
	for index, values := range metadata {
		row := index + 3
		_ = book.SetCellValue(essayGradingSheet, fmt.Sprintf("A%d", row), values[0])
		_ = book.SetCellValue(essayGradingSheet, fmt.Sprintf("B%d", row), values[1])
		_ = book.SetCellValue(essayGradingSheet, fmt.Sprintf("C%d", row), values[2])
	}

	headers := []string{"question_seq", "question_name", "section_name", "candidate_response", "max_score", "score", "comment"}
	for column, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(column+1, essayGradingHeaderRow)
		_ = book.SetCellValue(essayGradingSheet, cell, header)
	}
	for index, essay := range detail.GetEssays() {
		row := essayGradingHeaderRow + index + 1
		values := []any{essay.GetQuestionSeq(), essay.GetQuestionName(), essay.GetSectionName(), essay.GetCandidateResponse(), essay.GetMaxScore(), "", ""}
		for column, value := range values {
			cell, _ := excelize.CoordinatesToCellName(column+1, row)
			_ = book.SetCellValue(essayGradingSheet, cell, value)
		}
	}

	_ = book.SetColWidth(essayGradingSheet, "A", "A", 20)
	_ = book.SetColWidth(essayGradingSheet, "B", "B", 32)
	_ = book.SetColWidth(essayGradingSheet, "C", "C", 42)
	_ = book.SetColWidth(essayGradingSheet, "D", "D", 72)
	_ = book.SetColWidth(essayGradingSheet, "E", "F", 14)
	_ = book.SetColWidth(essayGradingSheet, "G", "G", 44)
	_ = book.SetRowHeight(essayGradingSheet, 2, 34)

	titleStyle, _ := book.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 16, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"1D4ED8"}, Pattern: 1}, Alignment: &excelize.Alignment{Vertical: "center"}})
	headerStyle, _ := book.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"334155"}, Pattern: 1}, Alignment: &excelize.Alignment{WrapText: true}})
	inputStyle, _ := book.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Color: []string{"FEF3C7"}, Pattern: 1}, Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"}})
	wrapStyle, _ := book.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"}})
	_ = book.SetCellStyle(essayGradingSheet, "A1", "G1", titleStyle)
	_ = book.SetCellStyle(essayGradingSheet, "A16", "G16", headerStyle)
	_ = book.SetCellStyle(essayGradingSheet, "B11", "B14", inputStyle)
	if len(detail.GetEssays()) > 0 {
		lastRow := essayGradingHeaderRow + len(detail.GetEssays())
		_ = book.SetCellStyle(essayGradingSheet, fmt.Sprintf("A%d", essayGradingHeaderRow+1), fmt.Sprintf("G%d", lastRow), wrapStyle)
		_ = book.SetCellStyle(essayGradingSheet, fmt.Sprintf("F%d", essayGradingHeaderRow+1), fmt.Sprintf("G%d", lastRow), inputStyle)
	}
	_ = book.SetPanes(essayGradingSheet, &excelize.Panes{Freeze: true, Split: true, YSplit: essayGradingHeaderRow, TopLeftCell: "A17", ActivePane: "bottomLeft"})
	return book, nil
}

func parseEssayGradingWorkbook(book *excelize.File, routeExamULID string, detail *gexampb.GetExamEssayDetailsResponse) (*essayGradingWorkbook, error) {
	rows, err := book.GetRows(essayGradingSheet)
	if err != nil {
		return nil, fmt.Errorf("grading workbook must contain the %q sheet", essayGradingSheet)
	}
	metadata := make(map[string]string)
	headerRow := -1
	for index, row := range rows {
		key := workbookCell(row, 0)
		if key == "question_seq" {
			headerRow = index
			break
		}
		if key != "" {
			metadata[key] = workbookCell(row, 1)
		}
	}
	if metadata["format_version"] != essayGradingFormat {
		return nil, fmt.Errorf("unsupported grading workbook format_version")
	}
	if metadata["exam_ulid"] != routeExamULID || metadata["exam_ulid"] != detail.GetExamUlid() {
		return nil, fmt.Errorf("grading workbook exam_ulid does not match the selected exam")
	}
	graderName := strings.TrimSpace(metadata["grader_name"])
	if graderName == "" || utf8.RuneCountInString(graderName) > 100 {
		return nil, fmt.Errorf("grader_name is required and must be at most 100 characters")
	}
	graderID := strings.TrimSpace(metadata["grader_id"])
	if graderID != "" {
		if _, err := ulid.ParseStrict(graderID); err != nil {
			return nil, fmt.Errorf("grader_id must be a valid ULID or left blank")
		}
	}
	isPassed, err := parseWorkbookBool(metadata["is_passed"])
	if err != nil {
		return nil, err
	}
	overallComment := strings.TrimSpace(metadata["overall_comment"])
	if utf8.RuneCountInString(overallComment) > 1000 {
		return nil, fmt.Errorf("overall_comment must be at most 1000 characters")
	}
	if headerRow < 0 {
		return nil, fmt.Errorf("grading workbook question table is missing")
	}

	detailBySequence := make(map[int32]*gexampb.ExamEssayItemDetail, len(detail.GetEssays()))
	for _, essay := range detail.GetEssays() {
		detailBySequence[essay.GetQuestionSeq()] = essay
	}
	items := make([]essayGradingWorkbookItem, 0, len(detailBySequence))
	seen := make(map[int32]struct{}, len(detailBySequence))
	var essayScore float64
	for index := headerRow + 1; index < len(rows); index++ {
		row := rows[index]
		if strings.TrimSpace(workbookCell(row, 0)) == "" {
			continue
		}
		sequenceValue, err := strconv.Atoi(strings.TrimSpace(workbookCell(row, 0)))
		if err != nil || sequenceValue <= 0 {
			return nil, fmt.Errorf("row %d has an invalid question_seq", index+1)
		}
		sequence := int32(sequenceValue)
		detailEssay, exists := detailBySequence[sequence]
		if !exists {
			return nil, fmt.Errorf("row %d question_seq %d does not belong to this exam", index+1, sequence)
		}
		if _, duplicate := seen[sequence]; duplicate {
			return nil, fmt.Errorf("question_seq %d is duplicated", sequence)
		}
		seen[sequence] = struct{}{}
		maxScore, err := parseWorkbookScore(workbookCell(row, 4), "max_score", index+1)
		if err != nil || maxScore <= 0 || maxScore != detailEssay.GetMaxScore() {
			return nil, fmt.Errorf("row %d has an invalid max_score", index+1)
		}
		score, err := parseWorkbookScore(workbookCell(row, 5), "score", index+1)
		if err != nil {
			return nil, err
		}
		if score < 0 || score > maxScore {
			return nil, fmt.Errorf("row %d score must be between 0 and max_score", index+1)
		}
		comment := strings.TrimSpace(workbookCell(row, 6))
		if utf8.RuneCountInString(comment) > 500 {
			return nil, fmt.Errorf("row %d comment must be at most 500 characters", index+1)
		}
		items = append(items, essayGradingWorkbookItem{
			QuestionSeq:       sequence,
			QuestionName:      detailEssay.GetQuestionName(),
			SectionName:       detailEssay.GetSectionName(),
			CandidateResponse: detailEssay.GetCandidateResponse(),
			MaxScore:          maxScore,
			Score:             score,
			Comment:           comment,
		})
		essayScore += score
	}
	if len(items) != len(detailBySequence) {
		return nil, fmt.Errorf("grading workbook must contain all %d essay questions", len(detailBySequence))
	}

	return &essayGradingWorkbook{
		ExamULID:       routeExamULID,
		CandidateName:  strings.TrimSpace(detail.GetCandidateFirstName() + " " + detail.GetCandidateLastName()),
		CandidateEmail: detail.GetCandidateEmail(),
		ProgramCode:    detail.GetProgramCode(),
		ExamCode:       detail.GetExamCode(),
		ExamForm:       detail.GetExamForm(),
		ObjectiveScore: detail.GetObjectiveScore(),
		GraderName:     graderName,
		GraderID:       graderID,
		IsPassed:       isPassed,
		OverallComment: overallComment,
		EssayScore:     essayScore,
		FinalScore:     detail.GetObjectiveScore() + essayScore,
		Items:          items,
	}, nil
}

func workbookCell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func parseWorkbookBool(value string) (bool, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "TRUE":
		return true, nil
	case "FALSE":
		return false, nil
	default:
		return false, fmt.Errorf("is_passed must be TRUE or FALSE")
	}
}

func parseWorkbookScore(value, field string, row int) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("row %d %s is required", row, field)
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("row %d has an invalid %s", row, field)
	}
	return parsed, nil
}
