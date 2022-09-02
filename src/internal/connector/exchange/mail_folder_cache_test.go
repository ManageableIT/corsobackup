package exchange

import (
	"context"
	stdpath "path"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/alcionai/corso/internal/connector/graph"
	"github.com/alcionai/corso/internal/path"
	"github.com/alcionai/corso/internal/tester"
)

const (
	// Need to use a hard-coded ID because GetAllFolderNamesForUser only gets
	// top-level folders right now.
	//nolint:lll
	testFolderID = "AAMkAGZmNjNlYjI3LWJlZWYtNGI4Mi04YjMyLTIxYThkNGQ4NmY1MwAuAAAAAADCNgjhM9QmQYWNcI7hCpPrAQDSEBNbUIB9RL6ePDeF3FIYAABl7AqpAAA="

	// Full folder path for the folder above.
	expectedFolderPath = "toplevel/subFolder/subsubfolder"
)

type mockContainer struct {
	id       *string
	name     *string
	parentID *string
}

//nolint:revive
func (m mockContainer) GetId() *string {
	return m.id
}

func (m mockContainer) GetDisplayName() *string {
	return m.name
}

//nolint:revive
func (m mockContainer) GetParentFolderId() *string {
	return m.parentID
}

type MailFolderCacheUnitSuite struct {
	suite.Suite
}

func TestMailFolderCacheUnitSuite(t *testing.T) {
	suite.Run(t, new(MailFolderCacheUnitSuite))
}

func (suite *MailFolderCacheUnitSuite) TestCheckRequiredValues() {
	id := uuid.NewString()
	name := "foo"
	parentID := uuid.NewString()
	emptyString := ""

	table := []struct {
		name  string
		c     mockContainer
		check assert.ErrorAssertionFunc
	}{
		{
			name: "NilID",
			c: mockContainer{
				id:       nil,
				name:     &name,
				parentID: &parentID,
			},
			check: assert.Error,
		},
		{
			name: "NilDisplayName",
			c: mockContainer{
				id:       &id,
				name:     nil,
				parentID: &parentID,
			},
			check: assert.Error,
		},
		{
			name: "NilParentFolderID",
			c: mockContainer{
				id:       &id,
				name:     &name,
				parentID: nil,
			},
			check: assert.Error,
		},
		{
			name: "EmptyID",
			c: mockContainer{
				id:       &emptyString,
				name:     &name,
				parentID: &parentID,
			},
			check: assert.Error,
		},
		{
			name: "EmptyDisplayName",
			c: mockContainer{
				id:       &id,
				name:     &emptyString,
				parentID: &parentID,
			},
			check: assert.Error,
		},
		{
			name: "EmptyParentFolderID",
			c: mockContainer{
				id:       &id,
				name:     &name,
				parentID: &emptyString,
			},
			check: assert.Error,
		},
		{
			name: "AllValues",
			c: mockContainer{
				id:       &id,
				name:     &name,
				parentID: &parentID,
			},
			check: assert.NoError,
		},
	}

	for _, test := range table {
		suite.T().Run(test.name, func(t *testing.T) {
			test.check(t, checkRequiredValues(test.c))
		})
	}
}

func newMockCachedContainer(name string) *mockCachedContainer {
	return &mockCachedContainer{
		id:          uuid.NewString(),
		parentID:    uuid.NewString(),
		displayName: name,
	}
}

type mockCachedContainer struct {
	id           string
	parentID     string
	displayName  string
	p            *path.Builder
	expectedPath string
}

//nolint:revive
func (m mockCachedContainer) GetId() *string {
	return &m.id
}

//nolint:revive
func (m mockCachedContainer) GetParentFolderId() *string {
	return &m.parentID
}

func (m mockCachedContainer) GetDisplayName() *string {
	return &m.displayName
}

func (m mockCachedContainer) Path() *path.Builder {
	return m.p
}

func (m *mockCachedContainer) SetPath(newPath *path.Builder) {
	m.p = newPath
}

// TestConfiguredMailFolderCacheUnitSuite cannot run its tests in parallel.
type ConfiguredMailFolderCacheUnitSuite struct {
	suite.Suite

	mc mailFolderCache

	allContainers []*mockCachedContainer
}

func (suite *ConfiguredMailFolderCacheUnitSuite) SetupTest() {
	suite.allContainers = []*mockCachedContainer{}

	for i := 0; i < 4; i++ {
		suite.allContainers = append(
			suite.allContainers,
			newMockCachedContainer(strings.Repeat("sub", i)+"folder"),
		)
	}

	// Base case for the recursive lookup.
	suite.allContainers[0].p = path.Builder{}.Append(suite.allContainers[0].displayName)
	suite.allContainers[0].expectedPath = suite.allContainers[0].displayName

	for i := 1; i < len(suite.allContainers); i++ {
		suite.allContainers[i].parentID = suite.allContainers[i-1].id
		suite.allContainers[i].expectedPath = stdpath.Join(
			suite.allContainers[i-1].expectedPath,
			suite.allContainers[i].displayName,
		)
	}

	suite.mc = mailFolderCache{cache: map[string]cachedContainer{}}

	for _, c := range suite.allContainers {
		suite.mc.cache[c.id] = c
	}
}

func TestConfiguredMailFolderCacheUnitSuite(t *testing.T) {
	suite.Run(t, new(ConfiguredMailFolderCacheUnitSuite))
}

func (suite *ConfiguredMailFolderCacheUnitSuite) TestLookupCachedFolderNoPathsCached() {
	ctx := context.Background()

	for _, c := range suite.allContainers {
		suite.T().Run(*c.GetDisplayName(), func(t *testing.T) {
			p, err := suite.mc.IDToPath(ctx, c.id)
			require.NoError(t, err)

			assert.Equal(t, c.expectedPath, p.String())
		})
	}
}

func (suite *ConfiguredMailFolderCacheUnitSuite) TestLookupCachedFolderCachesPaths() {
	t := suite.T()
	ctx := context.Background()
	c := suite.allContainers[len(suite.allContainers)-1]

	p, err := suite.mc.IDToPath(ctx, c.id)
	require.NoError(t, err)

	assert.Equal(t, c.expectedPath, p.String())

	c.parentID = "foo"

	p, err = suite.mc.IDToPath(ctx, c.id)
	require.NoError(t, err)

	assert.Equal(t, c.expectedPath, p.String())
}

func (suite *ConfiguredMailFolderCacheUnitSuite) TestLookupCachedFolderErrorsParentNotFound() {
	t := suite.T()
	ctx := context.Background()
	last := suite.allContainers[len(suite.allContainers)-1]
	almostLast := suite.allContainers[len(suite.allContainers)-2]

	delete(suite.mc.cache, almostLast.id)

	_, err := suite.mc.IDToPath(ctx, last.id)
	assert.Error(t, err)
}

func (suite *ConfiguredMailFolderCacheUnitSuite) TestLookupCachedFolderErrorsNotFound() {
	t := suite.T()
	ctx := context.Background()

	_, err := suite.mc.IDToPath(ctx, "foo")
	assert.Error(t, err)
}

type MailFolderCacheIntegrationSuite struct {
	suite.Suite
	gs graph.Service
}

func (suite *MailFolderCacheIntegrationSuite) SetupSuite() {
	t := suite.T()

	_, err := tester.GetRequiredEnvVars(tester.M365AcctCredEnvs...)
	require.NoError(t, err)

	a := tester.NewM365Account(t)
	require.NoError(t, err)

	m365, err := a.M365Config()
	require.NoError(t, err)

	service, err := createService(m365, false)
	require.NoError(t, err)

	suite.gs = service
}

func TestMailFolderCacheIntegrationSuite(t *testing.T) {
	if err := tester.RunOnAny(
		tester.CorsoCITests,
		tester.CorsoGraphConnectorTests,
	); err != nil {
		t.Skip()
	}

	suite.Run(t, new(MailFolderCacheIntegrationSuite))
}

func (suite *MailFolderCacheIntegrationSuite) TestDeltaFetch() {
	ctx := context.Background()
	t := suite.T()
	userID := tester.M365UserID(t)

	mfc := mailFolderCache{
		userID: userID,
		gs:     suite.gs,
	}

	require.NoError(t, mfc.Populate(ctx))

	p, err := mfc.IDToPath(ctx, testFolderID)
	require.NoError(t, err)

	assert.Equal(t, expectedFolderPath, p.String())
}