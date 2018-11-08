package xl

import (
	"regexp"
	"strings"

	"github.com/huandu/xstrings"
)

// check if string is in slice given
func isStringInSlice(s string, list []string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// check if string is empty
func isStringEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

// replace nested placeholders with XL placeholders
func replaceTemplatePlaceholders(processedTmpl string) string {
	// seems like XLD allows pretty much anything in a placeholder
	re := regexp.MustCompile(`\#\{([^{}]+|\s*)\}`)
	return re.ReplaceAllString(processedTmpl, `{{$1}}`)
}

func addSuffixIfNeeded(val, suffix string) string {
	if !strings.HasSuffix(val, suffix) {
		return val + suffix
	}
	return val
}

func toKebabCase(str string) string {
	return strings.Replace(xstrings.ToSnakeCase(str), "_", "-", -1)
}
