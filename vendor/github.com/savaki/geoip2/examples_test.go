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

package geoip2_test

import (
	"encoding/json"
	"os"
	"time"

	"github.com/savaki/geoip2"
	"golang.org/x/net/context"
)

func ExampleApi_City() {
	userId := os.Getenv("MAXMIND_USER_ID")
	licenseKey := os.Getenv("MAXMIND_LICENSE_KEY")
	api := geoip2.New(userId, licenseKey)

	resp, _ := api.City(nil, "1.2.3.4")
	json.NewEncoder(os.Stdout).Encode(resp)
}

func ExampleApi_Country() {
	userId := os.Getenv("MAXMIND_USER_ID")
	licenseKey := os.Getenv("MAXMIND_LICENSE_KEY")
	api := geoip2.New(userId, licenseKey)

	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	resp, _ := api.Country(ctx, "1.2.3.4")
	json.NewEncoder(os.Stdout).Encode(resp)
}

func ExampleApi_Insights() {
	userId := os.Getenv("MAXMIND_USER_ID")
	licenseKey := os.Getenv("MAXMIND_LICENSE_KEY")
	api := geoip2.New(userId, licenseKey)

	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	resp, _ := api.Insights(ctx, "1.2.3.4")
	json.NewEncoder(os.Stdout).Encode(resp)
}
