package domain

import "time"

type DriverApplication struct {
	ID, UserID                            string
	ContactPhone, FirstName, LastName     string
	PINCodeEncrypted                      []byte
	CarMake, CarModel, CarPlate, CarColor string
	CarYear, Seats                        int
	Status                                string
	ReviewedBy, RejectReason              *string
	ReviewedAt                            *time.Time
	CreatedAt                             time.Time
}
