//go:build wireinject

package container

import (
	"github.com/google/wire"
	"gorm.io/gorm"
)

func Init(db *gorm.DB) (*AppContainer, error) {
	panic(wire.Build(ProviderSet))
}
