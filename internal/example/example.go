package example

import (
	"context"
	"github.com/dpovey/goobie/pkg/goobie"
	"github.com/dpovey/goobie/pkg/stream"
)

var dropPerson = goobie.Update(`
 DROP TABLE IF EXISTS person
  `).Update()

var createPerson = goobie.Update(`
CREATE TABLE person (
      id   SERIAL,
      name VARCHAR NOT NULL UNIQUE,
      age  SMALLINT
    )
    `).Update()

var dropCountry = goobie.Update(`
		DROP TABLE IF EXISTS country
	`).Update()
var createCountry = goobie.Update(`
		CREATE TABLE country (
			code CHAR(3),
			name TEXT,
			population INTEGER,
			Gnp NUMERIC(10, 2)
		)
	`).Update()

func DroppingCountryTables(xa goobie.Transactor) error {
	_, err := dropCountry.Run(xa)
	if err != nil {
		return err
	}
	return nil
}
func CreatingCountryTables(xa goobie.Transactor) error {
	_, err := createCountry.Run(xa)
	if err != nil {
		return err
	}
	rows := []goobie.UpdateT{
		goobie.Update("INSERT INTO country (code, name, population, Gnp) VALUES (?, ?, ?, ?)", "USA", "United States", 1000000, 1000000.00),
		goobie.Update("INSERT INTO country (code, name, population, Gnp) VALUES (?, ?, ?, ?)", "GBR", "United Kingdom", 1000000, 1000000.00),
		goobie.Update("INSERT INTO country (code, name, population, Gnp) VALUES (?, ?, ?, ?)", "FRA", "France", 1000000, 1000000.00),
		goobie.Update("INSERT INTO country (code, name, population, Gnp) VALUES (?, ?, ?, ?)", "DEU", "Germany", 1000000, 1000000.00),
		goobie.Update("INSERT INTO country (code, name, population, Gnp) VALUES (?, ?, ?, ?)", "JPN", "Japan", 1000000, 1000000.00),
	}
	_, err = goobie.UpdateAll(rows...).Run(xa)
	return err
}

func ReadingRowsIntoSlices(xa goobie.Transactor) ([]string, error) {
	return goobie.Query0[string]("SELECT name FROM country ORDER BY name").ToSlice().Run(xa)
}

func Streaming(xa goobie.Transactor) ([]string, error) {
	s, _ := goobie.Query0[string]("SELECT name FROM country ORDER BY name").Stream().Run(xa)
	return s.Take(5).Collect(context.Background())
}

type Country struct {
	Code string
	Name string
	Pop  int
	Gnp  *float64
}

func MultipleColumns(xa goobie.Transactor) ([]Country, error) {
	s, _ := goobie.Query[Country]("SELECT code, name, population, Gnp FROM country ORDER BY code").Stream().Run(xa)
	return s.Take(5).Collect(context.Background())
}

func MultipleColumnsToMap(xa goobie.Transactor) (map[string]Country, error) {
	s, _ := goobie.Query[Country]("SELECT code, name, population, Gnp FROM country").Stream().Run(xa)
	return stream.ToMap(s.Take(5), context.Background(), func(c Country) (string, Country) {
		return c.Code, c
	})
}

func CreatingPersonTables(xa goobie.Transactor) (int64, error) {
	total, err := goobie.Map2[int64, int64, int64](dropPerson, createPerson)(func(a int64, b int64) int64 {
		return a + b
	}).Run(xa)
	if err != nil {
		return -1, err
	}
	return total, nil
}

func insert1(name string, age *int) goobie.UpdateOp {
	return goobie.Update(`
		INSERT INTO person (name, age)
		VALUES (?, ?)
	`, name, age).Update()
}

type Person struct {
	Id   int64
	Name string
	Age  *int
}

func InsertingData(xa goobie.Transactor) ([]Person, error) {
	// Note: outside a transaction
	age := new(int)
	*age = 12
	_, err := insert1("Alice", age).Run(xa)
	if err != nil {
		return nil, err
	}
	_, err = insert1("Bob", nil).Run(xa)
	if err != nil {
		return nil, err
	}
	return goobie.Query[Person]("SELECT id, name, age FROM person").ToSlice().Run(xa)
}

func UpdatingData(xa goobie.Transactor) (int64, error) {
	return goobie.Update("UPDATE person SET age = 15 where name = 'Alice'").Run(xa)
}

func RetrievingResults(xa goobie.Transactor, name string, age int) (Person, error) {
	var result Person
	var err error
	insert := goobie.Update(`
		INSERT INTO person (name, age)
		VALUES (?, ?)
		`, name, age)
	fetchId := goobie.Query0[int64](`SELECT lastval()`).Unique()

	err = xa.Transact(func(xa goobie.Transaction) error {
		_, err = insert.Run(xa)
		if err != nil {
			return err
		}
		id, err := fetchId.Run(xa)
		if err != nil {
			return err
		}
		result, err = goobie.Query[Person]("SELECT id, name, age FROM person WHERE id = ?", id).Unique().Run(xa)
		return err
	})

	return result, err
}
