package ml

var Humidities = map[string]*Humidity{
	"Bonsai big":   NewHumidity(1.490, 1.378, 40, 65),
	"Bonsai small": NewHumidity(1.490, 1.378, 40, 65),
}

type Humidity struct {
	minHumidity, maxHumidity     float64
	minPercentage, maxPercentage float64
}

func (h Humidity) MinHumidity() float64 {
	return h.minHumidity
}

func (h Humidity) MaxHumidity() float64 {
	return h.maxHumidity
}

func (h Humidity) calculate(percentage float64) float64 {
	return h.minHumidity + (h.maxHumidity-h.minHumidity)*percentage/100
}

func (h Humidity) LowHumidity() float64 {
	return h.calculate(h.minPercentage)
}

func (h Humidity) HighHumidity() float64 {
	return h.calculate(h.maxPercentage)
}

func (h Humidity) IsLow(v float64) bool {
	return v > h.LowHumidity()
}

func (h Humidity) IsHigh(v float64) bool {
	return v < h.HighHumidity()
}

func NewHumidity(minHumidity, maxHumidity, minPercentage, maxPercentage float64) *Humidity {
	return &Humidity{
		minHumidity:   minHumidity,
		maxHumidity:   maxHumidity,
		minPercentage: minPercentage,
		maxPercentage: maxPercentage,
	}
}
