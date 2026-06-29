package handler

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	profileShortTextMax  = 64
	profileAddressMax    = 160
	profileBioMax        = 500
	profilePostalCodeMax = 16
)

var postalCodePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 -]*[A-Za-z0-9]$`)

func normalizeProfileGender(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "":
		return ""
	case "male", "m", "man", "boy", "1", "男", "男性":
		return "Male"
	case "female", "f", "woman", "girl", "2", "女", "女性":
		return "Female"
	default:
		return ""
	}
}

func normalizeProfilePhone(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	if strings.HasPrefix(value, "+") {
		b.WriteByte('+')
	}
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func digitCount(value string) int {
	count := 0
	for _, r := range value {
		if r >= '0' && r <= '9' {
			count++
		}
	}
	return count
}

func validProfilePhone(value string, required bool) bool {
	phone := normalizeProfilePhone(value)
	if phone == "" {
		return !required
	}
	digits := digitCount(phone)
	return digits >= 7 && digits <= 15
}

func normalizeProfilePostalCode(value string) string {
	return strings.ToUpper(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func validProfilePostalCode(value string, required bool) bool {
	postalCode := normalizeProfilePostalCode(value)
	if postalCode == "" {
		return !required
	}
	return utf8.RuneCountInString(postalCode) >= 2 &&
		utf8.RuneCountInString(postalCode) <= profilePostalCodeMax &&
		postalCodePattern.MatchString(postalCode)
}

func trimProfileText(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	out := make([]rune, 0, maxRunes)
	for _, r := range value {
		out = append(out, r)
		if len(out) >= maxRunes {
			break
		}
	}
	return string(out)
}

func validateProfileTextLength(field string, value string, maxRunes int) error {
	if utf8.RuneCountInString(strings.TrimSpace(value)) > maxRunes {
		return fmt.Errorf("%s must be at most %d characters", field, maxRunes)
	}
	return nil
}

func normalizeAndValidateUserProfileInput(input *UserProfileInput) error {
	if input == nil {
		return nil
	}
	lengthChecks := []struct {
		field string
		value string
		max   int
	}{
		{"display_name", input.DisplayName, profileShortTextMax},
		{"first_name", input.FirstName, profileShortTextMax},
		{"last_name", input.LastName, profileShortTextMax},
		{"country", input.Country, profileShortTextMax},
		{"province", input.Province, profileShortTextMax},
		{"city", input.City, profileShortTextMax},
		{"address", input.Address, profileAddressMax},
		{"affiliation", input.Affiliation, profileShortTextMax},
		{"title", input.Title, profileShortTextMax},
		{"real_name", input.RealName, profileShortTextMax},
		{"bio", input.Bio, profileBioMax},
		{"education", input.Education, profileShortTextMax},
	}
	for _, check := range lengthChecks {
		if err := validateProfileTextLength(check.field, check.value, check.max); err != nil {
			return err
		}
	}
	input.DisplayName = trimProfileText(input.DisplayName, profileShortTextMax)
	input.FirstName = trimProfileText(input.FirstName, profileShortTextMax)
	input.LastName = trimProfileText(input.LastName, profileShortTextMax)
	input.Phone = normalizeProfilePhone(input.Phone)
	input.HomePhone = normalizeProfilePhone(input.HomePhone)
	input.WorkPhone = normalizeProfilePhone(input.WorkPhone)
	input.Country = trimProfileText(input.Country, profileShortTextMax)
	input.Province = trimProfileText(input.Province, profileShortTextMax)
	input.City = trimProfileText(input.City, profileShortTextMax)
	input.Address = trimProfileText(input.Address, profileAddressMax)
	input.PostalCode = normalizeProfilePostalCode(input.PostalCode)
	input.Affiliation = trimProfileText(input.Affiliation, profileShortTextMax)
	input.Title = trimProfileText(input.Title, profileShortTextMax)
	input.RealName = trimProfileText(input.RealName, profileShortTextMax)
	input.Bio = trimProfileText(input.Bio, profileBioMax)
	rawGender := strings.TrimSpace(input.Gender)
	input.Gender = normalizeProfileGender(input.Gender)
	input.Education = trimProfileText(input.Education, profileShortTextMax)

	if rawGender != "" && input.Gender == "" {
		return fmt.Errorf("gender is invalid")
	}
	if !validProfilePhone(input.HomePhone, false) {
		return fmt.Errorf("home_phone is invalid")
	}
	if !validProfilePhone(input.WorkPhone, false) {
		return fmt.Errorf("work_phone is invalid")
	}
	if !validProfilePostalCode(input.PostalCode, false) {
		return fmt.Errorf("postal_code is invalid")
	}
	return nil
}

func normalizeAndValidateSignupExamInput(input *SignupExamInput) error {
	if input == nil {
		return nil
	}
	lengthChecks := []struct {
		field string
		value string
		max   int
	}{
		{"first_name", input.FirstName, profileShortTextMax},
		{"middle_name", input.MiddleName, profileShortTextMax},
		{"last_name", input.LastName, profileShortTextMax},
		{"email", input.Email, profileShortTextMax},
		{"country", input.Country, profileShortTextMax},
		{"province", input.Province, profileShortTextMax},
		{"city", input.City, profileShortTextMax},
		{"address", input.Address, profileAddressMax},
	}
	for _, check := range lengthChecks {
		if err := validateProfileTextLength(check.field, check.value, check.max); err != nil {
			return err
		}
	}
	input.FirstName = trimProfileText(input.FirstName, profileShortTextMax)
	input.MiddleName = trimProfileText(input.MiddleName, profileShortTextMax)
	input.LastName = trimProfileText(input.LastName, profileShortTextMax)
	input.Email = trimProfileText(input.Email, profileShortTextMax)
	input.HomePhone = normalizeProfilePhone(input.HomePhone)
	input.WorkPhone = normalizeProfilePhone(input.WorkPhone)
	rawGender := strings.TrimSpace(input.Gender)
	input.Gender = normalizeProfileGender(input.Gender)
	input.Country = trimProfileText(input.Country, profileShortTextMax)
	input.Province = trimProfileText(input.Province, profileShortTextMax)
	input.City = trimProfileText(input.City, profileShortTextMax)
	input.Address = trimProfileText(input.Address, profileAddressMax)
	input.PostalCode = normalizeProfilePostalCode(input.PostalCode)

	if !validProfilePhone(input.HomePhone, true) {
		return fmt.Errorf("home_phone is invalid")
	}
	if !validProfilePhone(input.WorkPhone, false) {
		return fmt.Errorf("work_phone is invalid")
	}
	if rawGender == "" || input.Gender == "" {
		return fmt.Errorf("gender is invalid")
	}
	if !validProfilePostalCode(input.PostalCode, true) {
		return fmt.Errorf("postal_code is invalid")
	}
	return nil
}
