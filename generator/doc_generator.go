package generator

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/random-dwi/helm-doc/helm"
	"github.com/random-dwi/helm-doc/output"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"regexp"
	"strings"
)

type CommandFlags struct {
	Verbose            bool
	VerifyExamples     bool
	VerifyValues       bool
	VerifyDependencies bool
	Version            string
	RepoURL            string
	Username           string
	Password           string
	Keyring            string
	CertFile           string
	KeyFile            string
	CaFile             string
	Verify             bool
	Devel              bool
}

type ConfigDoc struct {
	Description  string
	DefaultValue interface{}
	ExampleValue interface{}
}

func GenerateDocs(c *chart.Chart, ignoredPrefixes []string, parentCharts map[*chart.Chart]*chart.Chart, flags CommandFlags) (map[string]*ConfigDoc, error) {

	var allValues = make(map[string]map[string]interface{})
	var valueSource []string

	var currentChart = c
	var currentPrefix = ""

	for currentChart != nil {
		valueSource = append(valueSource, currentChart.Metadata.Name)

		var rawValues []byte
		if currentChart.Values == nil {
			output.Debugf("chart has no values.yaml")
			rawValues = []byte("")
		} else {
			rawValues = []byte(currentChart.Values.Raw)
		}

		values, err := parseYaml(rawValues)

		if err != nil {
			return nil, fmt.Errorf("unable to read values for %s:%s: %v", currentChart.Metadata.Name, currentChart.Metadata.Version, err)
		}

		if currentPrefix != "" {
			val := findValueForKeyAndGlobal(currentPrefix, values, values["global"])
			values, _ = val.(map[string]interface{})
			currentPrefix = currentChart.Metadata.Name + "." + currentPrefix
		} else {
			currentPrefix = currentChart.Metadata.Name
		}

		allValues[currentChart.Metadata.Name] = values
		currentChart = parentCharts[currentChart]
	}

	definitions, err := findAndParseYaml(c.Files, "definitions.yaml")

	if err != nil {
		return nil, fmt.Errorf("unable to read definitions for %s:%s: %v", c.Metadata.Name, c.Metadata.Version, err)
	}

	examples, err := findAndParseYaml(c.Files, "examples.yaml")

	if err != nil {
		if flags.VerifyExamples {
			return nil, fmt.Errorf("unable to read examples for %s:%s: %v", c.Metadata.Name, c.Metadata.Version, err)
		} else {
			output.Warnf("unable to read examples for %s:%s: %v", c.Metadata.Name, c.Metadata.Version, err)
		}
	}

	return generate(definitions, allValues, valueSource, examples, ignoredPrefixes, flags), nil
}

func generate(definitions map[string]interface{}, allValues map[string]map[string]interface{}, valueSource []string, examples map[string]interface{}, ignoredPrefixes []string, flags CommandFlags) map[string]*ConfigDoc {

	docs := convertToConfigDocs("", definitions)

	if len(ignoredPrefixes) > 0 {
		for _, source := range valueSource {
			for key := range allValues[source] {
				if containsString(ignoredPrefixes, key) {
					delete(allValues[source], key)
				}
			}
		}
	}

	if flags.VerifyValues {
		missingKeys := validateDefaultValues("", definitions, allValues[valueSource[0]])
		if len(missingKeys) > 0 {
			var prefix = "\n\t"
			output.Failf("undocumented values detected: %s%s", prefix, strings.Join(missingKeys, prefix))
		}
	}

	docs = insertDefaultValues(docs, allValues, valueSource)

	if examples != nil {
		docs = insertExampleValues(docs, examples, flags)
	}

	return docs
}

func convertToConfigDocs(parentKey string, definitions map[string]interface{}) map[string]*ConfigDoc {

	var descriptions = map[string]*ConfigDoc{}

	for key, value := range definitions {

		var globalKey = key

		if parentKey != "" {
			globalKey = parentKey + "." + key
		}

		defMap, isMap := value.(map[string]interface{})
		if !isMap {
			description, isString := value.(string)
			if isString {
				descriptions[globalKey] = &ConfigDoc{Description: description}
			} else {
				defArray, isArray := value.([]interface{})
				if isArray {
					if len(defArray) == 1 {
						subDescriptions := convertToConfigDocs(globalKey+"[]", defArray[0].(map[string]interface{}))
						for k, v := range subDescriptions {
							descriptions[k] = v
						}
					} else {
						output.Failf("definition can only be array with length 1: %s (value: %v)", globalKey, value)
					}
				} else {
					output.Failf("definition has to be either a map or a string: %s (value: %v)", globalKey, value)
				}
			}
		} else {
			subDescriptions := convertToConfigDocs(globalKey, defMap)
			for k, v := range subDescriptions {
				descriptions[k] = v
			}
		}
	}

	return descriptions
}

func validateDefaultValues(parentKey string, definitions map[string]interface{}, values map[string]interface{}) []string {

	var missingKeys []string

	for key, value := range values {

		var globalKey = key

		if parentKey != "" {
			globalKey = parentKey + "." + key
		}

		valMap, isMap := value.(map[string]interface{})
		if isMap && len(valMap) > 0 {
			missingKeys = append(missingKeys, validateDefaultValues(globalKey, definitions, valMap)...)
			continue
		}

		valArray, isArray := value.([]interface{})
		if isArray {
			for _, row := range valArray {
				rowMap, isMap := row.(map[string]interface{})
				if isMap {
					missingKeys = append(missingKeys, validateDefaultValues(globalKey+"[]", definitions, rowMap)...)
				} else {
					// we have an array with elements that are not maps, so we need a doc for the array itself
					if !findDefinitionForKeyOrParentKey(globalKey, definitions) {
						foundKey := false
						for _, key := range missingKeys {
							if key == globalKey {
								foundKey = true
								break
							}
						}
						if !foundKey {
							missingKeys = append(missingKeys, globalKey)
						}
					}
				}
			}
			continue
		}

		if findValueForKey(globalKey, definitions, true) == nil {
			missingKeys = append(missingKeys, globalKey)
			continue
		}
	}

	return missingKeys
}

func insertDefaultValues(docs map[string]*ConfigDoc, allValues map[string]map[string]interface{}, valueSource []string) map[string]*ConfigDoc {

	for globalKey, configDoc := range docs {
		var defaultValue interface{} = nil
		for _, source := range valueSource {
			defaultValue = mergeValues(findValueForKey(globalKey, allValues[source], false), defaultValue)
		}
		configDoc.DefaultValue = defaultValue
	}

	return docs
}

func mergeValues(defaultParent interface{}, defaultChild interface{}) interface{} {
	parentMap, parentIsMap := defaultParent.(map[string]interface{})
	childMap, childIsMap := defaultChild.(map[string]interface{})
	if !parentIsMap || !childIsMap {
		// parent overwrites child
		if defaultParent != nil {
			return defaultParent
		} else {
			return defaultChild
		}
	} else {
		return helm.MergeValues(childMap, parentMap)
	}
}

func insertExampleValues(docs map[string]*ConfigDoc, examples map[string]interface{}, flags CommandFlags) map[string]*ConfigDoc {

	var missingExamples []string

	for globalKey, configDoc := range docs {
		configDoc.ExampleValue = findValueForKey(globalKey, examples, false)
		if flags.VerifyExamples && configDoc.ExampleValue == nil && configDoc.DefaultValue == nil {
			missingExamples = append(missingExamples, globalKey)
		}
	}

	if missingExamples != nil {
		var prefix = "\n\t"
		output.Failf("when --verify-examples is true an example needs to be provided for every config without default: %s%s", prefix, strings.Join(missingExamples, prefix))
	}

	return docs
}

// find definition for a given key or a parent key
func findDefinitionForKeyOrParentKey(globalKey string, definitions map[string]interface{}) bool {

	keys := strings.Split(globalKey, ".")
	currentMap := definitions

	for _, key := range keys {
		baseKey, isArray := isArrayKey(key)
		if _, exists := currentMap[baseKey]; exists {
			if !isArray {
				newMap, isMap := currentMap[baseKey].(map[string]interface{})
				if isMap {
					currentMap = newMap
				} else {
					_, isString := currentMap[baseKey].(string)
					return isString
				}
			} else {
				// it is an array
				newArray, isDefArray := currentMap[baseKey].([]interface{})
				if isDefArray {
					if len(newArray) != 1 {
						output.Failf("only arrays with length 1 allowed in definition: %s", globalKey)
					} else {
						newMap, isMap := newArray[0].(map[string]interface{})
						if isMap {
							currentMap = newMap
						} else {
							_, isString := currentMap[baseKey].(string)
							return isString
						}
					}
				} else {
					_, isString := currentMap[baseKey].(string)
					return isString
				}
			}
		} else {
			return false
		}
	}

	return false
}

func findValueForKeyAndGlobal(globalKey string, values map[string]interface{}, global interface{}) interface{} {

	keys := strings.Split(globalKey, ".")

	for i := range keys {
		leftMostKeys := keys[:len(keys)-i]
		joinedKey := strings.Join(leftMostKeys, ".")

		if subValues, exists := values[joinedKey]; exists {
			subKey := strings.TrimPrefix(strings.TrimPrefix(globalKey, joinedKey), ".")

			newMap, isMap := subValues.(map[string]interface{})
			if isMap {
				if subKey == "" {
					newMap["global"] = global
					return newMap
				} else {
					return findValueForKeyAndGlobal(subKey, newMap, global)
				}
			} else {
				globalMap := make(map[string]interface{})
				globalMap["global"] = global
				return globalMap
			}
		}
	}

	return nil
}

// find value for a given key or nil if it does not exist
// if `useParentValue` is true, instead of nil the parent value is returned if available
func findValueForKey(globalKey string, values map[string]interface{}, useParentValue bool) interface{} {

	keys := strings.Split(globalKey, ".")

	for i := range keys {
		leftMostKeys := keys[:len(keys)-i]
		joinedKey := strings.Join(leftMostKeys, ".")

		baseKey, isArray := isArrayKey(joinedKey)

		if subValues, exists := values[baseKey]; exists {
			subKey := strings.TrimPrefix(strings.TrimPrefix(globalKey, joinedKey), ".")

			if subKey == "" {
				return subValues
			} else if !isArray {
				newMap, isMap := subValues.(map[string]interface{})
				if isMap {
					return findValueForKey(subKey, newMap, useParentValue)
				} else {
					// we cannot go deeper but we have not found the full key
					if useParentValue {
						return subValues
					} else {
						return nil
					}
				}
			} else {
				subArray, isArray := subValues.([]interface{})
				if isArray {
					if len(subArray) == 0 {
						return nil
					} else {
						newMap, isMap := subArray[0].(map[string]interface{})
						if isMap {
							return findValueForKey(subKey, newMap, useParentValue)
						} else {
							// we cannot go deeper but we have not found the full key
							if useParentValue {
								return subArray[0]
							} else {
								return nil
							}
						}
					}
				} else {
					if useParentValue {
						return subValues
					} else {
						output.Failf("expected array in examples: %s", globalKey)
					}
				}
			}
		}
	}

	return nil
}

func isArrayKey(key string) (string, bool) {
	arrayRegex := regexp.MustCompile(`\[\]$`)
	if arrayRegex.MatchString(key) {
		return arrayRegex.ReplaceAllString(key, ""), true
	} else {
		return key, false
	}
}

func parseYaml(bytes []byte) (map[string]interface{}, error) {
	valueMap := map[string]interface{}{}

	if err := yaml.Unmarshal(bytes, &valueMap); err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to parse yaml: %s", err)
	}

	return valueMap, nil
}

func findAndParseYaml(files []*any.Any, filename string) (map[string]interface{}, error) {
	for _, file := range files {
		if file.TypeUrl == filename {
			return parseYaml(file.Value)
		}
	}
	return nil, fmt.Errorf("required file not found in chart: %s", filename)
}

func containsString(list []string, element string) bool {
	for _, it := range list {
		if it == element {
			return true
		}
	}
	return false
}
