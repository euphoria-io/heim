package proto

import (
	"encoding/json"
	"strconv"
	"time"

	"euphoria.io/heim/proto/snowflake"
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

func (t *Time) StdTime() time.Time { return time.Time(*t) }

func Now() Time { return Time(snowflake.Clock().UTC()) }
