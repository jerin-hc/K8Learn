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

func (r *StudentAppReconciler) reconcilerStudentApp(ctx context.Context, app *studentappv1.StudentApp, postgresService *corev1.Service, log logr.Logger) (bool, error) {
	// get secret for student app
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      app.Spec.Credentials.SecretName,
		Namespace: app.Namespace,
	}, secret); err != nil {
		return false, err
	}
	log.Info("Secret loaded for student app", "app", app.Name)

	// get or create ConfigMap
	configMap := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      getResourceName("app-config", app.Name),
		Namespace: app.Namespace,
	}, configMap); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
		configMap, err = r.createConfigMap(ctx, app, secret, postgresService)
		if err != nil {
			return false, err
		}
		log.Info("ConfigMap created for ", "app", app.Name)
	}

	// get or create Deployment
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      getResourceName("app-deploy", app.Name),
		Namespace: app.Namespace,
	}, deployment); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
		deployment, err = r.createDeployment(ctx, app, configMap, secret)
		if err != nil {
			return false, err
		}
		log.Info("Deployment created for ", "app", app.Name)
	}

	// get or create Service (NodePort)
	service := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      getResourceName("app-svc", app.Name),
		Namespace: app.Namespace,
	}, service); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
		_, err = r.createStudentService(ctx, app)
		if err != nil {
			return false, err
		}
		log.Info("Service created for ", "app", app.Name)
	}

	// check if deployment is ready
	if deployment.Status.ObservedGeneration != deployment.Generation {
		return false, nil
	}
	if deployment.Status.ReadyReplicas < *deployment.Spec.Replicas {
		return false, nil
	}

	return true, nil
}

func (r *StudentAppReconciler) createConfigMap(ctx context.Context, app *studentappv1.StudentApp, secret *corev1.Secret, service *corev1.Service) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: generateMeta("app-config", app),
		Data: map[string]string{
			"POSTGRES_HOST": service.Name,
			"POSTGRES_PORT": "5432",
			"POSTGRES_NAME": string(secret.Data[app.Spec.Credentials.DatabaseKey]),
		},
	}

	if err := r.Create(ctx, configMap); err != nil {
		return nil, fmt.Errorf("error creating configmap for %s: %w", app.Name, err)
	}

	return configMap, nil
}

func (r *StudentAppReconciler) createDeployment(ctx context.Context, app *studentappv1.StudentApp, configMap *corev1.ConfigMap, secret *corev1.Secret) (*appsv1.Deployment, error) {
	replicas := int32(2)
	deployment := &appsv1.Deployment{
		ObjectMeta: generateMeta("app-deploy", app),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "student-app",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "student-app",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "student-app",
							Image:           "docker.io/ij3rry/student-db:v0.0.2",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "POSTGRES_HOST",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: configMap.Name,
											},
											Key: "POSTGRES_HOST",
										},
									},
								},
								{
									Name: "POSTGRES_PORT",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: configMap.Name,
											},
											Key: "POSTGRES_PORT",
										},
									},
								},
								{
									Name: "POSTGRES_NAME",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: configMap.Name,
											},
											Key: "POSTGRES_NAME",
										},
									},
								},
								{
									Name: "POSTGRES_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: secret.Name,
											},
											Key: app.Spec.Credentials.UsernameKey,
										},
									},
								},
								{
									Name: "POSTGRES_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: secret.Name,
											},
											Key: app.Spec.Credentials.PasswordKey,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := r.Create(ctx, deployment); err != nil {
		return nil, fmt.Errorf("error creating deployment for %s: %w", app.Name, err)
	}

	return deployment, nil
}

func (r *StudentAppReconciler) createStudentService(ctx context.Context, app *studentappv1.StudentApp) (*corev1.Service, error) {
	service := &corev1.Service{
		ObjectMeta: generateMeta("app-svc", app),
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				"app": "student-app",
			},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}

	if err := r.Create(ctx, service); err != nil {
		return nil, fmt.Errorf("error creating service for %s: %w", app.Name, err)
	}

	return service, nil
}
