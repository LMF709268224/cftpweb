package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
	"google.golang.org/protobuf/proto"
)

func requestLocale(r *http.Request) string {
	if r == nil {
		return ""
	}
	raw := strings.TrimSpace(strings.Split(r.Header.Get("Accept-Language"), ",")[0])
	if separator := strings.Index(raw, ";"); separator >= 0 {
		raw = raw[:separator]
	}
	return normalizedLocale(raw)
}

func normalizedLocale(raw string) string {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "_", "-"))
	if raw == "" {
		return ""
	}

	parts := strings.Split(raw, "-")
	language := strings.ToLower(strings.TrimSpace(parts[0]))
	switch language {
	case "zh":
		if len(parts) == 1 {
			return "zh-CN"
		}
	case "en":
		if len(parts) == 1 {
			return "en-US"
		}
	}
	if len(parts) == 1 {
		return language
	}

	normalized := []string{language}
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if len(part) == 2 {
			part = strings.ToUpper(part)
		}
		normalized = append(normalized, part)
	}
	return strings.Join(normalized, "-")
}

func translatedText(base string, translated string) string {
	if value := strings.TrimSpace(translated); value != "" {
		return value
	}
	return base
}

func translationForLocale[T any](translations map[string]T, locale string) (T, bool) {
	if value, ok := translations[locale]; ok {
		return value, true
	}
	for key, value := range translations {
		if strings.EqualFold(key, locale) {
			return value, true
		}
	}
	var zero T
	return zero, false
}

func (h *Handler) courseTranslation(ctx context.Context, courseID string, locale string) *lmspb.CourseTranslation {
	locale = normalizedLocale(locale)
	courseID = strings.TrimSpace(courseID)
	if courseID == "" || locale == "" || h.Lms == nil {
		return nil
	}
	resp, err := h.Lms.GetCourseTranslations(ctx, &lmspb.GetCourseTranslationsRequest{
		CourseId: courseID,
		Locale:   locale,
	})
	if err != nil {
		slog.Warn("Failed to load course translation", "error", err, "course_id", courseID, "locale", locale)
		return nil
	}
	translation, _ := translationForLocale(resp.GetTranslations(), locale)
	return translation
}

func (h *Handler) localizedCourse(ctx context.Context, course *lmspb.Course, locale string) *lmspb.Course {
	if course == nil {
		return nil
	}
	translation := h.courseTranslation(ctx, course.GetCourseUlid(), locale)
	if translation == nil {
		return course
	}
	localized := proto.Clone(course).(*lmspb.Course)
	localized.Title = translatedText(localized.GetTitle(), translation.GetTitle())
	localized.Description = translatedText(localized.GetDescription(), translation.GetDescription())
	return localized
}

func (h *Handler) localizedCourseSummary(ctx context.Context, course *lmspb.CourseSummary, locale string) *lmspb.CourseSummary {
	if course == nil {
		return nil
	}
	translation := h.courseTranslation(ctx, course.GetCourseUlid(), locale)
	if translation == nil {
		return course
	}
	localized := proto.Clone(course).(*lmspb.CourseSummary)
	localized.Title = translatedText(localized.GetTitle(), translation.GetTitle())
	return localized
}

func (h *Handler) localizedResourcePack(ctx context.Context, pack *lmspb.ResourcePack, locale string) *lmspb.ResourcePack {
	locale = normalizedLocale(locale)
	if pack == nil || locale == "" || h.Lms == nil {
		return pack
	}
	resp, err := h.Lms.GetResourcePackTranslations(ctx, &lmspb.GetResourcePackTranslationsRequest{
		PackId: pack.GetPackId(),
		Locale: locale,
	})
	if err != nil {
		slog.Warn("Failed to load resource pack translation", "error", err, "pack_id", pack.GetPackId(), "locale", locale)
		return pack
	}
	translation, ok := translationForLocale(resp.GetTranslations(), locale)
	if !ok || translation == nil {
		return pack
	}
	localized := proto.Clone(pack).(*lmspb.ResourcePack)
	localized.Title = translatedText(localized.GetTitle(), translation.GetTitle())
	localized.Description = translatedText(localized.GetDescription(), translation.GetDescription())
	return localized
}

func (h *Handler) localizedResourcePackFile(ctx context.Context, file *lmspb.ResourcePackFile, locale string) *lmspb.ResourcePackFile {
	locale = normalizedLocale(locale)
	if file == nil || locale == "" || h.Lms == nil {
		return file
	}
	resp, err := h.Lms.GetResourcePackFileTranslations(ctx, &lmspb.GetResourcePackFileTranslationsRequest{
		FileId: file.GetFileId(),
		Locale: locale,
	})
	if err != nil {
		slog.Warn("Failed to load resource pack file translation", "error", err, "file_id", file.GetFileId(), "locale", locale)
		return file
	}
	translation, ok := translationForLocale(resp.GetTranslations(), locale)
	if !ok || translation == nil {
		return file
	}
	localized := proto.Clone(file).(*lmspb.ResourcePackFile)
	localized.Title = translatedText(localized.GetTitle(), translation.GetTitle())
	localized.Description = translatedText(localized.GetDescription(), translation.GetDescription())
	return localized
}

func (h *Handler) localizedPipeline(ctx context.Context, pipeline *gccpb.PipelineConfig, locale string) *gccpb.PipelineConfig {
	locale = normalizedLocale(locale)
	if pipeline == nil || locale == "" || h.Gcc == nil {
		return pipeline
	}

	localized := proto.Clone(pipeline).(*gccpb.PipelineConfig)
	stageNames := make([]string, len(localized.GetStages()))
	unitNames := make([][]string, len(localized.GetStages()))
	qualificationNames := map[string]string{}
	qualificationsByID := map[string][]*gccpb.Qualification{}
	var pipelineTranslation *gccpb.PipelineTranslation
	var qualificationNamesMu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, err := h.Gcc.GetPipelineTranslations(ctx, &gccpb.GetPipelineTranslationsRequest{
			PipelineId: localized.GetPipelineUlid(),
			Locale:     locale,
		})
		if err != nil {
			slog.Warn("Failed to load pipeline translation", "error", err, "pipeline_id", localized.GetPipelineUlid(), "locale", locale)
			return
		}
		pipelineTranslation, _ = translationForLocale(resp.GetTranslations(), locale)
	}()

	for stageIndex, stage := range localized.GetStages() {
		if stage == nil {
			continue
		}
		unitNames[stageIndex] = make([]string, len(stage.GetUnits()))
		stageIndex := stageIndex
		stage := stage
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := h.Gcc.GetStageTranslations(ctx, &gccpb.GetStageTranslationsRequest{
				StageId: stage.GetStageUlid(),
				Locale:  locale,
			})
			if err != nil {
				slog.Warn("Failed to load pipeline stage translation", "error", err, "stage_id", stage.GetStageUlid(), "locale", locale)
				return
			}
			if translation, ok := translationForLocale(resp.GetTranslations(), locale); ok && translation != nil {
				stageNames[stageIndex] = translation.GetName()
			}
		}()

		for unitIndex, unit := range stage.GetUnits() {
			if unit == nil {
				continue
			}
			unitIndex := unitIndex
			unit := unit
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp, err := h.Gcc.GetUnitTranslations(ctx, &gccpb.GetUnitTranslationsRequest{
					UnitId: unit.GetUnitUlid(),
					Locale: locale,
				})
				if err != nil {
					slog.Warn("Failed to load pipeline unit translation", "error", err, "unit_id", unit.GetUnitUlid(), "locale", locale)
					return
				}
				if translation, ok := translationForLocale(resp.GetTranslations(), locale); ok && translation != nil {
					unitNames[stageIndex][unitIndex] = translation.GetName()
				}
			}()
		}
	}
	for _, qualifications := range [][]*gccpb.Qualification{
		localized.GetUnlockQuals(),
		localized.GetCertsQuals(),
		localized.GetCerts(),
	} {
		for _, qualification := range qualifications {
			if qualification == nil {
				continue
			}
			qualificationID := strings.TrimSpace(qualification.GetQualUlid())
			if qualificationID == "" {
				continue
			}
			qualificationsByID[qualificationID] = append(qualificationsByID[qualificationID], qualification)
		}
	}
	for qualificationID := range qualificationsByID {
		qualificationID := qualificationID
		wg.Add(1)
		go func() {
			defer wg.Done()
			translation := h.credentialDefinitionTranslation(ctx, qualificationID, locale)
			if translation == nil || strings.TrimSpace(translation.GetName()) == "" {
				return
			}
			qualificationNamesMu.Lock()
			qualificationNames[qualificationID] = translation.GetName()
			qualificationNamesMu.Unlock()
		}()
	}
	wg.Wait()

	if pipelineTranslation != nil {
		localized.Name = translatedText(localized.GetName(), pipelineTranslation.GetName())
		localized.CategoryTips = translatedText(localized.GetCategoryTips(), pipelineTranslation.GetCategoryTips())
		localized.Description = translatedText(localized.GetDescription(), pipelineTranslation.GetDescription())
	}
	for stageIndex, stage := range localized.GetStages() {
		if stage == nil {
			continue
		}
		stage.Name = translatedText(stage.GetName(), stageNames[stageIndex])
		for unitIndex, unit := range stage.GetUnits() {
			if unit == nil {
				continue
			}
			unit.Name = translatedText(unit.GetName(), unitNames[stageIndex][unitIndex])
		}
	}
	for qualificationID, qualifications := range qualificationsByID {
		for _, qualification := range qualifications {
			qualification.NameHint = translatedText(qualification.GetNameHint(), qualificationNames[qualificationID])
		}
	}
	return localized
}

func (h *Handler) bundleTranslation(ctx context.Context, bundleID string, locale string) *mallpb.BundleTranslation {
	locale = normalizedLocale(locale)
	bundleID = strings.TrimSpace(bundleID)
	if bundleID == "" || locale == "" || h.Mall == nil {
		return nil
	}
	resp, err := h.Mall.GetBundleTranslations(ctx, &mallpb.GetBundleTranslationsRequest{
		BundleId: bundleID,
		Locale:   locale,
	})
	if err != nil {
		slog.Warn("Failed to load bundle translation", "error", err, "bundle_id", bundleID, "locale", locale)
		return nil
	}
	translation, _ := translationForLocale(resp.GetTranslations(), locale)
	return translation
}

func (h *Handler) localizedMembership(ctx context.Context, membership *gmbrpb.Membership, locale string) *gmbrpb.Membership {
	locale = normalizedLocale(locale)
	if membership == nil || locale == "" || h.Gmbr == nil {
		return membership
	}
	resp, err := h.Gmbr.GetMembershipTranslations(ctx, &gmbrpb.GetMembershipTranslationsRequest{
		MembershipId: membership.GetMembershipUlid(),
		Locale:       locale,
	})
	if err != nil {
		slog.Warn("Failed to load membership translation", "error", err, "membership_id", membership.GetMembershipUlid(), "locale", locale)
		return membership
	}
	translation, ok := translationForLocale(resp.GetTranslations(), locale)
	if !ok || translation == nil {
		return membership
	}

	localized := proto.Clone(membership).(*gmbrpb.Membership)
	localized.Name = translatedText(localized.GetName(), translation.GetName())
	localized.Description = translatedText(localized.GetDescription(), translation.GetDescription())
	localized.IdealFor = translatedText(localized.GetIdealFor(), translation.GetIdealFor())
	localized.FeaturesJson = localizedMembershipFeaturesJSON(
		localized.GetFeaturesJson(),
		translation.GetFeatureCategories(),
		translation.GetFeatureLabels(),
	)
	return localized
}

func localizedMembershipFeaturesJSON(raw string, categories map[string]string, labels map[string]string) string {
	if strings.TrimSpace(raw) == "" || (len(categories) == 0 && len(labels) == 0) {
		return raw
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw
	}
	localizeMembershipFeatureValue(value, categories, labels)
	encoded, err := json.Marshal(value)
	if err != nil {
		return raw
	}
	return string(encoded)
}

func localizeMembershipFeatureValue(value any, categories map[string]string, labels map[string]string) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			localizeMembershipFeatureValue(item, categories, labels)
		}
	case map[string]any:
		if category, ok := typed["category"].(string); ok {
			typed["category"] = translatedText(category, categories[normalizeTranslationKey(category)])
		}
		if label, ok := typed["label"].(string); ok {
			typed["label"] = translatedText(label, labels[normalizeTranslationKey(label)])
		}
		for _, item := range typed {
			localizeMembershipFeatureValue(item, categories, labels)
		}
	}
}

func normalizeTranslationKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	previousUnderscore := false
	for _, current := range value {
		isAlphaNumeric := current >= 'a' && current <= 'z' || current >= '0' && current <= '9'
		if isAlphaNumeric {
			builder.WriteRune(current)
			previousUnderscore = false
			continue
		}
		if builder.Len() > 0 && !previousUnderscore {
			builder.WriteByte('_')
			previousUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

func (h *Handler) localizedCredentialDefinition(ctx context.Context, definition *gcredspb.CredentialDefinition, locale string) *gcredspb.CredentialDefinition {
	locale = normalizedLocale(locale)
	if definition == nil || locale == "" || h.Creds == nil {
		return definition
	}
	translation := h.credentialDefinitionTranslation(ctx, definition.GetCredDefUlid(), locale)
	if translation == nil {
		return definition
	}

	localized := proto.Clone(definition).(*gcredspb.CredentialDefinition)
	localized.Name = translatedText(localized.GetName(), translation.GetName())
	localized.Description = translatedText(localized.GetDescription(), translation.GetDescription())
	localized.AcquisitionMethod = translatedText(localized.GetAcquisitionMethod(), translation.GetAcquisitionMethod())
	for _, constraint := range localized.GetFileConstraints() {
		if constraint == nil {
			continue
		}
		constraint.Name = translatedText(
			constraint.GetName(),
			translation.GetFileConstraintNames()[normalizeTranslationKey(constraint.GetName())],
		)
	}
	return localized
}

func (h *Handler) credentialDefinitionTranslation(ctx context.Context, credentialDefinitionID string, locale string) *gcredspb.CredDefTranslation {
	locale = normalizedLocale(locale)
	credentialDefinitionID = strings.TrimSpace(credentialDefinitionID)
	if credentialDefinitionID == "" || locale == "" || h.Creds == nil {
		return nil
	}
	resp, err := h.Creds.GetCredDefTranslations(ctx, &gcredspb.GetCredDefTranslationsRequest{
		CredDefId: credentialDefinitionID,
		Locale:    locale,
	})
	if err != nil {
		slog.Warn("Failed to load credential definition translation", "error", err, "cred_def_id", credentialDefinitionID, "locale", locale)
		return nil
	}
	translation, ok := translationForLocale(resp.GetTranslations(), locale)
	if !ok || translation == nil {
		return nil
	}
	return translation
}
