package ml

import (
	"errors"
	"time"

	"github.com/bruli/go-core/event"
)

const (
	SystemDisabledReason              = "system_disabled"
	RainingReason                     = "raining"
	AboveMaxThresholdReason           = "above_max_threshold"
	BelowMinThresholdReason           = "below_min_threshold"
	ModelPredictionReason             = "model_prediction"
	ModelNotEstimatedReason           = "model_not_estimated"
	ZoneRecentlyExecutedByModelReason = "zone_recently_executed_by_model"
	IsNightRangeReason                = "is_night_range"

	DefaultSecondsOnLowHumidity = 20
)

var ErrUnknownZone = errors.New("unknown zone")

type CalculatedWatering struct {
	event.BasicAggregateRoot
	isRaining       bool
	systemActivated bool
	executions      Executions
	calculated      bool
	timeFunc        func() time.Time
}

func (c *CalculatedWatering) Calculated() bool {
	return c.calculated
}

func (c *CalculatedWatering) allowedFromSystem(zone string, currentHumidity float64) (bool, error) {
	switch {
	case c.isRaining:
		c.Record(NewWateringSystemSkippedEvent(RainingReason))
		return false, nil
	case !c.systemActivated:
		c.Record(NewWateringSystemSkippedEvent(SystemDisabledReason))
		return false, nil
	}
	ex, ok := c.executions[zone]
	if !ok {
		return false, ErrUnknownZone
	}
	if ex.IsRecentlyExecuted() {
		c.Record(NewWateringZoneSkippedEvent(zone, ZoneRecentlyExecutedByModelReason, currentHumidity, nil, nil, nil))
		return false, nil
	}
	return true, nil
}

func (c *CalculatedWatering) FromPrediction(pred *Prediction, zh *ZoneHumidity) {
	now := c.timeFunc().Hour()
	if now > 22 || now <= 8 {
		c.Record(NewWateringSystemSkippedEvent(IsNightRangeReason))
		return
	}
	switch {
	case pred.shouldWater:
		c.Record(NewWateringRequestedEvent(
			pred.Zone(),
			ModelPredictionReason,
			pred.PredictedSeconds(),
			zh.CurrentHumidity(),
			zh.HumidityReference().TargetMoisture(),
			new(pred.ID()),
			new(pred.DecisionReason()),
			new(pred.WateringProba()),
		))
		return
	default:
	}
	c.Record(NewWateringZoneSkippedEvent(pred.Zone(), ModelNotEstimatedReason, zh.CurrentHumidity(), new(pred.ID()), new(pred.DecisionReason()), new(pred.WateringProba())))
}

func NewCalculatedWatering(
	isRaining bool,
	systemActivated bool,
	timeFunc func() time.Time,
	exec Executions,
	zonesHumidity []*ZoneHumidity,
) (*CalculatedWatering, error) {
	cp := CalculatedWatering{
		isRaining:       isRaining,
		systemActivated: systemActivated,
		executions:      exec,
		timeFunc:        timeFunc,
	}
	for _, zh := range zonesHumidity {
		switch {
		case zh.HumidityReference().IsHigh(zh.CurrentHumidity()):
			cp.Record(NewWateringZoneSkippedEvent(zh.Zone(), AboveMaxThresholdReason, zh.CurrentHumidity(), nil, nil, nil))
			cp.calculated = true
		case zh.HumidityReference().IsLow(zh.CurrentHumidity()):
			cp.calculated = true
			allowed, err := cp.allowedFromSystem(zh.Zone(), zh.CurrentHumidity())
			if err != nil {
				return nil, err
			}
			if allowed {
				cp.Record(NewWateringRequestedEvent(
					zh.Zone(),
					BelowMinThresholdReason,
					DefaultSecondsOnLowHumidity,
					zh.CurrentHumidity(),
					zh.HumidityReference().TargetMoisture(),
					nil,
					nil,
					nil,
				))
			}
		default:
		}
	}
	return &cp, nil
}
