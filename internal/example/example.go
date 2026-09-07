package example

import (
	"context"

	"github.com/dpovey/goobie/pkg/goobie"
	"github.com/dpovey/goobie/pkg/goobie/read"
)

var dropPerson = goobie.Update0(`
 DROP TABLE IF EXISTS person
  `)

var createPerson = goobie.Update0(`
CREATE TABLE person (
      id   SERIAL,
      name VARCHAR NOT NULL UNIQUE,
      age  SMALLINT
    )
    `)

var dropCountry = goobie.Update0(`
		DROP TABLE IF EXISTS country
	`)
var createCountry = goobie.Update0(`
		CREATE TABLE country (
			code CHAR(3),
			name TEXT,
			population INTEGER,
			Gnp NUMERIC(10, 2)
		)
	`)

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
	rows := []goobie.UpdateF{
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
	s, err := goobie.Query0[string]("SELECT name FROM country ORDER BY name").Stream().Run(xa)
	if err != nil {
		return nil, err
	}
	return s.Take(5).Collect(context.Background())
}

type Country struct {
	Code string
	Name string
	Pop  int
	Gnp  *float64
}

var countryRead = (read.Row4[string, string, int, *float64]{}).Map(
	func(code string, name string, population int, gnp *float64) Country {
		return Country{Code: code, Name: name, Pop: population, Gnp: gnp}
	},
)

func countryQuery(sql string, args ...any) goobie.QueryT[Country] {
	return goobie.QueryFrAndRead(goobie.Fr(sql, args...), countryRead)
}

func MultipleColumns(xa goobie.Transactor) ([]Country, error) {
	s, err := countryQuery(
		"SELECT code, name, population, Gnp FROM country ORDER BY code",
	).Stream().Run(xa)
	if err != nil {
		return nil, err
	}
	return s.Take(5).Collect(context.Background())
}

func MultipleColumnsToMap(xa goobie.Transactor) (map[string]Country, error) {
	s, err := countryQuery(
		"SELECT code, name, population, Gnp FROM country",
	).Stream().Run(xa)
	if err != nil {
		return nil, err
	}
	return s.Take(5).ToMap(context.Background(), func(c Country) (string, Country) {
		return c.Code, c
	})
}

func CreatingPersonTables(xa goobie.Transactor) (int64, error) {
	total, err := dropPerson.Op().Map2(createPerson.Op(), func(a int64, b int64) int64 {
		return a + b
	}).Run(xa)
	if err != nil {
		return -1, err
	}
	return total, nil
}

func insert1(name string, age *int) goobie.UpdateF {
	return goobie.Update(`
		INSERT INTO person (name, age)
		VALUES (?, ?)
	`, name, age)
}

type Person struct {
	Id   int64
	Name string
	Age  *int
}

type PersonInfo struct {
	Name string
	Age  *int
}

func InsertingData(xa goobie.Transactor) ([]Person, error) {
	age := new(int)
	*age = 12
	return insert1("Alice", age).Op().
		FlatMap(func(int64) goobie.Op[int64] {
			return insert1("Bob", nil).Op()
		}).
		FlatMap(func(int64) goobie.Op[[]Person] {
			return goobie.Query[Person]("SELECT id, name, age FROM person").ToSlice()
		}).
		Run(xa)
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
	operation := insert.Op().
		FlatMap(func(int64) goobie.Op[int64] {
			return fetchId
		}).
		FlatMap(func(id int64) goobie.Op[Person] {
			return goobie.Query[Person](
				"SELECT id, name, age FROM person WHERE id = ?",
				id,
			).Unique()
		})

	err = xa.Transact(func(xa goobie.Transaction) error {
		result, err = operation.Run(xa)
		return err
	})

	return result, err
}

func BatchUpdates(xa goobie.Transactor, updates [][]any) (int64, error) {
	// Use UpdateF instead of UpdateWithWrite for batch operations
	updateF := goobie.Update("INSERT INTO person (name, age) VALUES (?, ?)")
	return xa.RunUpdateMany(updateF, updates)
}
