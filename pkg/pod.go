package pod

import (
	"log"
	"context"
	"flag"
	"k8s.io/client-go/tools/clientcmd"
	"path/filepath"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/rest"
	"k8s.io/client-go/util/homedir"
	guuid "github.com/google/uuid"
)

type PodArgs struct {
	PodName			string
	PodNamespace 	string
	FileID 			string
	Input			string
	Output			string
}

func (pa PodArgs) Create() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// config, err := rest.InClusterConfig()
	// if err != nil {
	// 	return
	// }

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	podSpec := pa.GetPodObject()

	pod, err := client.CoreV1().Pods("argo-events").Create(context.TODO(), podSpec, metav1.CreateOptions{})
	if err != nil {
		log.Fatal(err)
	}

	if pod != nil {
		log.Printf("Successfully created Pod")
	}
}

func (pa PodArgs) GetPodObject() *core.Pod {
	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rebuild-" + guuid.New().String(),
			Namespace: "argo-events",
		},
		Spec: core.PodSpec{
			ImagePullSecrets: []core.LocalObjectReference{{Name: "regcred"}},
			Volumes: []core.Volume{
				{
					Name: "sourcedir",
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: "glasswallsource-pvc",
						},
					},
				},
				{
					Name: "targetdir",
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: "glasswalltarget-pvc",
						},
					},
				},
			},
			Containers: []core.Container{
				{
					Name: "rebuild",
					Image: "diggers/icap-rebuild",
					ImagePullPolicy: core.PullAlways,
					Env: []core.EnvVar{
						{Name: "FILE_ID", Value: pa.FileID},
						{Name: "INPUT_PATH", Value: pa.Input},
						{Name: "OUTPUT_PATH", Value: pa.Output},
					},
					VolumeMounts: []core.VolumeMount{
						{Name: "sourcedir",	MountPath: "/input"},
						{Name: "targetdir",	MountPath: "/output"},
					},
				},
			},
		},
	}
}