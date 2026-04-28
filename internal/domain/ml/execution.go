package ml

import "time"

const WateringCoolDown = time.Hour * 2

type Executions map[string]Execution

type Execution struct {
	zone          string
	executionTime time.Time
}

func (e Execution) Zone() string {
	return e.zone
}

func (e Execution) ExecutionTime() time.Time {
	return e.executionTime
}

func (e Execution) IsRecentlyExecuted() bool {
	return time.Since(e.executionTime) < WateringCoolDown
}

func NewExecution(zone string, executionTime time.Time) *Execution {
	return &Execution{zone: zone, executionTime: executionTime}
}
