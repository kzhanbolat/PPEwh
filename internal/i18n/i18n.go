package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var translations = map[string]map[string]string{}

func Load(dir string) error {
	langs := []string{"en", "ru"}
	for _, lang := range langs {
		path := filepath.Join(dir, lang+".json")
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		m := map[string]string{}
		if err := json.Unmarshal(b, &m); err != nil {
			return err
		}
		translations[lang] = m
	}
	return nil
}

func NormalizeLang(lang string) string {
	switch lang {
	case "ru":
		return "ru"
	default:
		return "en"
	}
}

func T(lang, key string) string {
	lang = NormalizeLang(lang)
	if tr, ok := translations[lang]; ok {
		if val, ok := tr[key]; ok {
			return val
		}
	}
	return key
}

func Tf(lang, key string, args ...any) string {
	return fmt.Sprintf(T(lang, key), args...)
}

