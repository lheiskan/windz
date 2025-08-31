// @vibe: 🤖 -- ai
package stations

import (
	"encoding/xml"
)

// WFSStationResponse represents the root WFS response for station queries
type WFSStationResponse struct {
	XMLName xml.Name           `xml:"FeatureCollection"`
	Members []WFSStationMember `xml:"member"`
}

// WFSStationMember represents a single station in the WFS response
type WFSStationMember struct {
	XMLName            xml.Name           `xml:"member"`
	MonitoringFacility MonitoringFacility `xml:"EnvironmentalMonitoringFacility"`
}

// MonitoringFacility represents the environmental monitoring facility (station)
type MonitoringFacility struct {
	XMLName    xml.Name      `xml:"EnvironmentalMonitoringFacility"`
	ID         string        `xml:"id,attr"`
	Identifier GMLIdentifier `xml:"identifier"`
	Names      []GMLName     `xml:"name"`
	StartDate  string        `xml:"operationalActivityPeriod>OperationalActivityPeriod>activityTime>TimePeriod>beginPosition"`
	Geometry   WFSGeometry   `xml:"representativePoint"`
	BelongsTo  []BelongsTo   `xml:"belongsTo"`
}

// GMLIdentifier represents an identifier element with codeSpace attribute
type GMLIdentifier struct {
	XMLName   xml.Name `xml:"identifier"`
	CodeSpace string   `xml:"codeSpace,attr"`
	Value     string   `xml:",chardata"`
}

// GMLName represents a name element with codeSpace attribute
type GMLName struct {
	XMLName   xml.Name `xml:"name"`
	CodeSpace string   `xml:"codeSpace,attr"`
	Value     string   `xml:",chardata"`
}

// WFSGeometry represents the geographic location of the station
type WFSGeometry struct {
	XMLName xml.Name `xml:"representativePoint"`
	Point   WFSPoint `xml:"Point"`
}

// WFSPoint represents a geographic point in WFS
type WFSPoint struct {
	XMLName     xml.Name `xml:"Point"`
	ID          string   `xml:"id,attr"`
	SrsName     string   `xml:"srsName,attr"`
	Coordinates string   `xml:"pos"`
}

// BelongsTo represents the network(s) the station belongs to
type BelongsTo struct {
	XMLName xml.Name `xml:"belongsTo"`
	Title   string   `xml:"title,attr"`
	Href    string   `xml:"href,attr"`
}
