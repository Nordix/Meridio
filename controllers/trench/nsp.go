package trench

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	imageNsp = "nsp"
)

type NspStatefulSet struct {
	trench *meridiov1alpha1.Trench
	model  *appsv1.StatefulSet
	exec   *common.Executor
}

func NewNspStatefulSet(e *common.Executor, t *meridiov1alpha1.Trench) (*NspStatefulSet, error) {
	l := &NspStatefulSet{
		trench: t.DeepCopy(),
		exec:   e,
	}

	// get model
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (i *NspStatefulSet) getEnvVars(allEnv []corev1.EnvVar) []corev1.EnvVar {
	ret := []corev1.EnvVar{}
	for _, env := range allEnv {
		switch env.Name {
		case "NSP_PORT":
			env.Value = fmt.Sprint(common.NspTargetPort)
		case "NSP_CONFIG_MAP_NAME":
			env.Value = common.ConfigMapName(i.trench)
		case "NSP_NAMESPACE":
			env.Value = i.trench.ObjectMeta.Namespace
		case "SPIFFE_ENDPOINT_SOCKET", "NSP_DATASOURCE", "NSP_LOG_LEVEL":
		default:
			i.exec.LogError(fmt.Errorf("env %s not expected", env.Name), "get env var error")
		}
		ret = append(ret, env)
	}
	return ret
}

func (i *NspStatefulSet) insertParameters(init *appsv1.StatefulSet) *appsv1.StatefulSet {
	// if status nsp statefulset parameters are specified in the cr, use those
	// else use the default parameters
	nspStatefulSetName := common.NSPStatefulSetName(i.trench)
	ret := init.DeepCopy()
	ret.ObjectMeta.Name = nspStatefulSetName
	ret.ObjectMeta.Namespace = i.trench.ObjectMeta.Namespace
	ret.ObjectMeta.Labels["app"] = nspStatefulSetName
	ret.Spec.Selector.MatchLabels["app"] = nspStatefulSetName
	ret.Spec.ServiceName = nspStatefulSetName
	ret.Spec.Template.ObjectMeta.Labels["app"] = nspStatefulSetName
	ret.Spec.Template.Spec.ServiceAccountName = common.ServiceAccountName(i.trench)

	ret.Spec.Template.Spec.ImagePullSecrets = common.GetImagePullSecrets()

	// check resource requirement annotation update, and save annotation into deployment for visibility
	oa, _ := common.GetResourceRequirementAnnotation(&init.ObjectMeta)
	if na, _ := common.GetResourceRequirementAnnotation(&i.trench.ObjectMeta); na != oa {
		common.SetResourceRequirementAnnotation(&i.trench.ObjectMeta, &ret.ObjectMeta)
	}

	for k, container := range ret.Spec.Template.Spec.Containers {
		switch name := container.Name; name {
		case "nsp":
			if container.Image == "" {
				container.Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, imageNsp, common.Tag)
				container.ImagePullPolicy = corev1.PullAlways
			}
			if container.LivenessProbe == nil {
				container.LivenessProbe = common.GetProbe(common.StartUpTimer,
					common.GetProbeCommand(false, "unix:///tmp/health.sock", ""))
			}
			if container.ReadinessProbe == nil {
				container.ReadinessProbe = common.GetProbe(common.ReadinessTimer,
					common.GetProbeCommand(true, fmt.Sprintf(":%d", common.NspPort), ""))
			}
			if container.LivenessProbe == nil {
				container.LivenessProbe = common.GetProbe(common.LivenessTimer,
					common.GetProbeCommand(false, "unix:///tmp/health.sock", ""))
			}
			container.Env = i.getEnvVars(container.Env)
			// set resource requirements for container (if not found, then values from model
			// are kept even upon updates, as getReconciledDesiredStatus() overwrites containers)
			if err := common.SetContainerResourceRequirements(&i.trench.ObjectMeta, &container); err != nil {
				i.exec.LogInfo(fmt.Sprintf("trench %s, %v", i.trench.ObjectMeta.Name, err))
			}
		default:
			i.exec.LogError(fmt.Errorf("container %s not expected", name), "get container error")
		}
		ret.Spec.Template.Spec.Containers[k] = container
	}

	return ret
}

func (i *NspStatefulSet) getModel() error {
	model, err := common.GetStatefulSetModel("deployment/nsp.yaml")
	if err != nil {
		return err
	}
	i.model = model
	return nil
}

func (i *NspStatefulSet) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: i.trench.ObjectMeta.Namespace,
		Name:      common.NSPStatefulSetName(i.trench),
	}
}

func (i *NspStatefulSet) getDesiredStatus() *appsv1.StatefulSet {
	return i.insertParameters(i.model)
}

// getReconciledDesiredStatus gets the desired status of nsp StatefulSet after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *NspStatefulSet) getReconciledDesiredStatus(cd *appsv1.StatefulSet) *appsv1.StatefulSet {
	template := cd.DeepCopy()
	template.Spec.Template.Spec.InitContainers = i.model.Spec.Template.Spec.InitContainers
	template.Spec.Template.Spec.Containers = i.model.Spec.Template.Spec.Containers
	template.Spec.Template.Spec.Volumes = i.model.Spec.Template.Spec.Volumes
	return i.insertParameters(template)
}

func (i *NspStatefulSet) getCurrentStatus() (*appsv1.StatefulSet, error) {
	currentStatus := &appsv1.StatefulSet{}
	selector := i.getSelector()
	err := i.exec.GetObject(selector, currentStatus)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return currentStatus, nil
}

func (i *NspStatefulSet) getAction() error {
	cs, err := i.getCurrentStatus()
	if err != nil {
		return err
	}
	if cs == nil {
		ds := i.getDesiredStatus()
		if err != nil {
			return err
		}
		i.exec.AddCreateAction(ds)
	} else {
		ds := i.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds.Spec, cs.Spec) {
			i.exec.AddUpdateAction(ds)
		}
	}
	return nil
}
