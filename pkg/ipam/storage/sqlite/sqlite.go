/*
Copyright (c) 2021 Nordix Foundation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sqlite

import (
	"context"
	"errors"
	"sync"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type SQLiteIPAMStorage struct {
	DB *gorm.DB
	mu sync.Mutex
}

func New(datastore string) (*SQLiteIPAMStorage, error) {
	db, err := gorm.Open(sqlite.Open(datastore), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	sqlis := &SQLiteIPAMStorage{
		DB: db,
	}
	err = sqlis.init()
	if err != nil {
		return nil, err
	}
	return sqlis, nil
}

func (sqlis *SQLiteIPAMStorage) Add(ctx context.Context, prefix types.Prefix) error {
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()
	model := prefixToModel(prefix)
	tx := sqlis.DB.Create(model)
	return tx.Error
}

func (sqlis *SQLiteIPAMStorage) Delete(ctx context.Context, prefix types.Prefix) error {
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()
	return sqlis.delete(prefix)
}

func (sqlis *SQLiteIPAMStorage) Get(ctx context.Context, name string, parent types.Prefix) (types.Prefix, error) {
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()
	prefix := prefix.New(name, "", parent)
	model := prefixToModel(prefix)
	var result *Prefix
	tx := sqlis.DB.First(&result, model)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	np := modelToPrefix(result, parent)
	return np, nil
}

func (sqlis *SQLiteIPAMStorage) GetChilds(ctx context.Context, prefix types.Prefix) ([]types.Prefix, error) {
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()
	return sqlis.getChilds(prefix)
}

func (sqlis *SQLiteIPAMStorage) init() error {
	err := sqlis.DB.AutoMigrate(&Prefix{})
	return err
}

func (sqlis *SQLiteIPAMStorage) getChilds(prefix types.Prefix) ([]types.Prefix, error) {
	model := prefixToModel(prefix)
	var results []*Prefix
	tx := sqlis.DB.Where("parent_id = ?", model.Id).Find(&results)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return []types.Prefix{}, nil
		}
		return nil, tx.Error
	}
	prefixList := []types.Prefix{}
	for _, result := range results {
		np := modelToPrefix(result, prefix)
		prefixList = append(prefixList, np)
	}
	return prefixList, nil
}

func (sqlis *SQLiteIPAMStorage) delete(prefix types.Prefix) error {
	childs, err := sqlis.getChilds(prefix)
	if err != nil {
		return err
	}
	for _, child := range childs {
		err = sqlis.delete(child)
		if err != nil {
			return err
		}
	}
	model := prefixToModel(prefix)
	tx := sqlis.DB.Delete(model)
	return tx.Error
}
