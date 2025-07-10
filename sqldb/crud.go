package sqldb

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/mpetavy/common"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type CRUD[T any] struct {
	Name       string
	Repository *Repository[T]
	URLPrefix  string
	PostURL    *common.RestURL
	ListURL    *common.RestURL
	GetURL     *common.RestURL
	PutURL     *common.RestURL
	DeleteURL  *common.RestURL
}

var (
	ErrModifiedID     = fmt.Errorf("not allowed modification of ID")
	ErrNotFound       = fmt.Errorf("record not found")
	ErrDuplicateFound = fmt.Errorf("duplicate record found")

	crudRead  = flag.Bool("crud.read", false, "Allowed to READ via CRUD REST endpoints")
	crudWrite = flag.Bool("crud.write", false, "Allowed to WRITE via CRUD REST endpoints")
)

const (
	resourceURI = "%s/{id}"
)

type CRUDHandlerFunc func(restURL *common.RestURL, description string, needsAuth bool, handler http.HandlerFunc)

func NewCrud[T any](crudHandlerFunc CRUDHandlerFunc, repository *Repository[T], basicAuth func(r *http.Request, username, password string) error, urlPrefix string) (*CRUD[T], error) {
	common.DebugFunc()

	var t T

	objectName := fmt.Sprintf("%T", t)
	objectName = objectName[strings.LastIndex(objectName, ".")+1:]

	crud := &CRUD[T]{
		Name:       objectName,
		Repository: repository,
		URLPrefix:  urlPrefix,
		PostURL:    common.NewRestURL(http.MethodPost, urlPrefix),
		ListURL:    common.NewRestURL(http.MethodGet, urlPrefix),
		GetURL:     common.NewRestURL(http.MethodGet, fmt.Sprintf(resourceURI, urlPrefix)),
		PutURL:     common.NewRestURL(http.MethodPut, fmt.Sprintf(resourceURI, urlPrefix)),
		DeleteURL:  common.NewRestURL(http.MethodDelete, fmt.Sprintf(resourceURI, urlPrefix)),
	}

	crud.ListURL.Params = []common.RestURLField{
		{
			Name:        "offset",
			Description: "offset to start data set read",
			Default:     "-1",
		},
		{
			Name:        "limit",
			Description: "limit data set read",
			Default:     "-1",
		},
	}

	if crudHandlerFunc != nil {
		crudHandlerFunc(crud.PostURL, fmt.Sprintf("Register %s object", objectName), true, common.BasicAuthHandler(true, basicAuth, common.TelemetryHandler(crud.PostHandler)))
		crudHandlerFunc(crud.ListURL, fmt.Sprintf("List all %s objects", objectName), true, common.BasicAuthHandler(true, basicAuth, common.TelemetryHandler(crud.ListHandler)))
		crudHandlerFunc(crud.GetURL, fmt.Sprintf("Get %s object", objectName), true, common.BasicAuthHandler(true, basicAuth, common.TelemetryHandler(crud.GetHandler)))
		crudHandlerFunc(crud.PutURL, fmt.Sprintf("Update %s object", objectName), true, common.BasicAuthHandler(true, basicAuth, common.TelemetryHandler(crud.PutHandler)))
		crudHandlerFunc(crud.DeleteURL, fmt.Sprintf("Delete %s object", objectName), true, common.BasicAuthHandler(true, basicAuth, common.TelemetryHandler(crud.DeleteHandler)))
	}

	return crud, nil
}

func (crud *CRUD[T]) PostHandler(w http.ResponseWriter, r *http.Request) {
	common.DebugFunc()

	defer func(start time.Time) {
		crud.PostURL.UpdateStats(start)
	}(time.Now())

	err := crud.PostURL.Validate(r)
	if common.Error(err) {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if !*crudWrite {
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)

		return
	}

	ids, err := func() ([]int64, error) {
		records, _, err := common.ReadBodyJSON[T](r.Body)
		if common.Error(err) {
			return nil, err
		}

		err = crud.Repository.SaveAll(records)
		if common.Error(err) {
			return nil, err
		}

		var ids []int64

		for _, record := range records {
			fieldID, err := common.GetStructValue(&record, "ID")
			if common.Error(err) {
				return nil, err
			}

			sqlFieldID := fieldID.Interface().(FieldInt64)

			ids = append(ids, sqlFieldID.Int64())
		}

		return ids, nil
	}()

	switch err {
	case nil:
		for _, id := range ids {
			common.Info("%s POST: %d", crud.Name, id)

			w.Header().Set(common.HEADER_LOCATION, crud.GetURL.Format(id))
		}

		w.WriteHeader(http.StatusCreated)
	case ErrDuplicateFound:
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (crud *CRUD[T]) GetHandler(w http.ResponseWriter, r *http.Request) {
	common.DebugFunc()

	defer func(start time.Time) {
		crud.GetURL.UpdateStats(start)
	}(time.Now())

	err := crud.GetURL.Validate(r)
	if common.Error(err) {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if !*crudRead {
		http.Error(w, "", http.StatusMethodNotAllowed)

		return
	}

	id, ba, err := func() (int, []byte, error) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if common.Error(err) {
			return 0, nil, err
		}

		record := new(T)

		if id != 0 {
			record, err = crud.Repository.Find(id)
			if common.Error(err) {
				return 0, nil, err
			}
		}

		ba, err := json.MarshalIndent(record, "", "    ")
		if common.Error(err) {
			return 0, nil, err
		}

		return id, ba, nil
	}()

	switch err {
	case nil:
		common.Info("%s GET: %d", crud.Name, id)

		common.Error(common.HTTPResponse(w, r, http.StatusOK, common.MimetypeApplicationJson.MimeType, len(ba), bytes.NewReader(ba)))
	case ErrNotFound:
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (crud *CRUD[T]) ListHandler(w http.ResponseWriter, r *http.Request) {
	common.DebugFunc()

	defer func(start time.Time) {
		crud.ListURL.UpdateStats(start)
	}(time.Now())

	err := crud.ListURL.Validate(r)
	if common.Error(err) {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if !*crudRead {
		http.Error(w, "", http.StatusMethodNotAllowed)

		return
	}

	ba, err := func() ([]byte, error) {
		offset, err := strconv.Atoi(crud.ListURL.ParamValue(r, "offset"))
		if common.Error(err) {
			return nil, err
		}

		limit, err := strconv.Atoi(crud.ListURL.ParamValue(r, "limit"))
		if common.Error(err) {
			return nil, err
		}

		records, err := crud.Repository.FindAll(offset, limit)
		if common.Error(err) {
			return nil, err
		}

		ba, err := json.MarshalIndent(&records, "", "    ")
		if common.Error(err) {
			return nil, err
		}

		return ba, nil
	}()

	switch err {
	case nil:
		common.Info("%s LIST", crud.Name)

		common.Error(common.HTTPResponse(w, r, http.StatusOK, common.MimetypeApplicationJson.MimeType, len(ba), bytes.NewReader(ba)))
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (crud *CRUD[T]) PutHandler(w http.ResponseWriter, r *http.Request) {
	common.DebugFunc()

	defer func(start time.Time) {
		crud.PutURL.UpdateStats(start)
	}(time.Now())

	err := crud.PutURL.Validate(r)
	if common.Error(err) {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if !*crudWrite {
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)

		return
	}

	var id int64

	id, err = func() (int64, error) {
		records, _, err := common.ReadBodyJSON[T](r.Body)
		if common.Error(err) {
			return 0, err
		}

		fieldID, err := common.GetStructValue(&records[0], "ID")
		if common.Error(err) {
			return 0, err
		}

		sqlFieldID := fieldID.Interface().(FieldInt64)

		id := sqlFieldID

		if r.PathValue("id") != "" {
			v, err := strconv.Atoi(r.PathValue("id"))
			if common.Error(err) {
				return 0, err
			}

			id.SetInt64(int64(v)) // #nosec G115

			if id.Int64() != sqlFieldID.Int64() {
				return 0, ErrModifiedID
			}
		}

		err = crud.Repository.Update(records[0])
		if common.Error(err) {
			return 0, err
		}

		return id.Int64(), nil
	}()

	switch err {
	case nil:
		common.Info("%s PUT: %d", crud.Name, id)

		w.WriteHeader(http.StatusOK)
	case ErrNotFound:
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (crud *CRUD[T]) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	common.DebugFunc()

	defer func(start time.Time) {
		crud.DeleteURL.UpdateStats(start)
	}(time.Now())

	err := crud.DeleteURL.Validate(r)
	if common.Error(err) {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if !*crudWrite {
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)

		return
	}

	var id int

	id, err = func() (int, error) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if common.Error(err) {
			return 0, err
		}

		err = crud.Repository.Delete(id)
		if common.Error(err) {
			return 0, err
		}

		return id, nil
	}()

	switch err {
	case nil:
		common.Info("%s DELETE: %d", crud.Name, id)

		w.WriteHeader(http.StatusOK)
	case ErrNotFound:
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}
