package sqldb

import (
	"database/sql"
	"encoding/json"
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

func (c FieldString) MarshalJSON() ([]byte, error) {
	if c.Valid {
		return json.Marshal(c.String)
	}

	return json.Marshal(nil)
}

func (c *FieldString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		c.String = ""
		c.Valid = false

		return nil
	}

	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	c.String = v
	c.Valid = true

	return nil
}

func (c *FieldString) SetString(v string) {
	common.Error(c.Scan(v))
}

func (c *FieldString) SetField(other FieldString) {
	c.String = other.String
	c.Valid = other.Valid
}

func (c *FieldString) SetNull() {
	c.String = ""
	c.Valid = false
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

func (c FieldInt64) MarshalJSON() ([]byte, error) {
	if c.Valid {
		return json.Marshal(c.Int64)
	}

	return json.Marshal(nil)
}

func (c *FieldInt64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		c.Int64 = 0
		c.Valid = false

		return nil
	}

	var v int64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	c.Int64 = v
	c.Valid = true

	return nil
}

func (c *FieldInt64) SetInt64(v int64) {
	common.Error(c.Scan(v))
}

func (c *FieldInt64) SetNull() {
	c.Int64 = 0
	c.Valid = false
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

func (c FieldTime) MarshalJSON() ([]byte, error) {
	if c.Valid {
		return json.Marshal(c.Time)
	}

	return json.Marshal(nil)
}

func (c *FieldTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		c.Time = time.Time{}
		c.Valid = false

		return nil
	}

	var v time.Time
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	c.Time = v
	c.Valid = true

	return nil
}

func (c *FieldTime) SetTime(v time.Time) {
	common.Error(c.Scan(v))
}

func (c *FieldTime) SetNull() {
	c.Time = time.Time{}
	c.Valid = false
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

func (c FieldBool) MarshalJSON() ([]byte, error) {
	if c.Valid {
		return json.Marshal(c.Bool)
	}

	return json.Marshal(nil)
}

func (c *FieldBool) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		c.Bool = false
		c.Valid = false

		return nil
	}

	var v bool
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	c.Bool = v
	c.Valid = true

	return nil
}

func (c *FieldBool) SetBool(v bool) {
	common.Error(c.Scan(v))
}

func (c *FieldBool) SetNull() {
	c.Bool = false
	c.Valid = false
}
