package main

import (
	"fmt"
	"log"
	"net/http"
)


func main(){
newMux := http.NewServeMux()
server := &http.Server{
	Addr: ":8080",
	Handler: newMux,
}
newMux.Handle("/",http.FileServer(http.Dir('.')))
log.Fatal(server.ListenAndServe())
callGenderize("eff")

}

func callGenderize(nameQuery string){
	url := fmt.Sprintf("https://api.genderize.io?name=%s&apikey=Elixir.YOUR_API_KEY",nameQuery)
	fmt.Println(url, nameQuery)
}