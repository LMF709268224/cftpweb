package handler

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	profileNameTextMax   = 20
	profileShortTextMax  = 64
	profileAddressMax    = 160
	profileBioMax        = 500
	profilePostalCodeMax = 16
	profileLinkMax       = 100
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
		{"first_name", input.FirstName, profileNameTextMax},
		{"last_name", input.LastName, profileNameTextMax},
		{"country", input.Country, profileShortTextMax},
		{"province", input.Province, profileShortTextMax},
		{"city", input.City, profileShortTextMax},
		{"address", input.Address, profileAddressMax},
		{"affiliation", input.Affiliation, profileShortTextMax},
		{"title", input.Title, profileShortTextMax},
		{"bio", input.Bio, profileBioMax},
		{"education", input.Education, profileShortTextMax},
	}
	for _, check := range lengthChecks {
		if err := validateProfileTextLength(check.field, check.value, check.max); err != nil {
			return err
		}
	}
	input.FirstName = trimProfileText(input.FirstName, profileNameTextMax)
	input.LastName = trimProfileText(input.LastName, profileNameTextMax)
	input.Phone = normalizeProfilePhone(input.Phone)
	input.HomePhone = normalizeProfilePhone(input.HomePhone)
	input.Country = trimProfileText(input.Country, profileShortTextMax)
	input.Province = trimProfileText(input.Province, profileShortTextMax)
	input.City = trimProfileText(input.City, profileShortTextMax)
	input.Address = trimProfileText(input.Address, profileAddressMax)
	input.PostalCode = normalizeProfilePostalCode(input.PostalCode)
	input.Affiliation = trimProfileText(input.Affiliation, profileShortTextMax)
	input.Title = trimProfileText(input.Title, profileShortTextMax)
	input.Bio = trimProfileText(input.Bio, profileBioMax)
	rawGender := strings.TrimSpace(input.Gender)
	input.Gender = normalizeProfileGender(input.Gender)
	input.Education = trimProfileText(input.Education, profileShortTextMax)
	for _, link := range []struct {
		field string
		value **string
		hosts []string
	}{
		{"homepage", &input.Homepage, nil},
		{"linkedin", &input.LinkedIn, []string{"linkedin.com"}},
		{"twitter", &input.Twitter, []string{"x.com", "twitter.com"}},
	} {
		if *link.value == nil {
			continue
		}
		value := strings.TrimSpace(**link.value)
		if err := validateProfileLink(link.field, value, link.hosts); err != nil {
			return err
		}
		*link.value = &value
	}

	if rawGender != "" && input.Gender == "" {
		return fmt.Errorf("gender is invalid")
	}
	if !validProfilePhone(input.HomePhone, false) {
		return fmt.Errorf("home_phone is invalid")
	}
	if !validProfilePhone(input.Phone, false) {
		return fmt.Errorf("phone is invalid")
	}
	if !validProfilePostalCode(input.PostalCode, false) {
		return fmt.Errorf("postal_code is invalid")
	}
	if input.Birthday != "" {
		if err := validateAdultBirthdate(input.Birthday, "birthday", time.Now().UTC()); err != nil {
			return err
		}
	}
	return nil
}

func validateAdultBirthdate(value, field string, today time.Time) error {
	if len(value) != len("2006-01-02") {
		return fmt.Errorf("%s must be a valid YYYY-MM-DD date", field)
	}
	birthdate, err := time.Parse("2006-01-02", value)
	if err != nil || birthdate.Format("2006-01-02") != value {
		return fmt.Errorf("%s must be a valid YYYY-MM-DD date", field)
	}
	day := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	if birthdate.After(day) {
		return fmt.Errorf("%s cannot be in the future", field)
	}
	if birthdate.After(day.AddDate(-18, 0, 0)) {
		return fmt.Errorf("%s requires age 18 or older", field)
	}
	return nil
}

func validateProfileLink(field, value string, allowedHosts []string) error {
	if value == "" {
		return nil
	}
	if err := validateProfileTextLength(field, value, profileLinkMax); err != nil {
		return err
	}
	parsed, err := url.Parse(value)
	if err != nil || (!strings.EqualFold(parsed.Scheme, "https") && !strings.EqualFold(parsed.Scheme, "http")) || parsed.Hostname() == "" || parsed.User != nil {
		return fmt.Errorf("%s must be an HTTP or HTTPS URL", field)
	}
	if len(allowedHosts) > 0 {
		host := strings.ToLower(parsed.Hostname())
		for _, allowed := range allowedHosts {
			if host == allowed || strings.HasSuffix(host, "."+allowed) {
				return nil
			}
		}
		return fmt.Errorf("%s has an invalid domain", field)
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
		{"first_name", input.FirstName, profileNameTextMax},
		{"last_name", input.LastName, profileNameTextMax},
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
	input.FirstName = trimProfileText(input.FirstName, profileNameTextMax)
	input.LastName = trimProfileText(input.LastName, profileNameTextMax)
	input.Email = trimProfileText(input.Email, profileShortTextMax)
	input.HomePhone = normalizeProfilePhone(input.HomePhone)
	input.Phone = normalizeProfilePhone(input.Phone)
	rawGender := strings.TrimSpace(input.Gender)
	input.Gender = normalizeProfileGender(input.Gender)
	input.Country = trimProfileText(input.Country, profileShortTextMax)
	input.Province = trimProfileText(input.Province, profileShortTextMax)
	input.City = trimProfileText(input.City, profileShortTextMax)
	input.Address = trimProfileText(input.Address, profileAddressMax)
	input.PostalCode = normalizeProfilePostalCode(input.PostalCode)

	if !validProfilePhone(input.HomePhone, false) {
		return fmt.Errorf("home_phone is invalid")
	}
	if !validProfilePhone(input.Phone, false) {
		return fmt.Errorf("phone is invalid")
	}
	if rawGender == "" || input.Gender == "" {
		return fmt.Errorf("gender is invalid")
	}
	if !validProfilePostalCode(input.PostalCode, true) {
		return fmt.Errorf("postal_code is invalid")
	}
	if err := validateAdultBirthdate(input.Birthdate, "birthdate", time.Now().UTC()); err != nil {
		return err
	}
	return nil
}
