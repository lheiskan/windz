package fmi

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
	GmlID          string   `xml:"id,attr"`
	SampledFeature Location `xml:"sampledFeature>LocationCollection>member>Location"`
	Shape          Shape    `xml:"shape"`
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
	RectifiedGridCoverage RectifiedGridCoverage `xml:"RectifiedGridCoverage"`
	MultiPointCoverage    MultiPointCoverage    `xml:"MultiPointCoverage"`
}

// RectifiedGridCoverage contains the grid coverage data
type RectifiedGridCoverage struct {
	GmlID      string    `xml:"id,attr"`
	Limits     Limits    `xml:"limits"`
	AxisLabels string    `xml:"axisLabels"`
	DomainSet  DomainSet `xml:"domainSet"`
	RangeSet   RangeSet  `xml:"rangeSet"`
	RangeType  RangeType `xml:"rangeType"`
}

// Limits contains grid limits
type Limits struct {
	GridEnvelope GridEnvelope `xml:"GridEnvelope"`
}

// GridEnvelope contains grid envelope
type GridEnvelope struct {
	Low  string `xml:"low"`
	High string `xml:"high"`
}

// DomainSet contains the domain set
type DomainSet struct {
	RectifiedGrid    RectifiedGrid    `xml:"RectifiedGrid"`
	SimpleMultiPoint SimpleMultiPoint `xml:"SimpleMultiPoint"`
}

// RectifiedGrid contains rectified grid information
type RectifiedGrid struct {
	Dimension  string `xml:"dimension,attr"`
	GmlID      string `xml:"id,attr"`
	SrsName    string `xml:"srsName,attr"`
	Limits     Limits `xml:"limits"`
	AxisLabels string `xml:"axisLabels"`
	Origin     Origin `xml:"origin"`
}

// Origin contains the origin point
type Origin struct {
	Point OriginPoint `xml:"Point"`
}

// OriginPoint represents the origin point
type OriginPoint struct {
	GmlID   string `xml:"id,attr"`
	SrsName string `xml:"srsName,attr"`
	Pos     string `xml:"pos"`
}

// RangeSet contains the range set data
type RangeSet struct {
	DataBlock DataBlock `xml:"DataBlock"`
}

// DataBlock contains the actual data values
type DataBlock struct {
	DoubleOrNilReasonTupleList string `xml:"doubleOrNilReasonTupleList"`
}

// RangeType contains range type information
type RangeType struct {
	DataRecord DataRecord `xml:"DataRecord"`
}

// DataRecord contains data record fields
type DataRecord struct {
	Fields []Field `xml:"field"`
}

// Field represents a data field
type Field struct {
	Name string `xml:"name,attr"`
	Href string `xml:"href,attr"`
}

// SimpleMultiPoint contains position data
type SimpleMultiPoint struct {
	GmlID     string `xml:"id,attr"`
	SrsName   string `xml:"srsName,attr"`
	Positions string `xml:"positions"`
}

// MultiPointCoverage contains multi-point coverage data
type MultiPointCoverage struct {
	GmlID     string    `xml:"id,attr"`
	DomainSet DomainSet `xml:"domainSet"`
	RangeSet  RangeSet  `xml:"rangeSet"`
	RangeType RangeType `xml:"rangeType"`
}
