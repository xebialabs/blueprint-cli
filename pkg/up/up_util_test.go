package up

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRepo(t *testing.T) {
	t.Run("should return repo with a branch name", func(t *testing.T) {
		repo := getRepo("xl-up")

		fmt.Println(repo.GetInfo())
		assert.Equal(t, repo.GetName(), XlUpBlueprint)
		assert.Equal(t, repo.GetProvider(), "github")
		assert.Contains(t, repo.GetInfo(), "Branch: xl-up")
	})
}

func TestMergeMaps(t *testing.T) {

	t.Run("should return empty map when the maps are empty", func(t *testing.T) {
		autoMap := make(map[string]string)
		providedMap := make(map[string]string)

		mergedMap, isConflict := mergeMaps(autoMap, providedMap)

		assert.Equal(t, isConflict, false)
		assert.Equal(t, len(mergedMap), 0)
	})

	t.Run("should merge map when provided map is empty", func(t *testing.T) {
		autoMap := make(map[string]string)
		autoMap["one"] = "1"
		autoMap["two"] = "2"
		autoMap["three"] = "3"
		autoMap["four"] = "4"

		providedMap := make(map[string]string)

		mergedMap, isConflict := mergeMaps(autoMap, providedMap)

		assert.Equal(t, isConflict, false)
		assert.Equal(t, len(mergedMap), 4)
		assert.Equal(t, mergedMap["one"], "1")
		assert.Equal(t, mergedMap["two"], "2")
		assert.Equal(t, mergedMap["three"], "3")
		assert.Equal(t, mergedMap["four"], "4")
	})

	t.Run("should merge map when auto map is empty", func(t *testing.T) {
		autoMap := make(map[string]string)

		providedMap := make(map[string]string)
		providedMap["one"] = "1"
		providedMap["two"] = "2"
		providedMap["three"] = "3"
		providedMap["four"] = "4"

		mergedMap, isConflict := mergeMaps(autoMap, providedMap)

		assert.Equal(t, isConflict, false)
		assert.Equal(t, len(mergedMap), 4)
		assert.Equal(t, mergedMap["one"], "1")
		assert.Equal(t, mergedMap["two"], "2")
		assert.Equal(t, mergedMap["three"], "3")
		assert.Equal(t, mergedMap["four"], "4")
	})

	t.Run("should merge map when there is no overlap", func(t *testing.T) {
		autoMap := make(map[string]string)
		autoMap["two"] = "2"
		autoMap["four"] = "4"

		providedMap := make(map[string]string)
		providedMap["one"] = "1"
		providedMap["three"] = "3"

		mergedMap, isConflict := mergeMaps(autoMap, providedMap)

		assert.Equal(t, isConflict, false)
		assert.Equal(t, len(mergedMap), 4)
		assert.Equal(t, mergedMap["one"], "1")
		assert.Equal(t, mergedMap["two"], "2")
		assert.Equal(t, mergedMap["three"], "3")
		assert.Equal(t, mergedMap["four"], "4")
	})

	t.Run("should merge map when there is overlap", func(t *testing.T) {
		autoMap := make(map[string]string)
		autoMap["one"] = "1"
		autoMap["two"] = "2"

		providedMap := make(map[string]string)
		providedMap["one"] = "one"
		providedMap["two"] = "two"

		mergedMap, isConflict := mergeMaps(autoMap, providedMap)

		assert.Equal(t, isConflict, true)
		assert.Equal(t, len(mergedMap), 2)
		assert.Equal(t, mergedMap["one"], "1")
		assert.Equal(t, mergedMap["two"], "2")
	})

	t.Run("should merge map when there is overlap", func(t *testing.T) {
		autoMap := make(map[string]string)
		autoMap["one"] = "1"
		autoMap["two"] = "2"
		autoMap["three"] = "3"

		providedMap := make(map[string]string)
		providedMap["one"] = "one"
		providedMap["two"] = "two"
		providedMap["four"] = "four"

		mergedMap, isConflict := mergeMaps(autoMap, providedMap)

		assert.Equal(t, isConflict, true)
		assert.Equal(t, len(mergedMap), 4)
		assert.Equal(t, mergedMap["one"], "1")
		assert.Equal(t, mergedMap["two"], "2")
		assert.Equal(t, mergedMap["three"], "3")
		assert.Equal(t, mergedMap["four"], "four")
	})

}

func TestDecideVersionMatch(t *testing.T) {
	t.Run("should throw error when the new version number is less than the installed one", func(t *testing.T) {
		msg, err := decideVersionMatch("10.0.0", "9.9.9")

		assert.Equal(t, msg, "")
		assert.Equal(t, err.Error(), "cannot downgrade the deployment from 10.0.0 to 9.9.9")
	})

	t.Run("should accept when the new version number is greater than the installed one", func(t *testing.T) {
		msg, err := decideVersionMatch("9.9.9", "9.9.10")

		assert.Equal(t, msg, "upgrading from 9.9.9 to 9.9.10")
		assert.Equal(t, err, nil)

		msg, err = decideVersionMatch("9.10.9", "9.10.10")

		assert.Equal(t, msg, "upgrading from 9.10.9 to 9.10.10")
		assert.Equal(t, err, nil)

		msg, err = decideVersionMatch("10.10.9", "10.10.10")

		assert.Equal(t, msg, "upgrading from 10.10.9 to 10.10.10")
		assert.Equal(t, err, nil)
	})

	t.Run("should throw error when the new version number is less than the installed one", func(t *testing.T) {
		msg, err := decideVersionMatch("10.0.0", "10.0.0")

		assert.Equal(t, msg, "")
		assert.Equal(t, err.Error(), "the given version 10.0.0 already exists")
	})
}
