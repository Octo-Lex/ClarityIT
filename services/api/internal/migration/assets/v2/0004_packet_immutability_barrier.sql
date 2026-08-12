-- WP-01 forward revision 0004: packet immutability barrier.
-- Authority: WP01-AUTH-2026-08-12 / WP01-G0.
-- This is a storage-level K-01 guard only. The full legal packet transition
-- matrix remains WP01-G2. A packet that has ever left draft may never return to
-- draft, otherwise a later update could bypass the proposed-subject immutability
-- trigger by presenting OLD.state='draft'.
-- No live provider mutation, revision-ledger write, BEGIN/COMMIT, or psql meta.

SET LOCAL ROLE clarityit_owner;
SET LOCAL search_path = pg_catalog, public;

CREATE FUNCTION kernel.prevent_packet_return_to_draft() RETURNS trigger
LANGUAGE plpgsql AS $function$
BEGIN
    IF OLD.state <> 'draft' AND NEW.state = 'draft' THEN
        RAISE EXCEPTION 'operation packet cannot return to draft after proposal';
    END IF;
    RETURN NEW;
END;
$function$;

CREATE TRIGGER operation_packets_no_return_to_draft
BEFORE UPDATE ON kernel.operation_packets
FOR EACH ROW EXECUTE FUNCTION kernel.prevent_packet_return_to_draft();

REVOKE EXECUTE ON FUNCTION kernel.prevent_packet_return_to_draft() FROM PUBLIC, clarityit_app;
