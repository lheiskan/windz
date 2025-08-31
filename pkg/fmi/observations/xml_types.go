package observations

import (
	"encoding/xml"
)

// FeatureCollection represents the root element of the FMI WFS XML response
type FeatureCollection struct {
	XMLName        xml.Name        `xml:"FeatureCollection"`
	NumberMatched  string          `xml:"numberMatched,attr"`
	NumberReturned string          `xml:"numberReturned,attr"`
	TimeStamp      string          `xml:"timeStamp,attr"`
	Members        []FeatureMember `xml:"member"`
}

// FeatureMember contains the observation data
type FeatureMember struct {
	GridSeriesObservation GridSeriesObservation `xml:"GridSeriesObservation"`
}

// GridSeriesObservation represents a grid series observation
type GridSeriesObservation struct {
	XMLName                xml.Name               `xml:"GridSeriesObservation"`
	GmlID                  string                 `xml:"id,attr"`
	PhenomenonTime         PhenomenonTime         `xml:"phenomenonTime"`
	ResultTime             ResultTime             `xml:"resultTime"`
	Procedure              Procedure              `xml:"procedure"`
	ObservedProperty       ObservedProperty       `xml:"observedProperty"`
	SpatialSamplingFeature SpatialSamplingFeature `xml:"featureOfInterest>SF_SpatialSamplingFeature"`
	Result                 Result                 `xml:"result"`
}

// PhenomenonTime contains the time period of observations
type PhenomenonTime struct {
	TimePeriod TimePeriod `xml:"TimePeriod"`
}

// TimePeriod represents a time period
type TimePeriod struct {
	GmlID         string `xml:"id,attr"`
	BeginPosition string `xml:"beginPosition"`
	EndPosition   string `xml:"endPosition"`
}

// ResultTime contains the result time
type ResultTime struct {
	TimeInstant TimeInstant `xml:"TimeInstant"`
}

// TimeInstant represents a time instant
type TimeInstant struct {
	GmlID        string `xml:"id,attr"`
	TimePosition string `xml:"timePosition"`
}

// Procedure contains procedure information
type Procedure struct {
	Href string `xml:"href,attr"`
}

// ObservedProperty contains observed property information
type ObservedProperty struct {
	Href string `xml:"href,attr"`
}

// SpatialSamplingFeature contains location and sampling information
type SpatialSamplingFeature struct {
	GmlID          string             `xml:"id,attr"`
	SampledFeature LocationCollection `xml:"sampledFeature>LocationCollection"`
	Shape          Shape              `xml:"shape"`
}

// LocationCollection contains multiple station locations
type LocationCollection struct {
	Members []LocationMember `xml:"member"`
}

// LocationMember wraps a Location
type LocationMember struct {
	Location Location `xml:"Location"`
}

// Location contains station location information
type Location struct {
	GmlID      string     `xml:"id,attr"`
	Identifier Identifier `xml:"identifier"`
	Names      []Name     `xml:"name"`
	Region     string     `xml:"region"`
}

// Identifier contains station identifier
type Identifier struct {
	CodeSpace string `xml:"codeSpace,attr"`
	Value     string `xml:",chardata"`
}

// Name contains station name with code space
type Name struct {
	CodeSpace string `xml:"codeSpace,attr"`
	Value     string `xml:",chardata"`
}

// Shape contains the geometric shape information
type Shape struct {
	MultiPoint MultiPoint `xml:"MultiPoint"`
}

// MultiPoint contains multiple points
type MultiPoint struct {
	GmlID        string        `xml:"id,attr"`
	SrsName      string        `xml:"srsName,attr"`
	PointMembers []PointMember `xml:"pointMember"`
}

// PointMember contains a point
type PointMember struct {
	Point Point `xml:"Point"`
}

// Point represents a geographic point
type Point struct {
	GmlID   string `xml:"id,attr"`
	SrsName string `xml:"srsName,attr"`
	Name    string `xml:"name"`
	Pos     string `xml:"pos"`
}

// Result contains the observation results
type Result struct {
	MultiPointCoverage MultiPointCoverage `xml:"MultiPointCoverage"`
}

// MultiPointCoverage contains the observation data
type MultiPointCoverage struct {
	GmlID     string    `xml:"id,attr"`
	DomainSet DomainSet `xml:"domainSet"`
	RangeSet  RangeSet  `xml:"rangeSet"`
}

// DomainSet contains the domain information
type DomainSet struct {
	SimpleMultiPoint SimpleMultiPoint `xml:"SimpleMultiPoint"`
}

// SimpleMultiPoint contains position information
type SimpleMultiPoint struct {
	GmlID        string `xml:"id,attr"`
	SrsName      string `xml:"srsName,attr"`
	SrsDimension string `xml:"srsDimension,attr"`
	Positions    string `xml:"positions"`
}

// RangeSet contains the data values
type RangeSet struct {
	DataBlock DataBlock `xml:"DataBlock"`
}

// DataBlock contains the actual observation data
type DataBlock struct {
	RangeParameters            string `xml:"rangeParameters"`
	DoubleOrNilReasonTupleList string `xml:"doubleOrNilReasonTupleList"`
}
