package ml

var Humidities = map[string]*Humidity{
	"Bonsai big":   NewHumidity(1.472, 1.300),
	"Bonsai small": NewHumidity(1.646, 1.22),
}

type Humidity struct {
	v40  float64
	v100 float64
}

func (h Humidity) V40() float64 {
	return h.v40
}

func (h Humidity) V100() float64 {
	return h.v100
}

func NewHumidity(v40, v100 float64) *Humidity {
	return &Humidity{
		v40:  v40,
		v100: v100,
	}
}

func (h Humidity) voltageForPercentage(p float64) float64 {
	return h.v100 + (h.v40-h.v100)*((100-p)/60)
}

func (h Humidity) LowHumidity() float64 {
	return h.v40
}

func (h Humidity) HighHumidity() float64 {
	return h.voltageForPercentage(60)
}

func (h Humidity) NotWorkingFineLimit() float64 {
	return h.voltageForPercentage(15)
}

func (h Humidity) IsLow(v float64) bool {
	return v >= h.LowHumidity() && v <= h.NotWorkingFineLimit()
}

func (h Humidity) IsHigh(v float64) bool {
	return v <= h.HighHumidity()
}

func (h Humidity) InRange(v float64) bool {
	return v <= h.LowHumidity() && v >= h.HighHumidity()
}
