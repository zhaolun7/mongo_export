package utils

import "time"

func GetTimestamp(dateStr string) int64 {
	tm2, _ := time.Parse("2006-01-02", dateStr)
	return tm2.Unix() - 8*3600
}

func GetTimestampFromHour(dateStr string) int64 {
	tm2, _ := time.Parse("2006-01-02-15", dateStr)
	return tm2.Unix() - 8*3600
}

func Utc2dateDay(timestamp int64) string {
	tm := time.Unix(timestamp, 0)
	return tm.Format("2006-01-02")
}

