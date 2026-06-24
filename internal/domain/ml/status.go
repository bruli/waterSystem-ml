package ml

import "context"

type Status struct {
	active  bool
	raining bool
}

func (s Status) Raining() bool {
	return s.raining
}

func (s Status) Active() bool {
	return s.active
}

func NewStatus(active, raining bool) *Status {
	return &Status{active: active, raining: raining}
}

type SystemStatus struct {
	repository StatusRepository
}

func (s SystemStatus) GetStatus(ctx context.Context) (*Status, error) {
	return s.repository.GetStatus(ctx)
}

func NewSystemStatus(repository StatusRepository) *SystemStatus {
	return &SystemStatus{repository: repository}
}
