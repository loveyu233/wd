package wd

type StructGormIDData struct {
	ID   uint64 `gorm:"column:id"`
	Data string `gorm:"column:data"`
}

type StructGormIDDateTime struct {
	ID       uint64   `gorm:"column:id"`
	DateTime DateTime `gorm:"column:date_time"`
}

type StructGormIDPtrDateTime struct {
	ID       uint64    `gorm:"column:id"`
	DateTime *DateTime `gorm:"column:date_time"`
}

type StructGormIDDateOnly struct {
	ID       uint64   `gorm:"column:id"`
	DateOnly DateOnly `gorm:"column:date_only"`
}

type StructGormIDPtrDateOnly struct {
	ID       uint64    `gorm:"column:id"`
	DateOnly *DateOnly `gorm:"column:date_only"`
}

type StructGormIDMonthDay struct {
	ID       uint64   `gorm:"column:id"`
	MonthDay MonthDay `gorm:"column:month_day"`
}

type StructGormIDPtrMonthDay struct {
	ID       uint64    `gorm:"column:id"`
	MonthDay *MonthDay `gorm:"column:month_day"`
}

type StructGormIDTimeOnly struct {
	ID       uint64   `gorm:"column:id"`
	TimeOnly TimeOnly `gorm:"column:time_only"`
}

type StructGormIDPtrTimeOnly struct {
	ID       uint64    `gorm:"column:id"`
	TimeOnly *TimeOnly `gorm:"column:time_only"`
}

type StructGormIDTimeHM struct {
	ID     uint64 `gorm:"column:id"`
	TimeHM TimeHM `gorm:"column:time_hm"`
}

type StructGormIDPtrTimeHM struct {
	ID     uint64  `gorm:"column:id"`
	TimeHM *TimeHM `gorm:"column:time_hm"`
}

type StructGormIDDataTwo struct {
	ID    uint64 `gorm:"column:id"`
	Data1 string `gorm:"column:data_1"`
	Data2 string `gorm:"column:data_2"`
}
