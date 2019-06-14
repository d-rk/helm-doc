package generator

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/random-dwi/helm-doc/output"
)

func Test_validateDefaultValues(t *testing.T) {
	type args struct {
		parentKey   string
		definitions string
		values      string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{name: "no_values", args: args{parentKey: "", definitions: "{}", values: "{}"}, want: nil},
		{name: "definition_missing", args: args{parentKey: "", definitions: `{}`, values: `{"key": 123}`}, want: []string{"key"}},
		{name: "definition_on_higher_level", args: args{parentKey: "", definitions: `{"parent": "docs"}`, values: `{"parent": {"child": 123}}`}, want: nil},
		{name: "definition_on_higher_level_inline", args: args{parentKey: "", definitions: `{"parent": "docs"}`, values: `{"parent.child": 123}`}, want: nil},
		{name: "definition_inline", args: args{parentKey: "", definitions: `{"parent.child": "docs"}`, values: `{"parent.child": 123}`}, want: nil},
		{name: "array_definition", args: args{parentKey: "", definitions: `{"parent": [{"child1": "doc1", "child2": "doc2"}]}`, values: `{"parent": []}`}, want: nil},
		{name: "array_definition_filled", args: args{parentKey: "", definitions: `{"parent": [{"child1": "doc1", "child2": "doc2"}]}`, values: `{"parent": [{"child1": 1, "child2": 2}]}`}, want: nil},
		{name: "array_definition_missing", args: args{parentKey: "", definitions: `{"parent": [{"child1": "doc1"}]}`, values: `{"parent": [{"child1": 1, "child2": 2}]}`}, want: []string{"parent[].child2"}},
		{name: "array_description", args: args{parentKey: "", definitions: `{"array": "array doc"}`, values: `{"array": [{"child1": 1, "child2": 2}]}`}, want: nil},
		{name: "string_array", args: args{parentKey: "", definitions: `{"array": "array doc"}`, values: `{"array": ["child1", "child2"]}`}, want: nil},
		{name: "string_array_missing", args: args{parentKey: "", definitions: `{}`, values: `{"array": ["child1", "child2"]}`}, want: []string{"array"}},
		{name: "int_array_missing", args: args{parentKey: "", definitions: `{}`, values: `{"array": [1,2,3]}`}, want: []string{"array"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateDefaultValues(tt.args.parentKey, parseJson(tt.args.definitions), parseJson(tt.args.values)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validateDefaultValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_findValueForKey(t *testing.T) {
	type args struct {
		globalKey      string
		values         string
		useParentValue bool
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{name: "not_found", args: args{globalKey: "global.secret", values: "{}"}, want: nil},
		{name: "find_int", args: args{globalKey: "global.secret", values: `{"global": {"secret": 123}}`}, want: 123.0},
		{name: "find_string", args: args{globalKey: "global.secret", values: `{"global": {"secret": "expected"}}`}, want: "expected"},
		{name: "find_inline", args: args{globalKey: "global.secret", values: `{"global.secret": "expected"}`}, want: "expected"},
		{name: "find_inline_complex", args: args{globalKey: "global.secret", values: `{"global": {}, "global.secret": "expected"}`}, want: "expected"},
		{name: "find_inline_complex2", args: args{globalKey: "global.secret.value", values: `{"global": {"secret.value": "expected"}}`}, want: "expected"},
		{name: "find_array", args: args{globalKey: "array[].secret.value", values: `{"array": [{"secret.value": "expected"}]}`}, want: "expected"},
		{name: "find_array_parent", args: args{globalKey: "array[].child", values: `{"array": "docs"}`, useParentValue: true}, want: "docs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findValueForKey(tt.args.globalKey, parseJson(tt.args.values), tt.args.useParentValue); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findValueForKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mergeValues(t *testing.T) {
	type args struct {
		defaultParent interface{}
		defaultChild  interface{}
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{name: "test_parent_nil", args: args{defaultParent: nil, defaultChild: parseJson(`{"global": "child"}`)}, want: parseJson(`{"global": "child"}`)},
		{name: "test_string_merge", args: args{defaultParent: parseJson(`{"global": "parent"}`), defaultChild: parseJson(`{"global": "child"}`)}, want: parseJson(`{"global": "parent"}`)},
		{name: "test_string_overwrite", args: args{defaultParent: `"global"`, defaultChild: parseJson(`{"global": "child"}`)}, want: `"global"`},
		{name: "test_map_overwrite", args: args{defaultParent: parseJson(`{"global": "parent"}`), defaultChild: `"somestring"`}, want: parseJson(`{"global": "parent"}`)},
		{name: "test_map_merge", args: args{defaultParent: parseJson(`{"global": {"hello": "parent"}}`), defaultChild: parseJson(`{"global": {"hello": "child", "other": "value"}}`)}, want: parseJson(`{"global": {"hello": "parent", "other": "value"}}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeValues(tt.args.defaultParent, tt.args.defaultChild); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func parseJson(value string) map[string]interface{} {
	valueMap := map[string]interface{}{}

	if err := json.Unmarshal([]byte(value), &valueMap); err != nil {
		output.Failf("error parsing json: %v", err)
	}

	return valueMap
}
