package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
	"google.golang.org/grpc"
)

type localizationCCClient struct {
	gccpb.CCServiceClient
}

func (localizationCCClient) GetPipelineTranslations(
	_ context.Context,
	_ *gccpb.GetPipelineTranslationsRequest,
	_ ...grpc.CallOption,
) (*gccpb.GetPipelineTranslationsResponse, error) {
	return &gccpb.GetPipelineTranslationsResponse{
		Translations: map[string]*gccpb.PipelineTranslation{
			"zh-CN": {
				Name:         "认证名称",
				CategoryTips: "分类提示",
				Description:  "认证描述",
			},
		},
	}, nil
}

func (localizationCCClient) GetStageTranslations(
	_ context.Context,
	_ *gccpb.GetStageTranslationsRequest,
	_ ...grpc.CallOption,
) (*gccpb.GetStageTranslationsResponse, error) {
	return &gccpb.GetStageTranslationsResponse{
		Translations: map[string]*gccpb.StageTranslation{
			"zh-CN": {Name: "第一阶段"},
		},
	}, nil
}

func (localizationCCClient) GetUnitTranslations(
	_ context.Context,
	_ *gccpb.GetUnitTranslationsRequest,
	_ ...grpc.CallOption,
) (*gccpb.GetUnitTranslationsResponse, error) {
	return &gccpb.GetUnitTranslationsResponse{
		Translations: map[string]*gccpb.UnitTranslation{
			"zh-CN": {Name: "课程单元"},
		},
	}, nil
}

type localizationCredentialClient struct {
	gcredspb.CredentialServiceClient
}

type localizationLMSClient struct {
	lmspb.LmsServiceClient
}

func (localizationLMSClient) GetCourseTranslations(
	_ context.Context,
	_ *lmspb.GetCourseTranslationsRequest,
	_ ...grpc.CallOption,
) (*lmspb.GetCourseTranslationsResponse, error) {
	return &lmspb.GetCourseTranslationsResponse{
		Translations: map[string]*lmspb.CourseTranslation{
			"zh-CN": {
				Title:       "课程标题",
				Description: "课程描述",
			},
		},
	}, nil
}

func (localizationLMSClient) GetResourcePackTranslations(
	_ context.Context,
	_ *lmspb.GetResourcePackTranslationsRequest,
	_ ...grpc.CallOption,
) (*lmspb.GetResourcePackTranslationsResponse, error) {
	return &lmspb.GetResourcePackTranslationsResponse{
		Translations: map[string]*lmspb.ResourcePackTranslation{
			"zh-CN": {
				Title:       "资源包标题",
				Description: "资源包描述",
			},
		},
	}, nil
}

func (localizationLMSClient) GetResourcePackFileTranslations(
	_ context.Context,
	_ *lmspb.GetResourcePackFileTranslationsRequest,
	_ ...grpc.CallOption,
) (*lmspb.GetResourcePackFileTranslationsResponse, error) {
	return &lmspb.GetResourcePackFileTranslationsResponse{
		Translations: map[string]*lmspb.ResourcePackFileTranslation{
			"zh-CN": {
				Title:       "文件标题",
				Description: "文件描述",
			},
		},
	}, nil
}

func (localizationCredentialClient) GetCredDefTranslations(
	_ context.Context,
	_ *gcredspb.GetCredDefTranslationsRequest,
	_ ...grpc.CallOption,
) (*gcredspb.GetCredDefTranslationsResponse, error) {
	return &gcredspb.GetCredDefTranslationsResponse{
		Translations: map[string]*gcredspb.CredDefTranslation{
			"zh-CN": {
				Name:              "工作经验证明",
				Description:       "提交工作经验证明材料",
				AcquisitionMethod: "提交后等待审核",
				FileConstraintNames: map[string]string{
					"employment_certificate": "雇佣证明",
				},
			},
		},
	}, nil
}

type localizationMembershipClient struct {
	gmbrpb.GmbrServiceClient
}

type localizationMallClient struct {
	mallpb.MallServiceClient
}

func (localizationMallClient) GetBundleTranslations(
	_ context.Context,
	_ *mallpb.GetBundleTranslationsRequest,
	_ ...grpc.CallOption,
) (*mallpb.GetBundleTranslationsResponse, error) {
	return &mallpb.GetBundleTranslationsResponse{
		Translations: map[string]*mallpb.BundleTranslation{
			"zh-CN": {
				Name:        "认证商品",
				Description: "认证商品描述",
			},
		},
	}, nil
}

func (localizationMembershipClient) GetMembershipTranslations(
	_ context.Context,
	_ *gmbrpb.GetMembershipTranslationsRequest,
	_ ...grpc.CallOption,
) (*gmbrpb.GetMembershipTranslationsResponse, error) {
	return &gmbrpb.GetMembershipTranslationsResponse{
		Translations: map[string]*gmbrpb.MembershipTranslation{
			"zh-CN": {
				Name:        "专业会员",
				Description: "会员方案描述",
				IdealFor:    "适合金融科技从业者",
				FeatureCategories: map[string]string{
					"events_seminars": "活动与研讨会",
				},
				FeatureLabels: map[string]string{
					"live_webinars": "在线研讨会",
				},
			},
		},
	}, nil
}

func TestRequestLocale(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{name: "Chinese short code", header: "zh", want: "zh-CN"},
		{name: "English short code", header: "en", want: "en-US"},
		{name: "Locale list", header: "zh-CN,zh;q=0.9,en;q=0.8", want: "zh-CN"},
		{name: "Underscore locale", header: "en_US", want: "en-US"},
		{name: "Missing locale", want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/", nil)
			request.Header.Set("Accept-Language", test.header)
			if got := requestLocale(request); got != test.want {
				t.Fatalf("requestLocale() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestLocalizedGLMSContent(t *testing.T) {
	handler := &Handler{Lms: localizationLMSClient{}}

	baseCourse := &lmspb.Course{
		CourseUlid:  "course-1",
		Title:       "Course title",
		Description: "Course description",
	}
	localizedCourse := handler.localizedCourse(context.Background(), baseCourse, "zh-CN")
	if localizedCourse.GetTitle() != "课程标题" || localizedCourse.GetDescription() != "课程描述" {
		t.Fatalf("course translation was not applied: %#v", localizedCourse)
	}
	if baseCourse.GetTitle() != "Course title" {
		t.Fatal("course localization mutated the source course")
	}

	basePack := &lmspb.ResourcePack{
		PackId:      "pack-1",
		Title:       "Resource pack",
		Description: "Resource pack description",
	}
	localizedPack := handler.localizedResourcePack(context.Background(), basePack, "zh-CN")
	if localizedPack.GetTitle() != "资源包标题" || localizedPack.GetDescription() != "资源包描述" {
		t.Fatalf("resource pack translation was not applied: %#v", localizedPack)
	}

	baseFile := &lmspb.ResourcePackFile{
		FileId:      "file-1",
		Title:       "Resource file",
		Description: "Resource file description",
	}
	localizedFile := handler.localizedResourcePackFile(context.Background(), baseFile, "zh-CN")
	if localizedFile.GetTitle() != "文件标题" || localizedFile.GetDescription() != "文件描述" {
		t.Fatalf("resource pack file translation was not applied: %#v", localizedFile)
	}
}

func TestLocalizedGLMSContentFallsBackToBaseLanguage(t *testing.T) {
	handler := &Handler{Lms: localizationLMSClient{}}
	base := &lmspb.Course{
		CourseUlid:  "course-1",
		Title:       "Course title",
		Description: "Course description",
	}

	localized := handler.localizedCourse(context.Background(), base, "en-US")
	if localized != base {
		t.Fatal("missing translation should return the original course")
	}
}

func TestLocalizedPipelineOverlaysConfiguredTranslations(t *testing.T) {
	handler := &Handler{
		Gcc:   localizationCCClient{},
		Creds: localizationCredentialClient{},
	}
	base := &gccpb.PipelineConfig{
		PipelineUlid: "pipeline-1",
		Name:         "Pipeline",
		CategoryTips: "Category",
		Description:  "Description",
		Stages: []*gccpb.StageConfig{{
			StageUlid: "stage-1",
			Name:      "Stage",
			Units: []*gccpb.UnitConfig{{
				UnitUlid: "unit-1",
				Name:     "Unit",
			}},
		}},
		FinalAuditQuals: []*gccpb.QualificationRequirement{{
			QualUlid: "credential-1",
			NameHint: "Work Experience Proof",
		}},
	}

	localized := handler.localizedPipeline(context.Background(), base, "zh-CN")
	if localized.GetName() != "认证名称" ||
		localized.GetCategoryTips() != "分类提示" ||
		localized.GetDescription() != "认证描述" {
		t.Fatalf("pipeline translation was not applied: %#v", localized)
	}
	if localized.GetStages()[0].GetName() != "第一阶段" {
		t.Fatalf("stage translation was not applied: %#v", localized.GetStages()[0])
	}
	if localized.GetStages()[0].GetUnits()[0].GetName() != "课程单元" {
		t.Fatalf("unit translation was not applied: %#v", localized.GetStages()[0].GetUnits()[0])
	}
	if localized.GetFinalAuditQuals()[0].GetNameHint() != "工作经验证明" {
		t.Fatalf("qualification translation was not applied: %#v", localized.GetFinalAuditQuals()[0])
	}
	if base.GetName() != "Pipeline" || base.GetStages()[0].GetName() != "Stage" {
		t.Fatal("localization mutated the source pipeline")
	}
}

func TestLocalizedCredentialDefinitionTranslatesFileConstraint(t *testing.T) {
	handler := &Handler{Creds: localizationCredentialClient{}}
	base := &gcredspb.CredentialDefinition{
		CredDefUlid:       "credential-1",
		Name:              "Work Experience Proof",
		Description:       "Upload evidence",
		AcquisitionMethod: "Submit for review",
		FileConstraints: []*gcredspb.CredentialFileConstraint{{
			Name: "Employment Certificate",
		}},
	}

	localized := handler.localizedCredentialDefinition(context.Background(), base, "zh-CN")
	if localized.GetName() != "工作经验证明" ||
		localized.GetDescription() != "提交工作经验证明材料" ||
		localized.GetAcquisitionMethod() != "提交后等待审核" {
		t.Fatalf("credential translation was not applied: %#v", localized)
	}
	if localized.GetFileConstraints()[0].GetName() != "雇佣证明" {
		t.Fatalf("file constraint translation was not applied: %#v", localized.GetFileConstraints()[0])
	}
	if base.GetFileConstraints()[0].GetName() != "Employment Certificate" {
		t.Fatal("localization mutated the source credential definition")
	}
}

func TestBundleTranslation(t *testing.T) {
	handler := &Handler{Mall: localizationMallClient{}}

	translation := handler.bundleTranslation(context.Background(), "bundle-1", "zh-CN")
	if translation == nil ||
		translation.GetName() != "认证商品" ||
		translation.GetDescription() != "认证商品描述" {
		t.Fatalf("bundle translation was not loaded: %#v", translation)
	}
}

func TestLocalizedMembershipTranslatesFeatureJSON(t *testing.T) {
	handler := &Handler{Gmbr: localizationMembershipClient{}}
	base := &gmbrpb.Membership{
		MembershipUlid: "membership-1",
		Name:           "Professional Membership",
		Description:    "Membership description",
		IdealFor:       "Fintech professionals",
		FeaturesJson:   `{"groups":[{"category":"Events & Seminars","items":[{"label":"Live Webinars"}]}]}`,
	}

	localized := handler.localizedMembership(context.Background(), base, "zh-CN")
	if localized.GetName() != "专业会员" ||
		localized.GetDescription() != "会员方案描述" ||
		localized.GetIdealFor() != "适合金融科技从业者" {
		t.Fatalf("membership translation was not applied: %#v", localized)
	}

	var features map[string]any
	if err := json.Unmarshal([]byte(localized.GetFeaturesJson()), &features); err != nil {
		t.Fatalf("localized features JSON is invalid: %v", err)
	}
	group := features["groups"].([]any)[0].(map[string]any)
	if group["category"] != "活动与研讨会" {
		t.Fatalf("feature category translation was not applied: %#v", group)
	}
	item := group["items"].([]any)[0].(map[string]any)
	if item["label"] != "在线研讨会" {
		t.Fatalf("feature label translation was not applied: %#v", item)
	}
	if base.GetName() != "Professional Membership" {
		t.Fatal("localization mutated the source membership")
	}
}
