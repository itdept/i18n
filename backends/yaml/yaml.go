package yaml

import (
	"errors"
	"fmt"
	"github.com/gobuffalo/packr/v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/itdept/i18n"
	"gopkg.in/yaml.v2"
)

var _ i18n.Backend = &Backend{}

// New new YAML backend for I18n
func New(paths ...string) *Backend {
	backend := &Backend{}

	var files []string
	for _, p := range paths {
		if file, err := os.Open(p); err == nil {
			defer file.Close()
			if fileInfo, err := file.Stat(); err == nil {
				if fileInfo.IsDir() {
					yamlFiles, _ := filepath.Glob(filepath.Join(p, "*.yaml"))
					files = append(files, yamlFiles...)

					ymlFiles, _ := filepath.Glob(filepath.Join(p, "*.yml"))
					files = append(files, ymlFiles...)
				} else if fileInfo.Mode().IsRegular() {
					files = append(files, p)
				}
			}
		}
	}
	for _, file := range files {
		if content, err := ioutil.ReadFile(file); err == nil {
			backend.contents = append(backend.contents, content)
		}
	}
	return backend
}

// New new YAML backend for I18n that implements backer to package the files
func NewWithPacker(paths ...string) (*Backend, error){

	backend := &Backend{}

	for _, path := range paths {
		box := packr.New("i18nConfig", path)
		filesInBox := box.List()

		for _, file := range filesInBox {
			if strings.Contains(file, ".yml") {
				s, err := box.Find(file)

				if err != nil {
					return nil ,err
				}
				backend.contents = append(backend.contents, s)
			}
		}
	}
	return backend, nil
}

// NewWithWalk has the same functionality as New but uses filepath.Walk to find all the translation files recursively.
func NewWithWalk(paths ...string) i18n.Backend {
	backend := &Backend{}

	var files []string
	for _, p := range paths {
		filepath.Walk(p, func(path string, fileInfo os.FileInfo, err error) error {
			if isYamlFile(fileInfo) {
				files = append(files, path)
			}
			return nil
		})
	}
	for _, file := range files {
		if content, err := ioutil.ReadFile(file); err == nil {
			backend.contents = append(backend.contents, content)
		}
	}

	return backend
}

// NewWithWalk has the same functionality as New but uses filepath.Walk to find all the translation files recursively.
func NewPackrWithWalk(paths ...string) (i18n.Backend, error) {
	backend := &Backend{}

	//var subDirs []string

	for _, path := range paths {

		log.Println("************************ current main path: ", path)

		box := packr.New("i18nConfig", path)
		filesInBox := box.List()

		log.Println(filesInBox)

		for _, contentPath := range filesInBox {
			if strings.Contains(contentPath, ".yml") {
				s, err := box.Find(contentPath)

				if err != nil {
					return nil ,err
				}
				backend.contents = append(backend.contents, s)
			} else {
				log.Println("************************** is folder path: ", contentPath)
				walkPath(contentPath)
			}
		}


		//box.Walk(func(subPath string, f file.File) error {
		//	log.Println("**************************** sub dir ", subPath)
		//
		//	subDirs = append(subDirs, subPath)
		//
		//	return nil
		//})

	}

	//log.Println(subDirs)

	return backend, nil
}

func walkPath (subDirPath string) {
	//box := packr.New("i18nConfig", subDirPath)
	//filesInBox := box.List()
}

func isYamlFile(fileInfo os.FileInfo) bool {
	if fileInfo == nil {
		return false
	}
	return fileInfo.Mode().IsRegular() && (strings.HasSuffix(fileInfo.Name(), ".yml") || strings.HasSuffix(fileInfo.Name(), ".yaml"))
}

func walkFilesystem(fs http.FileSystem, entry http.File, prefix string) [][]byte {
	var (
		contents [][]byte
		err      error
		isRoot   bool
	)
	if entry == nil {
		if entry, err = fs.Open("/"); err != nil {
			return nil
		}
		isRoot = true
		defer entry.Close()
	}
	fileInfo, err := entry.Stat()
	if err != nil {
		return nil
	}
	if !isRoot {
		prefix = prefix + fileInfo.Name() + "/"
	}
	if fileInfo.IsDir() {
		if entries, err := entry.Readdir(-1); err == nil {
			for _, e := range entries {
				if file, err := fs.Open(prefix + e.Name()); err == nil {
					defer file.Close()
					contents = append(contents, walkFilesystem(fs, file, prefix)...)
				}
			}
		}
	} else if isYamlFile(fileInfo) {
		if content, err := ioutil.ReadAll(entry); err == nil {
			contents = append(contents, content)
		}
	}
	return contents
}

// NewWithFilesystem initializes a backend that reads translation files from an http.FileSystem.
func NewWithFilesystem(fss ...http.FileSystem) i18n.Backend {
	backend := &Backend{}

	for _, fs := range fss {
		backend.contents = append(backend.contents, walkFilesystem(fs, nil, "/")...)
	}
	return backend
}

// Backend YAML backend
type Backend struct {
	contents [][]byte
}

func loadTranslationsFromYaml(locale string, value interface{}, scopes []string) (translations []*i18n.Translation) {
	switch v := value.(type) {
	case yaml.MapSlice:
		for _, s := range v {
			results := loadTranslationsFromYaml(locale, s.Value, append(scopes, fmt.Sprint(s.Key)))
			translations = append(translations, results...)
		}
	default:
		var translation = &i18n.Translation{
			Locale: locale,
			Key:    strings.Join(scopes, "."),
			Value:  fmt.Sprint(v),
		}
		translations = append(translations, translation)
	}
	return
}

// LoadYAMLContent load YAML content
func (backend *Backend) LoadYAMLContent(content []byte) (translations []*i18n.Translation, err error) {
	var slice yaml.MapSlice

	if err = yaml.Unmarshal(content, &slice); err == nil {
		for _, item := range slice {
			translations = append(translations, loadTranslationsFromYaml(item.Key.(string) /* locale */, item.Value, []string{})...)
		}
	}

	return translations, err
}

// LoadTranslations load translations from YAML backend
func (backend *Backend) LoadTranslations() (translations []*i18n.Translation) {
	for _, content := range backend.contents {
		if results, err := backend.LoadYAMLContent(content); err == nil {
			translations = append(translations, results...)
		} else {
			panic(err)
		}
	}
	return translations
}

// SaveTranslation save translation into YAML backend, not implemented
func (backend *Backend) SaveTranslation(t *i18n.Translation) error {
	return errors.New("not implemented")
}

// DeleteTranslation delete translation into YAML backend, not implemented
func (backend *Backend) DeleteTranslation(t *i18n.Translation) error {
	return errors.New("not implemented")
}
