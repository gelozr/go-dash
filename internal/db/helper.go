package db

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

func RecordExists(q *gorm.DB) (bool, error) {
	var hit int

	if err := q.Select("1").Limit(1).Scan(&hit).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("record exists: %w", err)
	}

	return hit == 1, nil
}
