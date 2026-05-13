package bridge

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	bridgev1 "github.com/sandy001-kki/Shukra/api/bridge/v1"
	"github.com/sandy001-kki/Shukra/internal/shadow"
)

type Server struct {
	bridgev1.AionosBridgeServer
	client client.WithWatch
	scheme *runtime.Scheme
}

func NewServer(c client.WithWatch, scheme *runtime.Scheme) *Server {
	return &Server{client: c, scheme: scheme}
}

func NewGRPCServer(tlsConfig *tls.Config, bridgeServer bridgev1.AionosBridgeServer, requireClientCert bool) *grpc.Server {
	opts := []grpc.ServerOption{}
	if requireClientCert {
		opts = append(opts,
			grpc.UnaryInterceptor(validateClientCertificateUnary),
			grpc.StreamInterceptor(validateClientCertificateStream),
		)
	}
	if tlsConfig != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}
	server := grpc.NewServer(opts...)
	bridgev1.RegisterAionosBridgeServer(server, bridgeServer)
	reflection.Register(server)
	return server
}

func (s *Server) EnsureShadowNamespace(ctx context.Context) error {
	ns := &corev1.Namespace{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: shadow.ShadowNamespace}, ns); err == nil {
		return nil
	} else if !apierrors.IsNotFound(err) {
		return err
	}
	return s.client.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: shadow.ShadowNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "shukra-bridge",
				"aionos.io/shadow-namespace":   "true",
			},
		},
	})
}

func validateClientCertificateUnary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if err := validateClientCertificate(ctx); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

func validateClientCertificateStream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := validateClientCertificate(ss.Context()); err != nil {
		return err
	}
	return handler(srv, ss)
}

func validateClientCertificate(ctx context.Context) error {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing peer information")
	}
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return status.Error(codes.Unauthenticated, "mTLS is required")
	}
	if len(tlsInfo.State.PeerCertificates) == 0 {
		return status.Error(codes.Unauthenticated, "client certificate is required")
	}
	if time.Now().After(tlsInfo.State.PeerCertificates[0].NotAfter) {
		return status.Error(codes.Unauthenticated, "client certificate is expired")
	}
	return nil
}

func intervalSeconds(requested int32) time.Duration {
	if requested <= 0 {
		requested = 10
	}
	if requested < 2 {
		requested = 2
	}
	return time.Duration(requested) * time.Second
}

func requiredNameNamespace(name, namespace string) error {
	if name == "" || namespace == "" {
		return fmt.Errorf("name and namespace are required")
	}
	return nil
}
