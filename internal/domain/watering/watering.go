package watering

type Watering struct {
	zone    string
	seconds int
}

func (w Watering) Zone() string {
	return w.zone
}

func (w Watering) Seconds() int {
	return w.seconds
}

func New(zone string, seconds int) *Watering {
	return &Watering{zone: zone, seconds: seconds}
}
