/*
Copyright (c) 2021-2022 Nordix Foundation

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

package common

import (
	"fmt"

	"github.com/go-logr/logr"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Executions interface {
	create(obj client.Object) error
	update(obj client.Object) error
	delete(obj client.Object) error
	RunAll(actions []Action) error
}

type Action interface {
	Run(e *Executor) (string, string, error)
}

type Executor struct {
	scheme  *runtime.Scheme
	client  client.Client
	ctx     context.Context
	owner   client.Object
	log     logr.Logger
	actions []Action
}

func NewExecutor(s *runtime.Scheme, c client.Client, ct context.Context, cr client.Object, l logr.Logger) *Executor {
	return &Executor{
		scheme: s,
		client: c,
		ctx:    ct,
		owner:  cr,
		log:    l.WithName("executor"),
	}
}

// Set the owner of created objects
func (e *Executor) SetOwner(cr client.Object) {
	e.owner = cr
}

func (e *Executor) GetOwner() client.Object {
	return e.owner
}

func (e *Executor) LogInfo(msg string) {
	e.log.Info(msg)
}

func (e *Executor) LogError(err error, msg string) {
	e.log.Error(err, msg)
}

func (e *Executor) GetObject(selector client.ObjectKey, obj client.Object) error {
	err := e.client.Get(e.ctx, selector, obj)
	if err != nil {
		return fmt.Errorf("failed to get objects: %w", err)
	}

	return nil
}

func (e *Executor) ListObject(obj client.ObjectList, opts ...client.ListOption) error {
	err := e.client.List(e.ctx, obj, opts...)
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	return nil
}

func (e *Executor) RunActions() error {
	for _, action := range e.actions {
		action, name, err := action.Run(e)
		if err != nil {
			e.log.Error(err, "execute action", "action", action, "objectName", name, "result", "failure")
			return fmt.Errorf("failed to run action for obj (%s): %w", name, err)
		}
		e.log.Info("execute action", "action", action, "objectName", name, "result", "success")
	}
	return nil
}

func AppendActions(actions ...Action) []Action {
	var ret []Action
	for _, action := range actions {
		if action != nil {
			ret = append(ret, action)
		}
	}
	return ret
}

func (e *Executor) appendActions(actions ...Action) {
	for _, action := range actions {
		if action != nil {
			e.actions = append(e.actions, action)
		}
	}
}

type createAction struct {
	obj    client.Object
	action string
}

type updateAction struct {
	obj    client.Object
	action string
}

type updateStatusAction struct {
	obj    client.Object
	action string
}

func (a createAction) Run(e *Executor) (string, string, error) {
	return a.action, a.obj.GetName(), e.create(a.obj)
}

func (a updateAction) Run(e *Executor) (string, string, error) {
	return a.action, a.obj.GetName(), e.update(a.obj)
}

func (a updateStatusAction) Run(e *Executor) (string, string, error) {
	return a.action, a.obj.GetName(), e.updateStatus(a.obj)
}

func (e *Executor) create(obj client.Object) error {
	err := e.SetControllerReference(obj)
	if err != nil {
		return err
	}
	// conflicts will happen when there are frequent actions
	if err = e.client.Create(e.ctx, obj); err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create obj (%s): %w", obj.GetName(), err)
	}
	return nil
}

func (e *Executor) update(obj client.Object) error {
	// conflicts will happen when there are frequent actions
	if err := e.client.Update(e.ctx, obj); err != nil && !errors.IsConflict(err) {
		return fmt.Errorf("failed to update obj (%s): %w", obj.GetName(), err)
	}
	return nil
}

func (e *Executor) updateStatus(obj client.Object) error {
	err := e.client.Status().Update(e.ctx, obj)
	if err != nil {
		// conflicts will happen when there are frequent actions
		if errors.IsConflict(err) {
			return nil
		}
		return fmt.Errorf("failed to update obj status (%s): %w", obj.GetName(), err)
	}
	return nil
}

func (e *Executor) SetControllerReference(obj client.Object) error {
	err := controllerutil.SetControllerReference(e.owner, obj, e.scheme)
	if err != nil {
		return fmt.Errorf("set controller reference error: %s", err)
	}
	return nil
}

// Append/update owner reference. Used when setting owner reference for custom resource
func (e *Executor) SetOwnerReference(obj client.Object, owners ...client.Object) error {
	if owners == nil {
		return fmt.Errorf("owner cannot be nil")
	}
	for _, owner := range owners {
		err := controllerutil.SetOwnerReference(owner, obj, e.scheme)
		if err != nil {
			return fmt.Errorf("set owner reference error: %s", err)
		}
	}
	return nil
}

func (e *Executor) AddCreateAction(obj client.Object) {
	name := obj.GetName()
	e.log.Info("add action", "action", "create", "objectName", name, "objectKind", obj.GetObjectKind().GroupVersionKind())
	e.appendActions(createAction{obj: obj, action: "create"})
}

func (e *Executor) AddUpdateAction(obj client.Object) {
	name := obj.GetName()
	e.log.Info("add action", "action", "update", "objectName", name, "objectKind", obj.GetObjectKind().GroupVersionKind())
	e.appendActions(updateAction{obj: obj, action: "update"})
}

func (e *Executor) AddUpdateStatusAction(obj client.Object) {
	name := obj.GetName()
	e.log.Info("add action", "action", "update status", "objectName", name, "objectKind", obj.GetObjectKind().GroupVersionKind())
	e.appendActions(updateStatusAction{obj: obj, action: "update status"})
}

type Reader struct {
	scheme *runtime.Scheme
	client client.Reader
	ctx    context.Context
	log    logr.Logger
}

func NewReader(s *runtime.Scheme, c client.Reader, ct context.Context, l logr.Logger) *Reader {
	return &Reader{
		scheme: s,
		client: c,
		ctx:    ct,
		log:    l.WithName("reader"),
	}
}

func (r *Reader) LogInfo(msg string) {
	r.log.Info(msg)
}

func (r *Reader) LogError(err error, msg string) {
	r.log.Error(err, msg)
}

func (r *Reader) GetObject(selector client.ObjectKey, obj client.Object) error {
	err := r.client.Get(r.ctx, selector, obj)
	if err != nil {
		return fmt.Errorf("reader failed to get objects: %w", err)
	}

	return nil
}

func (r *Reader) ListObject(obj client.ObjectList, opts ...client.ListOption) error {
	err := r.client.List(r.ctx, obj, opts...)
	if err != nil {
		return fmt.Errorf("reader failed to list objects: %w", err)
	}

	return nil
}
