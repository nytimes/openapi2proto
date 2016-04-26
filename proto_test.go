package openapi2proto

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestGenerateProto(t *testing.T) {
	tests := []struct {
		yaml             bool
		givenFixturePath string

		wantProto string
	}{
		{
			false,
			"fixtures/semantic_api.json",

			testSemanticWant,
		},
		{
			false,
			"fixtures/most_popular.json",

			testMostPopWant,
		},
		{
			true,
			"fixtures/spec.yaml",

			testUberProtoWant,
		},
		{
			false,
			"fixtures/spec.json",

			testUberProtoWant,
		},
	}

	for _, test := range tests {

		testSpec, err := ioutil.ReadFile(test.givenFixturePath)
		if err != nil {
			t.Fatalf("unable to open test fixture: ", err)
		}

		var testAPI APIDefinition
		if test.yaml {
			err = yaml.Unmarshal(testSpec, &testAPI)
			if err != nil {
				t.Fatalf("unable to unmarshal text fixture into APIDefinition: ", err)
			}
		} else {
			err = json.Unmarshal(testSpec, &testAPI)
			if err != nil {
				t.Fatalf("unable to unmarshal text fixture into APIDefinition: ", err)
			}

		}

		protoResult, err := GenerateProto(&testAPI)
		if err != nil {
			t.Fatalf("unable to generate protobuf from APIDefinition: ", err)
		}

		if test.wantProto != string(protoResult) {
			t.Errorf("testYaml expected:\n%q\nGOT:\n%s", test.wantProto, []byte(protoResult))
		}
	}
}

const testUberProtoWant = `syntax = "proto3";

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
`

const testMostPopWant = `syntax = "proto3";
import "google/protobuf/any.proto";

package themostpopularapi;

message Article {
    string abstract = 1;
    string byline = 2;
    string column = 3;
    DesFacet des_facet = 4;
    GeoFacet geo_facet = 5;
    google.protobuf.Any media = 6;
    OrgFacet org_facet = 7;
    PerFacet per_facet = 8;
    string published_date = 9;
    string section = 10;
    string source = 11;
    string title = 12;
    string url = 13;
}

message ArticleWithCountType {
    string abstract = 1;
    string byline = 2;
    string column = 3;
    string count_type = 4;
    DesFacet des_facet = 5;
    GeoFacet geo_facet = 6;
    message Media {
        string caption = 1;
        string copyright = 2;
        message Media_metadata {
            string format = 1;
            int32 height = 2;
            string url = 3;
            int32 width = 4;
        }
        Media_metadata media_metadata = 3;
        string subtype = 4;
        string type = 5;
    }
    repeated Media media = 7;
    OrgFacet org_facet = 8;
    PerFacet per_facet = 9;
    string published_date = 10;
    string section = 11;
    string source = 12;
    string title = 13;
    string url = 14;
}

message DesFacet {
}

message GeoFacet {
}

message OffSet {
}

message OrgFacet {
}

message PerFacet {
}

message Section {
}

message SharedTypes {
}

message TimePeriod {
}
`
const testSemanticWant = `syntax = "proto3";

package thesemanticapi;

message Concept {
    repeated ConceptRelation ancestors = 1;
    message Article_list {
        message Result {
            string body = 1;
            string byline = 2;
            message Concepts {
                repeated string nytd_des = 1;
                repeated string nytd_org = 2;
                repeated string nytd_per = 3;
            }
            Concepts concepts = 3;
            string date = 4;
            string document_type = 5;
            string title = 6;
            string type_of_material = 7;
            string url = 8;
        }
        repeated Result results = 1;
        int32 total = 2;
    }
    Article_list article_list = 2;
    message Combination {
        string combination_note = 1;
        int32 combination_source_concept_id = 2;
        string combination_source_concept_name = 3;
        string combination_source_concept_type = 4;
        int32 combination_target_concept_id = 5;
        string combination_target_concept_name = 6;
        string combination_target_concept_type = 7;
    }
    repeated Combination combinations = 3;
    string concept_created = 4;
    int32 concept_id = 5;
    string concept_name = 6;
    string concept_status = 7;
    string concept_type = 8;
    string concept_updated = 9;
    repeated ConceptRelation descendants = 10;
    int32 is_times_tag = 11;
    message Link {
        int32 concept_id = 1;
        string concept_name = 2;
        string concept_status = 3;
        string concept_type = 4;
        int32 is_times_tag = 5;
        string link = 6;
        int32 link_id = 7;
        string link_type = 8;
        string mapping_type = 9;
        string relation = 10;
    }
    repeated Link links = 12;
    message Scope_note {
        string scope_note = 1;
        string scope_note_name = 2;
        string scope_note_type = 3;
    }
    repeated Scope_note scope_notes = 13;
    string search_api_query = 14;
    message Taxonomy {
        int32 source_concept_id = 1;
        string source_concept_name = 2;
        string source_concept_type = 3;
        string source_concept_vernacular = 4;
        int32 target_concept_id = 5;
        string target_concept_name = 6;
        string target_concept_type = 7;
        string target_concept_vernacular = 8;
        string taxonomic_relation = 9;
        string taxonomic_verification_status = 10;
    }
    repeated Taxonomy taxonomy = 15;
    string vernacular = 16;
}

message ConceptRelation {
    enum Class {
        CLASS_NYTD_GEO = 0;
        CLASS_NYTD_PER = 1;
        CLASS_NYTD_ORG = 2;
        CLASS_NYTD_DES = 3;
    }
    Class class = 1;
    string concept_created = 2;
    int32 concept_id = 3;
    string concept_name = 4;
    string concept_status = 5;
    string concept_type = 6;
    string concept_updated = 7;
    int32 is_times_tag = 8;
    string vernacular = 9;
}

message TestModel {
    enum Category {
        CATEGORY_NYTD_GEO = 0;
        CATEGORY_NYTD_PER = 1;
        CATEGORY_NYTD_ORG = 2;
        CATEGORY_NYTD_DES = 3;
    }
    Category category = 1;
    message Class {
        string something = 1;
    }
    repeated Class class = 2;
    bool test_bool = 3;
}
`
