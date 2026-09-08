package yandex

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)

func oauthServer(t *testing.T, status int, body string) (*httptest.Server, *url.Values) {
	t.Helper()
	got := &url.Values{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		*got = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

func TestExchangeSendsFormAndParsesToken(t *testing.T) {
	srv, got := oauthServer(t, 200,
		`{"access_token":"acc","refresh_token":"ref","expires_in":3600,"token_type":"bearer"}`)
	o := &OAuth{ClientID: "cid", ClientSecret: "sec", TokenURL: srv.URL,
		Now: func() time.Time { return fixedNow }}

	tok, err := o.Exchange(context.Background(), "1234567")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	want := url.Values{
		"grant_type": {"authorization_code"}, "code": {"1234567"},
		"client_id": {"cid"}, "client_secret": {"sec"},
	}
	if got.Encode() != want.Encode() {
		t.Errorf("form = %v, want %v", *got, want)
	}
	if tok.AccessToken != "acc" || tok.RefreshToken != "ref" {
		t.Errorf("token = %+v", tok)
	}
	if !tok.ExpiresAt.Equal(fixedNow.Add(time.Hour)) {
		t.Errorf("ExpiresAt = %v, want %v", tok.ExpiresAt, fixedNow.Add(time.Hour))
	}
}

func TestRefreshUsesRefreshGrant(t *testing.T) {
	srv, got := oauthServer(t, 200,
		`{"access_token":"acc2","refresh_token":"ref2","expires_in":10,"token_type":"bearer"}`)
	o := &OAuth{ClientID: "cid", ClientSecret: "sec", TokenURL: srv.URL}

	tok, err := o.Refresh(context.Background(), "ref1")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if got.Get("grant_type") != "refresh_token" || got.Get("refresh_token") != "ref1" {
		t.Errorf("form = %v", *got)
	}
	if tok.AccessToken != "acc2" || tok.RefreshToken != "ref2" {
		t.Errorf("token = %+v", tok)
	}
}

func TestExchangeReturnsOAuthError(t *testing.T) {
	srv, _ := oauthServer(t, 400, `{"error":"invalid_grant","error_description":"Code has expired"}`)
	o := &OAuth{ClientID: "cid", ClientSecret: "sec", TokenURL: srv.URL}

	_, err := o.Exchange(context.Background(), "bad")
	if err == nil || !strings.Contains(err.Error(), "invalid_grant") || !strings.Contains(err.Error(), "Code has expired") {
		t.Fatalf("err = %v, want invalid_grant with description", err)
	}
}

func TestExchangeRejectsNonJSON(t *testing.T) {
	srv, _ := oauthServer(t, 502, `<html>Bad gateway</html>`)
	o := &OAuth{ClientID: "cid", ClientSecret: "sec", TokenURL: srv.URL}

	_, err := o.Exchange(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("err = %v, want mention of HTTP 502", err)
	}
}

func TestLoginURL(t *testing.T) {
	o := &OAuth{ClientID: "cid"}
	want := "https://oauth.yandex.ru/authorize?client_id=cid&response_type=code"
	if got := o.LoginURL(); got != want {
		t.Errorf("LoginURL = %q, want %q", got, want)
	}
}
