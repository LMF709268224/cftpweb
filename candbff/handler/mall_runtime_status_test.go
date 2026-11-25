package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	gexampb "github.com/afnandelfin620-star/cftptest/cftp/gexam"
	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"google.golang.org/grpc"
)

type runtimeExamClientStub struct {
	gexampb.GExamServiceClient
	detail      *gexampb.ExamDetail
	err         error
	requestedID string
}

func (s *runtimeExamClientStub) GetExamDetail(_ context.Context, req *gexampb.GetExamRequest, _ ...grpc.CallOption) (*gexampb.ExamDetail, error) {
	s.requestedID = req.GetExamUlid()
	return s.detail, s.err
}

func TestRuntimeLearningProgressPreservesZeroAndMissing(t *testing.T) {
	config := PipelineConfig{Stages: []StageConfig{{Units: []UnitConfig{
		{GlmsCourseUlid: "zero"}, {GlmsCourseUlid: "partial"}, {GlmsCourseUlid: "missing"},
	}}}}
	mergeLearningAccess(&config, []*lmspb.CandidateEnrollmentSummary{
		{CourseUlid: "zero", Status: "learning", ProgressPercentage: 0},
		{CourseUlid: "partial", Status: "learning", ProgressPercentage: 40},
	})
	encoded, err := json.Marshal(config.Stages[0].Units)
	if err != nil {
		t.Fatal(err)
	}
	var units []map[string]any
	if err := json.Unmarshal(encoded, &units); err != nil {
		t.Fatal(err)
	}
	if progress, ok := units[0]["learning_progress_percentage"]; !ok || progress != float64(0) {
		t.Fatalf("real zero progress lost: %s", encoded)
	}
	if units[1]["learning_progress_percentage"] != float64(40) {
		t.Fatalf("wrong partial progress: %s", encoded)
	}
	if _, ok := units[2]["learning_progress_percentage"]; ok {
		t.Fatalf("missing enrollment fabricated progress: %s", encoded)
	}
}

func TestRuntimeExamUsesCurrentAttemptAndChecksOwnership(t *testing.T) {
	for _, tc := range []struct {
		name        string
		examID      string
		candidateID string
		unitID      string
		err         error
		want        bool
	}{
		{"current booked exam", "current-exam", "candidate", "unit", nil, true},
		{"old attempt", "old-exam", "candidate", "unit", nil, false},
		{"different candidate", "current-exam", "someone-else", "unit", nil, false},
		{"different unit", "current-exam", "candidate", "another-unit", nil, false},
		{"service unavailable", "current-exam", "candidate", "unit", errors.New("unavailable"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := PipelineConfig{Stages: []StageConfig{{StageUlid: "stage-config", Units: []UnitConfig{{UnitUlid: "unit-config", ExamUlid: "exam-config"}}}}}
			mergeRuntimeStatuses(&config, &gprogpb.GetPipelineDetailRsp{Stages: []*gprogpb.StageDetail{{
				Stage:       &gprogpb.StageSummary{StageCcUlid: "stage-config"},
				CourseUnits: []*gprogpb.CourseUnitSummary{{CourseUnitCcUlid: "unit-config", CourseUnitUlid: "unit", ExamUlid: "current-exam"}},
			}}})
			client := &runtimeExamClientStub{detail: &gexampb.ExamDetail{
				ExamUlid: tc.examID, CandidateUlid: tc.candidateID, CourseUnitUlid: tc.unitID,
				ExamStatus: "SCHEDULED", AppointmentStartTime: "2026-12-01T08:00:00Z",
			}, err: tc.err}
			(&Handler{Gexam: client}).enrichRuntimeExams(context.Background(), "candidate", &config)
			unit := config.Stages[0].Units[0]
			if client.requestedID != "current-exam" || unit.ExamUlid != "exam-config" {
				t.Fatalf("confused config and runtime exam: %+v", unit)
			}
			if (unit.CurrentExam != nil) != tc.want {
				t.Fatalf("unexpected exam visibility: %+v", unit.CurrentExam)
			}
			if tc.want && (unit.CurrentExam.Status != "SCHEDULED" || unit.CurrentExam.AppointmentStartTime != "2026-12-01T08:00:00Z") {
				t.Fatalf("lost appointment data: %+v", unit.CurrentExam)
			}
		})
	}
}
