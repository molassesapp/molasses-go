package molasses

import (
	"hash/crc32"
	"math"
	"strings"
)

type feature struct {
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
}

type environment struct {
	TeamID      string `json:"teamId"`
	Name        string `json:"name"`
	APIKeyValue string `json:"apiKey"`
	Features    []feature
}

type segmentType string

var (
	alwaysControl    segmentType = "alwaysControl"
	alwaysExperiment segmentType = "alwaysExperiment"
	everyoneElse     segmentType = "everyoneElse"
)

type operator string

var (
	all                  operator = "all"
	in                   operator = "in"
	nin                  operator = "nin"
	equals               operator = "equals"
	doesNotEqual         operator = "doesNotEqual"
	contains             operator = "contains"
	doesNotContain       operator = "doesNotContain"
	greaterThan          operator = "greaterThan"
	lessThan             operator = "lessThan"
	greaterThanOrEqualTo operator = "greaterThanOrEqualTo"
	lessThanOrEqualTo    operator = "lessThanOrEqualTo"
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
	Params map[string]string
}

func containsStr(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
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
	for _, constraint := range s.UserConstraints {
		userValue, paramExists := user.Params[constraint.UserParam]
		if constraint.UserParam == "id" {
			paramExists = true
			userValue = user.ID
		}
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
	}
	return false
}
