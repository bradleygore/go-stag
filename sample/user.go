package sample

import (
	"time"

	m "github.com/bradleygore/stag/sample/nested"
)

type User struct {
	m.Model
	FirstName     string     `json:"fName" db:"first_name"`
	LastName      string     `json:"lName" db:"last_name"`
	Age           uint16     `json:"age" db:"age_years"`
	DOB           *time.Time `json:"dob" db:"bday"`
	DBSkip        int64      `json:"dbSkip" db:"-"`
	JSONBlankName []float64  `json:",omitempty" db:"json_blank_name"`
}

type PowerUser struct {
	User
	SpecialPower string    `json:"superPower" db:"super_power"`
	JSONSkip     uint      `json:"-" db:"json_skip"`
	DBBlankName  []float64 `json:"dbBlanker" db:",bogus"`
}
