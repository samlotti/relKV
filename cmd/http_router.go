package cmd

import (
	"github.com/gorilla/mux"
	"net/http"
)

func (b *BucketsDb) newHTTPRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/status", b.status).Methods(http.MethodGet)

	dataRouter := router.NewRoute().Subrouter()

	if value, ok := EnvironmentInstance.LookupEnv("SECRET"); ok && len(value) > 5 {
		auth := NewAuthSecret(value)
		dataRouter.Use(auth.Middleware)
		b.authsecret = auth
	}

	if b.allowCreate {
		dataRouter.HandleFunc("/{bucket}", b.createBucket).Methods(http.MethodPut)
	}

	dataRouter.HandleFunc("/", b.listBuckets).Methods(http.MethodGet)
	dataRouter.HandleFunc("/{bucket}", b.listKeys).Methods(http.MethodGet)

	// order is important
	dataRouter.HandleFunc("/get/{bucket}", b.getKeys).Methods(http.MethodPost)

	dataRouter.HandleFunc("/{bucket}/{key:.*}", b.setKey).Methods(http.MethodPost)

	// Get a single entry
	dataRouter.HandleFunc("/{bucket}/{key:.*}", b.getKey).Methods(http.MethodGet)

	dataRouter.HandleFunc("/{bucket}/{key:.*}", b.delKey).Methods(http.MethodDelete)

	return router
}
