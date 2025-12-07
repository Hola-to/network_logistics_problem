package validators

import (
	"fmt"

	commonv1 "logistics/gen/go/logistics/common/v1"
)

// ValidateRequest валидирует входящий запрос
func ValidateRequest(graph *commonv1.Graph) []*commonv1.ValidationError {
	var errors []*commonv1.ValidationError

	if graph == nil {
		errors = append(errors, &commonv1.ValidationError{
			Field:   "graph",
			Message: "Граф не может быть nil",
			Code:    "NIL_GRAPH",
		})
		return errors
	}

	return errors
}

// ValidateAlgorithmChoice валидирует выбор алгоритма
func ValidateAlgorithmChoice(algo commonv1.Algorithm) []*commonv1.ValidationError {
	var errors []*commonv1.ValidationError

	validAlgorithms := map[commonv1.Algorithm]bool{
		commonv1.Algorithm_ALGORITHM_EDMONDS_KARP:   true,
		commonv1.Algorithm_ALGORITHM_DINIC:          true,
		commonv1.Algorithm_ALGORITHM_MIN_COST:       true,
		commonv1.Algorithm_ALGORITHM_PUSH_RELABEL:   true,
		commonv1.Algorithm_ALGORITHM_FORD_FULKERSON: true,
	}

	if algo != commonv1.Algorithm_ALGORITHM_UNSPECIFIED && !validAlgorithms[algo] {
		errors = append(errors, &commonv1.ValidationError{
			Field:   "algorithm",
			Message: fmt.Sprintf("Неизвестный алгоритм: %s", algo),
			Code:    "INVALID_ALGORITHM",
		})
	}

	return errors
}

// ValidateThreshold валидирует пороговые значения
func ValidateThreshold(value float64, fieldName string, min, max float64) []*commonv1.ValidationError {
	var errors []*commonv1.ValidationError

	if value < min || value > max {
		errors = append(errors, &commonv1.ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("Значение должно быть в диапазоне [%.2f, %.2f], получено: %.2f", min, max, value),
			Code:    "INVALID_THRESHOLD",
		})
	}

	return errors
}

// ValidatePagination валидирует параметры пагинации
func ValidatePagination(page, pageSize int32) []*commonv1.ValidationError {
	var errors []*commonv1.ValidationError

	if page < 0 {
		errors = append(errors, &commonv1.ValidationError{
			Field:   "page",
			Message: fmt.Sprintf("Номер страницы не может быть отрицательным: %d", page),
			Code:    "INVALID_PAGINATION",
		})
	}

	if pageSize < 0 {
		errors = append(errors, &commonv1.ValidationError{
			Field:   "page_size",
			Message: fmt.Sprintf("Размер страницы не может быть отрицательным: %d", pageSize),
			Code:    "INVALID_PAGINATION",
		})
	}

	if pageSize > 1000 {
		errors = append(errors, &commonv1.ValidationError{
			Field:   "page_size",
			Message: fmt.Sprintf("Размер страницы слишком большой: %d (максимум 1000)", pageSize),
			Code:    "INVALID_PAGINATION",
		})
	}

	return errors
}
