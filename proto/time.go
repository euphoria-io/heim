package proto

import (
	"encoding/json"
	"strconv"
	"time"
)

type Time time.Time

func (t Time) MarshalJSON() ([]byte, error) {
	if time.Time(t).IsZero() {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(time.Time(t).Unix(), 10)), nil
}

func (t *Time) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*t = Time{}
		return nil
	}
	var unix int64
	if err := json.Unmarshal(data, &unix); err != nil {
		return err
	}
	*t = Time(time.Unix(unix, 0).UTC())
	return nil
}

func Now() Time { return Time(time.Now().UTC()) }
