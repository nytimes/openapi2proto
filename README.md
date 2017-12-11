# openapi2proto [![Build Status](https://travis-ci.org/NYTimes/openapi2proto.svg?branch=master)](https://travis-ci.org/NYTimes/openapi2proto)

This tool will accept an OpenAPI/Swagger definition (yaml or JSON) and generate a Protobuf v3 schema and gRPC service definition from it.

## Install

To install, have Go installed with `$GOPATH/bin` on your `$PATH` and then:
```
go get -u github.com/NYTimes/openapi2proto/cmd/openapi2proto
```

## Run

There are 3 CLI flags for using the tool:
* `-spec` to point to the appropriate OpenAPI spec file
* `-options` to include google.api.http options for [grpc-gateway](https://github.com/gengo/grpc-gateway) users. This is disabled by default.
* `-out` to have the output written to a file rather than `Stdout. Defaults to `Stdout` if this is not specified`

## Protobuf Tags
* To allow for more control over how your protobuf schema evolves, all parameters and property definitions will accept an optional extension parameter, `x-proto-tag`, that will overide the generated tag with the value supplied.

## External Files
* Any externally referenced Open API spec will be fetched and inlined.
* Any externally referenced Protobuf files will be added as imports.
  * Example usage: `$ref: "google/protobuf/timestamp.proto#/google.protobuf.Timestamp"`

## Caveats

* Fields with scalar types that can also be "null" will get wrapped with one of the `google.protobuf.*Value` types.
* Fields with that have more than 1 type and the second type is not "null" will be replaced with the `google.protobuf.Any` type.
* Endpoints that respond with an array will be wrapped with a message type that has a single field, 'items', that contains the array.
* Only "200" and "201" responses are inspected for determining the expected return value for RPC endpoints.
* To prevent enum collisions and to match the [protobuf style guide](https://developers.google.com/protocol-buffers/docs/style#enums), enum values will be `CAPITALS_WITH_UNDERSCORES` and nested enum values and will have their parent types prepended.


## Example

```
╰─➤  openapi2proto -spec swagger.yaml -options
syntax = "proto3";

import "google/protobuf/empty.proto";

import "google/api/annotations.proto";

package swaggerpetstore;

message GetPetsRequest {
    // maximum number of results to return
    int32 limit = 1;
    // tags to filter by
    repeated string tags = 2;
}

message PostPetsRequest {
    // Pet to add to the store
    Pet pet = 1;
}

message GetPetsIdRequest {
    // ID of pet to fetch
    int64 id = 1;
}

message DeletePetsIdRequest {
    // ID of pet to delete
    int64 id = 1;
}

message Pet {
    int64 id = 1;
    string name = 2;
    string tag = 3;
}

message Pets {
    repeated Pet pets = 1;
}

service SwaggerPetstoreService {
    // Returns all pets from the system that the user has access to
    rpc GetPets(GetPetsRequest) returns (Pets) {
      option (google.api.http) = {
        get: "/api/pets"
      };
    }
    // Creates a new pet in the store.  Duplicates are allowed
    rpc PostPets(PostPetsRequest) returns (Pet) {
      option (google.api.http) = {
        post: "/api/pets"
        body: "pet"
      };
    }
    // Returns a user based on a single ID, if the user does not have access to the pet
    rpc GetPetsId(GetPetsIdRequest) returns (Pet) {
      option (google.api.http) = {
        get: "/api/pets/{id}"
      };
    }
    // deletes a single pet based on the ID supplied
    rpc DeletePetsId(DeletePetsIdRequest) returns (google.protobuf.Empty) {
      option (google.api.http) = {
        delete: "/api/pets/{id}"
      };
    }
}
```
