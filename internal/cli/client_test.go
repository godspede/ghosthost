// internal/cli/client_test.go
package cli

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/godspede/ghosthost/internal/admin"
)

type stubCore struct{}

func (stubCore) Secret() string { return "s" }
func (stubCore) Share(admin.ShareRequest) (admin.SharePayload, error) {
	return admin.SharePayload{SchemaVersion: "1", ID: "id", Token: "tok", URL: "u"}, nil
}
func (stubCore) Revoke(string) error { return nil }
func (stubCore) Reshare(id string) (admin.SharePayload, error) {
	return admin.SharePayload{SchemaVersion: "1", ID: id}, nil
}
func (stubCore) List() admin.ListResponse     { return admin.ListResponse{SchemaVersion: "1"} }
func (stubCore) Status() admin.StatusResponse { return admin.StatusResponse{SchemaVersion: "1"} }
func (stubCore) Stop()                        {}
func (stubCore) Info(query string) (admin.InfoPayload, error) {
	return admin.InfoPayload{
		SharePayload: admin.SharePayload{SchemaVersion: "1", ID: "infoid", Token: "tok", URL: "u"},
		SrcPath:      "/abs/path",
	}, nil
}

func TestClient_Info(t *testing.T) {
	srv := httptest.NewServer(admin.NewHandler(stubCore{}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Secret: "s", HTTP: srv.Client()}
	got, err := c.Info(context.Background(), "abc/?&=+") // include chars that must be escaped
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "infoid" || got.SrcPath != "/abs/path" {
		t.Errorf("got %+v", got)
	}
}

func TestClient_Share(t *testing.T) {
	srv := httptest.NewServer(admin.NewHandler(stubCore{}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Secret: "s", HTTP: srv.Client()}
	p, err := c.Share(context.Background(), admin.ShareRequest{SrcPath: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "id" {
		t.Fatalf("got %+v", p)
	}
}
