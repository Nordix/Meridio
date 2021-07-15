package trench

import (
	"fmt"

	"github.com/go-logr/logr"
	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
	client client.Client
	ctx    context.Context
	cr     *meridiov1alpha1.Trench
	log    logr.Logger
}

func NewExecutor(s *runtime.Scheme, c client.Client, ct context.Context, cr *meridiov1alpha1.Trench) *Executor {
	return &Executor{
		scheme: s,
		client: c,
		ctx:    ct,
		cr:     cr,
		log:    log.Log.WithName("reconciler"),
	}
}

func (e *Executor) runAll(actions []Action) error {
	for _, action := range actions {
		msg, err := action.Run(e)
		if err != nil {
			e.log.Error(err, msg, "result", "failure")
			return err
		}
		e.log.Info(msg, "result", "succeess")
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

func (a createAction) Run(e *Executor) (string, error) {
	return a.msg, e.create(a.obj)
}

func (a updateAction) Run(e *Executor) (string, error) {
	return a.msg, e.update(a.obj)
}

func (e *Executor) create(obj client.Object) error {
	err := controllerutil.SetControllerReference(e.cr, obj, e.scheme)
	if err != nil {
		return fmt.Errorf("set reference error: %s", err)
	}

	return e.client.Create(e.ctx, obj)
}

func (e *Executor) update(obj client.Object) error {
	err := controllerutil.SetControllerReference(e.cr, obj, e.scheme)
	if err != nil {
		return fmt.Errorf("set reference error: %s", err)
	}

	err = e.client.Update(e.ctx, obj)
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
	return e.client.Delete(e.ctx, obj)
}

func newCreateAction(obj client.Object, msg string) Action {
	return createAction{obj: obj, msg: msg}
}

func newUpdateAction(obj client.Object, msg string) Action {
	return updateAction{obj: obj, msg: msg}
}
