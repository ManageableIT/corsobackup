package config_test

import (
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/alcionai/corso/cli/config"
	"github.com/alcionai/corso/pkg/repository"
	"github.com/alcionai/corso/pkg/storage"
)

const (
	configFileTemplate = `
bucket = '%s'
endpoint = 's3.amazonaws.com'
prefix = 'test-prefix'
provider = 'S3'
tenantid = '%s'
`
)

type ConfigSuite struct {
	suite.Suite
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}

func (suite *ConfigSuite) TestReadRepoConfigBasic() {
	// Generate test config file
	b := "read-repo-config-basic-bucket"
	tID := "6f34ac30-8196-469b-bf8f-d83deadbbbba"
	testConfigData := fmt.Sprintf(configFileTemplate, b, tID)
	testConfigFilePath := path.Join(suite.T().TempDir(), "corso.toml")
	err := ioutil.WriteFile(testConfigFilePath, []byte(testConfigData), 0700)
	assert.NoError(suite.T(), err)

	// Configure viper to read test config file
	viper.SetConfigFile(testConfigFilePath)

	// Read and validate config
	s3Cfg, account, err := config.ReadRepoConfig()
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), b, s3Cfg.Bucket)
	assert.Equal(suite.T(), tID, account.TenantID)
}

func (suite *ConfigSuite) TestWriteReadConfig() {
	// Configure viper to read test config file
	tempDir := suite.T().TempDir()
	testConfigFilePath := path.Join(tempDir, "corso.toml")
	err := config.InitConfig(testConfigFilePath)
	assert.NoError(suite.T(), err)

	s3Cfg := storage.S3Config{Bucket: "write-read-config-bucket"}
	account := repository.Account{TenantID: "6f34ac30-8196-469b-bf8f-d83deadbbbbd"}
	err = config.WriteRepoConfig(s3Cfg, account)
	assert.NoError(suite.T(), err)

	readS3Cfg, readAccount, err := config.ReadRepoConfig()
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), s3Cfg, readS3Cfg)
	assert.Equal(suite.T(), account, readAccount)
}