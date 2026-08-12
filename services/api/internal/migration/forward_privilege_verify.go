package migration

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// verifyForwardRuntimePrivileges protects the G1 message-envelope privilege
// boundary that is expressed with PostgreSQL column ACLs in revision 0005.
// Relation/schema/function/default ACLs remain part of the frozen target
// manifest; these column-level grants are verified explicitly so a narrowed
// runtime privilege cannot silently drift while the relation ACL is unchanged.
func verifyForwardRuntimePrivileges(ctx context.Context, conn *pgx.Conn) error {
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE clarityit_owner`); err != nil {
		return fmt.Errorf("forward privilege verification boundary: %w", err)
	}

	var (
		outboxPublishedUpdate bool
		outboxAttemptsUpdate  bool
		outboxPayloadUpdate   bool
		inboxProcessedUpdate  bool
		inboxResultUpdate     bool
		inboxPayloadUpdate    bool
		inboxDelete           bool
	)
	if err := tx.QueryRow(ctx, `SELECT
		has_column_privilege('clarityit_app','kernel.outbox_messages','published_at','UPDATE'),
		has_column_privilege('clarityit_app','kernel.outbox_messages','publish_attempts','UPDATE'),
		has_column_privilege('clarityit_app','kernel.outbox_messages','payload_digest','UPDATE'),
		has_column_privilege('clarityit_app','kernel.inbox_messages','processed_at','UPDATE'),
		has_column_privilege('clarityit_app','kernel.inbox_messages','result','UPDATE'),
		has_column_privilege('clarityit_app','kernel.inbox_messages','payload_digest','UPDATE'),
		has_table_privilege('clarityit_app','kernel.inbox_messages','DELETE')`).Scan(
		&outboxPublishedUpdate,
		&outboxAttemptsUpdate,
		&outboxPayloadUpdate,
		&inboxProcessedUpdate,
		&inboxResultUpdate,
		&inboxPayloadUpdate,
		&inboxDelete,
	); err != nil {
		return fmt.Errorf("%w: message privilege query: %v", ErrForwardManifest, err)
	}

	if !outboxPublishedUpdate || !outboxAttemptsUpdate || outboxPayloadUpdate ||
		!inboxProcessedUpdate || !inboxResultUpdate || inboxPayloadUpdate || inboxDelete {
		return fmt.Errorf(
			"%w: message column privilege posture mismatch",
			ErrForwardManifest,
		)
	}
	return nil
}
