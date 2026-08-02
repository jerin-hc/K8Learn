package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	studentappv1 "ij3rry.com/student-operator/api/v1"
)

func (r *StudentAppReconciler) recouncilerPostgres(ctx context.Context, app *studentappv1.StudentApp, service *corev1.Service, log logr.Logger) (bool, error) {
	//get secrets for postgres
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      app.Spec.Credentials.SecretName,
		Namespace: app.Namespace,
	}, secret); err != nil {
		return false, err
	}
	log.Info("Postgres secret loaded for ", "app", app.Name)

	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      getResourceName("postgres-pvc", app.Name),
		Namespace: app.Namespace,
	}, pvc); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
		pvc, err = r.createPVC(ctx, app)
		if err != nil {
			return false, err
		}
		log.Info("PVC created for ", "app", app.Name)
	}

	if err := r.Get(ctx, types.NamespacedName{
		Name:      getResourceName("postgres-service", app.Name),
		Namespace: app.Namespace,
	}, service); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
		service, err = r.createHeadlesService(ctx, app)
		if err != nil {
			return false, err
		}
		log.Info("Service created for ", "app", app.Name)

	}

	sts := &appsv1.StatefulSet{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      getResourceName("postgres-sts", app.Name),
		Namespace: app.Namespace,
	}, sts); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
		sts, err = r.createSTS(ctx, app, service, secret, pvc)
		if err != nil {
			return false, err
		}
		log.Info("StatefulSet created for ", "app", app.Name)
	}

	if sts.Status.ObservedGeneration != sts.Generation {
		return false, nil
	}
	if sts.Status.ReadyReplicas < *sts.Spec.Replicas {
		return false, nil
	}
	return true, nil
}

func (r *StudentAppReconciler) createPVC(ctx context.Context, app *studentappv1.StudentApp) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: generateMeta("postgres-pvc", app),
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: app.Spec.StorageSize,
				},
			},
		},
	}

	if err := r.Create(ctx, pvc); err != nil {
		return nil, fmt.Errorf("error creating persistent volume claim for %s: %w", app.Name, err)
	}

	return pvc, nil
}

func (r *StudentAppReconciler) createHeadlesService(ctx context.Context, app *studentappv1.StudentApp) (*corev1.Service, error) {
	service := &corev1.Service{
		ObjectMeta: generateMeta("postgres-service", app),
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector: map[string]string{
				"app": "postgres",
			},
			Ports: []corev1.ServicePort{{
				Port:       5432,
				TargetPort: intstr.FromInt(5432),
			}},
		},
	}
	if err := r.Create(ctx, service); err != nil {
		return nil, fmt.Errorf("error creating headles service for %s: %w", app.Name, err)
	}
	return service, nil
}

func (r *StudentAppReconciler) createSTS(ctx context.Context, app *studentappv1.StudentApp, service *corev1.Service, secret *corev1.Secret, pvc *corev1.PersistentVolumeClaim) (*appsv1.StatefulSet, error) {
	replicas := int32(1)
	sts := &appsv1.StatefulSet{
		ObjectMeta: generateMeta("postgres-sts", app),
		Spec: appsv1.StatefulSetSpec{
			ServiceName: service.Name,
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: service.Spec.Selector,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "postgres",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "postgres",
							Image: "postgres:15-alpine",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5432,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "POSTGRES_USER",
									Value: string(secret.Data[app.Spec.Credentials.UsernameKey]),
								},
								{
									Name:  "POSTGRES_PASSWORD",
									Value: string(secret.Data[app.Spec.Credentials.PasswordKey]),
								},
								{
									Name:  "POSTGRES_DB",
									Value: string(secret.Data[app.Spec.Credentials.DatabaseKey]),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "postgres-storage",
									MountPath: "/var/lib/postgresql/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "postgres-storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvc.Name,
								},
							},
						},
					},
				},
			},
		},
	}

	if err := r.Create(ctx, sts); err != nil {
		return nil, fmt.Errorf("error creating statefulset for %s: %w", app.Name, err)
	}

	return sts, nil
}
