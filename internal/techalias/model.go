package techalias

import (
	"time"
)

// TechnologyAlias represents alternative names or abbreviations for a technology.
// For example, "JavaScript" might have aliases like "JS" or "ECMAScript".
type TechnologyAlias struct {
	ID           int       `json:"id" db:"id"`
	TechnologyID int       `json:"technology_id" db:"technology_id"`
	Alias        string    `json:"alias" db:"alias"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
