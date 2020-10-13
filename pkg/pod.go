package pod

import (
	"fmt"
	"log"
	"os"
	"context" 

	guuid "github.com/google/uuid"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type PodArgs struct {
	PodNamespace	string
	Client			*kubernetes.Clientset
	FileID			string
	Input			string
	Output			string
}

func NewPodArgs(fileId, input, output string) (podArgs *PodArgs, err error){
	podNamespace := os.Getenv("POD_NAMESPACE")

	if podNamespace == "" {
		err = fmt.Errorf("init failed: POD_NAMESPACE environment variable not set")
		return
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	podArgs = &PodArgs{
		PodNamespace: podNamespace,
		Client:	client,
		FileID: fileId,
		Input: input,
		Output: output,
	}
	return
}

func (pa PodArgs) CreatePod() error {
	podSpec := pa.GetPodObject()

	pod, err := pa.Client.CoreV1().Pods("argo-events").Create(context.TODO(), podSpec, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	if pod != nil {
		log.Printf("Successfully created Pod")
	}

	return nil
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
					ImagePullPolicy: core.PullIfNotPresent,
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