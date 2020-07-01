package unittests

import (
	"fmt"
	"log"
	"sync"

	"github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"
)

func getUniqueName(str, a string) string {
	return fmt.Sprintf("%+v-%+v", str, a)
}

// StartTestManager adds recFn
func StartTestManager(mgr manager.Manager, g *gomega.GomegaWithT) (chan struct{}, *sync.WaitGroup) {
	stop := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		g.Expect(mgr.Start(stop)).NotTo(gomega.HaveOccurred())
	}()
	return stop, wg
}

func CreateCluster() (*cluster.Provider, string, string, error) {
	clusterName := "kubectl-mtb"
	nodeList := []v1alpha4.Node{}

	nodeCount := 1
	for i := 0; i < nodeCount; i++ {
		node := v1alpha4.Node{
			Role: v1alpha4.NodeRole("control-plane"),
		}
		nodeList = append(nodeList, node)
	}
	config := &v1alpha4.Cluster{
		Nodes: nodeList,
	}
	options := cluster.CreateWithV1Alpha4Config(config)
	newProvider := cluster.NewProvider()
	if err := newProvider.Create(
		clusterName,
		options,
	); err != nil {
		fmt.Println(err.Error())
		return newProvider, "", clusterName, err
	}
	//fmt.Printf("ctx.KubeConfigPath(): %s\n", newProvider.List)
	kubeConfig, err := newProvider.KubeConfig(clusterName, false)

	if err != nil {
		log.Println(err.Error())
	}

	fmt.Println("cluster is created")

	return newProvider, kubeConfig, clusterName, nil
}

func CreateTenants() {

}

func getClientSet(configPath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func DeleteCluster(provider *cluster.Provider, name string, configPath string) error {

	err := provider.Delete(name, configPath)
	fmt.Println(err)
	if err != nil {
		fmt.Println(err.Error())
	}

	return err

}

// func main() {

// 	provider, config, name, err := CreateCluster()
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	// fmt.Println(provider, config, name)

// 	err = DeleteCluster(provider, name, config)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// }
