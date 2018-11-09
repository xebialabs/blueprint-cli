package xl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsStringInSlice(t *testing.T) {
	t.Run("should check if given string is in slice", func(t *testing.T) {
		assert.Equal(t, true, isStringInSlice("there", []string{"there", "it", "is"}))
		assert.Equal(t, false, isStringInSlice("not there", []string{"there", "it", "is"}))
		assert.Equal(t, false, isStringInSlice("not there", []string{}))
		assert.Equal(t, false, isStringInSlice("not there", nil))
	})
}

func TestIsStringEmpty(t *testing.T) {
	t.Run("should check if given string is empty", func(t *testing.T) {
		assert.Equal(t, true, isStringEmpty(""))
		assert.Equal(t, true, isStringEmpty(" "))
		assert.Equal(t, true, isStringEmpty("          "))
		assert.Equal(t, false, isStringEmpty("not empty string"))
		assert.Equal(t, false, isStringEmpty("!@#$%^&*()_+"))
	})
}

func TestReplaceTemplatePlaceholders(t *testing.T) {
	t.Run("should replace custom placeholders with XLD format", func(t *testing.T) {
		assert.Equal(t, "", replaceTemplatePlaceholders(""))
		assert.Equal(t, "${test", replaceTemplatePlaceholders("${test"))
		assert.Equal(t, "${{test}", replaceTemplatePlaceholders("${{test}"))
		assert.Equal(t, "#{{test}}", replaceTemplatePlaceholders("#{{test}}"))
		assert.Equal(t, "{{test}}", replaceTemplatePlaceholders("#{test}"))
		assert.Equal(t, "{{ test }}", replaceTemplatePlaceholders("#{ test }"))
		assert.Equal(t, "$\\{{test\\}}", replaceTemplatePlaceholders("$\\{{test\\}}"))
		assert.Equal(
			t,
			"SPRING_DATASOURCE_URL: jdbc:mysql://{{MYSQL_DB_ADDRESS}}:{{MYSQL_DB_PORT}}/store?useUnicode=true&characterEncoding=utf8&useSSL=false",
			replaceTemplatePlaceholders("SPRING_DATASOURCE_URL: jdbc:mysql://#{MYSQL_DB_ADDRESS}:#{MYSQL_DB_PORT}/store?useUnicode=true&characterEncoding=utf8&useSSL=false"),
		)
		assert.Equal(t, `
templates:
- name: {{.GetName}}-ecs-dictionary-${xlrPlaceholder}
	type: template.udm.Dictionary-${ xlrPlaceholder }
	entries:
	MYSQL_DB_PORT: '{{%finalPort%}}'
	MYSQL_DB_PORT: '{{finalPort}}'
	MYSQL_DB_PORT: '{{finalPort123}}'
	MYSQL_DB_PORT: '{{finalPort.123_123-123}}'
	MYSQL_DB_PORT: '{{finalPort.123_123-@@123##}}'
	MYSQL_DB_PORT: '{{ finalPort.123_ 123-$@@123## }}'
	SPRING_DATASOURCE_URL: jdbc:mysql://{{MYSQL_DB_ADDRESS}}:{{MYSQL_DB_PORT}}/store?useUnicode=true&characterEncoding=utf8&useSSL=false
- name: {{.GetName}}-ecs-alb-dictionary
	type: template.udm.Dictionary
	entries:
	ALB_DNS_NAME: '{{%dnsName%}}'
		`, replaceTemplatePlaceholders(`
templates:
- name: {{.GetName}}-ecs-dictionary-${xlrPlaceholder}
	type: template.udm.Dictionary-${ xlrPlaceholder }
	entries:
	MYSQL_DB_PORT: '#{%finalPort%}'
	MYSQL_DB_PORT: '#{finalPort}'
	MYSQL_DB_PORT: '#{finalPort123}'
	MYSQL_DB_PORT: '#{finalPort.123_123-123}'
	MYSQL_DB_PORT: '#{finalPort.123_123-@@123##}'
	MYSQL_DB_PORT: '#{ finalPort.123_ 123-$@@123## }'
	SPRING_DATASOURCE_URL: jdbc:mysql://#{MYSQL_DB_ADDRESS}:#{MYSQL_DB_PORT}/store?useUnicode=true&characterEncoding=utf8&useSSL=false
- name: {{.GetName}}-ecs-alb-dictionary
	type: template.udm.Dictionary
	entries:
	ALB_DNS_NAME: '#{%dnsName%}'
		`))
	})
}

func TestAddSuffixIfNeeded(t *testing.T) {
	t.Run("should add given suffix to value", func(t *testing.T) {
		assert.Equal(t, "http://test.com/", addSuffixIfNeeded("http://test.com", "/"))
		assert.Equal(t, "myfile.yaml.tmpl", addSuffixIfNeeded("myfile.yaml", ".tmpl"))
	})
	t.Run("should not add suffix to value", func(t *testing.T) {
		assert.Equal(t, "http://test.com/", addSuffixIfNeeded("http://test.com/", "/"))
		assert.Equal(t, "myfile.yaml.tmpl", addSuffixIfNeeded("myfile.yaml.tmpl", ".tmpl"))
	})
}

func TestToKebabCase(t *testing.T) {
	t.Run("should convert strings to kebab case", func(t *testing.T) {
		assert.Equal(t, "", toKebabCase(""))
		assert.Equal(t, "test-string-in-camel", toKebabCase("test-string-in-camel"))
		assert.Equal(t, "test-string-in-camel", toKebabCase("testStringInCamel"))
		assert.Equal(t, "test-string-with-space", toKebabCase("test String with space"))
		assert.Equal(t, "test-string-with-space", toKebabCase("test_string_with_space"))
		assert.Equal(t, "test-my-project-123", toKebabCase("test my project 123"))
	})
}
