package safety

type Plan struct {
	PlannedMutations []PlannedMutation `json:"planned_mutations"`
}

type PlannedMutation struct {
	Operation  string `json:"operation"`
	ResourceID string `json:"resource_id,omitempty"`
	Before     any    `json:"before,omitempty"`
	After      any    `json:"after,omitempty"`
}

func NewPlan() *Plan {
	return &Plan{
		PlannedMutations: make([]PlannedMutation, 0),
	}
}

func (p *Plan) Add(op, id string, before, after any) {
	p.PlannedMutations = append(p.PlannedMutations, PlannedMutation{
		Operation:  op,
		ResourceID: id,
		Before:     before,
		After:      after,
	})
}
