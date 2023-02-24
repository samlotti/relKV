package cmd

import (
	"net/http"
)

type AuthSecret struct {
	secret string
}

func NewAuthSecret(secret string) *AuthSecret {
	mw := &AuthSecret{}
	mw.secret = secret
	return mw
}

func (mw *AuthSecret) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		tkn := getHeaderKey("tkn", r)
		//fmt.Printf("tkn:%s\n", tkn)
		//fmt.Printf("sec:%s\n", mw.secret)
		if mw.secret != tkn {
			SendError(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		} else {
			next.ServeHTTP(w, r)
		}

	})
}
