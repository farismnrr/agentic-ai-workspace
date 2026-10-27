package database

import (
	"context"
	"testing"
)

func TestPlanApprovalMigrationExtendsAnExistingPlanDatabase(t *testing.T) {
	db := rawDatabase(t)
	ctx := context.Background()
	applyBefore(t, db, "060_plan_approvals.sql")
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON;
		INSERT INTO users(id,name,email,password_hash,created_at) VALUES ('u','User','u@example.test','hash',1);
		INSERT INTO projects(id,user_id,name,created_at) VALUES ('p','u','Project',1);
		INSERT INTO tasks(id,project_id,user_id,title,status,mode,provider_id,model,created_at,updated_at,source,work_id)
		VALUES ('t','p','u','External task','needs-input','risky','','',2,2,'ChatGPT','w');
		INSERT INTO task_events(task_id,seq,id,at,kind,body) VALUES ('t',1,'event-1',2,'plan_review','{}');
		INSERT INTO external_works(id,task_id,user_id,project_id,client,state,created_at,updated_at,plan_state)
		VALUES ('w','t','u','p','ChatGPT','completed',2,2,'approved');
		INSERT INTO external_plan_revisions(id,work_id,task_id,rev_no,plan_rev,steps,prd,research,qa,small_task,
		 summary,digest,event_id,status,created_at)
		VALUES ('r','w','t',1,1,'[]','requirements','research','answers',0,'summary','digest',
		'event-1','approved',2);`); err != nil {
		t.Fatal(err)
	}
	before := journal(t, db)
	if err := migrate(ctx, db); err != nil {
		t.Fatalf("migrate existing submitted-plan database: %v", err)
	}
	after := journal(t, db)
	for name, checksum := range before {
		if after[name] != checksum {
			t.Errorf("migration changed earlier journal entry %s", name)
		}
	}
	if after["060_plan_approvals.sql"] == "" {
		t.Fatal("approval migration was not recorded")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO external_plan_decisions
		(id,revision_id,work_id,task_id,user_id,project_id,decision,source,client,idem_key,req_hash,note,decided_at)
		VALUES ('d','r','w','t','u','p','approved','app','','approve-key','hash','',3)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO external_plan_handoffs
		(id,decision_id,work_id,state,next_attempt_at) VALUES ('h','d','w','pending',3)`); err != nil {
		t.Fatal(err)
	}
}
