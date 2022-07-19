package store

import (
	"context"

	"github.com/kopia/kopia/repo/manifest"
	"github.com/pkg/errors"

	"github.com/alcionai/corso/internal/model"
	"github.com/alcionai/corso/pkg/backup"
)

// GetBackup gets a single backup by id.
func (w Wrapper) GetBackup(ctx context.Context, backupID model.ID) (*backup.Backup, error) {
	b := backup.Backup{}
	err := w.Get(ctx, model.BackupSchema, backupID, &b)
	if err != nil {
		return nil, errors.Wrap(err, "getting backup")
	}
	return &b, nil
}

// GetDetailsFromBackupID retrieves all backups in the model store.
func (w Wrapper) GetBackups(ctx context.Context) ([]backup.Backup, error) {
	bms, err := w.GetIDsForType(ctx, model.BackupSchema, nil)
	if err != nil {
		return nil, err
	}
	bs := make([]backup.Backup, len(bms))
	for i, bm := range bms {
		b := backup.Backup{}
		err := w.GetWithModelStoreID(ctx, model.BackupSchema, bm.ModelStoreID, &b)
		if err != nil {
			return nil, err
		}
		bs[i] = b
	}
	return bs, nil
}

// GetDetails gets the backup details by ID.
func (w Wrapper) GetDetails(ctx context.Context, detailsID manifest.ID) (*backup.Details, error) {
	d := backup.Details{}
	err := w.GetWithModelStoreID(ctx, model.BackupDetailsSchema, detailsID, &d)
	if err != nil {
		return nil, errors.Wrap(err, "getting details")
	}
	return &d, nil
}

// GetDetailsFromBackupID retrieves the backup.Details within the specified backup.
func (w Wrapper) GetDetailsFromBackupID(ctx context.Context, backupID model.ID) (*backup.Details, *backup.Backup, error) {
	b, err := w.GetBackup(ctx, backupID)
	if err != nil {
		return nil, nil, err
	}

	d, err := w.GetDetails(ctx, manifest.ID(b.DetailsID))
	if err != nil {
		return nil, nil, err
	}

	return d, b, nil
}