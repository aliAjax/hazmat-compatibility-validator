package application

import (
	"sort"

	container "github.com/enterprise-labs/hazmat-compatibility-validator/internal/container/domain"
)

func cloneContainers(values []container.Container) []container.Container {
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	return values
}
