package GoMicroservice //_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
)

var app App

func TestMain(m *testing.M) {
	app.Initialize(
		"postgres",
		"postgres",
		"postgres")

	ensureTableExists()
	code := m.Run()
	clearTable()
	os.Exit(code)
}

func ensureTableExists() {
	if _, err := app.DB.Exec(tableCreationQuery); err != nil {
		log.Fatal(err)
	}
}

func clearTable() {
	app.DB.Exec("DELETE FROM products")
	app.DB.Exec("ALTER SEQUENCE products_id_seq RESTART WITH 1")
}

const tableCreationQuery = `CREATE TABLE IF NOT EXISTS products
(
    id SERIAL,
    name TEXT NOT NULL,
    price NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    CONSTRAINT products_pkey PRIMARY KEY (id)
)`

func TestEmptyTable(t *testing.T) {
	clearTable()

	request, _ := http.NewRequest("GET", "/products", nil)
	response := executeRequest(request)

	checkResponseCode(t, http.StatusOK, response.Code)

	if body := response.Body.String(); body != "[]" {
		t.Errorf("Expected an empty array. Got %s", body)
	}
}

func executeRequest(request *http.Request) *httptest.ResponseRecorder {
	responseRecorder := httptest.NewRecorder()
	app.Router.ServeHTTP(responseRecorder, request)

	return responseRecorder
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func TestGetNonExistentProduct(t *testing.T) {
	clearTable()

	request, _ := http.NewRequest("GET", "/product/11", nil)
	response := executeRequest(request)

	checkResponseCode(t, http.StatusNotFound, response.Code)

	var m map[string]string
	json.Unmarshal(response.Body.Bytes(), &m)
	if m["error"] != "Product not found" {
		t.Errorf("Expected the 'error' key of the response to be set to 'Product not found'. Got '%s'", m["error"])
	}
}

func TestCreateProduct(t *testing.T) {

	clearTable()

	var jsonStr = []byte(`{"name":"test product", "price": 11.22}`)
	request, _ := http.NewRequest("POST", "/product", bytes.NewBuffer(jsonStr))
	request.Header.Set("Content-Type", "application/json")

	response := executeRequest(request)
	checkResponseCode(t, http.StatusCreated, response.Code)

	var m map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	if m["name"] != "test product" {
		t.Errorf("Expected product name to be 'test product'. Got '%v'", m["name"])
	}

	if m["price"] != 11.22 {
		t.Errorf("Expected product price to be '11.22'. Got '%v'", m["price"])
	}

	// the id is compared to 1.0 because JSON unmarshaling converts numbers to
	// floats, when the target is app map[string]interface{}
	if m["id"] != 1.0 {
		t.Errorf("Expected product ID to be '1'. Got '%v'", m["id"])
	}

	body, _ := ioutil.ReadAll(response.Body)
	fmt.Print("\t" + string(body) + "\n")

}

func TestGetProduct(t *testing.T) {
	clearTable()
	addProducts(1)

	request, _ := http.NewRequest("GET", "/product/1", nil)
	response := executeRequest(request)

	body, _ := ioutil.ReadAll(response.Body)
	fmt.Print("\t" + string(body) + "\n")

	checkResponseCode(t, http.StatusOK, response.Code)
}

func addProducts(count int) {
	if count < 1 {
		count = 1
	}

	for i := 0; i < count; i++ {
		app.DB.Exec("INSERT INTO products(name, price) VALUES($1, $2)", "Product "+strconv.Itoa(i), (i+1.0)*10)
	}
}

func TestUpdateProduct(t *testing.T) {

	clearTable()
	addProducts(1)

	request, _ := http.NewRequest("GET", "/product/1", nil)
	response := executeRequest(request)
	var originalProduct map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &originalProduct)

	var jsonStr = []byte(`{"name":"test product - updated name", "price": 11.22}`)
	request, _ = http.NewRequest("PUT", "/product/1", bytes.NewBuffer(jsonStr))
	request.Header.Set("Content-Type", "application/json")

	response = executeRequest(request)

	checkResponseCode(t, http.StatusOK, response.Code)

	var m map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	if m["id"] != originalProduct["id"] {
		t.Errorf("Expected the id to remain the same (%v). Got %v", originalProduct["id"], m["id"])
	}

	if m["name"] == originalProduct["name"] {
		t.Errorf("Expected the name to change from '%v' to '%v'. Got '%v'", originalProduct["name"], m["name"], m["name"])
	}

	if m["price"] == originalProduct["price"] {
		t.Errorf("Expected the price to change from '%v' to '%v'. Got '%v'", originalProduct["price"], m["price"], m["price"])
	}

	body, _ := ioutil.ReadAll(response.Body)
	fmt.Print("\t" + string(body) + "\n")
}

func TestDeleteProduct(t *testing.T) {
	clearTable()
	addProducts(1)

	request, _ := http.NewRequest("GET", "/product/1", nil)
	response := executeRequest(request)
	checkResponseCode(t, http.StatusOK, response.Code)

	request, _ = http.NewRequest("DELETE", "/product/1", nil)
	response = executeRequest(request)

	body, _ := ioutil.ReadAll(response.Body)
	fmt.Print("\t" + string(body) + "\n")

	checkResponseCode(t, http.StatusOK, response.Code)

	request, _ = http.NewRequest("GET", "/product/1", nil)
	response = executeRequest(request)
	checkResponseCode(t, http.StatusNotFound, response.Code)
}

func TestFindProductsByNameConcreteProduct(t *testing.T) {
	clearTable()
	addProducts(3)

	query := url.Values{}
	query.Set("name", "Product 1")
	requestUrl := "/products/name?" + query.Encode()
	request, _ := http.NewRequest("GET", requestUrl, nil)

	response := executeRequest(request)

	checkResponseCode(t, http.StatusOK, response.Code)

	var products []product
	if err := json.Unmarshal(response.Body.Bytes(), &products); err != nil {
		t.Error("Error parsing body!")
	}

	if len(products) != 1 {
		log.Fatalf("Expected 1 product. Received %d products.", len(products))
	}

	checkResponseCode(t, http.StatusOK, response.Code)
}

func TestFindProductsByNameMultipleProducts(t *testing.T) {
	clearTable()
	addProducts(3)

	query := url.Values{}
	query.Set("name", "Product")
	requestUrl := "/products/name?" + query.Encode()
	request, _ := http.NewRequest("GET", requestUrl, nil)

	response := executeRequest(request)

	checkResponseCode(t, http.StatusOK, response.Code)

	var products []product
	if err := json.Unmarshal(response.Body.Bytes(), &products); err != nil {
		t.Error("Error parsing body!")
	}

	if len(products) != 3 {
		log.Fatalf("Expected 3 products. Received %d product(s).", len(products))
	}

	checkResponseCode(t, http.StatusOK, response.Code)
}

func TestDeleteProductByName(t *testing.T) {
	clearTable()

	var jsonStr = []byte(`{"name":"test", "price": 11.22}`)

	request, _ := http.NewRequest("POST", "/product", bytes.NewBuffer(jsonStr))
	request.Header.Set("Content-Type", "application/json")

	response := executeRequest(request)
	checkResponseCode(t, http.StatusCreated, response.Code)

	query := url.Values{}
	query.Set("name", "test")
	requestUrl := "/product/name?" + query.Encode()
	request, _ = http.NewRequest("DELETE", requestUrl, nil)
	response = executeRequest(request)

	body, _ := ioutil.ReadAll(response.Body)
	fmt.Print("\t" + string(body) + "\n")

	checkResponseCode(t, http.StatusOK, response.Code)

	request, _ = http.NewRequest("GET", "/product/1", nil)
	response = executeRequest(request)
	checkResponseCode(t, http.StatusNotFound, response.Code)
}

func TestGetProductsTotalPrice(t *testing.T) {
	clearTable()
	addProducts(2)

	request, _ := http.NewRequest("GET", "/product/total", nil)
	response := executeRequest(request)

	var totalPrice float64
	json.Unmarshal(response.Body.Bytes(), &totalPrice)

	if totalPrice != 30.0 {
		log.Fatalf("Expected total price 30.0, got %f", totalPrice)
	}

	checkResponseCode(t, http.StatusOK, response.Code)
}
