package testutils

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"strings"
)

type MockDockerhubServer struct {
	Ts *httptest.Server
	ImageTag map[string]string
}

func SetUpMockDockerhubServer(imageTag map[string]string) *MockDockerhubServer {
	router := mux.NewRouter()

	router.HandleFunc("/v2/users/login", func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte(`{"token": "test token"}`))
	}).Methods("GET")

	router.HandleFunc("/v2/namespaces/{namespace}/repositories/{repo}/images/{digest}/tags",
		func(res http.ResponseWriter, req *http.Request) {
			vars := mux.Vars(req)
			digest := vars["digest"]

			if tag, ok := imageTag[digest]; ok {
				res.Write([]byte(fmt.Sprintf(`{"results": [{"tag": "%s", "is_current": true}]}`, tag)))
			} else {
				res.Write([]byte(`{"results": nil}`))
			}
		}).Methods("GET")

	router.HandleFunc("/v2/namespaces/{namespace}/repositories/{repo}/images",
		func(res http.ResponseWriter, req *http.Request) {
			var resSlice []string
			for image, tag := range imageTag {
				imageStr := fmt.Sprintf(`{"digest": "%s", "tags": ["tag": "%s", "is_current": true]}`, image, tag)
				resSlice = append(resSlice, imageStr)
			}
			resString := strings.Join(resSlice, ",")
			res.Write([]byte(fmt.Sprintf(`{"results": [%s]}`, resString)))
		})

	ts := httptest.NewServer(router)
	defer ts.Close()

	mds := MockDockerhubServer{
		Ts: ts,
		ImageTag: imageTag,
	}

	return &mds
}

func (mds *MockDockerhubServer) PushNewTag (tag string, image string) {
	mds.ImageTag[image] = tag
}
