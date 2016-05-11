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
* `-yaml` to specify the OpenAPI spec file format. (`-yaml=false` for JSON)
* `-options` to include google.api.http options for [grpc-gateway](https://github.com/gengo/grpc-gateway) users. This is disabled by default.

## Caveats

* Fields with scalar types that can also be "null" will get wrapped with one of the `google.protobuf.*Value` types.
* Fields with that have more than 1 type and the second type is not "null" will be replaced with the `google.protobuf.Any` type. 
* Endpoints that respond with an array will be wrapped with a message type that has a single field, 'items', that contains the array.
* Only "200" and "201" responses are inspected for determining the expected return value for RPC endpoints.
* To prevent enum collisions and to match language conventions, enum values will be upper-cased and nested enum values and will have their parent types prepended.

## Example
```
╰─ openapi2proto -spec books.json -yaml=false
syntax = "proto3";

import "google/protobuf/struct.proto";

package booksapi;

message GetListsformatRequest {
    string bestsellers_date = 1;
    string date = 2;
    enum GetListsformatRequest_Format {
        GETLISTSFORMATREQUEST_FORMAT_JSON = 0;
        GETLISTSFORMATREQUEST_FORMAT_JSONP = 1;
    }
    GetListsformatRequest_Format format = 3;
    string isbn = 4;
    string list = 5;
    int32 offset = 6;
    string published_date = 7;
    int32 rank = 8;
    int32 rank_last_week = 9;
    enum GetListsformatRequest_Sort_order {
        GETLISTSFORMATREQUEST_SORT_ORDER_ASC = 0;
        GETLISTSFORMATREQUEST_SORT_ORDER_DESC = 1;
    }
    GetListsformatRequest_Sort_order sort_order = 10;
    int32 weeks_on_list = 11;
}

message GetListsformatResponse {
    string copyright = 1;
    string last_modified = 2;
    int32 num_results = 3;
    message Result {
        string amazon_product_url = 1;
        int32 asterisk = 2;
        string bestsellers_date = 3;
        message Book_detail {
            string age_group = 1;
            string author = 2;
            string contributor = 3;
            string contributor_note = 4;
            string description = 5;
            int32 price = 6;
            string primary_isbn10 = 7;
            string primary_isbn13 = 8;
            string publisher = 9;
            string title = 10;
        }
        repeated Book_detail book_details = 4;
        int32 dagger = 5;
        string display_name = 6;
        message Isbn {
            string isbn10 = 1;
            string isbn13 = 2;
        }
        repeated Isbn isbns = 7;
        string list_name = 8;
        string published_date = 9;
        int32 rank = 10;
        int32 rank_last_week = 11;
        message Review {
            string article_chapter_link = 1;
            string book_review_link = 2;
            string first_chapter_link = 3;
            string sunday_review_link = 4;
        }
        repeated Review reviews = 12;
        int32 weeks_on_list = 13;
    }
    repeated Result results = 4;
    string status = 5;
}

message GetListsBestSellersHistoryRequest {
    string age_group = 1;
    string author = 2;
    string contributor = 3;
    string isbn = 4;
    string price = 5;
    string publisher = 6;
    string title = 7;
}

message GetListsBestSellersHistoryResponse {
    string copyright = 1;
    int32 num_results = 2;
    message Result {
        string age_group = 1;
        string author = 2;
        string contributor = 3;
        string contributor_note = 4;
        string description = 5;
        message Isbn {
            string isbn10 = 1;
            string isbn13 = 2;
        }
        repeated Isbn isbns = 6;
        int32 price = 7;
        string publisher = 8;
        message Ranks_history {
            int32 asterisk = 1;
            string bestsellers_date = 2;
            int32 dagger = 3;
            string display_name = 4;
            string list_name = 5;
            string primary_isbn10 = 6;
            string primary_isbn13 = 7;
            string published_date = 8;
            int32 rank = 9;
            google.protobuf.NullValue ranks_last_week = 10;
            int32 weeks_on_list = 11;
        }
        repeated Ranks_history ranks_history = 9;
        message Review {
            string article_chapter_link = 1;
            string book_review_link = 2;
            string first_chapter_link = 3;
            string sunday_review_link = 4;
        }
        repeated Review reviews = 10;
        string title = 11;
    }
    repeated Result results = 3;
    string status = 4;
}

message GetListsNamesformatRequest {
    string api_key = 1;
    enum GetListsNamesformatRequest_Format {
        GETLISTSNAMESFORMATREQUEST_FORMAT_JSON = 0;
        GETLISTSNAMESFORMATREQUEST_FORMAT_JSONP = 1;
    }
    GetListsNamesformatRequest_Format format = 2;
}

message GetListsNamesformatResponse {
    string copyright = 1;
    int32 num_results = 2;
    message Result {
        string display_name = 1;
        string list_name = 2;
        string list_name_encoded = 3;
        string newest_published_date = 4;
        string oldest_published_date = 5;
        string updated = 6;
    }
    repeated Result results = 3;
    string status = 4;
}

message GetListsOverviewformatRequest {
    string api_key = 1;
    enum GetListsOverviewformatRequest_Format {
        GETLISTSOVERVIEWFORMATREQUEST_FORMAT_JSON = 0;
        GETLISTSOVERVIEWFORMATREQUEST_FORMAT_JSONP = 1;
    }
    GetListsOverviewformatRequest_Format format = 2;
    string published_date = 3;
}

message GetListsOverviewformatResponse {
    string copyright = 1;
    int32 num_results = 2;
    message Results {
        string bestsellers_date = 1;
        message List {
            message Book {
                string age_group = 1;
                string author = 2;
                string contributor = 3;
                string contributor_note = 4;
                string created_date = 5;
                string description = 6;
                int32 price = 7;
                string primary_isbn10 = 8;
                string primary_isbn13 = 9;
                string publisher = 10;
                int32 rank = 11;
                string title = 12;
                string updated_date = 13;
            }
            repeated Book books = 1;
            string display_name = 2;
            int32 list_id = 3;
            string list_image = 4;
            string list_name = 5;
            string updated = 6;
        }
        repeated List lists = 2;
        string published_date = 3;
    }
    Results results = 3;
    string status = 4;
}

message GetListsDateListRequest {
    string bestsellers_date = 1;
    string date = 2;
    int32 isbn = 3;
    string list = 4;
    string list_name = 5;
    int32 offset = 6;
    string published_date = 7;
    string rank = 8;
    int32 rank_last_week = 9;
    enum GetListsDateListRequest_Sort_order {
        GETLISTSDATELISTREQUEST_SORT_ORDER_ASC = 0;
        GETLISTSDATELISTREQUEST_SORT_ORDER_DESC = 1;
    }
    GetListsDateListRequest_Sort_order sort_order = 10;
    int32 weeks_on_list = 11;
}

message GetListsDateListResponse {
    string copyright = 1;
    string last_modified = 2;
    int32 num_results = 3;
    message Results {
        string bestsellers_date = 1;
        message Book {
            string age_group = 1;
            string amazon_product_url = 2;
            string article_chapter_link = 3;
            int32 asterisk = 4;
            string author = 5;
            string book_image = 6;
            string book_review_link = 7;
            string contributor = 8;
            string contributor_note = 9;
            int32 dagger = 10;
            string description = 11;
            string first_chapter_link = 12;
            message Isbn {
                string isbn10 = 1;
                string isbn13 = 2;
            }
            repeated Isbn isbns = 13;
            int32 price = 14;
            string primary_isbn10 = 15;
            string primary_isbn13 = 16;
            string publisher = 17;
            int32 rank = 18;
            int32 rank_last_week = 19;
            string sunday_review_link = 20;
            string title = 21;
            int32 weeks_on_list = 22;
        }
        repeated Book books = 2;
        message Correction {
        }
        repeated Correction corrections = 3;
        string display_name = 4;
        string list_name = 5;
        int32 normal_list_ends_at = 6;
        string published_date = 7;
        string updated = 8;
    }
    Results results = 4;
    string status = 5;
}

message GetReviewsformatRequest {
    string api_key = 1;
    string author = 2;
    enum GetReviewsformatRequest_Format {
        GETREVIEWSFORMATREQUEST_FORMAT_JSON = 0;
        GETREVIEWSFORMATREQUEST_FORMAT_JSONP = 1;
    }
    GetReviewsformatRequest_Format format = 3;
    int32 isbn = 4;
    string title = 5;
}

message GetReviewsformatResponse {
    string copyright = 1;
    int32 num_results = 2;
    message Result {
        string book_author = 1;
        string book_title = 2;
        string byline = 3;
        repeated string isbn13 = 4;
        string publication_dt = 5;
        string summary = 6;
        string url = 7;
    }
    repeated Result results = 3;
    string status = 4;
}

service BooksAPIService {
    rpc GetListsformat(GetListsformatRequest) returns (GetListsformatResponse) {}
    rpc GetListsBestSellersHistory(GetListsBestSellersHistoryRequest) returns (GetListsBestSellersHistoryResponse) {}
    rpc GetListsNamesformat(GetListsNamesformatRequest) returns (GetListsNamesformatResponse) {}
    rpc GetListsOverviewformat(GetListsOverviewformatRequest) returns (GetListsOverviewformatResponse) {}
    rpc GetListsDateList(GetListsDateListRequest) returns (GetListsDateListResponse) {}
    rpc GetReviewsformat(GetReviewsformatRequest) returns (GetReviewsformatResponse) {}
}
```
