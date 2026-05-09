package safety

import "testing"

func TestNewPlan(t *testing.T) {
	plan := NewPlan()
	if plan == nil || len(plan.PlannedMutations) != 0 {
		t.Fatalf("NewPlan() = %#v", plan)
	}
}

func TestPlanAdd(t *testing.T) {
	plan := NewPlan()
	plan.Add("update", "id-1", map[string]string{"before": "a"}, map[string]string{"after": "b"})

	if got, want := len(plan.PlannedMutations), 1; got != want {
		t.Fatalf("len(PlannedMutations) = %d, want %d", got, want)
	}
	mut := plan.PlannedMutations[0]
	if mut.Operation != "update" || mut.ResourceID != "id-1" {
		t.Fatalf("Add() mutation = %#v", mut)
	}
}

func TestCheckReadAndMutationBranches(t *testing.T) {
	tests := []struct {
		name      string
		tier      OperationTier
		readOnly  bool
		dryRun    bool
		confirmed bool
		wantErr   bool
	}{
		{"read allowed", TierRead, false, false, false, false},
		{"read only blocks mutation", TierMutation, true, false, true, true},
		{"dry run allows mutation", TierMutation, false, true, false, false},
		{"confirm required", TierMutation, false, false, false, true},
		{"confirmed mutation allowed", TierMutation, false, false, true, false},
		{"remote action confirm required", TierRemoteAction, false, false, false, true},
		{"destructive confirm required", TierDestructive, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Check(tt.tier, tt.readOnly, tt.dryRun, tt.confirmed)
			if tt.wantErr && err == nil {
				t.Fatal("Check() error = nil, want failure")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Check() error = %v, want nil", err)
			}
		})
	}
}
