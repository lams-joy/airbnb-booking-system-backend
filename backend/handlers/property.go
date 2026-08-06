package handlers

import (
"encoding/json"
"errors"
"net/http"
"strconv"

"github.com/go-chi/chi/v5"
"github.com/jackc/pgx/v5"

"airbnb-booking-system/db"
)

type Property struct {
ID             string  `json:"id"`
HostID         string  `json:"host_id"`
Title          string  `json:"title"`
Description    string  `json:"description"`
PricePerNight  float64 `json:"price_per_night"`
MaxGuests      int     `json:"max_guests"`
Latitude       float64 `json:"latitude"`
Longitude      float64 `json:"longitude"`
DistanceMeters float64 `json:"distance_meters,omitempty"`
}

type CreatePropertyRequest struct {
HostID        string  `json:"host_id"`
Title         string  `json:"title"`
Description   string  `json:"description"`
PricePerNight float64 `json:"price_per_night"`
MaxGuests     int     `json:"max_guests"`
Latitude      float64 `json:"latitude"`
Longitude     float64 `json:"longitude"`
}

// CreateProperty handles host listing generation with spatial points
func CreateProperty(w http.ResponseWriter, r *http.Request) {
var req CreatePropertyRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
http.Error(w, "Invalid request payload", http.StatusBadRequest)
return
}

if req.HostID == "" || req.Title == "" || req.PricePerNight <= 0 || req.MaxGuests <= 0 {
http.Error(w, "Missing required fields or invalid price/guest limits", http.StatusBadRequest)
return
}

query := `
INSERT INTO properties (host_id, title, description, price_per_night, max_guests, location)
VALUES ($1, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography)
RETURNING id, created_at;
`

var propertyID string
var createdAt string

err := db.Pool.QueryRow(
r.Context(), query,
req.HostID, req.Title, req.Description, req.PricePerNight, req.MaxGuests, req.Longitude, req.Latitude,
).Scan(&propertyID, &createdAt)

if err != nil {
http.Error(w, "Failed to create property: "+err.Error(), http.StatusInternalServerError)
return
}

w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusCreated)
json.NewEncoder(w).Encode(map[string]interface{}{
"id":         propertyID,
"message":    "Property created successfully",
"created_at": createdAt,
})
}

// SearchProperties queries listings within a given radius in kilometers using ST_DWithin
func SearchProperties(w http.ResponseWriter, r *http.Request) {
latStr := r.URL.Query().Get("lat")
lngStr := r.URL.Query().Get("lng")
radiusStr := r.URL.Query().Get("radius_km")

if latStr == "" || lngStr == "" {
http.Error(w, "lat and lng query parameters are required", http.StatusBadRequest)
return
}

lat, err1 := strconv.ParseFloat(latStr, 64)
lng, err2 := strconv.ParseFloat(lngStr, 64)
radiusKM, err3 := strconv.ParseFloat(radiusStr, 64)

if err1 != nil || err2 != nil {
http.Error(w, "Invalid latitude or longitude", http.StatusBadRequest)
return
}

if err3 != nil || radiusKM <= 0 {
radiusKM = 20.0 // Default 20km radius
}

radiusMeters := radiusKM * 1000.0

query := `
SELECT 
id, host_id, title, description, price_per_night, max_guests,
ST_Y(location::geometry) AS latitude,
ST_X(location::geometry) AS longitude,
ST_Distance(location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) AS distance_meters
FROM properties
WHERE ST_DWithin(location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
ORDER BY distance_meters ASC
LIMIT 50;
`

rows, err := db.Pool.Query(r.Context(), query, lng, lat, radiusMeters)
if err != nil {
http.Error(w, "Query execution error: "+err.Error(), http.StatusInternalServerError)
return
}
defer rows.Close()

properties := []Property{}
for rows.Next() {
var p Property
if err := rows.Scan(&p.ID, &p.HostID, &p.Title, &p.Description, &p.PricePerNight, &p.MaxGuests, &p.Latitude, &p.Longitude, &p.DistanceMeters); err != nil {
http.Error(w, "Error scanning row: "+err.Error(), http.StatusInternalServerError)
return
}
properties = append(properties, p)
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(properties)
}

// GetPropertyByID fetches single listing by UUID
func GetPropertyByID(w http.ResponseWriter, r *http.Request) {
propertyID := chi.URLParam(r, "id")

query := `
SELECT 
id, host_id, title, description, price_per_night, max_guests,
ST_Y(location::geometry) AS latitude,
ST_X(location::geometry) AS longitude
FROM properties
WHERE id = $1;
`

var p Property
err := db.Pool.QueryRow(r.Context(), query, propertyID).Scan(
&p.ID, &p.HostID, &p.Title, &p.Description, &p.PricePerNight, &p.MaxGuests, &p.Latitude, &p.Longitude,
)

if err != nil {
if errors.Is(err, pgx.ErrNoRows) {
http.Error(w, "Property not found", http.StatusNotFound)
return
}
http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(p)
}
