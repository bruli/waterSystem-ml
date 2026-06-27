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
	calculatedZones map[string]bool
	zonesHumidity   map[string]*ZoneHumidity
	timeFunc        func() time.Time
}

func (c *CalculatedWatering) Calculated() bool {
	for _, calc := range c.calculatedZones {
		if !calc {
			return false
		}
	}
	return true
}

func (c *CalculatedWatering) PendingPredictionZones() []string {
	zones := make([]string, 0)
	for zone, calculated := range c.calculatedZones {
		if !calculated {
			zones = append(zones, zone)
		}
	}
	return zones
}

func (c *CalculatedWatering) markAllCalculated() {
	for zone := range c.calculatedZones {
		c.calculatedZones[zone] = true
	}
}

func (c *CalculatedWatering) allowedFromSystem() bool {
	switch {
	case c.isRaining:
		c.Record(NewWateringSystemSkippedEvent(RainingReason))
		c.markAllCalculated()
		return false
	case !c.systemActivated:
		c.Record(NewWateringSystemSkippedEvent(SystemDisabledReason))
		c.markAllCalculated()
		return false
	case c.isNightRange():
		c.Record(NewWateringSystemSkippedEvent(IsNightRangeReason))
		c.markAllCalculated()
		return false
	default:
		return true
	}
}

func (c *CalculatedWatering) allowedFromZone(zone string, currentHumidity float64) (bool, error) {
	ex, ok := c.executions[zone]
	if !ok {
		return false, ErrUnknownZone
	}
	if ex.IsRecentlyExecuted() {
		c.Record(NewWateringZoneSkippedEvent(zone, ZoneRecentlyExecutedByModelReason, currentHumidity, nil, nil, nil))
		c.calculatedZones[zone] = true
		return false, nil
	}
	return true, nil
}

func (c *CalculatedWatering) isNightRange() bool {
	hour := c.timeFunc().Hour()
	return hour > 22 || hour <= 8
}

func (c *CalculatedWatering) FromPrediction(pred *Prediction, zh *ZoneHumidity) {
	if pred == nil {
		return
	}
	if c.calculatedZones[pred.Zone()] {
		return
	}
	if zh == nil {
		return
	}

	c.calculatedZones[pred.Zone()] = true

	predictionID := pred.ID()
	decisionReason := pred.DecisionReason()
	wateringProba := pred.WateringProba()

	if pred.shouldWater {
		c.Record(NewWateringRequestedEvent(
			pred.Zone(),
			ModelPredictionReason,
			pred.PredictedSeconds(),
			zh.CurrentHumidity(),
			zh.HumidityReference().TargetMoisture(),
			&predictionID,
			&decisionReason,
			&wateringProba,
		))
		return
	}

	c.Record(NewWateringZoneSkippedEvent(
		pred.Zone(),
		ModelNotEstimatedReason,
		zh.CurrentHumidity(),
		&predictionID,
		&decisionReason,
		&wateringProba,
	))
}

func NewCalculatedWatering(
	isRaining bool,
	systemActivated bool,
	timeFunc func() time.Time,
	exec Executions,
	zonesHumidity []*ZoneHumidity,
	pendingPredictionLogsZones map[string]bool,
) (*CalculatedWatering, error) {
	calcZones := make(map[string]bool, len(zonesHumidity))
	zoneHumMap := make(map[string]*ZoneHumidity, len(zonesHumidity))
	for _, zh := range zonesHumidity {
		calcZones[zh.Zone()] = false
		zoneHumMap[zh.Zone()] = zh
	}

	cw := CalculatedWatering{
		isRaining:       isRaining,
		systemActivated: systemActivated,
		executions:      exec,
		timeFunc:        timeFunc,
		calculatedZones: calcZones,
		zonesHumidity:   zoneHumMap,
	}

	if !cw.allowedFromSystem() {
		return &cw, nil
	}

	for _, zh := range zonesHumidity {
		zone := zh.Zone()
		currentHumidity := zh.CurrentHumidity()
		if _, ok := pendingPredictionLogsZones[zone]; ok {
			cw.Record(NewWateringZoneSkippedEvent(
				zh.Zone(),
				PredictionPendingValidationReason,
				zh.CurrentHumidity(),
				nil,
				nil,
				nil,
			))
			cw.calculatedZones[zh.Zone()] = true
			continue
		}
		allowed, err := cw.allowedFromZone(zone, currentHumidity)
		if err != nil {
			return nil, err
		}
		if !allowed {
			continue
		}

		switch {
		case zh.HumidityReference().IsHigh(currentHumidity):
			cw.Record(NewWateringZoneSkippedEvent(zone, AboveMaxThresholdReason, currentHumidity, nil, nil, nil))
			cw.calculatedZones[zone] = true
		case zh.HumidityReference().IsLow(currentHumidity):
			cw.Record(NewWateringRequestedEvent(
				zone,
				BelowMinThresholdReason,
				DefaultSecondsOnLowHumidity,
				currentHumidity,
				zh.HumidityReference().TargetMoisture(),
				nil,
				nil,
				nil,
			))
			cw.calculatedZones[zone] = true
		default:
		}
	}

	return &cw, nil
}
