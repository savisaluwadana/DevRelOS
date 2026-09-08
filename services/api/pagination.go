package main

import (
	"net/http"

	"github.com/savisaluwadana/DevRelOS/internal/storage"
)

// requestPage reads ?limit= and ?offset= for a list endpoint.
//
// Out-of-range values fall back to the storage defaults rather than erroring,
// matching how the endpoints that already accepted ?limit= behaved. Callers get
// a bounded query even when neither parameter is supplied.
func requestPage(r *http.Request) storage.Page {
	return storage.Page{
		Limit:  intQuery(r, "limit", 0),
		Offset: intQuery(r, "offset", 0),
	}
}
