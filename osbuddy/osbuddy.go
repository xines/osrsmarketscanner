package osbuddy

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// summaryURL
const summaryURL = `https://rsbuddy.com/exchange/summary.json`

type item struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Members         bool   `json:"members"`
	Sp              int    `json:"sp"`
	BuyAverage      int    `json:"buy_average"`
	BuyQuantity     int    `json:"buy_quantity"`
	SellAverage     int    `json:"sell_average"`
	SellQuantity    int    `json:"sell_quantity"`
	OverallAverage  int    `json:"overall_average"`
	OverallQuantity int    `json:"overall_quantity"`
}

// GeItems is used for item storage
var GeItems = map[string]item{}

// CachedItems adds Cache support for item lookup
var CachedItems = make([]item, 0)

// GetItemPriceByID takes a item id and returns item price
func GetItemPriceByID(id int) (int, error) {
	priceFound := 0

	for _, v := range GeItems {
		if v.ID == id {
			fmt.Printf("FOUND: %s - (BUY-ang: %d / SELL-ang: %d)", v.Name, v.BuyAverage, v.SellAverage)
			priceFound = v.BuyAverage
			break
		}
	}

	return priceFound, nil
}

// GetItemNameByID gets a item name from it's current id
func GetItemNameByID(id int) (string, error) {
	var itemFound string
	for _, v := range GeItems {
		if v.ID == id {
			fmt.Printf("GetItemNameByID FOUND: %s", v.Name)
			itemFound = v.Name
			break
		}
	}

	return itemFound, nil
}

// GetItemDataByName get all avilable datas from item name, and return (item) struct
func GetItemDataByName(name string) (item, error) {

	var itemFound item
	found := false

	for _, v := range GeItems {
		if strings.ToLower(v.Name) == name {
			itemFound = v
			found = true
			break
		} else if strings.Contains(strings.ToLower(v.Name), name) {
			fmt.Printf("Could not find %s but found %s\n", name, v.Name)
			itemFound = v
			found = true
			break
		}
	}

	if !found {
		return itemFound, fmt.Errorf("item not found")
	}

	return itemFound, nil
}

// UpdateGEItems Updates all items from json into global mapped struct to be used by other functions
func UpdateGEItems() error {
	resp, err := http.Get(summaryURL)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&GeItems); err != nil {
		return err
	}

	log.Print("Ge-Items updated successfully.\n")

	return nil
}
