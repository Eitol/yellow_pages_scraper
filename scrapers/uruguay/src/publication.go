package scraper

import (
	"fmt"
	"strconv"
	"strings"
)

type Category struct {
	URL  string `json:"url" bson:"url"`
	Name string `json:"name" bson:"name"`
}

func (c *Category) CleanName() string {
	return strings.Split(strings.Split(c.URL, "q_")[1], "/")[0]
}

type PhoneType string

const (
	Fixed = PhoneType("office")
	Mobil = PhoneType("mobil")
	Fax   = PhoneType("fax")
)

type Phone struct {
	PhoneType    PhoneType `json:"phoneType" bson:"phoneType"`
	Number       string    `json:"number" bson:"number"`
	IsMain       bool      `json:"isMain" bson:"isMain"`
	HaveWhatsapp bool      `json:"haveWhatsapp" bson:"haveWhatsapp"`
}

type DayOfWeek string

const (
	UnknownDay = DayOfWeek("unknown")
	Monday     = DayOfWeek("monday")
	Tuesday    = DayOfWeek("tuesday")
	Wednesday  = DayOfWeek("wednesday")
	Thursday   = DayOfWeek("thursday")
	Friday     = DayOfWeek("friday")
	Saturday   = DayOfWeek("saturday")
	Sunday     = DayOfWeek("sunday")
)

type HourRange struct {
	Start string `json:"start" bson:"start"`
	End   string `json:"end" bson:"end"`
}

type Timetable = map[DayOfWeek][]HourRange

type Coordinates struct {
	Latitude  float64 `json:"latitude" bson:"latitude"`
	Longitude float64 `json:"longitude" bson:"longitude"`
}

func (c Coordinates) String() string {
	lat := strconv.FormatFloat(c.Latitude, 'f', 6, 64)
	lng := strconv.FormatFloat(c.Latitude, 'f', 6, 64)
	return fmt.Sprintf("[%s, %s]", lat, lng)
}

type AddressType string

const (
	Office    = AddressType("office")
	House     = AddressType("house")
	Apartment = AddressType("apartment")
)

type Address struct {
	AddressType   AddressType `json:"addressType" bson:"addressType"`
	Number        string      `json:"number" bson:"number"`
	StreetAddress string      `json:"streetAddress" bson:"streetAddress"`
	Locality      string      `json:"locality" bson:"locality"`
	Zip           string      `json:"zip" bson:"zip"`
	City          string      `json:"city" bson:"city"`
}

type Publication struct {
	Name           string      `json:"name" bson:"name"`
	Phones         []Phone     `json:"phones" bson:"phones"`
	Mails          []string    `json:"mails" bson:"mails"`
	Address        Address     `json:"address" bson:"address"`
	Coordinates    Coordinates `json:"coordinates" bson:"coordinates"`
	Webs           []string    `json:"webs" bson:"webs"`
	Categories     []string    `json:"categories" bson:"categories"`
	Timetable      Timetable   `json:"timetable" bson:"timetable"`
	PublicationUrl string      `json:"publicationUrl" bson:"publicationUrl"`
	Information    string      `json:"information" bson:"information"`
}
