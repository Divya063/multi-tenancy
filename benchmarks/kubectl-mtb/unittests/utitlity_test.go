package unittests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sigs.k8s.io/kind/pkg/cluster"
	"testing"

	tenancyv1alpha1 "github.com/kubernetes-sigs/multi-tenancy/tenant/pkg/apis/tenancy/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tenant2 "github.com/kubernetes-sigs/multi-tenancy/tenant/pkg/controller/tenant"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"k8s.io/client-go/kubernetes/scheme"
	"github.com/kubernetes-sigs/multi-tenancy/tenant/pkg/apis"
)

var cfg *rest.Config
var c client.Client
var err error

func TestMain(m *testing.M) {
	provider, kubeConfig, clusterName, err := setup()
	tr := true
	apis.AddToScheme(scheme.Scheme)
	fmt.Println()
	t := &envtest.Environment{
		CRDDirectoryPaths:  []string{filepath.Join("..", "crds")},
		UseExistingCluster: &tr,
	}

	if cfg, err = t.Start(); err != nil {
		fmt.Println(err)
	}
	//fmt.Println(cfg)
	code := m.Run()
	t.Stop()
	teardown(provider, kubeConfig, clusterName)
	os.Exit(code)
}

func TestCreateTenant(t *testing.T) {

	tenant := &tenancyv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: "check",
		},
		Spec: tenancyv1alpha1.TenantSpec{
			TenantAdminNamespaceName: "t1admin",
		},
	}

	instance := &tenancyv1alpha1.TenantNamespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "t1admin",
			Namespace: tenant.Spec.TenantAdminNamespaceName,
		},
		Spec: tenancyv1alpha1.TenantNamespaceSpec{
			Name: "t1admi",
		},
	}

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		fmt.Println(err)
	}
	c = mgr.GetClient()
	err = tenant2.Add(mgr)
	if err != nil {
		fmt.Println(err)
	}


	err = c.Create(context.TODO(), tenant)
	if err != nil {
		fmt.Println(err)
	}

	err = c.Create(context.TODO(), instance)
	if err != nil {
		fmt.Println(err)
	}

}

func setup() (*cluster.Provider, string, string, error){
	// Do something here.

	provider, kubeConfig, clusterName, err := CreateCluster()


	if err != nil {
		fmt.Printf("\033[1;36m%s\033[0m", "> Setup completed\n")
	} else {
		fmt.Printf("\033[1;36m%s\033[0m", "> Setup failed\n")
		fmt.Printf(err.Error())
	}

	return provider, kubeConfig, clusterName, err
}

func teardown(provider *cluster.Provider, kubeConfig string, cluster string) {


	DeleteCluster(provider, cluster, kubeConfig)
	fmt.Printf("\033[1;36m%s\033[0m", "> Teardown completed")
	fmt.Printf("\n")
}
