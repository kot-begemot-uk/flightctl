package store

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/flightctl/flightctl/internal/flterrors"
	"github.com/flightctl/flightctl/internal/store/model"
	"github.com/flightctl/flightctl/internal/util"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const retryIterations = 10


var (
	ErrorInvalidFieldKey   = errors.New("invalid field filter key")
	ErrorInvalidFieldValue = errors.New("invalid field filter value")
)

type CreateOrUpdateMode string

const (
	ModeCreateOnly     CreateOrUpdateMode = "create-only"
	ModeUpdateOnly     CreateOrUpdateMode = "update-only"
	ModeCreateOrUpdate CreateOrUpdateMode = "create-or-update"
)

func BuildBaseListQuery(query *gorm.DB, orgId uuid.UUID, listParams ListParams) *gorm.DB {
	query = query.Where("org_id = ?", orgId).Order("name")
	invertLabels := false
	if listParams.InvertLabels != nil && *listParams.InvertLabels {
		invertLabels = true
	}
	query = LabelSelectionQuery(query, listParams.Labels, invertLabels)
	query = FieldFilterSelectionQuery(query, listParams.Filter)

	queryStr, args := createOrQuery("owner", listParams.Owners)
	if len(queryStr) > 0 {
		query = query.Where(queryStr, args...)
	}

	if listParams.FleetName != nil {
		query = query.Where("fleet_name = ?", *listParams.FleetName)
	}
	return query
}

func AddPaginationToQuery(query *gorm.DB, limit int, cont *Continue) *gorm.DB {
	if limit == 0 {
		return query
	}
	query = query.Limit(limit)
	if cont != nil {
		query = query.Where("name >= ?", cont.Name)
	}

	return query
}

func CountRemainingItems(query *gorm.DB, lastItemName string) int64 {
	var count int64
	query.Where("name >= ?", lastItemName).Count(&count)
	return count
}

func GetNonNilFieldsFromResource(resource model.Resource) []string {
	ret := []string{}
	if resource.Generation != nil {
		ret = append(ret, "generation")
	}
	if resource.Labels != nil {
		ret = append(ret, "labels")
	}
	if resource.Owner != nil {
		ret = append(ret, "owner")
	}
	if resource.Annotations != nil {
		ret = append(ret, "annotations")
	}

	if resource.Generation != nil {
		ret = append(ret, "generation")
	}

	if resource.ResourceVersion != nil {
		ret = append(ret, "resource_version")
	}

	return ret
}

// LabelSelectionQuery applies a label-based selection query to the given GORM DB query.
// It takes a map of labels and a GORM DB query as input.
// The function returns the modified DB query.
func LabelSelectionQuery(query *gorm.DB, labels map[string]string, inverse bool) *gorm.DB {
	if len(labels) == 0 {
		return query
	}

	arrayLabels := util.LabelMapToArray(&labels)

	// we do this instead of constructing the query string directly because of the Where
	// function implementation, finding a ? in the string will trigger one path, @ in the
	// string will trigger another path that looks for a pre-stored  database query.
	arrayPlaceholders := []string{}
	arrayValues := []interface{}{}
	for _, v := range arrayLabels {
		arrayPlaceholders = append(arrayPlaceholders, "?")
		arrayValues = append(arrayValues, v)
	}

	queryString := fmt.Sprintf("labels @> ARRAY[%s]", strings.Join(arrayPlaceholders, ","))

	if inverse {
		return query.Not(queryString, arrayValues...)
	}
	return query.Where(queryString, arrayValues...)
}

// FieldFilterSelectionQuery takes a GORM DB query and a map of search parameters. To search for a key-value pair in the
// in a JSON object use the key to reflect location in the JSON data and the value to reflect the value to search for.
// example map[string]string{"status.config.summary.status": "UpToDate"} will search status.config.summary.status for the
// value "UpToDate".
// To search for multiple values in the same field, separate the values with a comma.
func FieldFilterSelectionQuery(query *gorm.DB, fieldMap map[string][]string) *gorm.DB {
	queryStr, args := createQueryFromFilterMap(fieldMap)
	if len(queryStr) > 0 {
		query = query.Where(queryStr, args...)
	}

	return query
}

func createQueryFromFilterMap(fieldMap map[string][]string) (string, []interface{}) {
	var queryParams []string
	var args []interface{}

	for key, values := range fieldMap {
		if key == "" || values == nil || len(values) == 0 {
			continue
		}

		orQuery, queryArgs := createOrQuery(createParamsFromKey(key), values)
		if len(orQuery) > 0 {
			queryParams = append(queryParams, orQuery)
			args = append(args, queryArgs...)
		}
	}

	var query string
	// join all query conditions with 'OR'
	if len(queryParams) > 0 {
		query = strings.Join(queryParams, " OR ")
	}

	return query, args
}

func createParamsFromKey(key string) string {
	parts := strings.Split(key, ".")
	params := ""
	for i, part := range parts {
		if i == 0 {
			params += part
		} else if i == len(parts)-1 {
			// prefix last part with the ->> operator for JSONB fetching text
			params += fmt.Sprintf(" ->> '%s'", part)
		} else {
			// prefix intermediate parts with the -> operator for JSONB
			params += fmt.Sprintf(" -> '%s'", part)
		}
	}
	return params
}

// createOrQuery can return empty `queryStr`/`args` (ie if `key` or `values` params are empty).
// The caller is expected to check the size of `queryStr`/`args` before constructing a GORM query.
func createOrQuery(key string, values []string) (string, []interface{}) {
	var queryStr string
	var queryParams []string
	var args []interface{}

	if key == "" {
		return queryStr, args
	}

	for _, val := range values {
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}
		queryParams = append(queryParams, fmt.Sprintf("%s = ?", key))
		args = append(args, val)
	}

	if len(queryParams) > 0 {
		queryStr = strings.Join(queryParams, " OR ")
	}
	return queryStr, args
}

func getExistingRecord[R any](db *gorm.DB, name string, orgId uuid.UUID) (*R, error) {
	var existingRecord R
	if err := db.Where("name = ? and org_id = ?", name, orgId).First(&existingRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, flterrors.ErrorFromGormError(err)
	}
	return &existingRecord, nil
}

func retryCreateOrUpdate[A any](fn func() (*A, bool, bool, error)) (*A, bool, error) {
	var (
		a              *A
		created, retry bool
		err            error
	)
	i := 0
	for a, created, retry, err = fn(); retry && i < retryIterations; a, created, retry, err = fn() {
		i++
	}
	return a, created, err
}

func retryUpdate(fn func() (bool, error)) error {
	var (
		retry bool
		err   error
	)
	i := 0
	for retry, err = fn(); retry && i < retryIterations; retry, err = fn() {
		i++
	}
	return err
}


// validateFieldKey validates a field key. Valid characters are [a-zA-Z.]
func validateFieldKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	for _, char := range key {
		if !unicode.IsLetter(char) && char != '.' {
			return "", fmt.Errorf("%w: %s", ErrorInvalidFieldKey, key)
		}
	}
	return key, nil
}

// validateFieldValue validates a field value. Valid characters are [a-zA-Z0-9,-.]
func validateFieldValue(value string) (string, error) {
	value = strings.TrimSpace(value)
	for _, char := range value {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != ',' && char != '-' && char != '.' {
			return "", fmt.Errorf("%w: %s", ErrorInvalidFieldValue, value)
		}
	}
	return value, nil
}

func ConvertFieldFilterParamsToMap(params []string) (map[string][]string, error) {
	fieldMap := make(map[string][]string)
	if len(params) == 0 {
		return fieldMap, nil
	}

	for _, selectorStr := range params {
		selectorStr = strings.TrimSpace(selectorStr)
		if selectorStr == "" {
			continue
		}

		pairs := strings.Split(selectorStr, "=")
		if len(pairs) == 1 {
			return nil, fmt.Errorf("invalid selector format: %s", selectorStr)
		}

		key, err := validateFieldKey(pairs[0])
		if err != nil {
			return nil, err
		}
		value, err := validateFieldValue(pairs[1])
		if err != nil {
			return nil, err
		}
		fieldMap[key] = append(fieldMap[key], value)
	}

	return fieldMap, nil
}

// ConvertStatusFilterParamsToMap converts statusFilter query params to to a validated filterMap map.
func ConvertStatusFilterParamsToMap(params []string) (map[string][]string, error) {
	fieldMap := make(map[string][]string)
	if len(params) == 0 {
		return fieldMap, nil
	}

	for _, selectorStr := range params {
		selectorStr = strings.TrimSpace(selectorStr)
		if selectorStr == "" {
			continue
		}

		pairs := strings.Split(selectorStr, "=")
		if len(pairs) == 1 {
			return nil, fmt.Errorf("invalid selector format: %s", selectorStr)
		}

		key, err := validateFieldKey(pairs[0])
		if err != nil {
			return nil, err
		}
		value, err := validateFieldValue(pairs[1])
		if err != nil {
			return nil, err
		}
		fieldMap[key] = append(fieldMap[key], value)
	}

	return fieldMap, nil
}
