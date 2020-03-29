package rule

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type DirectoryRule struct {
	basePath string
}

func NewDirectoryRule(basePath string) DirectoryRule {
	return DirectoryRule{
		basePath: basePath,
	}
}

func (d DirectoryRule) IsRoutable(relativePath string) bool {
	absolutePath, err := filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), d.basePath, relativePath))
	if err != nil {
		log.Printf("Error trying to get absolute path of '%s': %w", relativePath, err)
		return false
	}

	{
		exists, err := Exists(absolutePath)
		if err != nil {
			log.Printf("Error checking if file exists: %w", err)
			return false
		}
		if !exists {
			return false
		}
		stat, _ := os.Stat(absolutePath)

		// TODO assert that we can't get below the base of "blarg"
		return stat.IsDir()
	}
}

func (d DirectoryRule) Handle(w http.ResponseWriter, r *http.Request) {
}

// TODO put this in my library
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
