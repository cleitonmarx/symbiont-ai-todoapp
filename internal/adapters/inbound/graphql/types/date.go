package types

import (
	"fmt"
	"io"
	"strconv"
	"time"
)

// Date is a custom GraphQL scalar type for date values (without time).
type Date time.Time

func (d *Date) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("Date must be a string")
	}
	parsed, err := time.Parse(time.DateOnly, str)
	if err != nil {
		return err
	}
	*d = Date(parsed)
	return nil
}

func (d Date) MarshalGQL(w io.Writer) {
	t := time.Time(d)
	fmt.Fprintf(w, `"%s"`, t.Format(time.DateOnly)) //nolint:errcheck
}

func (d *Date) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	return d.UnmarshalGQL(s)
}

func (d Date) MarshalJSON() ([]byte, error) {
	t := time.Time(d)
	return fmt.Appendf(nil, `"%s"`, t.Format(time.DateOnly)), nil
}
