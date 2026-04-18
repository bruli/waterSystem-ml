package ml

const (
	MinHumidity   = 1378
	MaxHumidity   = 1490
	MinPercentage = 40
	MaxPercentage = 65
)

func LowHumidity() float64 {
	return MinHumidity + (MaxHumidity-MinHumidity)*MinPercentage/100
}

func HighHumidity() float64 {
	return MinHumidity + (MaxHumidity-MinHumidity)*MaxPercentage/100
}
