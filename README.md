# openapi2proto [![Build Status](https://travis-ci.org/NYTimes/openapi2proto.svg?branch=master)](https://travis-ci.org/NYTimes/openapi2proto)

This tool will accept an OpenAPI/Swagger definition (yaml or JSON) and generate a Protobuf v3 schema and gRPC service definition from it.

## Install

To install, have Go installed with `$GOPATH/bin` on your `$PATH` and then:
```
go get -u github.com/NYTimes/openapi2proto/cmd/openapi2proto
```

## Run

There are 2 CLI flags for using the tool: 
* `-spec` to point to the appropriate OpenAPI spec file
* `-options` to include google.api.http options for [grpc-gateway](https://github.com/gengo/grpc-gateway) users. This is disabled by default.

## Protobuf Tags
* To allow for more control over how your protobuf schema evolves, all parameters and property definitions will accept an optional extension parameter, `x-proto-tag`, that will overide the generated tag with the value supplied.

## External Files
* Any externally referenced Open API spec will be fetched and inlined.
* Any externally referenced Protobuf files will be added as imports.
  * Example usage: `$ref: "google/protobuf/timestamp.proto#/google.protobuf.Timestamp'`

## Caveats

* Fields with scalar types that can also be "null" will get wrapped with one of the `google.protobuf.*Value` types.
* Fields with that have more than 1 type and the second type is not "null" will be replaced with the `google.protobuf.Any` type. 
* Endpoints that respond with an array will be wrapped with a message type that has a single field, 'items', that contains the array.
* Only "200" and "201" responses are inspected for determining the expected return value for RPC endpoints.
* To prevent enum collisions and to match the [protobuf style guide](https://developers.google.com/protocol-buffers/docs/style#enums), enum values will be `CAPITALS_WITH_UNDERSCORES` and nested enum values and will have their parent types prepended.


## Example:
