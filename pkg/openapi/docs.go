package openapi

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type ApiDocumentation struct {
	Apis []ApiEndpoint `json:"apis"`
}

// ApiInfo represents basic API information
type ApiInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
	GeneratedAt string `json:"generated_at"`
}

// ApiServer represents API server information
type ApiServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ApiEndpoint represents API endpoint information
type ApiEndpoint struct {
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	Description string         `json:"description"`
	Request     *ApiSchema     `json:"request,omitempty"`
	Response    *ApiSchema     `json:"response,omitempty"`
	Errors      []apiError     `json:"errors,omitempty"`
	Parameters  []ApiParameter `json:"parameters,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
}

// ApiSchema represents data structure information
type ApiSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]ApiProperty `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
	Example    interface{}            `json:"example,omitempty"`
}

// ApiProperty represents property information
type ApiProperty struct {
	Type           string                 `json:"type"`
	Description    string                 `json:"description,omitempty"`
	Required       bool                   `json:"required,omitempty"`
	Example        interface{}            `json:"example,omitempty"`
	Format         string                 `json:"format,omitempty"`
	Properties     map[string]ApiProperty `json:"properties,omitempty"`     // Properties of nested objects
	RequiredFields []string               `json:"requiredFields,omitempty"` // Required fields of nested objects
	Items          *ApiProperty           `json:"items,omitempty"`          // Type information of array items
}

// ApiParameter represents parameter information
type ApiParameter struct {
	Name        string `json:"name"`
	In          string `json:"in"` // path, query, header, body
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// GenerateApiDocumentation generates complete API documentation
func (e *Endpoint) GenerateApiDocumentation(info ApiInfo, servers []ApiServer) *ApiDocumentation {
	if info.GeneratedAt == "" {
		info.GeneratedAt = time.Now().Format(time.RFC3339)
	}

	doc := &ApiDocumentation{
		Apis: make([]ApiEndpoint, 0, len(e.apilist)),
	}

	for _, router := range e.apilist {
		endpoint := e.generateEndpointDoc(router)
		doc.Apis = append(doc.Apis, endpoint)
	}

	return doc
}

// generateEndpointDoc generates documentation for a single route
func (e *Endpoint) generateEndpointDoc(router ApiRouter) ApiEndpoint {
	endpoint := ApiEndpoint{
		Method:      router.Method,
		Path:        router.Path,
		Description: router.Name,
		Parameters:  e.extractPathParameters(router.Path),
		Tags:        extractTags(router.Path),
		Errors:      make([]apiError, 0),
	}

	for _, err := range router.Errors {
		endpoint.Errors = append(endpoint.Errors, apiError{
			Code:    err.Code(),
			Message: err.Message(),
		})
	}

	// Generate request structure documentation
	if router.Request != nil {
		endpoint.Request = e.generateSchemaDoc(router.Request)
		// If there's a manually set request example, use it instead of auto-generated example
		if router.RequestExample != nil {
			endpoint.Request.Example = router.RequestExample
		}
	}

	// Generate response structure documentation
	if router.Response != nil {
		// First generate the actual response data schema
		actualResponseSchema := e.generateSchemaDoc(router.Response)

		// Wrap in pin response format
		endpoint.Response = &ApiSchema{
			Type: "object",
			Properties: map[string]ApiProperty{
				"data": {
					Type:           actualResponseSchema.Type,
					Description:    "Response data",
					Properties:     actualResponseSchema.Properties,
					RequiredFields: actualResponseSchema.Required,
					Items:          nil, // Will be set below if needed
				},
				"meta": {
					Type:        "object",
					Description: "Metadata (optional)",
				},
				"trace_id": {
					Type:        "string",
					Description: "Request trace ID",
				},
				"error": {
					Type:        "object",
					Description: "Error information (only present when error occurs)",
					Properties: map[string]ApiProperty{
						"message": {
							Type:        "string",
							Description: "Human readable error message",
						},
						"type": {
							Type:        "string",
							Description: "Error type (e.g., 'user')",
						},
						"key": {
							Type:        "string",
							Description: "Error code for programmatic handling",
						},
					},
					RequiredFields: []string{"message", "type", "key"},
				},
			},
			Required: []string{}, // No required fields as response structure varies
		}

		// Handle array responses
		if actualResponseSchema.Type == "array" {
			endpoint.Response.Properties["data"] = ApiProperty{
				Type:        "array",
				Description: "Response data",
				Items: &ApiProperty{
					Type:           "object",
					Properties:     actualResponseSchema.Properties,
					RequiredFields: actualResponseSchema.Required,
				},
			}
		}

		// Set wrapped example (success case)
		if router.ResponseExample != nil {
			endpoint.Response.Example = map[string]interface{}{
				"data":     router.ResponseExample,
				"trace_id": "example-trace-id-123",
			}
		} else if actualResponseSchema.Example != nil {
			endpoint.Response.Example = map[string]interface{}{
				"data":     actualResponseSchema.Example,
				"trace_id": "example-trace-id-123",
			}
		}
	}

	return endpoint
}

// generateSchemaDoc generates data structure documentation
func (e *Endpoint) generateSchemaDoc(obj interface{}) *ApiSchema {
	if obj == nil {
		return nil
	}

	objType := reflect.TypeOf(obj)
	objValue := reflect.ValueOf(obj)

	// If it's a pointer, get the pointed type and value
	if objType.Kind() == reflect.Ptr {
		if objValue.IsNil() {
			objValue = reflect.New(objType.Elem())
		}
		objType = objType.Elem()
		objValue = objValue.Elem()
	}

	schema := &ApiSchema{
		Type:       "object",
		Properties: make(map[string]ApiProperty),
		Required:   make([]string, 0),
	}

	// Handle struct
	if objType.Kind() == reflect.Struct {
		for i := 0; i < objType.NumField(); i++ {
			field := objType.Field(i)

			// Skip private fields
			if !field.IsExported() {
				continue
			}

			// Get JSON tag
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}

			fieldName := field.Name
			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" {
					fieldName = parts[0]
				}
			}

			prop := ApiProperty{
				Type:        e.getTypeString(field.Type),
				Description: e.buildDescription(field),
			}

			// Check if required
			bindingTag := field.Tag.Get("binding")
			if strings.Contains(bindingTag, "required") {
				prop.Required = true
				schema.Required = append(schema.Required, fieldName)
			}

			// Handle nested objects
			if field.Type.Kind() == reflect.Struct || (field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct) {
				nestedType := field.Type
				if nestedType.Kind() == reflect.Ptr {
					nestedType = nestedType.Elem()
				}

				// Skip time types
				if !(nestedType.PkgPath() == "time" && nestedType.Name() == "Time") {
					nestedSchema := e.generateSchemaDoc(reflect.New(nestedType).Interface())
					if nestedSchema != nil {
						prop.Properties = nestedSchema.Properties
						prop.RequiredFields = nestedSchema.Required
					}
				}
			}

			// Handle array types
			if field.Type.Kind() == reflect.Slice || field.Type.Kind() == reflect.Array {
				elemType := field.Type.Elem()
				if elemType.Kind() == reflect.Struct || (elemType.Kind() == reflect.Ptr && elemType.Elem().Kind() == reflect.Struct) {
					if elemType.Kind() == reflect.Ptr {
						elemType = elemType.Elem()
					}

					itemSchema := e.generateSchemaDoc(reflect.New(elemType).Interface())
					if itemSchema != nil {
						prop.Items = &ApiProperty{
							Type:           "object",
							Properties:     itemSchema.Properties,
							RequiredFields: itemSchema.Required,
						}
					}
				} else {
					prop.Items = &ApiProperty{
						Type: e.getTypeString(elemType),
					}
				}
			}

			// Don't set example values to properties

			schema.Properties[fieldName] = prop
		}

		// Generate overall example
		schema.Example = e.generateStructExample(objType)
	}

	return schema
}

// buildDescription builds description information, merging description and binding information
func (e *Endpoint) buildDescription(field reflect.StructField) string {
	desc := field.Tag.Get("description")
	bindingTag := field.Tag.Get("binding")

	var parts []string
	if desc != "" {
		parts = append(parts, desc)
	}

	if bindingTag != "" {
		// Parse binding tags
		bindingParts := strings.Split(bindingTag, ",")
		for _, part := range bindingParts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "oneof=") {
				enumValues := strings.TrimPrefix(part, "oneof=")
				values := strings.Fields(enumValues)
				if len(values) > 0 {
					parts = append(parts, fmt.Sprintf("One of: %s", strings.Join(values, ", ")))
				}
			} else if strings.HasPrefix(part, "min=") {
				parts = append(parts, fmt.Sprintf("Min: %s", strings.TrimPrefix(part, "min=")))
			} else if strings.HasPrefix(part, "max=") {
				parts = append(parts, fmt.Sprintf("Max: %s", strings.TrimPrefix(part, "max=")))
			} else if strings.HasPrefix(part, "len=") {
				parts = append(parts, fmt.Sprintf("Length: %s", strings.TrimPrefix(part, "len=")))
			}
		}
	}

	return strings.Join(parts, " ")
}

// getTypeString gets type string
func (e *Endpoint) getTypeString(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map:
		return "object"
	case reflect.Ptr:
		return e.getTypeString(t.Elem())
	case reflect.Struct:
		// Special handling for time types
		if t.PkgPath() == "time" && t.Name() == "Time" {
			return "string"
		}
		return "object"
	default:
		return "string"
	}
}

// getExampleValue gets example value
func (e *Endpoint) getExampleValue(t reflect.Type) interface{} {
	switch t.Kind() {
	case reflect.String:
		return "example"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return 1
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return 1
	case reflect.Float32, reflect.Float64:
		return 1.0
	case reflect.Bool:
		return true
	case reflect.Slice, reflect.Array:
		return []interface{}{}
	case reflect.Map:
		return map[string]interface{}{}
	case reflect.Ptr:
		return e.getExampleValue(t.Elem())
	case reflect.Struct:
		if t.PkgPath() == "time" && t.Name() == "Time" {
			return "2023-01-01T00:00:00Z"
		}
		return e.generateStructExample(t)
	default:
		return "example"
	}
}

// generateStructExample generates struct example
func (e *Endpoint) generateStructExample(t reflect.Type) interface{} {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return e.getExampleValue(t)
	}

	example := make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
		}

		example[fieldName] = e.getExampleValue(field.Type)
	}

	return example
}

// extractPathParameters extracts path parameters
func (e *Endpoint) extractPathParameters(path string) []ApiParameter {
	var params []ApiParameter
	parts := strings.Split(path, "/")

	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			paramName := strings.TrimPrefix(part, ":")
			params = append(params, ApiParameter{
				Name:        paramName,
				In:          "path",
				Type:        "string",
				Required:    true,
				Description: fmt.Sprintf("%s parameter", paramName),
			})
		}
	}

	return params
}

// extractTags extracts tags from path
func extractTags(path string) []string {
	parts := strings.Split(path, "/")
	if len(parts) > 1 && parts[1] != "" {
		return []string{parts[1]}
	}
	return []string{"default"}
}

// GetApiDocumentation gets API documentation (JSON format)
func (e *Endpoint) GetApiDocumentation() *ApiDocumentation {
	info := ApiInfo{
		Title:       "project API",
		Description: "project Platform API Documentation",
		Version:     "1.0.0",
	}

	servers := []ApiServer{
		{
			URL:         "https://api.project.com",
			Description: "Production Environment",
		},
		{
			URL:         "https://api-dev.project.com",
			Description: "Development Environment",
		},
	}

	return e.GenerateApiDocumentation(info, servers)
}
