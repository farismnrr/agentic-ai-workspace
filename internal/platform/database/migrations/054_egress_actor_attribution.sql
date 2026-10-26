-- Keep activity.user_id as the project owner: activity visibility depends on it.
-- Replace only these grant triggers; migration 053 stays immutable in the journal.
-- Detail includes trusted actor IDs/roles, never the user-supplied request reason.
DROP TRIGGER egress_request_activity;
DROP TRIGGER egress_decision_activity;

CREATE TRIGGER egress_request_activity AFTER INSERT ON egress_grants BEGIN
  INSERT INTO activity (id, user_id, space, client, tool, summary, outcome, detail, at, project_id, task_id)
  SELECT lower(hex(randomblob(16))), p.user_id, 'coding', NEW.id, 'egress_request',
    'HTTPS access requested: ' || NEW.host, 'ok',
    'requester=' || NEW.requested_by_kind || ':' || NEW.requested_by || '; owner=' || p.user_id,
    NEW.created_at, NEW.project_id, NEW.task_id
  FROM projects p WHERE p.id = NEW.project_id;
END;

CREATE TRIGGER egress_decision_activity AFTER UPDATE OF status ON egress_grants
WHEN OLD.status <> NEW.status BEGIN
  INSERT INTO activity (id, user_id, space, client, tool, summary, outcome, detail, at, project_id, task_id)
  SELECT lower(hex(randomblob(16))), p.user_id, 'coding', NEW.id,
    CASE NEW.status WHEN 'approved' THEN 'egress_approve'
      WHEN 'denied' THEN 'egress_deny' ELSE 'egress_revoke' END,
    'HTTPS access ' || NEW.status || ': ' || NEW.host, 'ok',
    'decider=' || CASE WHEN NEW.decided_by LIKE 'system:%' THEN 'system'
      WHEN NEW.decided_by = p.user_id THEN 'owner'
      WHEN EXISTS (SELECT 1 FROM users u WHERE u.id=NEW.decided_by AND u.role='admin') THEN 'admin'
      ELSE 'person' END || ':' || NEW.decided_by ||
    '; requester=' || NEW.requested_by_kind || ':' || NEW.requested_by || '; owner=' || p.user_id,
    NEW.decided_at, NEW.project_id, NEW.task_id
  FROM projects p WHERE p.id = NEW.project_id;
END;
