package crud

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/flaboy/pin"
)

type JsonRawMessage json.RawMessage

func (j JsonRawMessage) To(target interface{}) error {
	if j == nil {
		return nil
	}
	if err := json.Unmarshal(j, target); err != nil {
		return err
	}
	return nil
}

type QueryForm struct {
	Filter     json.RawMessage `json:"filter"`
	Pagination Pagination      `json:"pagination"`
}

type Pagination struct {
	Page  int   `json:"page"`
	Size  int   `json:"size"`
	Total int64 `json:"total"`
}

type QueryResult struct {
	Items      interface{} `json:"items"`
	Pagination *Pagination `json:"pagination"`
	Addition   interface{} `json:"addition"`
}

func (f *QueryForm) BindJSON(c *pin.Context) error {
	if err := c.BindJSON(f); err != nil {
		fmt.Println("error", err)
		return err
	}
	if f.Pagination.Size == 0 {
		f.Pagination.Size = 20
	}
	if f.Pagination.Page == 0 {
		f.Pagination.Page = 1
	}
	return nil
}

func (f *QueryForm) Parse(c *pin.Context) error {
	f.Pagination = Pagination{}
	f.Filter = json.RawMessage([]byte(c.Context.Query("filter")))
	pagestr := c.Context.Query("page")
	if pagestr != "" {
		f.Pagination.Page, _ = strconv.Atoi(pagestr)
	}
	page_size_str := c.Context.Query("page_size")
	if page_size_str != "" {
		f.Pagination.Size, _ = strconv.Atoi(page_size_str)
	}
	if f.Pagination.Size == 0 {
		f.Pagination.Size = 20
	}
	if f.Pagination.Page == 0 {
		f.Pagination.Page = 1
	}
	return nil
}

type Sort struct {
	Column string `json:"column"`
	Order  string `json:"order"` // "asc" or "desc"
}

type QueryContext struct {
	Filter     interface{}
	Pagination *Pagination
	Sort       *Sort
}

func (q *QueryContext) Parse(f JsonRawMessage, c *pin.Context) error {
	if err := f.To(&q.Filter); err != nil {
		return err
	}
	q.Pagination = &Pagination{}
	q.Pagination.Page, _ = strconv.Atoi(c.Context.Query("page"))
	q.Pagination.Size, _ = strconv.Atoi(c.Context.Query("page_size"))
	return nil
}

func (q *QueryContext) GetFilter() (map[string]interface{}, error) {
	filter := make(map[string]interface{})
	val := reflect.ValueOf(q.Filter)
	if val.Kind() == reflect.Map {
		for _, key := range val.MapKeys() {
			filter[key.String()] = val.MapIndex(key).Interface()
		}
		return filter, nil
	}
	return nil, fmt.Errorf("filter is not a valid map")
}

func (q *QueryContext) GetPagination() *Pagination {
	return q.Pagination
}

func (q *QueryContext) SetFilter(filter interface{}) {
	q.Filter = filter
}

func (q *QueryContext) SetPagination(pagination *Pagination) {
	q.Pagination = pagination
}

func (q *QueryContext) AddToFilter(key string, value interface{}) {
	val := reflect.ValueOf(q.Filter)
	if val.Kind() == reflect.Map {
		val.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	}
}

func (q *QueryContext) RemoveFromFilter(key string) {
	val := reflect.ValueOf(q.Filter)
	if val.Kind() == reflect.Map {
		val.SetMapIndex(reflect.ValueOf(key), reflect.Value{})
	}
}

func (q *QueryContext) ClearFilter() {
	val := reflect.ValueOf(q.Filter)
	if val.Kind() == reflect.Map {
		val.Set(reflect.MakeMap(val.Type()))
	}
}

func (q *QueryContext) SetPage(page int) {
	q.Pagination.Page = page
}

func (q *QueryContext) SetPageSize(size int) {
	q.Pagination.Size = size
}

func (q *QueryContext) GetPage() int {
	return q.Pagination.Page
}

func (q *QueryContext) GetPageSize() int {
	return q.Pagination.Size
}

func (q *QueryContext) GetTotal() int64 {
	return q.Pagination.Total
}

func (q *QueryContext) SetTotal(total int64) {
	q.Pagination.Total = total
}

func (q *QueryContext) ToQueryForm() *QueryForm {
	return &QueryForm{
		Filter:     json.RawMessage(q.Filter.([]byte)),
		Pagination: *q.Pagination,
	}
}

func (q *QueryContext) FromQueryForm(f *QueryForm) {
	q.Filter = string(f.Filter)
	q.Pagination = &f.Pagination
}

func (q *QueryContext) String() string {
	var b strings.Builder
	b.WriteString("Filter: ")
	filter, _ := json.Marshal(q.Filter)
	b.Write(filter)
	b.WriteString(", Pagination: ")
	pagination, _ := json.Marshal(q.Pagination)
	b.Write(pagination)
	return b.String()
}

// BindQuery 简化的查询绑定 API
// 自动解析查询参数到过滤器结构体，并返回 QueryContext
// filter 必须是指向结构体的指针
func BindQuery(c *pin.Context, filter interface{}) (*QueryContext, error) {
	// 绑定过滤器
	if err := bindFilterFromQuery(c, filter); err != nil {
		return nil, err
	}

	// 创建分页信息 - 使用带前缀的参数名避免冲突
	page, _ := strconv.Atoi(c.DefaultQuery("pagination-page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pagination-size", "20"))

	pagination := &Pagination{
		Page: page,
		Size: size,
	}

	// 创建排序信息
	var sort *Sort
	if column := c.Query("sort.column"); column != "" {
		order := c.DefaultQuery("sort.order", "asc")
		sort = &Sort{
			Column: column,
			Order:  order,
		}
	}

	return &QueryContext{
		Filter:     filter,
		Pagination: pagination,
		Sort:       sort,
	}, nil
}

// bindFilterFromQuery 从查询参数绑定到指定的 Filter 结构体
// filter 必须是指向结构体的指针
func bindFilterFromQuery(c *pin.Context, filter interface{}) error {
	v := reflect.ValueOf(filter)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return nil // 如果不是结构体指针，直接返回
	}

	elem := v.Elem()
	typ := elem.Type()

	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		fieldType := typ.Field(i)

		// 跳过不可设置的字段
		if !field.CanSet() {
			continue
		}

		// 获取查询参数名（优先使用 json tag，然后使用字段名）
		paramName := getParamName(fieldType)
		queryValue := c.Query(paramName)

		// 如果查询参数为空，跳过
		if queryValue == "" {
			continue
		}

		// 根据字段类型设置值
		if err := setFieldValue(field, fieldType, queryValue); err != nil {
			return err
		}
	}

	return nil
}

// getParamName 获取查询参数名
func getParamName(field reflect.StructField) string {
	// 优先使用 json tag
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		// 处理 json:",omitempty" 这种情况
		if parts := strings.Split(jsonTag, ","); len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	// 使用字段名（转为小写首字母）
	name := field.Name
	if len(name) > 0 {
		return strings.ToLower(name[:1]) + name[1:]
	}
	return name
}

// setFieldValue 设置字段值
func setFieldValue(field reflect.Value, fieldType reflect.StructField, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int64:
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			field.SetInt(intVal)
		}

	case reflect.Slice:
		return setSliceValue(field, fieldType, value)

	default:
		// 对于其他类型，保持零值
	}

	return nil
}

// setSliceValue 设置切片类型的值
func setSliceValue(field reflect.Value, fieldType reflect.StructField, value string) error {
	elemType := field.Type().Elem()

	// 获取分隔符（默认为逗号）
	delimiter := getDelimiter(fieldType)

	// 分割字符串
	parts := strings.Split(value, delimiter)

	// 创建切片
	slice := reflect.MakeSlice(field.Type(), 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		elemValue := reflect.New(elemType).Elem()

		switch elemType.Kind() {
		case reflect.Int:
			if intVal, err := strconv.Atoi(part); err == nil {
				elemValue.SetInt(int64(intVal))
				slice = reflect.Append(slice, elemValue)
			}
		case reflect.Int64:
			if int64Val, err := strconv.ParseInt(part, 10, 64); err == nil {
				elemValue.SetInt(int64Val)
				slice = reflect.Append(slice, elemValue)
			}
		case reflect.String:
			elemValue.SetString(part)
			slice = reflect.Append(slice, elemValue)
		}
	}

	field.Set(slice)
	return nil
}

// getDelimiter 获取分隔符
func getDelimiter(fieldType reflect.StructField) string {
	if filterTag := fieldType.Tag.Get("filter"); filterTag != "" {
		// 解析 filter tag，例如 "delimiter:;"
		parts := strings.Split(filterTag, ":")
		if len(parts) == 2 && parts[0] == "delimiter" {
			return parts[1]
		}
	}
	return "," // 默认分隔符
}

// ParseDateRange 解析日期范围字符串为时间戳数组
// 支持格式：
// 1. "timestamp1,timestamp2" - 直接的时间戳
// 2. "2024-01-01,2024-01-31" - 日期字符串
func ParseDateRange(dateRangeStr string) ([]int64, error) {
	if dateRangeStr == "" {
		return nil, nil
	}

	parts := strings.Split(dateRangeStr, ",")
	if len(parts) != 2 {
		return nil, nil
	}

	var result []int64

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// 尝试解析为时间戳
		if timestamp, err := strconv.ParseInt(part, 10, 64); err == nil {
			result = append(result, timestamp)
			continue
		}

		// 尝试解析为日期字符串
		if t, err := time.Parse("2006-01-02", part); err == nil {
			result = append(result, t.Unix())
			continue
		}

		// 如果都解析失败，返回 nil
		return nil, nil
	}

	return result, nil
}
