package webhookreadiness

import (
	"net/http"
	"os"
)

func ready(w http.ResponseWriter, req *http.Request) {

}

func startServer() {
	http.HandleFunc("/ready", ready)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		os.Exit(1)
	}
}
