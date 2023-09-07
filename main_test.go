package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	_ "net/http/pprof"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_getData(t *testing.T) {
	type args struct {
		resp   string
		status int
	}
	tests := []struct {
		name    string
		args    args
		want    *Response
		wantErr bool
	}{
		{
			name: "okmetrics",
			args: args{
				resp:   `{ "kitty": { "id": "1", "ownerId": "1", "ownerFirstName": "John", "ownerLastName": "Doe", "contributionsCounter": 1, "totalCollectedAmount": 100 }, "available": 1 }`,
				status: http.StatusOK,
			},
			want:    &Response{Kitty: Kitty{ID: "1", OwnerID: "1", OwnerFirstName: "John", OwnerLastName: "Doe", ContributionsCounter: 1, TotalCollectedAmount: 100}, Available: 1},
			wantErr: false,
		}, {
			name: "statusError",
			args: args{
				resp:   ``,
				status: http.StatusServiceUnavailable,
			},
			want:    &Response{},
			wantErr: true,
		}, {
			name: "jsonError",
			args: args{
				resp:   `ahhaha [ crappy { json`,
				status: http.StatusOK,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.args.status)
				w.Write([]byte(tt.args.resp))
			}))
			r, err := getData(context.TODO(), server.URL)
			server.Close()

			assert.False(t, (err != nil) != tt.wantErr)
			if !tt.wantErr {
				assert.Equal(t, r.Kitty.ID, tt.want.Kitty.ID)
				assert.Equal(t, r.Kitty.OwnerID, tt.want.Kitty.OwnerID)
				assert.Equal(t, r.Kitty.OwnerFirstName, tt.want.Kitty.OwnerFirstName)
				assert.Equal(t, r.Kitty.OwnerLastName, tt.want.Kitty.OwnerLastName)
				assert.Equal(t, r.Kitty.ContributionsCounter, tt.want.Kitty.ContributionsCounter)
				assert.Equal(t, r.Kitty.TotalCollectedAmount, tt.want.Kitty.TotalCollectedAmount)
				assert.Equal(t, r.Available, 1)
			}
		})
	}
}

func Test_registerPrometheusMetrics(t *testing.T) {
	t.Run("register", func(t *testing.T) {
		got, err := registerPrometheusMetrics()
		assert.NoError(t, err)
		assert.NotNil(t, got.contributionsCounterGauge)
		assert.NotNil(t, got.totalCollectedAmountGauge)
	})
}

func Test_parseConfig(t *testing.T) {
	type args struct {
		uuid  string
		port  string
		delay string
	}
	tests := []struct {
		name    string
		args    args
		want    *config
		wantErr bool
	}{
		{
			name:    "okconfig",
			want:    &config{url: "https://api.lyf.eu/public/api/kitties/6666", delay: 66 * time.Second, port: 1234},
			wantErr: false,
			args: args{
				uuid:  "6666",
				port:  "1234",
				delay: "66s",
			},
		},
		{
			name:    "errNonNumericPort",
			want:    &config{},
			wantErr: true,
			args: args{
				uuid:  "6666",
				port:  "azerty",
				delay: "66",
			},
		},
		{
			name:    "errNoTimeExtension",
			want:    &config{url: "https://api.lyf.eu/public/api/kitties/6666", delay: 66 * time.Second, port: 1234},
			wantErr: true,
			args: args{
				uuid:  "6666",
				port:  "1234",
				delay: "66",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("LYF_KITTY_UUID", tt.args.uuid)
			os.Setenv("LYF_PORT", tt.args.port)
			os.Setenv("LYF_DELAY", tt.args.delay)
			got, err := parseConfig()

			// the assert way
			assert.False(t, (err != nil) != tt.wantErr)
			if !tt.wantErr {
				assert.Equal(t, got, tt.want)
			}

			// the stdlib way
			// if (err != nil) != tt.wantErr {
			// 	t.Errorf("parseConfig() error = %v, wantErr %v", err, tt.wantErr)
			// 	return
			// }
			// if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
			// 	t.Errorf("parseConfig() = %v, want %v", got, tt.want)
			// 	return
			// }
		})
	}
}
