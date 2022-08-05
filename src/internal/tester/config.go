package tester

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/alcionai/corso/pkg/account"
)

const (
	// S3 config
	testCfgBucket   = "bucket"
	testCfgEndpoint = "endpoint"
	testCfgPrefix   = "prefix"

	// M365 config
	testCfgTenantID = "tenantid"
	testCfgUserID   = "m365userid"
)

// test specific env vars
const (
	EnvCorsoM365TestUserID     = "CORSO_M356_TEST_USER_ID"
	EnvCorsoTestConfigFilePath = "CORSO_TEST_CONFIG_FILE"
)

// global to hold the test config results.
var testConfig map[string]string

// call this instead of returning testConfig directly.
func cloneTestConfig() map[string]string {
	if testConfig == nil {
		return map[string]string{}
	}
	clone := map[string]string{}
	for k, v := range testConfig {
		clone[k] = v
	}
	return clone
}

func NewTestViper() (*viper.Viper, error) {
	vpr := viper.New()

	configFilePath := os.Getenv(EnvCorsoTestConfigFilePath)
	if len(configFilePath) == 0 {
		return vpr, nil
	}

	// Or use a custom file location
	fileName := path.Base(configFilePath)
	ext := path.Ext(configFilePath)
	if len(ext) == 0 {
		return nil, errors.New("corso_test requires an extension")
	}

	vpr.SetConfigFile(configFilePath)
	vpr.AddConfigPath(path.Dir(configFilePath))
	vpr.SetConfigType(ext[1:])
	fileName = strings.TrimSuffix(fileName, ext)
	vpr.SetConfigName(fileName)

	return vpr, nil
}

// reads a corso configuration file with values specific to
// local integration test controls.  Populates values with
// defaults where standard.
func readTestConfig() (map[string]string, error) {
	if testConfig != nil {
		return cloneTestConfig(), nil
	}

	vpr, err := NewTestViper()
	if err != nil {
		return nil, err
	}

	// only error if reading an existing file failed.  No problem if we're missing files.
	if err = vpr.ReadInConfig(); err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		if !ok {
			return nil, errors.Wrap(err, "reading config file: "+viper.ConfigFileUsed())
		}
	}

	testEnv := map[string]string{}
	fallbackTo(testEnv, testCfgBucket, vpr.GetString(testCfgBucket), "test-corso-repo-init")
	fallbackTo(testEnv, testCfgEndpoint, vpr.GetString(testCfgEndpoint), "s3.amazonaws.com")
	fallbackTo(testEnv, testCfgPrefix, vpr.GetString(testCfgPrefix))
	fallbackTo(testEnv, testCfgTenantID, os.Getenv(account.TenantID), vpr.GetString(testCfgTenantID))
	fallbackTo(testEnv, testCfgUserID, os.Getenv(EnvCorsoM365TestUserID), vpr.GetString(testCfgTenantID), "lidiah@8qzvrj.onmicrosoft.com")
	testEnv[EnvCorsoTestConfigFilePath] = os.Getenv(EnvCorsoTestConfigFilePath)

	testConfig = testEnv
	return cloneTestConfig(), nil
}

// MakeTempTestConfigClone makes a copy of the test config file in a temp directory.
// This allows tests which interface with reading and writing to a config file
// (such as the CLI) to safely manipulate file contents without amending the user's
// original file.
//
// Returns a filepath string pointing to the location of the temp file.
func MakeTempTestConfigClone(t *testing.T) (*viper.Viper, string, error) {
	cfg, err := readTestConfig()
	if err != nil {
		return nil, "", err
	}

	fName := path.Base(os.Getenv(EnvCorsoTestConfigFilePath))
	if len(fName) == 0 || fName == "." || fName == "/" {
		fName = ".corso_test.toml"
	}
	tDir := t.TempDir()
	tDirFp := path.Join(tDir, fName)

	if _, err := os.Create(tDirFp); err != nil {
		return nil, "", err
	}

	vpr := viper.New()
	vpr.SetConfigFile(tDirFp)
	vpr.AddConfigPath(tDir)
	vpr.SetConfigType(path.Ext(fName))
	vpr.SetConfigName(fName)

	for k, v := range cfg {
		vpr.Set(k, v)
	}

	if err := vpr.WriteConfig(); err != nil {
		return nil, "", err
	}

	return vpr, tDirFp, nil
}

// writes the first non-zero valued string to the map at the key.
// fallback priority should match viper ordering (manually handled
// here since viper fails to provide fallbacks on fileNotFoundErr):
// manual overrides > flags > env vars > config file > default value
func fallbackTo(m map[string]string, key string, fallbacks ...string) {
	for _, fb := range fallbacks {
		if len(fb) > 0 {
			m[key] = fb
			return
		}
	}
}