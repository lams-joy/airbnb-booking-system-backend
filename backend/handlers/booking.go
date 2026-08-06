package handlers

import (
"context"
"encoding/json"
"fmt"
"net/http"
"time"

"airbnb-booking-system/db"
)

type HoldRequest struct {
PropertyID string `json:"property_id"`
GuestID    string `json:"guest_id"`
CheckIn    string `json:"check_in"`  // Format: YYYY-MM-DD
CheckOut   string `json:"check_out"` // Format: YYYY-MM-DD
}

type ConfirmBookingRequest struct {
PropertyID string  `json:"property_id"`
GuestID    string  `json:"guest_id"`
CheckIn    string  `json:"check_in"`
CheckOut   string  `json:"check_out"`
TotalPrice float64 `json:"total_price"`
}

// HoldDates reserves dates in Redis for 10 minutes while guest checks out
func HoldDates(w http.ResponseWriter, r *http.Request) {
var req HoldRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
http.Error(w, "Invalid request payload", http.StatusBadRequest)
return
}

if req.PropertyID == "" || req.GuestID == "" || req.CheckIn == "" || req.CheckOut == "" {
http.Error(w, "Missing required fields", http.StatusBadRequest)
return
}

// 1. Check if dates are already confirmed in PostgreSQL
var count int
checkQuery := `
SELECT COUNT(*) FROM bookings
WHERE property_id = $1 
  AND status = 'confirmed'
  AND daterange(check_in, check_out, '[)') && daterange($2::date, $3::date, '[)');
`
err := db.Pool.QueryRow(r.Context(), checkQuery, req.PropertyID, req.CheckIn, req.CheckOut).Scan(&count)
if err != nil {
http.Error(w, "Database verification failed: "+err.Error(), http.StatusInternalServerError)
return
}

if count > 0 {
http.Error(w, "These dates are already confirmed for another booking", http.StatusConflict)
return
}

// 2. Attempt atomic lock in Redis (Key: hold:{property_id}:{check_in}_{check_out})
redisKey := fmt.Sprintf("hold:%s:%s_%s", req.PropertyID, req.CheckIn, req.CheckOut)
ctx := context.Background()

// SETNX sets key only if it does not exist with a 10-minute TTL
acquired, err := db.RDB.SetNX(ctx, redisKey, req.GuestID, 10*time.Minute).Result()
if err != nil {
http.Error(w, "Redis lock error: "+err.Error(), http.StatusInternalServerError)
return
}

if !acquired {
http.Error(w, "Another guest is currently holding these dates. Try again in 10 minutes.", http.StatusConflict)
return
}

w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(map[string]interface{}{
"message":    "Dates held successfully for 10 minutes",
"hold_key":   redisKey,
"expires_in": "10m",
})
}

// ConfirmBooking converts a temporary hold into a confirmed PostgreSQL reservation
func ConfirmBooking(w http.ResponseWriter, r *http.Request) {
var req ConfirmBookingRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
http.Error(w, "Invalid request payload", http.StatusBadRequest)
return
}

// 1. Insert reservation into PostgreSQL
// If dates overlap with another confirmed booking, the GiST EXCLUDE constraint will raise an error
query := `
INSERT INTO bookings (property_id, guest_id, check_in, check_out, total_price, status)
VALUES ($1, $2, $3, $4, $5, 'confirmed')
RETURNING id, created_at;
`

var bookingID string
var createdAt string

err := db.Pool.QueryRow(
r.Context(), query,
req.PropertyID, req.GuestID, req.CheckIn, req.CheckOut, req.TotalPrice,
).Scan(&bookingID, &createdAt)

if err != nil {
http.Error(w, "Booking failed (dates may overlap with existing reservation): "+err.Error(), http.StatusConflict)
return
}

// 2. Clean up temporary Redis hold
redisKey := fmt.Sprintf("hold:%s:%s_%s", req.PropertyID, req.CheckIn, req.CheckOut)
db.RDB.Del(context.Background(), redisKey)

w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusCreated)
json.NewEncoder(w).Encode(map[string]interface{}{
"booking_id": bookingID,
"status":     "confirmed",
"message":    "Reservation confirmed successfully",
"created_at": createdAt,
})
}
