package dates

import (
	"time"
	"math"
	"fmt"
)

func int64DateParser(unixTime int64, tz *time.Location) (time.Time, string, error) {
	numDigits := int(math.Log10(float64(unixTime))) + 1
	precision := ""
	var parsedTime time.Time
	switch {
	case numDigits <= 5: // Excel timestamps
		parsedTime, precision, err := float64DateParser(float64(unixTime), tz)
		if err != nil {
			return time.Time{}, "", err
		}
		return parsedTime, precision, nil
	case numDigits > 5 && numDigits <= 10: // Unix timestamps with 5 to 10 digits
		parsedTime = time.Unix(unixTime, 0)
		precision = SecondPrecision
	case numDigits == 13: // milliseconds
		parsedTime = time.Unix(0, unixTime*int64(time.Millisecond))
		precision = MillisecondPrecision
	case numDigits == 16: // microseconds
		parsedTime = time.Unix(0, unixTime*int64(time.Microsecond))
		precision = MicrosecondPrecision
	case numDigits == 19: // nanoseconds
		parsedTime = time.Unix(0, unixTime)
		precision = NanosecondPrecision
	default:
		return time.Time{}, "", fmt.Errorf("unknown Unix timestamp length: %d digits", numDigits)
	}
	return parsedTime, precision, nil
}