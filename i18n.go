package i18n

import (
	"errors"
	"strings"

	"github.com/qor/cache"
	"github.com/qor/cache/memory"
	"github.com/theplant/cldr"
)

// Default default locale for i18n
var Default = "en-US"

// I18n struct that hold all translations
type I18n struct {
	scope           string
	value           string
	Backends        []Backend
	FallbackLocales map[string][]string
	fallbackLocales []string
	cacheStore      cache.CacheStoreInterface
}

// Backend defined methods that needs for translation backend
type Backend interface {
	LoadTranslations() []*Translation
	SaveTranslation(*Translation) error
	DeleteTranslation(*Translation) error
}

// Translation is a struct for translations, including Translation Key, Locale, Value
type Translation struct {
	Key     string
	Locale  string
	Value   string
	Backend Backend `json:"-"`
}

// New initialize I18n with backends
func New(backends ...Backend) *I18n {
	i18n := I18n{Backends: backends, cacheStore: memory.New()}
	i18n.loadToCacheStore()
	return &i18n
}

// SetCacheStore set i18n's cache store
func (i18n *I18n) SetCacheStore(cacheStore cache.CacheStoreInterface) {
	i18n.cacheStore = cacheStore
	i18n.loadToCacheStore()
}

func (i18n *I18n) loadToCacheStore() {
	backends := i18n.Backends
	for i := len(backends) - 1; i >= 0; i-- {
		var backend = backends[i]
		for _, translation := range backend.LoadTranslations() {
			i18n.AddTranslation(translation)
		}
	}
}

// LoadTranslations load translations as map `map[locale]map[key]*Translation`
func (i18n *I18n) LoadTranslations() map[string]map[string]*Translation {
	var translations = map[string]map[string]*Translation{}

	for i := len(i18n.Backends); i > 0; i-- {
		backend := i18n.Backends[i-1]
		for _, translation := range backend.LoadTranslations() {
			if translations[translation.Locale] == nil {
				translations[translation.Locale] = map[string]*Translation{}
			}
			translations[translation.Locale][translation.Key] = translation
		}
	}
	return translations
}

// AddTranslation add translation
func (i18n *I18n) AddTranslation(translation *Translation) error {
	return i18n.cacheStore.Set(cacheKey(translation.Locale, translation.Key), translation)
}

// SaveTranslation save translation
func (i18n *I18n) SaveTranslation(translation *Translation) error {
	for _, backend := range i18n.Backends {
		if backend.SaveTranslation(translation) == nil {
			i18n.AddTranslation(translation)
			return nil
		}
	}

	return errors.New("failed to save translation")
}

// DeleteTranslation delete translation
func (i18n *I18n) DeleteTranslation(translation *Translation) (err error) {
	for _, backend := range i18n.Backends {
		backend.DeleteTranslation(translation)
	}

	return i18n.cacheStore.Delete(cacheKey(translation.Locale, translation.Key))
}

// Scope i18n scope
func (i18n *I18n) Scope(scope string) *I18n {
	i18nIntermediate := I18n{cacheStore: i18n.cacheStore, scope: scope, value: i18n.value, Backends: i18n.Backends, FallbackLocales: i18n.FallbackLocales, fallbackLocales: i18n.fallbackLocales}
	return &i18nIntermediate
}

// Default default value of translation if key is missing
func (i18n *I18n) Default(value string) *I18n {
	i18nIntermediate := I18n{cacheStore: i18n.cacheStore, scope: i18n.scope, value: value, Backends: i18n.Backends, FallbackLocales: i18n.FallbackLocales, fallbackLocales: i18n.fallbackLocales}
	return &i18nIntermediate
}

// Fallbacks fallback to locale if translation doesn't exist in specified locale
func (i18n *I18n) Fallbacks(locale ...string) *I18n {
	i18nIntermediate := I18n{cacheStore: i18n.cacheStore, scope: i18n.scope, value: i18n.value, Backends: i18n.Backends, FallbackLocales: i18n.FallbackLocales, fallbackLocales: locale}
	return &i18nIntermediate
}

// T translate with locale, key and arguments
func (i18n *I18n) T(locale, key string, args ...interface{}) string {
	var (
		value           = i18n.value
		translationKey  = key
		fallbackLocales = i18n.fallbackLocales
	)

	if locale == "" {
		locale = Default
	}

	if locales, ok := i18n.FallbackLocales[locale]; ok {
		fallbackLocales = append(fallbackLocales, locales...)
	}
	fallbackLocales = append(fallbackLocales, Default)

	if i18n.scope != "" {
		translationKey = strings.Join([]string{i18n.scope, key}, ".")
	}

	var translation Translation
	if err := i18n.cacheStore.Unmarshal(cacheKey(locale, key), &translation); err != nil || translation.Value == "" {
		for _, fallbackLocale := range fallbackLocales {
			if err := i18n.cacheStore.Unmarshal(cacheKey(fallbackLocale, key), &translation); err == nil && translation.Value != "" {
				break
			}
		}

		if translation.Value == "" {
			// Get default translation if not translated
			if err := i18n.cacheStore.Unmarshal(cacheKey(Default, key), &translation); err != nil || translation.Value == "" {
				// If not initialized
				var defaultBackend Backend
				if len(i18n.Backends) > 0 {
					defaultBackend = i18n.Backends[0]
				}
				translation = Translation{Key: translationKey, Value: value, Locale: locale, Backend: defaultBackend}

				// Save translation
				i18n.SaveTranslation(&translation)
			}
		}
	}

	if translation.Value != "" {
		value = translation.Value
	} else {
		value = key
	}

	if str, err := cldr.Parse(locale, value, args...); err == nil {
		value = str
	}

	return value
}

// func getLocaleFromContext(context *qor.Context) string {
// 	if locale := utils.GetLocale(context); locale != "" {
// 		return locale
// 	}

// 	return Default
// }

type availableLocalesInterface interface {
	AvailableLocales() []string
}

type viewableLocalesInterface interface {
	ViewableLocales() []string
}

type editableLocalesInterface interface {
	EditableLocales() []string
}

// func getAvailableLocales(req *http.Request, currentUser qor.CurrentUser) []string {
// 	if user, ok := currentUser.(viewableLocalesInterface); ok {
// 		return user.ViewableLocales()
// 	}

// 	if user, ok := currentUser.(availableLocalesInterface); ok {
// 		return user.AvailableLocales()
// 	}
// 	return []string{Default}
// }

// func getEditableLocales(req *http.Request, currentUser qor.CurrentUser) []string {
// 	if user, ok := currentUser.(editableLocalesInterface); ok {
// 		return user.EditableLocales()
// 	}

// 	if user, ok := currentUser.(availableLocalesInterface); ok {
// 		return user.AvailableLocales()
// 	}
// 	return []string{Default}
// }

func cacheKey(strs ...string) string {
	return strings.Join(strs, "/")
}
