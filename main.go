package GoMicroservice

import "os"

func main() {
	a := App{}
	a.Initialize(
		os.Getenv("APP_DB_USERNAME"),
		os.Getenv("APP_DB_PASSWORD"),
		os.Getenv("APP_DB_NAME"))

	//set env variables (bash):
	//export APP_DB_USERNAME=postgres
	//export APP_DB_PASSWORD=<whatever password you use>
	//export APP_DB_NAME=postgres

	a.Run(":8010")
}
