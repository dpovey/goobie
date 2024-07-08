package example

import (
	"database/sql"
	sqltx "github.com/dpovey/goobie/pkg/goobie/sql"
	_ "github.com/lib/pq"
	"log"
	"os"
	"reflect"
	"testing"
	"time"
)

var xa *sqltx.Transactor

// TestMain sets up the database connection for all tests
func TestMain(m *testing.M) {
	connStr := "user=testuser password=testpassword dbname=testdb sslmode=disable host=localhost port=5432"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	// Wait for the database to be ready
	for i := 0; i < 10; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		log.Fatalf("Database not ready: %v", err)
	}

	xa = sqltx.NewSqlTransactor(db)

	// Setup country tables
	err = DroppingCountryTables(xa)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	err = CreatingCountryTables(xa)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Setup person tables
	_, err = CreatingPersonTables(xa)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Run the tests
	code := m.Run()

	// Clean up
	_ = db.Close()
	os.Exit(code)
}

func TestInsertingAndData(t *testing.T) {
	// Insert
	got, err := InsertingData(xa)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	want := []Person{
		{Id: 1, Name: "Alice", Age: &[]int{12}[0]},
		{Id: 2, Name: "Bob", Age: nil},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Got: %v, Want: %v", got, want)
	}

	// Update
	nrUpdated, err := UpdatingData(xa)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if nrUpdated != 1 {
		t.Errorf("Got: %v, Want: 1", got)
	}
}

func TestReadingRowsIntoSlices(t *testing.T) {
	got, err := ReadingRowsIntoSlices(xa)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	want := []string{"France", "Germany", "Japan", "United Kingdom", "United States"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Got: %v, Want: %v", got, want)
	}
}

func TestStreaming(t *testing.T) {
	got, err := Streaming(xa)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	want := []string{"France", "Germany", "Japan", "United Kingdom", "United States"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Got: %v, Want: %v", got, want)
	}
}

func TestMultipleColumns(t *testing.T) {
	got, err := MultipleColumns(xa)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	want := []Country{
		{Code: "DEU", Name: "Germany", Pop: 1000000, Gnp: &[]float64{1000000.00}[0]},
		{Code: "FRA", Name: "France", Pop: 1000000, Gnp: &[]float64{1000000.00}[0]},
		{Code: "GBR", Name: "United Kingdom", Pop: 1000000, Gnp: &[]float64{1000000.00}[0]},
		{Code: "JPN", Name: "Japan", Pop: 1000000, Gnp: &[]float64{1000000.00}[0]},
		{Code: "USA", Name: "United States", Pop: 1000000, Gnp: &[]float64{1000000.00}[0]},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Got: %v, Want: %v", got, want)
	}
}

func TestMultipleColumnsToMap(t *testing.T) {
	m, err := MultipleColumnsToMap(xa)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	want := map[string]Country{
		"DEU": {Code: "DEU", Name: "Germany", Pop: 1000000, Gnp: &[]float64{1000000.00}[0]},
		"FRA": {Code: "FRA", Name: "France", Pop: 1000000, Gnp: &[]float64{1000000.00}[0]},
		"GBR": {Code: "GBR", Name: "United Kingdom", Pop: 1000000, Gnp: &[]float64{1000000.00}[0]},
		"JPN": {Code: "JPN", Name: "Japan", Pop: 1000000, Gnp: &[]float64{1000000.00}[0]},
		"USA": {Code: "USA", Name: "United States", Pop: 1000000, Gnp: &[]float64{1000000.00}[0]},
	}
	if !reflect.DeepEqual(m, want) {
		t.Errorf("Got: %v, Want: %v", m, want)
	}
}

func TestRetrievingResults(t *testing.T) {
	got, err := RetrievingResults(xa, "John Doe", 29)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	want := Person{
		Id:   0,
		Name: "John Doe",
		Age:  &[]int{29}[0],
	}
	if want.Name != got.Name || *want.Age != *got.Age {
		t.Errorf("Got: %v, Want: %v", got, want)
	}
}
