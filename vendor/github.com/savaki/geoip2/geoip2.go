//	Copyright 2015 Matt Ho
//
//	Licensed under the Apache License, Version 2.0 (the "License");
//	you may not use this file except in compliance with the License.
//	You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
//	Unless required by applicable law or agreed to in writing, software
//	distributed under the License is distributed on an "AS IS" BASIS,
//	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	See the License for the specific language governing permissions and
//	limitations under the License.

package geoip2

import (
	"encoding/json"
	"net/http"

	"golang.org/x/net/context"
)

type Api struct {
	doFunc     func(ctx context.Context, req *http.Request) (*http.Response, error)
	userId     string
	licenseKey string
}

func New(userId, licenseKey string) *Api {
	api := &Api{
		userId:     userId,
		licenseKey: licenseKey,
	}
	return WithClient(api, http.DefaultClient)
}

func WithClient(api *Api, client *http.Client) *Api {
	return WithClientFunc(api, wrap(client.Do))
}

func WithClientFunc(api *Api, ctxFunc func(context.Context, *http.Request) (*http.Response, error)) *Api {
	return &Api{
		doFunc:     ctxFunc,
		userId:     api.userId,
		licenseKey: api.licenseKey,
	}
}

func wrap(doFunc func(*http.Request) (*http.Response, error)) func(context.Context, *http.Request) (*http.Response, error) {
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		return doFunc(req)
	}
}

func (a *Api) Country(ctx context.Context, ipAddress string) (Response, error) {
	return a.fetch(ctx, "https://geoip.maxmind.com/geoip/v2.1/country/", ipAddress)
}

func (a *Api) City(ctx context.Context, ipAddress string) (Response, error) {
	return a.fetch(ctx, "https://geoip.maxmind.com/geoip/v2.1/city/", ipAddress)
}

func (a *Api) Insights(ctx context.Context, ipAddress string) (Response, error) {
	return a.fetch(ctx, "https://geoip.maxmind.com/geoip/v2.1/insights/", ipAddress)
}

func (a *Api) fetch(ctx context.Context, prefix, ipAddress string) (Response, error) {
	req, err := http.NewRequest("GET", prefix+ipAddress, nil)
	if err != nil {
		return Response{}, err
	}

	// authorize the request
	// http://dev.maxmind.com/geoip/geoip2/web-services/#Authorization
	req.SetBasicAuth(a.userId, a.licenseKey)

	// execute the request
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := a.doFunc(ctx, req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	// handle errors that may occur
	// http://dev.maxmind.com/geoip/geoip2/web-services/#Response_Headers
	if resp.StatusCode >= 400 && resp.StatusCode < 600 {
		v := Error{}
		err := json.NewDecoder(resp.Body).Decode(&v)
		if err != nil {
			return Response{}, err
		}

		return Response{}, v
	}

	// parse the response body
	// http://dev.maxmind.com/geoip/geoip2/web-services/#Response_Body
	response := Response{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	return response, err
}
