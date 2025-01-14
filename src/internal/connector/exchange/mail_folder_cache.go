package exchange

import (
	"context"

	"github.com/pkg/errors"

	"github.com/alcionai/corso/src/internal/connector/graph"
	"github.com/alcionai/corso/src/internal/connector/support"
	"github.com/alcionai/corso/src/pkg/path"
)

var _ graph.ContainerResolver = &mailFolderCache{}

// mailFolderCache struct used to improve lookup of directories within exchange.Mail
// cache map of cachedContainers where the  key =  M365ID
// nameLookup map: Key: DisplayName Value: ID
type mailFolderCache struct {
	*containerResolver
	enumer containersEnumerator
	getter containerGetter
	userID string
}

// populateMailRoot manually fetches directories that are not returned during Graph for msgraph-sdk-go v. 40+
// rootFolderAlias is the top-level directory for exchange.Mail.
// DefaultMailFolder is the traditional "Inbox" for exchange.Mail
// Action ensures that cache will stop at appropriate level.
// @error iff the struct is not properly instantiated
func (mc *mailFolderCache) populateMailRoot(
	ctx context.Context,
) error {
	for _, fldr := range []string{rootFolderAlias, DefaultMailFolder} {
		var directory string

		f, err := mc.getter.GetContainerByID(ctx, mc.userID, fldr)
		if err != nil {
			return errors.Wrap(err, "fetching root folder"+support.ConnectorStackErrorTrace(err))
		}

		if fldr == DefaultMailFolder {
			directory = DefaultMailFolder
		}

		temp := graph.NewCacheFolder(f, path.Builder{}.Append(directory))
		if err := mc.addFolder(temp); err != nil {
			return errors.Wrap(err, "initializing mail resolver")
		}
	}

	return nil
}

// Populate utility function for populating the mailFolderCache.
// Number of Graph Queries: 1.
// @param baseID: M365ID of the base of the exchange.Mail.Folder
// @param baseContainerPath: the set of folder elements that make up the path
// for the base container in the cache.
func (mc *mailFolderCache) Populate(
	ctx context.Context,
	baseID string,
	baseContainerPath ...string,
) error {
	if err := mc.init(ctx); err != nil {
		return err
	}

	err := mc.enumer.EnumerateContainers(ctx, mc.userID, "", mc.addFolder)
	if err != nil {
		return err
	}

	if err := mc.populatePaths(ctx); err != nil {
		return errors.Wrap(err, "mail resolver")
	}

	return nil
}

// init ensures that the structure's fields are initialized.
// Fields Initialized when cache == nil:
// [mc.cache]
func (mc *mailFolderCache) init(
	ctx context.Context,
) error {
	if mc.containerResolver == nil {
		mc.containerResolver = newContainerResolver()
	}

	return mc.populateMailRoot(ctx)
}
