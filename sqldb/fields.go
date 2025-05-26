package sqldb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/mpetavy/common"
	"time"
)

type FieldString struct {
	sql.NullString
}

func NewFieldString(v ...string) FieldString {
	if len(v) == 0 {
		return FieldString{}
	}

	o := FieldString{}
	common.Error(o.Scan(v[0]))

	return o
}

func (c FieldString) String() string {
	if c.NullString.Valid {
		return fmt.Sprintf("%v", c.NullString.String)
	}

	return ""
}

func (c FieldString) MarshalJSON() ([]byte, error) {
	if c.NullString.Valid {
		return json.Marshal(c.NullString.String)
	}

	return json.Marshal(nil)
}

func (c *FieldString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		c.NullString.String = ""
		c.NullString.Valid = false

		return nil
	}

	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	c.NullString.String = v
	c.NullString.Valid = true

	return nil
}

func (c *FieldString) GetString(v string) {
	common.Error(c.Scan(v))
}

func (c *FieldString) SetString(v string) {
	common.Error(c.Scan(v))
}

func (c *FieldString) SetField(other FieldString) {
	c.NullString.String = other.NullString.String
	c.NullString.Valid = other.Valid
}

func (c *FieldString) SetNull() {
	c.NullString.String = ""
	c.NullString.Valid = false
}

type FieldInt64 struct {
	sql.NullInt64
}

func NewFieldInt64(v ...int64) FieldInt64 {
	if len(v) == 0 {
		return FieldInt64{}
	}

	o := FieldInt64{}
	common.Error(o.Scan(v[0]))

	return o
}

func (c FieldInt64) String() string {
	if c.NullInt64.Valid {
		return fmt.Sprintf("%v", c.NullInt64.Int64)
	}

	return ""
}

func (c FieldInt64) MarshalJSON() ([]byte, error) {
	if c.NullInt64.Valid {
		return json.Marshal(c.NullInt64.Int64)
	}

	return json.Marshal(nil)
}

func (c *FieldInt64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		c.NullInt64.Int64 = 0
		c.NullInt64.Valid = false

		return nil
	}

	var v int64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	c.NullInt64.Int64 = v
	c.NullInt64.Valid = true

	return nil
}

func (c *FieldInt64) Int64() int64 {
	if c.NullInt64.Valid {
		return c.NullInt64.Int64
	}

	return 0
}

func (c *FieldInt64) SetInt64(v int64) {
	common.Error(c.Scan(v))
}

func (c *FieldInt64) SetNull() {
	c.NullInt64.Int64 = 0
	c.NullInt64.Valid = false
}

type FieldTime struct {
	sql.NullTime
}

func NewFieldTime(v ...time.Time) FieldTime {
	if len(v) == 0 {
		return FieldTime{}
	}

	o := FieldTime{}
	common.Error(o.Scan(v[0]))

	return o
}

func (c FieldTime) String() string {
	if c.NullTime.Valid {
		return c.NullTime.Time.Format(time.RFC3339)
	}

	return ""
}

func (c FieldTime) MarshalJSON() ([]byte, error) {
	if c.NullTime.Valid {
		return json.Marshal(c.NullTime.Time.UTC())
	}

	return json.Marshal(nil)
}

func (c *FieldTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		c.NullTime.Time = time.Time{}
		c.NullTime.Valid = false

		return nil
	}

	var v time.Time
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	c.NullTime.Time = v
	c.NullTime.Valid = true

	return nil
}

func (c *FieldTime) Time() time.Time {
	if c.NullTime.Valid {
		return c.NullTime.Time
	}

	return time.Time{}
}

func (c *FieldTime) SetTime(v time.Time) {
	common.Error(c.Scan(v))
}

func (c *FieldTime) SetNull() {
	c.NullTime.Time = time.Time{}
	c.NullTime.Valid = false
}

type FieldBool struct {
	sql.NullBool
}

func NewFieldBool(v ...bool) FieldBool {
	if len(v) == 0 {
		return FieldBool{}
	}

	o := FieldBool{}
	common.Error(o.Scan(v[0]))

	return o
}

func (c FieldBool) String() string {
	if c.NullBool.Valid {
		return fmt.Sprintf("%v", c.NullBool.Bool)
	}

	return ""
}

func (c FieldBool) MarshalJSON() ([]byte, error) {
	if c.NullBool.Valid {
		return json.Marshal(c.NullBool.Bool)
	}

	return json.Marshal(nil)
}

func (c *FieldBool) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		c.NullBool.Bool = false
		c.NullBool.Valid = false

		return nil
	}

	var v bool
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	c.NullBool.Bool = v
	c.NullBool.Valid = true

	return nil
}

func (c *FieldBool) SetBool(v bool) {
	common.Error(c.Scan(v))
}

func (c *FieldBool) SetNull() {
	c.NullBool.Bool = false
	c.NullBool.Valid = false
}
