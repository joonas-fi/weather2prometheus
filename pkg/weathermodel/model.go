// shape-standardized version of what openweathermap provides
package weathermodel

import (
	"time"
)

type Observation struct {
	Timestamp time.Time `json:"timestamp"`

	Temperature      float64 `json:"temperature"`       // [°C]
	AirPressure      int     `json:"air_pressure"`      // [hPa]
	RelativeHumidity int     `json:"relative_humidity"` // [%]

	Wind WindSpec `json:"wind"`
}

type WindSpec struct {
	Speed     float64 `json:"speed"`     // [m/s]
	Direction int     `json:"direction"` // [°]
}
