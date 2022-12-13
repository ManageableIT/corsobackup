package kopia

import (
	"bytes"
	"context"
	"io"
	stdpath "path"
	"testing"

	"github.com/kopia/kopia/fs"
	"github.com/kopia/kopia/snapshot/snapshotfs"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/alcionai/corso/src/internal/connector/mockconnector"
	"github.com/alcionai/corso/src/internal/data"
	"github.com/alcionai/corso/src/internal/tester"
	"github.com/alcionai/corso/src/pkg/backup/details"
	"github.com/alcionai/corso/src/pkg/path"
)

func expectDirs(
	t *testing.T,
	entries []fs.Entry,
	dirs []string,
	exactly bool,
) {
	t.Helper()

	if exactly {
		require.Len(t, entries, len(dirs))
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}

	assert.Subset(t, names, dirs)
}

//revive:disable:context-as-argument
func getDirEntriesForEntry(
	t *testing.T,
	ctx context.Context,
	entry fs.Entry,
) []fs.Entry {
	//revive:enable:context-as-argument
	d, ok := entry.(fs.Directory)
	require.True(t, ok, "returned entry is not a directory")

	entries, err := fs.GetAllEntries(ctx, d)
	require.NoError(t, err)

	return entries
}

// ---------------
// unit tests
// ---------------
type limitedRangeReader struct {
	readLen int
	io.ReadCloser
}

func (lrr *limitedRangeReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		// Not well specified behavior, defer to underlying reader.
		return lrr.ReadCloser.Read(p)
	}

	toRead := lrr.readLen
	if len(p) < toRead {
		toRead = len(p)
	}

	return lrr.ReadCloser.Read(p[:toRead])
}

type VersionReadersUnitSuite struct {
	suite.Suite
}

func TestVersionReadersUnitSuite(t *testing.T) {
	suite.Run(t, new(VersionReadersUnitSuite))
}

func (suite *VersionReadersUnitSuite) TestWriteAndRead() {
	inputData := []byte("This is some data for the reader to test with")
	table := []struct {
		name         string
		readVersion  uint32
		writeVersion uint32
		check        assert.ErrorAssertionFunc
	}{
		{
			name:         "SameVersionSucceeds",
			readVersion:  42,
			writeVersion: 42,
			check:        assert.NoError,
		},
		{
			name:         "DifferentVersionsFail",
			readVersion:  7,
			writeVersion: 42,
			check:        assert.Error,
		},
	}

	for _, test := range table {
		suite.T().Run(test.name, func(t *testing.T) {
			baseReader := bytes.NewReader(inputData)

			reversible := &restoreStreamReader{
				expectedVersion: test.readVersion,
				ReadCloser: newBackupStreamReader(
					test.writeVersion,
					io.NopCloser(baseReader),
				),
			}

			defer reversible.Close()

			allData, err := io.ReadAll(reversible)
			test.check(t, err)

			if err != nil {
				return
			}

			assert.Equal(t, inputData, allData)
		})
	}
}

func readAllInParts(
	t *testing.T,
	partLen int,
	reader io.ReadCloser,
) ([]byte, int) {
	res := []byte{}
	read := 0
	tmp := make([]byte, partLen)

	for {
		n, err := reader.Read(tmp)
		if errors.Is(err, io.EOF) {
			break
		}

		require.NoError(t, err)

		read += n
		res = append(res, tmp[:n]...)
	}

	return res, read
}

func (suite *VersionReadersUnitSuite) TestWriteHandlesShortReads() {
	t := suite.T()
	inputData := []byte("This is some data for the reader to test with")
	version := uint32(42)
	baseReader := bytes.NewReader(inputData)
	versioner := newBackupStreamReader(version, io.NopCloser(baseReader))
	expectedToWrite := len(inputData) + int(versionSize)

	// "Write" all the data.
	versionedData, writtenLen := readAllInParts(t, 1, versioner)
	assert.Equal(t, expectedToWrite, writtenLen)

	// Read all of the data back.
	baseReader = bytes.NewReader(versionedData)
	reader := &restoreStreamReader{
		expectedVersion: version,
		// Be adversarial and only allow reads of length 1 from the byte reader.
		ReadCloser: &limitedRangeReader{
			readLen:    1,
			ReadCloser: io.NopCloser(baseReader),
		},
	}
	readData, readLen := readAllInParts(t, 1, reader)
	// This reports the bytes read and returned to the user, excluding the version
	// that is stripped off at the start.
	assert.Equal(t, len(inputData), readLen)
	assert.Equal(t, inputData, readData)
}

type CorsoProgressUnitSuite struct {
	suite.Suite
	targetFilePath path.Path
	targetFileName string
}

func TestCorsoProgressUnitSuite(t *testing.T) {
	suite.Run(t, new(CorsoProgressUnitSuite))
}

func (suite *CorsoProgressUnitSuite) SetupSuite() {
	p, err := path.Builder{}.Append(
		testInboxDir,
		"testFile",
	).ToDataLayerExchangePathForCategory(
		testTenant,
		testUser,
		path.EmailCategory,
		true,
	)
	require.NoError(suite.T(), err)

	suite.targetFilePath = p
	suite.targetFileName = suite.targetFilePath.ToBuilder().Dir().String()
}

type testInfo struct {
	info       *itemDetails
	err        error
	totalBytes int64
}

var finishedFileTable = []struct {
	name               string
	cachedItems        func(fname string, fpath path.Path) map[string]testInfo
	expectedBytes      int64
	expectedNumEntries int
	err                error
}{
	{
		name: "DetailsExist",
		cachedItems: func(fname string, fpath path.Path) map[string]testInfo {
			return map[string]testInfo{
				fname: {
					info:       &itemDetails{details.ItemInfo{}, fpath},
					err:        nil,
					totalBytes: 100,
				},
			}
		},
		expectedBytes: 100,
		// 1 file and 5 folders.
		expectedNumEntries: 6,
	},
	{
		name: "PendingNoDetails",
		cachedItems: func(fname string, fpath path.Path) map[string]testInfo {
			return map[string]testInfo{
				fname: {
					info: nil,
					err:  nil,
				},
			}
		},
		expectedNumEntries: 0,
	},
	{
		name: "HadError",
		cachedItems: func(fname string, fpath path.Path) map[string]testInfo {
			return map[string]testInfo{
				fname: {
					info: &itemDetails{details.ItemInfo{}, fpath},
					err:  assert.AnError,
				},
			}
		},
		expectedNumEntries: 0,
	},
	{
		name: "NotPending",
		cachedItems: func(fname string, fpath path.Path) map[string]testInfo {
			return nil
		},
		expectedNumEntries: 0,
	},
}

func (suite *CorsoProgressUnitSuite) TestFinishedFile() {
	for _, test := range finishedFileTable {
		suite.T().Run(test.name, func(t *testing.T) {
			bd := &details.Details{}
			cp := corsoProgress{
				UploadProgress: &snapshotfs.NullUploadProgress{},
				deets:          bd,
				pending:        map[string]*itemDetails{},
			}

			ci := test.cachedItems(suite.targetFileName, suite.targetFilePath)

			for k, v := range ci {
				cp.put(k, v.info)
			}

			require.Len(t, cp.pending, len(ci))

			for k, v := range ci {
				cp.FinishedFile(k, v.err)
			}

			assert.Empty(t, cp.pending)
			assert.Len(t, bd.Entries, test.expectedNumEntries)
		})
	}
}

func (suite *CorsoProgressUnitSuite) TestFinishedFileBuildsHierarchy() {
	t := suite.T()
	// Order of folders in hierarchy from root to leaf (excluding the item).
	expectedFolderOrder := suite.targetFilePath.ToBuilder().Dir().Elements()

	// Setup stuff.
	bd := &details.Details{}
	cp := corsoProgress{
		UploadProgress: &snapshotfs.NullUploadProgress{},
		deets:          bd,
		pending:        map[string]*itemDetails{},
	}

	deets := &itemDetails{details.ItemInfo{}, suite.targetFilePath}
	cp.put(suite.targetFileName, deets)
	require.Len(t, cp.pending, 1)

	cp.FinishedFile(suite.targetFileName, nil)

	// Gather information about the current state.
	var (
		curRef     *details.DetailsEntry
		refToEntry = map[string]*details.DetailsEntry{}
	)

	for i := 0; i < len(bd.Entries); i++ {
		e := &bd.Entries[i]
		if e.Folder == nil {
			continue
		}

		refToEntry[e.ShortRef] = e

		if e.Folder.DisplayName == expectedFolderOrder[len(expectedFolderOrder)-1] {
			curRef = e
		}
	}

	// Actual tests start here.
	var rootRef *details.DetailsEntry

	// Traverse the details entries from leaf to root, following the ParentRef
	// fields. At the end rootRef should point to the root of the path.
	for i := len(expectedFolderOrder) - 1; i >= 0; i-- {
		name := expectedFolderOrder[i]

		require.NotNil(t, curRef)
		assert.Equal(t, name, curRef.Folder.DisplayName)

		rootRef = curRef
		curRef = refToEntry[curRef.ParentRef]
	}

	// Hierarchy root's ParentRef = "" and map will return nil.
	assert.Nil(t, curRef)
	require.NotNil(t, rootRef)
	assert.Empty(t, rootRef.ParentRef)
}

func (suite *CorsoProgressUnitSuite) TestFinishedHashingFile() {
	for _, test := range finishedFileTable {
		suite.T().Run(test.name, func(t *testing.T) {
			bd := &details.Details{}
			cp := corsoProgress{
				UploadProgress: &snapshotfs.NullUploadProgress{},
				deets:          bd,
				pending:        map[string]*itemDetails{},
			}

			ci := test.cachedItems(suite.targetFileName, suite.targetFilePath)

			for k, v := range ci {
				cp.FinishedHashingFile(k, v.totalBytes)
			}

			assert.Empty(t, cp.pending)
			assert.Equal(t, test.expectedBytes, cp.totalBytes)
		})
	}
}

type HierarchyBuilderUnitSuite struct {
	suite.Suite
	testPath path.Path
}

func (suite *HierarchyBuilderUnitSuite) SetupSuite() {
	tmp, err := path.FromDataLayerPath(
		stdpath.Join(
			testTenant,
			path.ExchangeService.String(),
			testUser,
			path.EmailCategory.String(),
			testInboxDir,
		),
		false,
	)
	require.NoError(suite.T(), err)

	suite.testPath = tmp
}

func TestHierarchyBuilderUnitSuite(t *testing.T) {
	suite.Run(t, new(HierarchyBuilderUnitSuite))
}

func (suite *HierarchyBuilderUnitSuite) TestBuildDirectoryTree() {
	tester.LogTimeOfTest(suite.T())
	ctx, flush := tester.NewContext()

	defer flush()

	t := suite.T()
	tenant := "a-tenant"
	user1 := testUser
	user1Encoded := encodeAsPath(user1)
	user2 := "user2"
	user2Encoded := encodeAsPath(user2)

	p2, err := path.FromDataLayerPath(
		stdpath.Join(
			tenant,
			service,
			user2,
			category,
			testInboxDir,
		),
		false,
	)
	require.NoError(t, err)

	// Encode user names here so we don't have to decode things later.
	expectedFileCount := map[string]int{
		user1Encoded: 5,
		user2Encoded: 42,
	}
	expectedServiceCats := map[string]ServiceCat{
		serviceCatTag(suite.testPath): {},
		serviceCatTag(p2):             {},
	}
	expectedResourceOwners := map[string]struct{}{
		suite.testPath.ResourceOwner(): {},
		p2.ResourceOwner():             {},
	}

	progress := &corsoProgress{pending: map[string]*itemDetails{}}

	collections := []data.Collection{
		mockconnector.NewMockExchangeCollection(
			suite.testPath,
			expectedFileCount[user1Encoded],
		),
		mockconnector.NewMockExchangeCollection(
			p2,
			expectedFileCount[user2Encoded],
		),
	}

	// Returned directory structure should look like:
	// - a-tenant
	//   - exchange
	//     - user1
	//       - emails
	//         - Inbox
	//           - 5 separate files
	//     - user2
	//       - emails
	//         - Inbox
	//           - 42 separate files
	dirTree, oc, err := inflateDirTree(ctx, collections, progress)
	require.NoError(t, err)

	assert.Equal(t, expectedServiceCats, oc.ServiceCats)
	assert.Equal(t, expectedResourceOwners, oc.ResourceOwners)

	assert.Equal(t, encodeAsPath(testTenant), dirTree.Name())

	entries, err := fs.GetAllEntries(ctx, dirTree)
	require.NoError(t, err)

	expectDirs(t, entries, encodeElements(service), true)

	entries = getDirEntriesForEntry(t, ctx, entries[0])
	expectDirs(t, entries, encodeElements(user1, user2), true)

	for _, entry := range entries {
		userName := entry.Name()

		entries = getDirEntriesForEntry(t, ctx, entry)
		expectDirs(t, entries, encodeElements(category), true)

		entries = getDirEntriesForEntry(t, ctx, entries[0])
		expectDirs(t, entries, encodeElements(testInboxDir), true)

		entries = getDirEntriesForEntry(t, ctx, entries[0])
		assert.Len(t, entries, expectedFileCount[userName])
	}

	totalFileCount := 0
	for _, c := range expectedFileCount {
		totalFileCount += c
	}

	assert.Len(t, progress.pending, totalFileCount)
}

func (suite *HierarchyBuilderUnitSuite) TestBuildDirectoryTree_MixedDirectory() {
	ctx, flush := tester.NewContext()
	defer flush()

	subdir := "subfolder"

	p2, err := suite.testPath.Append(subdir, false)
	require.NoError(suite.T(), err)

	expectedServiceCats := map[string]ServiceCat{
		serviceCatTag(suite.testPath): {},
		serviceCatTag(p2):             {},
	}
	expectedResourceOwners := map[string]struct{}{
		suite.testPath.ResourceOwner(): {},
		p2.ResourceOwner():             {},
	}

	// Test multiple orders of items because right now order can matter. Both
	// orders result in a directory structure like:
	// - a-tenant
	//   - exchange
	//     - user1
	//       - emails
	//         - Inbox
	//           - subfolder
	//             - 5 separate files
	//           - 42 separate files
	table := []struct {
		name   string
		layout []data.Collection
	}{
		{
			name: "SubdirFirst",
			layout: []data.Collection{
				mockconnector.NewMockExchangeCollection(
					p2,
					5,
				),
				mockconnector.NewMockExchangeCollection(
					suite.testPath,
					42,
				),
			},
		},
		{
			name: "SubdirLast",
			layout: []data.Collection{
				mockconnector.NewMockExchangeCollection(
					suite.testPath,
					42,
				),
				mockconnector.NewMockExchangeCollection(
					p2,
					5,
				),
			},
		},
	}

	for _, test := range table {
		suite.T().Run(test.name, func(t *testing.T) {
			progress := &corsoProgress{pending: map[string]*itemDetails{}}

			dirTree, oc, err := inflateDirTree(ctx, test.layout, progress)
			require.NoError(t, err)

			assert.Equal(t, expectedServiceCats, oc.ServiceCats)
			assert.Equal(t, expectedResourceOwners, oc.ResourceOwners)

			assert.Equal(t, encodeAsPath(testTenant), dirTree.Name())

			entries, err := fs.GetAllEntries(ctx, dirTree)
			require.NoError(t, err)

			expectDirs(t, entries, encodeElements(service), true)

			entries = getDirEntriesForEntry(t, ctx, entries[0])
			expectDirs(t, entries, encodeElements(testUser), true)

			entries = getDirEntriesForEntry(t, ctx, entries[0])
			expectDirs(t, entries, encodeElements(category), true)

			entries = getDirEntriesForEntry(t, ctx, entries[0])
			expectDirs(t, entries, encodeElements(testInboxDir), true)

			entries = getDirEntriesForEntry(t, ctx, entries[0])
			// 42 files and 1 subdirectory.
			assert.Len(t, entries, 43)

			// One of these entries should be a subdirectory with items in it.
			subDirs := []fs.Directory(nil)
			for _, e := range entries {
				d, ok := e.(fs.Directory)
				if !ok {
					continue
				}

				subDirs = append(subDirs, d)
				assert.Equal(t, encodeAsPath(subdir), d.Name())
			}

			require.Len(t, subDirs, 1)

			entries = getDirEntriesForEntry(t, ctx, entries[0])
			assert.Len(t, entries, 5)
		})
	}
}

func (suite *HierarchyBuilderUnitSuite) TestBuildDirectoryTree_Fails() {
	p2, err := path.Builder{}.Append(testInboxDir).ToDataLayerExchangePathForCategory(
		"tenant2",
		"user2",
		path.EmailCategory,
		false,
	)
	require.NoError(suite.T(), err)

	table := []struct {
		name   string
		layout []data.Collection
	}{
		{
			"MultipleRoots",
			// Directory structure would look like:
			// - tenant1
			//   - exchange
			//     - user1
			//       - emails
			//         - Inbox
			//           - 5 separate files
			// - tenant2
			//   - exchange
			//     - user2
			//       - emails
			//         - Inbox
			//           - 42 separate files
			[]data.Collection{
				mockconnector.NewMockExchangeCollection(
					suite.testPath,
					5,
				),
				mockconnector.NewMockExchangeCollection(
					p2,
					42,
				),
			},
		},
		{
			"NoCollectionPath",
			[]data.Collection{
				mockconnector.NewMockExchangeCollection(
					nil,
					5,
				),
			},
		},
	}

	for _, test := range table {
		ctx, flush := tester.NewContext()
		defer flush()

		suite.T().Run(test.name, func(t *testing.T) {
			_, _, err := inflateDirTree(ctx, test.layout, nil)
			assert.Error(t, err)
		})
	}
}