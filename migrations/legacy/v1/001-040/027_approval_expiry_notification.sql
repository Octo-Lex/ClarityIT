-- Migration 027: Approval expiry notification tracking
-- v1.1.0 Track 4: Approval Escalation and Expiry Notifications

-- Add column to track when expiry warning was first sent
-- Prevents repeated "expiring" notifications for the same approval
ALTER TABLE approval_requests
    ADD COLUMN IF NOT EXISTS expiring_notified_at TIMESTAMPTZ;
