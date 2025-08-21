package models

// common data type definition

type Exchanges struct {
	Exchanges []Exchange
}

type Exchange struct {
	Name string
}

type Currencies struct {
	Currencies []Currency
}

type Currency struct {
	Id string
}

// exchange rate data
