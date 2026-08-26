package models

import (
	"strings"
)

const (
	ConnMobile4GNormal = "MOBILE_4G_NORMAL"
	ConnMobile4GHigh   = "MOBILE_4G_HIGH"
	ConnSatelliteIrid  = "SATELLITE_IRIDIUM"
)

func ClassifyConnectivity(partName, model string) string {
	s := strings.ToUpper(partName + " " + model)
	has := func(sub string) bool { return strings.Contains(s, sub) }

	switch {
	case has("IRIDIUM") || has("SATELLITE") || has("SAT"):
		return ConnSatelliteIrid
	case has("HIGH") || has("HS") || has("HIGHSPEED"):
		return ConnMobile4GHigh
	case has("4G") || has("MOBILE") || has("LTE") || has("NORMAL"):
		return ConnMobile4GNormal
	default:
		return ""
	}
}

func NormalizeConnectivity(raw string) string {
	if v := ClassifyConnectivity(raw, ""); v != "" {
		return v
	}
	up := strings.ToUpper(strings.TrimSpace(raw))
	switch up {
	case ConnMobile4GNormal, ConnMobile4GHigh, ConnSatelliteIrid:
		return up
	}
	return ""
}
