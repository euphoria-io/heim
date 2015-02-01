package snowflake

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/sdming/gosnow"
)

type Snowflaker interface {
	Next() (uint64, error)
}

var Clock = func() time.Time { return time.Now() }
var Epoch = time.Date(2014, 12, 0, 0, 0, 0, 0, time.UTC)
var DefaultSnowflaker Snowflaker

var SeqCounter uint64

const seqIDMask = (1 << gosnow.SequenceBits) - 1

func init() {
	gosnow.Since = Epoch.UnixNano() / 1000000
	var err error
	DefaultSnowflaker, err = gosnow.Default()
	if err != nil {
		panic(err)
	}
}

type Snowflake uint64

func New() (Snowflake, error) {
	snowflake, err := DefaultSnowflaker.Next()
	if err != nil {
		return Snowflake(0), err
	}
	return Snowflake(snowflake), nil
}

func NewFromTime(t time.Time) Snowflake {
	timestampMillis := (t.UnixNano() - Epoch.UnixNano()) / int64(time.Millisecond)
	workerID := gosnow.DefaultWorkId()
	seqID := atomic.AddUint64(&SeqCounter, 1)

	return Snowflake(
		(uint64(timestampMillis) << (gosnow.WorkerIdBits + gosnow.SequenceBits)) |
			(uint64(workerID) << gosnow.SequenceBits) |
			(seqID & seqIDMask))
}

func NewFromString(s string) (Snowflake, error) {
	snowflake, err := strconv.ParseUint(s, 36, 64)
	if err != nil {
		return Snowflake(0), err
	}
	return Snowflake(snowflake), nil
}

func (s Snowflake) String() string {
	if s == 0 {
		return ""
	}
	return fmt.Sprintf("%013s", strconv.FormatUint(uint64(s), 36))
}

func (s Snowflake) GoString() string { return fmt.Sprintf("%v", s.String()) }

func (s *Snowflake) FromString(str string) error {
	if str == "" {
		*s = 0
		return nil
	}

	i, err := strconv.ParseUint(str, 36, 64)
	if err != nil {
		return err
	}

	*s = Snowflake(i)
	return nil
}

func (s Snowflake) MarshalJSON() ([]byte, error) { return json.Marshal(s.String()) }

func (s *Snowflake) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	return s.FromString(str)
}

func (s Snowflake) Time() time.Time {
	timestampMillis := uint64(s) >> (gosnow.WorkerIdBits + gosnow.SequenceBits)
	return Epoch.Add(time.Duration(timestampMillis) * time.Millisecond)
}

func (s Snowflake) IsZero() bool                    { return s == 0 }
func (s Snowflake) Before(reference Snowflake) bool { return s < reference }
