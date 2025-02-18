package api

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/microsoft/kiota-abstractions-go/serialization"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
	"github.com/pkg/errors"

	"github.com/alcionai/corso/src/internal/connector/graph"
	"github.com/alcionai/corso/src/internal/connector/support"
)

// ---------------------------------------------------------------------------
// controller
// ---------------------------------------------------------------------------

func (c Client) Contacts() Contacts {
	return Contacts{c}
}

// Contacts is an interface-compliant provider of the client.
type Contacts struct {
	Client
}

// ---------------------------------------------------------------------------
// methods
// ---------------------------------------------------------------------------

// CreateContactFolder makes a contact folder with the displayName of folderName.
// If successful, returns the created folder object.
func (c Contacts) CreateContactFolder(
	ctx context.Context,
	user, folderName string,
) (models.ContactFolderable, error) {
	requestBody := models.NewContactFolder()
	temp := folderName
	requestBody.SetDisplayName(&temp)

	return c.stable.Client().UsersById(user).ContactFolders().Post(ctx, requestBody, nil)
}

// DeleteContactFolder deletes the ContactFolder associated with the M365 ID if permissions are valid.
// Errors returned if the function call was not successful.
func (c Contacts) DeleteContactFolder(
	ctx context.Context,
	user, folderID string,
) error {
	return c.stable.Client().UsersById(user).ContactFoldersById(folderID).Delete(ctx, nil)
}

// RetrieveContactDataForUser is a GraphRetrievalFun that returns all associated fields.
func (c Contacts) RetrieveContactDataForUser(
	ctx context.Context,
	user, m365ID string,
) (serialization.Parsable, error) {
	return c.stable.Client().UsersById(user).ContactsById(m365ID).Get(ctx, nil)
}

// GetAllContactFolderNamesForUser is a GraphQuery function for getting
// ContactFolderId and display names for contacts. All other information is omitted.
// Does not return the default Contact Folder
func (c Contacts) GetAllContactFolderNamesForUser(
	ctx context.Context,
	user string,
) (serialization.Parsable, error) {
	options, err := optionsForContactFolders([]string{"displayName", "parentFolderId"})
	if err != nil {
		return nil, err
	}

	return c.stable.Client().UsersById(user).ContactFolders().Get(ctx, options)
}

func (c Contacts) GetContainerByID(
	ctx context.Context,
	userID, dirID string,
) (graph.Container, error) {
	ofcf, err := optionsForContactFolderByID([]string{"displayName", "parentFolderId"})
	if err != nil {
		return nil, errors.Wrap(err, "options for contact folder")
	}

	return c.stable.Client().
		UsersById(userID).
		ContactFoldersById(dirID).
		Get(ctx, ofcf)
}

// EnumerateContainers iterates through all of the users current
// contacts folders, converting each to a graph.CacheFolder, and calling
// fn(cf) on each one.  If fn(cf) errors, the error is aggregated
// into a multierror that gets returned to the caller.
// Folder hierarchy is represented in its current state, and does
// not contain historical data.
func (c Contacts) EnumerateContainers(
	ctx context.Context,
	userID, baseDirID string,
	fn func(graph.CacheFolder) error,
) error {
	service, err := c.service()
	if err != nil {
		return err
	}

	var (
		errs   *multierror.Error
		fields = []string{"displayName", "parentFolderId"}
	)

	ofcf, err := optionsForContactChildFolders(fields)
	if err != nil {
		return errors.Wrapf(err, "options for contact child folders: %v", fields)
	}

	builder := service.Client().
		UsersById(userID).
		ContactFoldersById(baseDirID).
		ChildFolders()

	for {
		resp, err := builder.Get(ctx, ofcf)
		if err != nil {
			return errors.Wrap(err, support.ConnectorStackErrorTrace(err))
		}

		for _, fold := range resp.GetValue() {
			if err := checkIDAndName(fold); err != nil {
				errs = multierror.Append(err, errs)
				continue
			}

			temp := graph.NewCacheFolder(fold, nil)

			err = fn(temp)
			if err != nil {
				errs = multierror.Append(err, errs)
				continue
			}
		}

		if resp.GetOdataNextLink() == nil {
			break
		}

		builder = users.NewItemContactFoldersItemChildFoldersRequestBuilder(*resp.GetOdataNextLink(), service.Adapter())
	}

	return errs.ErrorOrNil()
}

// ---------------------------------------------------------------------------
// item pager
// ---------------------------------------------------------------------------

var _ itemPager = &contactPager{}

type contactPager struct {
	gs      graph.Servicer
	builder *users.ItemContactFoldersItemContactsDeltaRequestBuilder
	options *users.ItemContactFoldersItemContactsDeltaRequestBuilderGetRequestConfiguration
}

func (p *contactPager) getPage(ctx context.Context) (pageLinker, error) {
	return p.builder.Get(ctx, p.options)
}

func (p *contactPager) setNext(nextLink string) {
	p.builder = users.NewItemContactFoldersItemContactsDeltaRequestBuilder(nextLink, p.gs.Adapter())
}

func (p *contactPager) valuesIn(pl pageLinker) ([]getIDAndAddtler, error) {
	return toValues[models.Contactable](pl)
}

func (c Contacts) GetAddedAndRemovedItemIDs(
	ctx context.Context,
	user, directoryID, oldDelta string,
) ([]string, []string, DeltaUpdate, error) {
	service, err := c.service()
	if err != nil {
		return nil, nil, DeltaUpdate{}, err
	}

	var (
		errs       *multierror.Error
		resetDelta bool
	)

	options, err := optionsForContactFoldersItemDelta([]string{"parentFolderId"})
	if err != nil {
		return nil, nil, DeltaUpdate{}, errors.Wrap(err, "getting query options")
	}

	if len(oldDelta) > 0 {
		builder := users.NewItemContactFoldersItemContactsDeltaRequestBuilder(oldDelta, service.Adapter())
		pgr := &contactPager{service, builder, options}

		added, removed, deltaURL, err := getItemsAddedAndRemovedFromContainer(ctx, pgr)
		// note: happy path, not the error condition
		if err == nil {
			return added, removed, DeltaUpdate{deltaURL, false}, errs.ErrorOrNil()
		}
		// only return on error if it is NOT a delta issue.
		// on bad deltas we retry the call with the regular builder
		if graph.IsErrInvalidDelta(err) == nil {
			return nil, nil, DeltaUpdate{}, err
		}

		resetDelta = true
		errs = nil
	}

	builder := service.Client().UsersById(user).ContactFoldersById(directoryID).Contacts().Delta()
	pgr := &contactPager{service, builder, options}

	added, removed, deltaURL, err := getItemsAddedAndRemovedFromContainer(ctx, pgr)
	if err != nil {
		return nil, nil, DeltaUpdate{}, err
	}

	return added, removed, DeltaUpdate{deltaURL, resetDelta}, errs.ErrorOrNil()
}
