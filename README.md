# openapi2proto

This tool will accept an OpenAPI/Swagger definition (yaml or JSON) and generate a Protobuf v3 schema from it.

## Install

To install, have Go installed with `$GOPATH/bin` on your `$PATH` and then:
```
go get -u github.com/NYTimes/openapi2proto/cmd/openapi2proto
```

## Run

There are 2 CLI flags for using the tool: 
* `-spec` to point to the appropriate OpenAPI spec file
* `-yaml` to specify the OpenAPI spec file format. (`-yaml=false` for JSON) 

Example:
```
╰─ openapi2proto -spec spec.yaml
syntax = "proto3";

package uberapi;

message Activities {
    int32 count = 1;
    repeated Activity history = 2;
    int32 limit = 3;
    int32 offset = 4;
}

message Activity {
    string uuid = 1;
}

message Error {
    int32 code = 1;
    string fields = 2;
    string message = 3;
}

message PriceEstimate {
    string currency_code = 1;
    string display_name = 2;
    string estimate = 3;
    int32 high_estimate = 4;
    int32 low_estimate = 5;
    string product_id = 6;
    int32 surge_multiplier = 7;
}

message Product {
    string capacity = 1;
    string description = 2;
    string display_name = 3;
    string image = 4;
    string product_id = 5;
}

message Profile {
    string email = 1;
    string first_name = 2;
    string last_name = 3;
    string picture = 4;
    string promo_code = 5;
}
```
