package ml

type Prediction struct {
	zone             string
	shouldWater      bool
	predictedSeconds float64
	decisionReason   string
}

func (p Prediction) Zone() string {
	return p.zone
}

func (p Prediction) ShouldWater() bool {
	return p.shouldWater
}

func (p Prediction) PredictedSeconds() float64 {
	return p.predictedSeconds
}

func (p Prediction) DecisionReason() string {
	return p.decisionReason
}

func NewPrediction(
	zone string,
	shouldWater bool,
	predictedSeconds float64,
	decisionReason string,
) *Prediction {
	return &Prediction{
		zone:             zone,
		shouldWater:      shouldWater,
		predictedSeconds: predictedSeconds,
		decisionReason:   decisionReason,
	}
}
