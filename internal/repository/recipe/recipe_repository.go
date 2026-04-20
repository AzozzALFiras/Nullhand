package recipe

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	recipemodel "github.com/AzozzALFiras/Nullhand/internal/model/recipe"
)

const defaultFileRelative = ".nullhand/recipes.json"

// Path returns the absolute path to the user recipes file.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("recipe repo: home dir: %w", err)
	}
	return filepath.Join(home, defaultFileRelative), nil
}

// Exists reports whether the user recipes file is present on disk.
func Exists() bool {
	path, err := Path()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Load merges the user recipes file (if any) over the provided defaults.
// User entries override defaults with the same name. If the user file is
// corrupt, defaults are returned with a logged warning — never an error,
// so the bot always starts with a working recipe set.
func Load(defaults map[string]recipemodel.Recipe) map[string]recipemodel.Recipe {
	merged := make(map[string]recipemodel.Recipe, len(defaults))
	for name, r := range defaults {
		r.Name = name
		merged[name] = r
	}

	path, err := Path()
	if err != nil {
		return merged
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("recipe repo: read %s: %v (using defaults only)", path, err)
		}
		return merged
	}

	var file recipemodel.File
	if err := json.Unmarshal(data, &file); err != nil {
		log.Printf("recipe repo: parse %s: %v (using defaults only)", path, err)
		return merged
	}
	for name, r := range file.Recipes {
		r.Name = name
		merged[name] = r
	}
	return merged
}

// Save writes the given recipes to the user file at 0600 permissions.
func Save(recipes map[string]recipemodel.Recipe) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("recipe repo: mkdir: %w", err)
	}
	file := recipemodel.File{Version: 1, Recipes: recipes}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("recipe repo: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("recipe repo: write: %w", err)
	}
	return nil
}
