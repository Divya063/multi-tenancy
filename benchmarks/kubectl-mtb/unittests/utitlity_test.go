package unittests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	v1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/kind/pkg/cluster"

	"github.com/kubernetes-sigs/multi-tenancy/tenant/pkg/apis"
	tenancyv1alpha1 "github.com/kubernetes-sigs/multi-tenancy/tenant/pkg/apis/tenancy/v1alpha1"
	tenant2 "github.com/kubernetes-sigs/multi-tenancy/tenant/pkg/controller/tenant"
	tenantnamespace "github.com/kubernetes-sigs/multi-tenancy/tenant/pkg/controller/tenantnamespace"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var cfg *rest.Config
var c client.Client
var err error

const timeout = time.Second * 5

func TestMain(m *testing.M) {
	// provider, kubeConfig, clusterName, err := setup()
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
	// teardown(provider, kubeConfig, clusterName)
	os.Exit(code)
}

func TestCreateTenant(t *testing.T) {

	g := gomega.NewGomegaWithT(t)

	// sa := corev1.ServiceAccount{
	// 	TypeMeta: metav1.TypeMeta{
	// 		Kind: "ServiceAccount",
	// 	},
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      "t1-admin1",
	// 		Namespace: "default",
	// 	},
	// }
	tenant := &tenancyv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tenant-sample",
		},
		Spec: tenancyv1alpha1.TenantSpec{
			TenantAdminNamespaceName: "tenant1admin",
			TenantAdmins: []v1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "t1-user1",
					Namespace: "default",
				},
			},
		},
	}

	instance := &tenancyv1alpha1.TenantNamespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tenantnamespace-sample",
			Namespace: tenant.Spec.TenantAdminNamespaceName,
		},
		Spec: tenancyv1alpha1.TenantNamespaceSpec{
			Name: "t1-ns1",
		},
	}

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		fmt.Println(err)
	}

	c = mgr.GetClient()

	//start tenant controller
	recFnTenant, _ := tenant2.SetupTestReconcile(tenant2.SetupNewReconciler(mgr))
	g.Expect(tenant2.AddManager(mgr, recFnTenant)).NotTo(gomega.HaveOccurred())

	//start tenantnamespace control
	recFnTenantNS, _ := tenantnamespace.SetupTestReconcile(tenantnamespace.NewReconciler(mgr))
	fmt.Println(recFnTenantNS)
	g.Expect(tenantnamespace.AddManager(mgr, recFnTenantNS)).NotTo(gomega.HaveOccurred())
	// err = tenantnamespace.Add(mgr)
	// if err != nil {
	// 	fmt.Println(err)
	// }

	stopMgr, mgrStopped := StartTestManager(mgr, g)
	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()
	// err = c.Create(context.TODO(), &sa)
	// if err != nil {
	// 	t.Logf("Failed to create tenant admin: %+v with error: %+v", sa.ObjectMeta.Name, err)
	// 	return
	// }

	err = c.Create(context.TODO(), tenant)
	if err != nil {
		fmt.Println(err)
	}
	err = c.Create(context.TODO(), instance)
	if err != nil {
		fmt.Println(err)
	}

}

func setup() (*cluster.Provider, string, string, error) {
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
