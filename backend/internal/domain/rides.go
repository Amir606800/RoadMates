package domain

import "time"

type District struct {
	ID             int
	NameAZ, Slug   string
	RequiresPermit bool
}

type Ride struct {
	ID, DriverID, DriverFirstName, DriverLastName string
	From, To                                      District
	DepartureAt                                   time.Time
	SeatsTotal, SeatsAvailable                    int
	PriceCents                                    int64
	Notes, Status                                 string
	CreatedAt                                     time.Time
}

type RideSearch struct {
	FromID, ToID int
	Date         string // Azerbaijan calendar date (UTC+04:00).
}
