package reconciler

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	lbImage = "load-balancer"
)

type LoadBalancer struct {
	currentStatus *appsv1.Deployment
	desiredStatus *appsv1.Deployment
}

func (l *LoadBalancer) getModel() (*appsv1.Deployment, error) {
	return getDeploymentModel("deployment/load-balancer.yaml")
}

func (l *LoadBalancer) insertParamters(dep *appsv1.Deployment, cr *meridiov1alpha1.Trench) *appsv1.Deployment {
	// if status load-balancer deployment parameters are specified in the cr, use those
	// else use the default parameters
	dep.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	dep.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s/%s:%s", Registry, Organization, lbImage, Tag)
	dep.Spec.Template.Spec.Containers[0].ImagePullPolicy = PullPolicy
	dep.Spec.Template.Spec.Containers[0].LivenessProbe = GetLivenessProbe(cr)
	dep.Spec.Template.Spec.Containers[0].ReadinessProbe = GetReadinessProbe(cr)
	return dep
}

func (l *LoadBalancer) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      "load-balancer",
	}
}

func (l *LoadBalancer) getCurrentStatus(ctx context.Context, cr *meridiov1alpha1.Trench, client client.Client) error {
	currentState := &appsv1.Deployment{}
	err := client.Get(ctx, l.getSelector(cr), currentState)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	l.currentStatus = currentState.DeepCopy()
	return nil
}

func (l *LoadBalancer) getDesiredStatus(cr *meridiov1alpha1.Trench) error {
	dep, err := l.getModel()
	if err != nil {
		return err
	}
	l.desiredStatus = l.insertParamters(dep, cr)
	return nil
}

// getReconciledDesiredStatus gets the desired status of load-balancer deployment after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *LoadBalancer) getReconciledDesiredStatus(lb *appsv1.Deployment, cr *meridiov1alpha1.Trench) {
	lb = i.insertParamters(lb, cr)
	i.desiredStatus = lb
}

func (l *LoadBalancer) getAction(e *Executor, cr *meridiov1alpha1.Trench) (Action, error) {
	var action Action
	err := l.getCurrentStatus(e.ctx, cr, e.client)
	if err != nil {
		return nil, err
	}
	if l.currentStatus == nil {
		err = l.getDesiredStatus(cr)
		if err != nil {
			return nil, err
		}
		e.log.Info("load-balancer", "add action", "create")
		action = newCreateAction(l.desiredStatus, "create load-balncer deployment")
	} else {
		l.getReconciledDesiredStatus(l.currentStatus, cr)
		if !equality.Semantic.DeepEqual(l.desiredStatus, l.currentStatus) {
			e.log.Info("load-balancer", "add action", "update")
			action = newUpdateAction(l.desiredStatus, "update load-balncer deployment")
		}
	}
	return action, nil
}
