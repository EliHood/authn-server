package accounts

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/keratin/authn-server/api"
	"github.com/keratin/authn-server/services"
)

func patchAccountUnlock(app *api.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			panic(err)
		}

		err = services.AccountUnlocker(app.AccountStore, id)
		if err != nil {
			if fe, ok := err.(services.FieldErrors); ok {
				api.WriteJson(w, http.StatusNotFound, api.ServiceErrors{fe})
				return
			}

			panic(err)
		}

		w.WriteHeader(http.StatusOK)
	}
}
