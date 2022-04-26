package trench

import (
	"fmt"
	"strconv"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	imageIpam = "ipam"
)

type IpamStatefulSet struct {
	trench *meridiov1alpha1.Trench
	model  *appsv1.StatefulSet
	exec   *common.Executor
}

func NewIPAM(e *common.Executor, t *meridiov1alpha1.Trench) (*IpamStatefulSet, error) {
	l := &IpamStatefulSet{
		trench: t.DeepCopy(),
		exec:   e,
	}

	// get model
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (i *IpamStatefulSet) getEnvVars(allEnv []corev1.EnvVar) []corev1.EnvVar {
	// if envVars are set in the cr, use the values
	// else return default envVars
	env := []corev1.EnvVar{
		{
			Name:  "IPAM_PORT",
			Value: strconv.Itoa(common.IpamPort),
		},
		{
			Name:  "IPAM_NAMESPACE",
			Value: i.trench.ObjectMeta.Namespace,
		},
		{
			Name:  "IPAM_TRENCH_NAME",
			Value: i.trench.ObjectMeta.GetName(),
		},
		{
			Name:  "IPAM_NSP_SERVICE",
			Value: common.NSPServiceWithPort(i.trench),
		},
		{
			Name:  "IPAM_PREFIX_IPV4",
			Value: common.SubnetPoolIpv4,
		},
		{
			Name:  "IPAM_PREFIX_IPV6",
			Value: common.SubnetPoolIpv6,
		},
		{
			Name:  "IPAM_CONDUIT_PREFIX_LENGTH_IPV4",
			Value: common.ConduitPrefixLengthIpv4,
		},
		{
			Name:  "IPAM_CONDUIT_PREFIX_LENGTH_IPV6",
			Value: common.ConduitPrefixLengthIpv6,
		},
		{
			Name:  "IPAM_NODE_PREFIX_LENGTH_IPV4",
			Value: common.NodePrefixLengthIpv4,
		},
		{
			Name:  "IPAM_NODE_PREFIX_LENGTH_IPV6",
			Value: common.NodePrefixLengthIpv6,
		},
		{
			Name:  "IPAM_IP_FAMILY",
			Value: common.GetIPFamily(i.trench),
		},
	}

	for _, e := range allEnv {
		// append all hard coded envVars
		if e.Name == "SPIFFE_ENDPOINT_SOCKET" ||
			e.Name == "IPAM_DATASOURCE" ||
			e.Name == "IPAM_LOG_LEVEL" {
			env = append(env, e)
		}
	}

	return env
}

func (i *IpamStatefulSet) insertParameters(dep *appsv1.StatefulSet) *appsv1.StatefulSet {
	// if status ipam statefulset parameters are specified in the cr, use those
	// else use the default parameters
	ret := dep.DeepCopy()
	ipamStatefulSetName := common.IPAMStatefulSetName(i.trench)
	ret.ObjectMeta.Name = ipamStatefulSetName
	ret.ObjectMeta.Namespace = i.trench.ObjectMeta.Namespace
	ret.ObjectMeta.Labels["app"] = ipamStatefulSetName
	ret.Spec.Selector.MatchLabels["app"] = ipamStatefulSetName
	ret.Spec.Template.ObjectMeta.Labels["app"] = ipamStatefulSetName

	ret.Spec.ServiceName = ipamStatefulSetName

	ret.Spec.Template.Spec.ImagePullSecrets = common.GetImagePullSecrets()

	// check resource requirement annotation update, and save annotation into deployment for visibility
	oa, _ := common.GetResourceRequirementAnnotation(&dep.ObjectMeta)
	if na, _ := common.GetResourceRequirementAnnotation(&i.trench.ObjectMeta); na != oa {
		common.SetResourceRequirementAnnotation(&i.trench.ObjectMeta, &ret.ObjectMeta)
	}

	for x, container := range ret.Spec.Template.Spec.Containers {
		switch name := container.Name; name {
		case "ipam":
			if container.Image == "" {
				container.Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, imageIpam, common.Tag)
				container.ImagePullPolicy = corev1.PullAlways
			}
			if container.StartupProbe == nil {
				container.StartupProbe = common.GetProbe(common.StartUpTimer,
					common.GetProbeCommand(false, "unix:///tmp/health.sock", ""))
			}
			if container.ReadinessProbe == nil {
				container.ReadinessProbe = common.GetProbe(common.ReadinessTimer,
					common.GetProbeCommand(true, fmt.Sprintf(":%d", common.IpamPort), ""))
			}
			if container.LivenessProbe == nil {
				container.LivenessProbe = common.GetProbe(common.LivenessTimer,
					common.GetProbeCommand(false, "unix:///tmp/health.sock", ""))
			}
			container.Env = i.getEnvVars(ret.Spec.Template.Spec.Containers[0].Env)
			// set resource requirements for container (if not found, then values from model
			// are kept even upon updates, as getReconciledDesiredStatus() overwrites containers)
			if err := common.SetContainerResourceRequirements(&i.trench.ObjectMeta, &container); err != nil {
				i.exec.LogInfo(fmt.Sprintf("trench %s, %v", i.trench.ObjectMeta.Name, err))
			}
		default:
			i.exec.LogError(fmt.Errorf("container %s not expected", name), "get container error")
		}
		ret.Spec.Template.Spec.Containers[x] = container
	}
	return ret
}

func (i *IpamStatefulSet) getModel() error {
	model, err := common.GetStatefulSetModel("deployment/ipam.yaml")
	if err != nil {
		return err
	}
	i.model = model
	return nil
}

func (i *IpamStatefulSet) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: i.trench.ObjectMeta.Namespace,
		Name:      common.IPAMStatefulSetName(i.trench),
	}
}

func (i *IpamStatefulSet) getDesiredStatus() *appsv1.StatefulSet {
	return i.insertParameters(i.model)
}

// getReconciledDesiredStatus gets the desired status of ipam statefulset after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *IpamStatefulSet) getReconciledDesiredStatus(cd *appsv1.StatefulSet) *appsv1.StatefulSet {
	template := cd.DeepCopy()
	template.Spec.Template.Spec.InitContainers = i.model.Spec.Template.Spec.InitContainers
	template.Spec.Template.Spec.Containers = i.model.Spec.Template.Spec.Containers
	template.Spec.Template.Spec.Volumes = i.model.Spec.Template.Spec.Volumes
	return i.insertParameters(template)
}

func (i *IpamStatefulSet) getCurrentStatus() (*appsv1.StatefulSet, error) {
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

func (i *IpamStatefulSet) getAction() error {
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
