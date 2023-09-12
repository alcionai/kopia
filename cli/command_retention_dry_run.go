package cli

import (
	"context"

	"github.com/kopia/kopia/repo"
	"github.com/kopia/kopia/repo/maintenance"
)

type commandRetentionDryRun struct{}

func (c *commandRetentionDryRun) setup(svc appServices, parent commandParent) {
	cmd := parent.Command("retention-dry-run", "Count blobs that will have retention extended")

	cmd.Action(svc.directRepositoryWriteAction(c.run))
}

func (c *commandRetentionDryRun) run(ctx context.Context, rep repo.DirectRepositoryWriter) error {

	_, err := maintenance.ExtendBlobRetentionTime(
		ctx,
		rep,
		maintenance.ExtendBlobRetentionTimeOptions{DryRun: true},
	)

	//nolint:wrapcheck
	return err
}
