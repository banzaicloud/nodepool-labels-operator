package npls

import (
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	v1alpha1 "github.com/banzaicloud/nodepool-labels-operator/pkg/apis/nodepoollabelset/v1alpha1"
	clientset "github.com/banzaicloud/nodepool-labels-operator/pkg/client/clientset/versioned"
)

type LabelSet map[string]string
type NodepoolLabelSets map[string]LabelSet

type Manager struct {
	namespace string
	clientset clientset.Interface
}

func NewNPLSManager(k8sConfig *rest.Config, namespace string) (*Manager, error) {
	clientset, err := clientset.NewForConfig(k8sConfig)
	if err != nil {
		return nil, emperror.Wrap(err, "could not get k8s npls clientset")
	}

	return &Manager{
		namespace: namespace,
		clientset: clientset,
	}, nil
}

func (m *Manager) Get(name string) (LabelSet, error) {
	npls, err := m.clientset.LabelsV1alpha1().NodePoolLabelSets(m.namespace).Get(name, v1.GetOptions{})
	if err != nil {
		return nil, emperror.WrapWith(err, "could not get npls", "name", name)
	}

	return LabelSet(npls.Spec.Labels), nil
}

func (m *Manager) GetAll() (NodepoolLabelSets, error) {
	nplss, err := m.clientset.LabelsV1alpha1().NodePoolLabelSets(m.namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, emperror.Wrap(err, "could not list npls resources")
	}

	sets := make(NodepoolLabelSets)
	for _, npls := range nplss.Items {
		sets[npls.Name] = LabelSet(npls.Spec.Labels)
	}

	return sets, nil
}

func (m *Manager) Sync(sets NodepoolLabelSets) error {
	merr := emperror.NewMultiErrorBuilder()
	for poolName, labelSet := range sets {
		if len(labelSet) == 0 {
			err := m.Delete(poolName)
			if err != nil {
				merr.Add(err)
			}
			continue
		}
		err := m.UpdateOrCreate(poolName, labelSet)
		if err != nil {
			merr.Add(err)
		}
	}

	return merr.ErrOrNil()
}

func (m *Manager) UpdateOrCreate(name string, labelSet LabelSet) error {
	err := m.Update(name, labelSet)
	if err != nil && k8serrors.IsNotFound(errors.Cause(err)) {
		err = m.Create(name, labelSet)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func (m *Manager) Update(name string, labelSet LabelSet) error {
	npls, err := m.clientset.LabelsV1alpha1().NodePoolLabelSets(m.namespace).Get(name, v1.GetOptions{})
	if err != nil {
		return emperror.WrapWith(err, "could not get npls", "name", name)
	}

	npls.Spec.Labels = labelSet
	_, err = m.clientset.LabelsV1alpha1().NodePoolLabelSets(m.namespace).Update(npls)
	if err != nil {
		return emperror.WrapWith(err, "could not update npls", "name", name)
	}

	return nil
}

func (m *Manager) Delete(name string) error {
	err := m.clientset.LabelsV1alpha1().NodePoolLabelSets(m.namespace).Delete(name, &v1.DeleteOptions{})
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return emperror.WrapWith(err, "could not delete npls", "name", name)
	}

	return nil
}

func (m *Manager) Create(name string, labelSet LabelSet) error {
	_, err := m.clientset.LabelsV1alpha1().NodePoolLabelSets(m.namespace).Create(
		&v1alpha1.NodePoolLabelSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: m.namespace,
			},
			Spec: v1alpha1.NodePoolLabelSetSpec{
				Labels: labelSet,
			},
		},
	)

	if err != nil {
		return emperror.WrapWith(err, "could not create npls", "name", name)
	}

	return nil
}
