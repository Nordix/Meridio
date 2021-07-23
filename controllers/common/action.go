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
	Run(e *Executor) (string, error)
}

type Executor struct {
	scheme *runtime.Scheme
	Client client.Client
	Ctx    context.Context
	Cr     client.Object
	Log    logr.Logger
}

func NewExecutor(s *runtime.Scheme, c client.Client, ct context.Context, cr client.Object, l logr.Logger) *Executor {
	return &Executor{
		scheme: s,
		Client: c,
		Ctx:    ct,
		Cr:     cr,
		Log:    l.WithName("executor"),
	}
}

func (e *Executor) RunAll(actions []Action) error {
	for _, action := range actions {
		msg, err := action.Run(e)
		if err != nil {
			e.Log.Error(err, msg, "result", "failure")
			return err
		}
		e.Log.Info(msg, "result", "succeess")
	}
	return nil
}

type createAction struct {
	obj client.Object
	msg string
}

type updateAction struct {
	obj client.Object
	msg string
}

type updateStatusAction struct {
	obj client.Object
	msg string
}

func (a createAction) Run(e *Executor) (string, error) {
	return a.msg, e.create(a.obj)
}

func (a updateAction) Run(e *Executor) (string, error) {
	return a.msg, e.update(a.obj)
}

func (a updateStatusAction) Run(e *Executor) (string, error) {
	return a.msg, e.updateStatus(a.obj)
}

func (e *Executor) create(obj client.Object) error {
	err := controllerutil.SetControllerReference(e.Cr, obj, e.scheme)
	if err != nil {
		return fmt.Errorf("set reference error: %s", err)
	}

	return e.Client.Create(e.Ctx, obj)
}

func (e *Executor) update(obj client.Object) error {
	err := controllerutil.SetControllerReference(e.Cr, obj, e.scheme)
	if err != nil {
		return fmt.Errorf("set reference error: %s", err)
	}

	err = e.Client.Update(e.Ctx, obj)
	if err != nil {
		// conflicts will happen when there are frequent actions
		if errors.IsConflict(err) {
			return nil
		}
		return err
	}
	return nil
}

func (e *Executor) updateStatus(obj client.Object) error {
	err := e.Client.Status().Update(e.Ctx, obj)
	if err != nil {
		// conflicts will happen when there are frequent actions
		if errors.IsConflict(err) {
			return nil
		}
		return err
	}
	return nil
}

func (e *Executor) delete(obj client.Object) error {
	return e.Client.Delete(e.Ctx, obj)
}

func NewCreateAction(obj client.Object, msg string) Action {
	return createAction{obj: obj, msg: msg}
}

func NewUpdateAction(obj client.Object, msg string) Action {
	return updateAction{obj: obj, msg: msg}
}

func NewUpdateStatusAction(obj client.Object, msg string) Action {
	return updateStatusAction{obj: obj, msg: msg}
}
