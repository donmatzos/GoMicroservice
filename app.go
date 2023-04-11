package GoMicroservice

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type App struct {
	Router *mux.Router
	DB     *sql.DB
}

func (app *App) Initialize(user, password, dbname string) {
	connectionString :=
		fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, password, dbname)

	var err error
	app.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	app.Router = mux.NewRouter()
	app.initializeRoutes()
}

func (app *App) Run(address string) {
	log.Fatal(http.ListenAndServe(address, app.Router))
}

func (app *App) getProduct(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(writer, http.StatusBadRequest, "Invalid product ID")
		return
	}

	p := product{ID: id}
	if err := p.getProduct(app.DB); err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(writer, http.StatusNotFound, "Product not found")
		default:
			respondWithError(writer, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(writer, http.StatusOK, p)
}

func respondWithError(writer http.ResponseWriter, code int, message string) {
	respondWithJSON(writer, code, map[string]string{"error": message})
}

func respondWithJSON(writer http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	writer.Write(response)
}

func (app *App) getProducts(writer http.ResponseWriter, request *http.Request) {
	count, _ := strconv.Atoi(request.FormValue("count"))
	start, _ := strconv.Atoi(request.FormValue("start"))

	if count > 10 || count < 1 {
		count = 10
	}
	if start < 0 {
		start = 0
	}

	products, err := getProducts(app.DB, start, count)
	if err != nil {
		respondWithError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(writer, http.StatusOK, products)
}

func (app *App) createProduct(writer http.ResponseWriter, request *http.Request) {
	var p product
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&p); err != nil {
		respondWithError(writer, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer request.Body.Close()

	if err := p.createProduct(app.DB); err != nil {
		respondWithError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(writer, http.StatusCreated, p)
}

func (app *App) updateProduct(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(writer, http.StatusBadRequest, "Invalid product ID")
		return
	}

	var p product
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&p); err != nil {
		respondWithError(writer, http.StatusBadRequest, "Invalid resquest payload")
		return
	}
	defer request.Body.Close()
	p.ID = id

	if err := p.updateProduct(app.DB); err != nil {
		respondWithError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(writer, http.StatusOK, p)
}

func (app *App) deleteProduct(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(writer, http.StatusBadRequest, "Invalid Product ID")
		return
	}

	p := product{ID: id}
	if err := p.deleteProduct(app.DB); err != nil {
		respondWithError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(writer, http.StatusOK, map[string]string{"result": "success"})
}

func (app *App) getProductsByName(writer http.ResponseWriter, request *http.Request) {
	productName := request.FormValue("name")

	products, err := getProductsByName(app.DB, productName)
	if err != nil {
		respondWithError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(writer, http.StatusOK, products)
}

func (app *App) deleteProductByName(writer http.ResponseWriter, request *http.Request) {
	productName := request.FormValue("name")

	err := deleteProductByName(app.DB, productName)
	if err != nil {
		respondWithError(writer, http.StatusBadRequest, "Invalid Product Name")
		return
	}

	if err := deleteProductByName(app.DB, productName); err != nil {
		respondWithError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(writer, http.StatusOK, map[string]string{"result": "success"})
}

func (app *App) getProductsTotalPrice(writer http.ResponseWriter, _ *http.Request) {

	products, err := getProductsTotalPrice(app.DB)
	if err != nil {
		respondWithError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(writer, http.StatusOK, products)
}

func (app *App) initializeRoutes() {
	app.Router.HandleFunc("/products", app.getProducts).Methods("GET")
	app.Router.HandleFunc("/products/name", app.getProductsByName).Methods("GET")
	app.Router.HandleFunc("/product", app.createProduct).Methods("POST")
	app.Router.HandleFunc("/product/{id:[0-9]+}", app.getProduct).Methods("GET")
	app.Router.HandleFunc("/product/{id:[0-9]+}", app.updateProduct).Methods("PUT")
	app.Router.HandleFunc("/product/{id:[0-9]+}", app.deleteProduct).Methods("DELETE")
	app.Router.HandleFunc("/product/name", app.deleteProductByName).Methods("DELETE")
	app.Router.HandleFunc("/product/total", app.getProductsTotalPrice).Methods("GET")
}
