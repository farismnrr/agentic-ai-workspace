-- Keep an immutable action trail in the existing activity log in the SAME SQLite
-- transaction as each grant create/decision. No user-provided reason or credential is logged.
CREATE TRIGGER egress_request_activity AFTER INSERT ON egress_grants BEGIN
  INSERT INTO activity (id, user_id, space, client, tool, summary, outcome, detail, at, project_id, task_id)
  SELECT lower(hex(randomblob(16))), p.user_id, 'coding', NEW.id, 'egress_request',
    'HTTPS access requested: ' || NEW.host, 'ok', NEW.requested_by_kind,
    NEW.created_at, NEW.project_id, NEW.task_id
  FROM projects p WHERE p.id = NEW.project_id;
END;

CREATE TRIGGER egress_decision_activity AFTER UPDATE OF status ON egress_grants
WHEN OLD.status <> NEW.status BEGIN
  INSERT INTO activity (id, user_id, space, client, tool, summary, outcome, detail, at, project_id, task_id)
  SELECT lower(hex(randomblob(16))), p.user_id, 'coding', NEW.id,
    CASE NEW.status WHEN 'approved' THEN 'egress_approve'
      WHEN 'denied' THEN 'egress_deny' ELSE 'egress_revoke' END,
    'HTTPS access ' || NEW.status || ': ' || NEW.host, 'ok', NEW.decided_by,
    NEW.decided_at, NEW.project_id, NEW.task_id
  FROM projects p WHERE p.id = NEW.project_id;
END;

CREATE INDEX egress_grants_expiration_idx ON egress_grants (expires_at, status);
