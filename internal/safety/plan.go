package safety

type Plan struct {
	PlannedMutations []PlannedMutation `json:"planned_mutations"`
}

type PlannedMutation struct {
	Operation  string      `json:"operation"`
	ResourceID string      `json:"resource_id,omitempty"`
	Before     interface{} `json:"before,omitempty"`
	After      interface{} `json:"after,omitempty"`
}

func NewPlan() *Plan {
	return &Plan{
		PlannedMutations: make([]PlannedMutation, 0),
	}
}

func (p *Plan) Add(op, id string, before, after interface{}) {
	p.PlannedMutations = append(p.PlannedMutations, PlannedMutation{
		Operation:  op,
		ResourceID: id,
		Before:     before,
		After:      after,
	})
}
