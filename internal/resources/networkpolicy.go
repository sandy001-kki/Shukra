// This file builds the NetworkPolicy resource from canonical v1beta1 spec
// rules. It exists so tenancy and ingress/egress translation stay centralized.
package resources

import (
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func NetworkPolicy(appEnv *appsv1beta1.AppEnvironment) *networkingv1.NetworkPolicy {
	ingressRules := appEnv.Spec.Security.NetworkPolicy.IngressRules
	if len(ingressRules) == 0 {
		// Default to same-namespace traffic on the application port when network
		// policy is conceptually enabled but the user did not specify rules.
		ingressRules = []networkingv1.NetworkPolicyIngressRule{{
			From: []networkingv1.NetworkPolicyPeer{{
				PodSelector: &metav1.LabelSelector{},
			}},
		}}
	}
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name(appEnv, "netpol"),
			Namespace: appEnv.Namespace,
			Labels:    appEnv.Labels("networkpolicy"),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: appEnv.Labels("app")},
			Ingress:     ingressRules,
			Egress:      appEnv.Spec.Security.NetworkPolicy.EgressRules,
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
		},
	}
}
