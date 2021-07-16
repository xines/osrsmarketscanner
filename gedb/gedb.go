package gedb

//go:generate go run github.com/objectbox/objectbox-go/cmd/objectbox-gogen

// GeDatas Object Box struct
type GeDatas struct {
	ID              uint64
	ItemID          int64
	Name            string
	Members         bool
	Sp              int64
	BuyAverage      int64
	BuyQuantity     int64
	SellAverage     int64
	SellQuantity    int64
	OverallAverage  int64
	OverallQuantity int64
	Date            int64
}
