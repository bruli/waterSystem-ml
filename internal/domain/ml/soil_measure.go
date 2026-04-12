package ml

type SoilMeasure struct {
	zone     string
	humidity float64
}

func (s SoilMeasure) Zone() string {
	return s.zone
}

func (s SoilMeasure) Humidity() float64 {
	return s.humidity
}

func NewSoilMeasure(zone string, humidity float64) *SoilMeasure {
	return &SoilMeasure{zone: zone, humidity: humidity}
}
