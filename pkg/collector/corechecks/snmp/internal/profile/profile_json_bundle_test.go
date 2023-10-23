package profile

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/DataDog/datadog-agent/pkg/config"
)

func Test_loadBundleJSONProfiles(t *testing.T) {
	defaultTestConfdPath, _ := filepath.Abs(filepath.Join("..", "test", "zipprofiles.d"))
	SetGlobalProfileConfigMap(nil)
	config.Datadog.Set("confd_path", defaultTestConfdPath)

	defaultProfiles, err := loadBundleJSONProfiles()
	assert.Nil(t, err)

	var actualProfiles []string
	for key := range defaultProfiles {
		actualProfiles = append(actualProfiles, key)
	}

	expectedProfiles := []string{
		"default-p1",      // yaml default profile
		"my-profile-name", // downloaded json profile
		"profile-from-ui", // downloaded json profile
	}
	assert.ElementsMatch(t, expectedProfiles, actualProfiles)
}