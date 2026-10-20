package application_test

import "testing"

func TestATaskOnAPlanShowsHowFullTheProgramsWindowIs(t *testing.T) {
	t.Parallel()

	program := &scriptedProgram{say: "Done.", used: 40_000, window: 200_000}
	r := newPlanLeadRig(program)
	task := r.startOnPlan(t, "Fix the button")
	r.run(task.ID)

	shown := r.task(t, task.ID)
	if shown.ContextUsed != 40_000 || shown.ContextWindow != 200_000 {
		t.Fatalf("meter = %d of %d, want 40000 of 200000", shown.ContextUsed, shown.ContextWindow)
	}
}

func TestAProgramThatReportsNoWindowGetsTheModelsKnownOne(t *testing.T) {
	t.Parallel()

	program := &scriptedProgram{say: "Done.", used: 40_000}
	r := newPlanLeadRig(program)
	r.windows.tokens = 272_000
	task := r.startOnPlan(t, "Fix the button")
	r.run(task.ID)

	if shown := r.task(t, task.ID); shown.ContextWindow != 272_000 {
		t.Fatalf("window = %d, want the model's own", shown.ContextWindow)
	}
}
