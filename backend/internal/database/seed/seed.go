package seed

import "gorm.io/gorm"

func SeedLocalData(db *gorm.DB) error {
	_ = db
	return nil
}
