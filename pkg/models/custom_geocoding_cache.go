package models

// CustomGeocodingCache represents a cached geocoding result stored in database (fork-local feature).
// The primary key is the SHA-256 hex of the normalized query, so no uuid type is consumed.
// No-hit results are also cached (negative cache, Hit=false) to comply with the provider usage policies.
type CustomGeocodingCache struct {
	QueryHash       string  `xorm:"'query_hash' VARCHAR(64) PK"`
	Query           string  `xorm:"'query' VARCHAR(255) NOT NULL DEFAULT ''"`
	Provider        string  `xorm:"'provider' VARCHAR(32) NOT NULL DEFAULT ''"`
	Latitude        float64 `xorm:"'latitude' NOT NULL DEFAULT 0"`
	Longitude       float64 `xorm:"'longitude' NOT NULL DEFAULT 0"`
	DisplayName     string  `xorm:"'display_name' VARCHAR(255) NOT NULL DEFAULT ''"`
	Confidence      byte    `xorm:"'confidence' NOT NULL DEFAULT 0"`
	Hit             bool    `xorm:"'hit' NOT NULL DEFAULT false"`
	CreatedUnixTime int64   `xorm:"'created_unix_time' NOT NULL DEFAULT 0"`
}
