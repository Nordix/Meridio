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
	"fmt"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type SQLiteIPAMStorage struct {
	DB *gorm.DB
	mu sync.Mutex
}

func NewStorage(datastore string) (*SQLiteIPAMStorage, error) {
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

func (sqlis *SQLiteIPAMStorage) Add(ctx context.Context, prefix string, child string) error {
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()
	var p Prefix
	tx := sqlis.DB.First(&p, "Prefix", prefix)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) { // Create
			tx = sqlis.DB.Create(&Prefix{
				Prefix: prefix,
				Childs: SerializeChilds([]string{child}),
			})
			return tx.Error
		}
		return tx.Error
	}
	childs := DeserializeChilds(p.Childs)
	if childExists(childs, child) {
		return fmt.Errorf("child %v already exists in %v", child, prefix)
	}
	childs = append(childs, child)
	tx = sqlis.DB.Save(&Prefix{
		Prefix: prefix,
		Childs: SerializeChilds(childs),
	})
	return tx.Error
}

func (sqlis *SQLiteIPAMStorage) Delete(ctx context.Context, prefix string, child string) error {
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()
	var p Prefix
	tx := sqlis.DB.First(&p, "Prefix", prefix)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		return tx.Error
	}
	childs := DeserializeChilds(p.Childs)
	if len(childs) <= 0 {
		tx = sqlis.DB.Delete(&Prefix{}, prefix)
		return tx.Error
	}
	childs = removeChild(childs, child)
	tx = sqlis.DB.Save(&Prefix{
		Prefix: prefix,
		Childs: SerializeChilds(childs),
	})
	return tx.Error
}

func (sqlis *SQLiteIPAMStorage) Get(ctx context.Context, prefix string) ([]string, error) {
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()
	return sqlis.get(ctx, prefix)
}

func (sqlis *SQLiteIPAMStorage) get(ctx context.Context, prefix string) ([]string, error) {
	var p Prefix
	tx := sqlis.DB.First(&p, "Prefix", prefix)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return []string{}, nil
		}
		return nil, tx.Error
	}
	return DeserializeChilds(p.Childs), nil
}

func (sqlis *SQLiteIPAMStorage) init() error {
	err := sqlis.DB.AutoMigrate(&Prefix{})
	return err
}

func removeChild(childs []string, child string) []string {
	index := -1
	for i, c := range childs {
		if c == child {
			index = i
			break
		}
	}
	if index <= -1 {
		return childs
	}
	return append(childs[:index], childs[index+1:]...)
}

func childExists(childs []string, child string) bool {
	for _, c := range childs {
		if c == child {
			return true
		}
	}
	return false
}
