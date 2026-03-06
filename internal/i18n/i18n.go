package i18n

import (
	"embed"
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed locales/*.yaml
var localeFS embed.FS

var (
	initOnce sync.Once
	initErr  error

	activeLang string
	locales    map[string]map[string]string
)

// MustInit initializes localization and panics on failure.
func MustInit() {
	if err := Init(); err != nil {
		panic(err)
	}
}

// Init initializes localization runtime once.
func Init() error {
	initOnce.Do(func() {
		locales = map[string]map[string]string{}
		for _, lang := range []string{"zh", "en"} {
			path := fmt.Sprintf("locales/%s.yaml", lang)
			data, err := localeFS.ReadFile(path)
			if err != nil {
				initErr = err
				return
			}
			bundle := map[string]string{}
			if err := yaml.Unmarshal(data, &bundle); err != nil {
				initErr = err
				return
			}
			locales[lang] = bundle
		}

		activeLang = detectLanguage()
		if _, ok := locales[activeLang]; !ok {
			activeLang = "zh"
		}
	})
	return initErr
}

// T translates key for current language.
func T(key string) string {
	_ = Init()
	if bundle, ok := locales[activeLang]; ok {
		if value, exists := bundle[key]; exists {
			return value
		}
	}
	if zh, ok := locales["zh"]; ok {
		if value, exists := zh[key]; exists {
			return value
		}
	}
	return key
}

// Tf translates key and formats the value.
func Tf(key string, args ...interface{}) string {
	return fmt.Sprintf(T(key), args...)
}

func detectLanguage() string {
	candidates := []string{
		os.Getenv("TUNMAN_LANG"),
		os.Getenv("SKILLS_LANG"),
		os.Getenv("LANG"),
		os.Getenv("LC_ALL"),
	}
	for _, raw := range candidates {
		if normalized := normalize(raw); normalized != "" {
			return normalized
		}
	}
	return "zh"
}

func normalize(value string) string {
	if value == "" {
		return ""
	}
	v := strings.ToLower(value)
	v = strings.ReplaceAll(v, "-", "_")
	if strings.HasPrefix(v, "zh") {
		return "zh"
	}
	if strings.HasPrefix(v, "en") {
		return "en"
	}
	return ""
}
