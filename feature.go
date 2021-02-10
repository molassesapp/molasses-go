package molasses

import (
	"errors"
	"fmt"
	"hash/crc32"
	"math"
	"strconv"
	"strings"
)

type feature struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Description string `json:"description"`
	Version     string `json:"version"`
	// Variants []Variant        `json:"variants"`
	Active   bool             `json:"active"`
	Segments []featureSegment `json:"segments"`
}

type userConstraint struct {
	Operator      operator `json:"operator"`
	Values        string   `json:"values"`
	UserParam     string   `json:"userParam"`
	UserParamType string   `json:"userParamType"`
}

type featureSegment struct {
	SegmentType     segmentType      `json:"segmentType"`
	UserConstraints []userConstraint `json:"userConstraints"`
	Percentage      int              `json:"percentage"`
	Constraint      operator         `json:"constraint"`
}

type segmentType string

var (
	alwaysControl    segmentType = "alwaysControl"
	alwaysExperiment segmentType = "alwaysExperiment"
	everyoneElse     segmentType = "everyoneElse"
)

type operator string

var (
	any            operator = "any"
	in             operator = "in"
	nin            operator = "nin"
	equals         operator = "equals"
	gte            operator = "gte"
	gt             operator = "gt"
	lt             operator = "lt"
	lte            operator = "lte"
	doesNotEqual   operator = "doesNotEqual"
	contains       operator = "contains"
	doesNotContain operator = "doesNotContain"
)

func containsParamValue(listAsString string, a string) bool {
	list := strings.Split(listAsString, ",")
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// User - The representation of your user
type User struct {
	ID     string
	Params map[string]interface{}
}

func isActive(f feature, user *User) bool {
	if !f.Active {
		return false
	}
	// if there is no user just return true
	if user == nil {
		return true
	}

	// Build a config map:
	segmentMap := map[string]featureSegment{}
	for _, s := range f.Segments {
		switch s.SegmentType {
		case alwaysControl:

			segmentMap["alwaysControl"] = s
			continue
		case alwaysExperiment:
			segmentMap["alwaysExperiment"] = s
			continue
		case everyoneElse:
			segmentMap["everyoneElse"] = s
			continue
		}
	}
	// check if they should have the control always
	if alwaysControlSegment, ok := segmentMap["alwaysControl"]; ok && isUserInSegment(*user, alwaysControlSegment) {
		return false
	}
	// check if they should have the experiment always
	if alwaysExperimentSegment, ok := segmentMap["alwaysExperiment"]; ok && isUserInSegment(*user, alwaysExperimentSegment) {
		return true
	}

	return getUserPercentage(*user, segmentMap["everyoneElse"])

}

func getUserPercentage(user User, segment featureSegment) bool {
	if segment.Percentage == 100 {
		return true
	}

	c := float64(crc32.ChecksumIEEE([]byte(user.ID)))
	v := math.Abs(math.Mod(c, 100.0))

	return v < float64(segment.Percentage)
}

func isUserInSegment(user User, s featureSegment) bool {
	constraintsToBeMet := len(s.UserConstraints)
	if s.Constraint == any {
		constraintsToBeMet = 1
	}
	constraintsMet := 0
	for i := 0; i < len(s.UserConstraints); i++ {
		constraint := s.UserConstraints[i]
		userValue, paramExists := user.Params[constraint.UserParam]
		if constraint.UserParam == "id" {
			paramExists = true
			userValue = user.ID
		}
		switch constraint.UserParamType {
		case "number":
			v, err := getFloat64Value(userValue)
			if err != nil {
				continue
			}
			if meetsConstraintForNumber(v, paramExists, constraint) {
				constraintsMet = constraintsMet + 1
			}
		case "bool":
			v, err := getBoolValue(userValue)
			if err != nil {
				continue
			}
			if meetsConstraintForBool(v, paramExists, constraint) {
				constraintsMet = constraintsMet + 1
			}
		default:
			v, err := getStringValue(userValue)
			if err != nil {
				continue
			}
			if meetsConstraintForString(v, paramExists, constraint) {
				constraintsMet = constraintsMet + 1
			}

		}

	}
	return constraintsMet >= constraintsToBeMet
}

func getFloat64Value(value interface{}) (float64, error) {
	switch v := value.(type) {
	case bool:
		return 0.0, errors.New("not valid value")
	case string:
		return strconv.ParseFloat(v, 10)
	case uint:
	case int:
		return float64(v), nil
	case float64:
		return v, nil
	}
	return 0.0, errors.New("not valid value")
}

func getBoolValue(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	case int:
	case float64:
		return v > 0.0, nil
	}
	return false, errors.New("not valid value")
}

func getStringValue(value interface{}) (string, error) {
	switch v := value.(type) {
	case bool:
		return strconv.FormatBool(v), nil
	case string:
		return v, nil
	case int:
	case float64:
		return fmt.Sprintf("%v", v), nil
	}
	return "", errors.New("not valid value")
}

func meetsConstraintForBool(userValue bool, paramExists bool, constraint userConstraint) bool {
	values, err := strconv.ParseBool(constraint.Values)
	if err != nil {
		return false
	}
	switch constraint.Operator {
	case equals:
		if paramExists && userValue == values {
			return true
		}
	case doesNotEqual:
		if paramExists && userValue != values {
			return true
		}
	default:
		return false
	}
	return false
}

func meetsConstraintForNumber(userValue float64, paramExists bool, constraint userConstraint) bool {
	values, err := strconv.ParseFloat(constraint.Values, 10)
	if err != nil {
		return false
	}
	switch constraint.Operator {
	case equals:
		if paramExists && userValue == values {
			return true
		}
	case doesNotEqual:
		if paramExists && userValue != values {
			return true
		}
	case gt:
		if paramExists && userValue > values {
			return true
		}
	case lt:
		if paramExists && userValue < values {
			return true
		}
	case gte:
		if paramExists && userValue >= values {
			return true
		}
	case lte:
		if paramExists && userValue <= values {
			return true
		}
	default:
		return false
	}
	return false
}

func meetsConstraintForString(userValue string, paramExists bool, constraint userConstraint) bool {
	switch constraint.Operator {
	case in:
		if paramExists && containsParamValue(constraint.Values, userValue) {
			return true
		}
	case nin:
		if paramExists && !containsParamValue(constraint.Values, userValue) {
			return true
		}
	case equals:
		if paramExists && userValue == constraint.Values {
			return true
		}
	case doesNotEqual:
		if paramExists && userValue != constraint.Values {
			return true
		}
	case contains:
		if paramExists && strings.Contains(userValue, constraint.Values) {
			return true
		}
	case doesNotContain:
		if paramExists && !strings.Contains(userValue, constraint.Values) {
			return true
		}
	default:
		return false
	}
	return false
}
