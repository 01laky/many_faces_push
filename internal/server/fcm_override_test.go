package server

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"firebase.google.com/go/v4/messaging"
	pushv1 "github.com/01laky/many_faces_push/gen/manyfaces/push/v1"
	"github.com/01laky/many_faces_push/internal/fcmclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubResolver struct {
	wireJSON string
	client   fcmMulticastSender
	err      error
}

func (s *stubResolver) Resolve(_ context.Context, wireJSON string) (fcmMulticastSender, error) {
	s.wireJSON = wireJSON
	if s.err != nil {
		return nil, s.err
	}
	return s.client, nil
}

const validServiceAccountJSON = `{
  "type": "service_account",
  "project_id": "demo-project",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIB\n-----END PRIVATE KEY-----\n",
  "client_email": "firebase-adminsdk@test.iam.gserviceaccount.com"
}`

// APC-G1: wire FCM block takes precedence over env resolver path.
func TestPushService_SendPush_wireFCMUsed(t *testing.T) {
	stub := &stubFCM{
		batch: &messaging.BatchResponse{
			Responses: []*messaging.SendResponse{{Success: true, MessageID: "m1"}},
		},
	}
	resolver := &stubResolver{client: stub}
	svc := NewPushServiceWithResolver(resolver, slog.Default())
	_, err := svc.SendPush(context.Background(), &pushv1.SendPushRequest{
		RegistrationTokens: []string{"tok"},
		TitleLocKey:        "k",
		BodyLocKey:         "b",
		Fcm:                &pushv1.FcmCredentialsConfig{ServiceAccountJson: validServiceAccountJSON},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if resolver.wireJSON != validServiceAccountJSON {
		t.Fatalf("expected wire json, got %q", resolver.wireJSON)
	}
	if stub.calls != 1 {
		t.Fatalf("expected 1 call, got %d", stub.calls)
	}
}

// APC-G2: empty wire block falls back to env client via resolver.
func TestPushService_SendPush_envFallback(t *testing.T) {
	stub := &stubFCM{
		batch: &messaging.BatchResponse{
			Responses: []*messaging.SendResponse{{Success: true}},
		},
	}
	resolver := &stubResolver{client: stub}
	svc := NewPushServiceWithResolver(resolver, slog.Default())
	_, err := svc.SendPush(context.Background(), &pushv1.SendPushRequest{
		RegistrationTokens: []string{"tok"},
		TitleLocKey:        "k",
		BodyLocKey:         "b",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if resolver.wireJSON != "" {
		t.Fatalf("expected empty wire json, got %q", resolver.wireJSON)
	}
}

// APC-G3: incomplete wire JSON returns InvalidArgument from resolver error mapping.
func TestPushService_SendPush_incompleteWireJSON(t *testing.T) {
	resolver := &stubResolver{err: errors.New("private_key is invalid or truncated")}
	svc := NewPushServiceWithResolver(resolver, slog.Default())
	_, err := svc.SendPush(context.Background(), &pushv1.SendPushRequest{
		RegistrationTokens: []string{"tok"},
		TitleLocKey:        "k",
		BodyLocKey:         "b",
		Fcm:                &pushv1.FcmCredentialsConfig{ServiceAccountJson: `{"type":"service_account"}`},
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("code: %v", err)
	}
}

// APC-G4: TestFcmCredentials validates JSON shape.
func TestPushService_TestFcmCredentials_validJSON(t *testing.T) {
	svc := NewPushServiceWithResolver(&stubResolver{}, slog.Default())
	resp, err := svc.TestFcmCredentials(context.Background(), &pushv1.TestFcmCredentialsRequest{
		Fcm: &pushv1.FcmCredentialsConfig{ServiceAccountJson: validServiceAccountJSON},
	})
	if err != nil {
		t.Fatalf("rpc: %v", err)
	}
	if !resp.Valid || resp.ProjectId != "demo-project" {
		t.Fatalf("resp: %+v", resp)
	}
}

func TestFcmclientValidateServiceAccountJSON(t *testing.T) {
	projectID, err := fcmclient.ValidateServiceAccountJSON(validServiceAccountJSON)
	if err != nil || projectID != "demo-project" {
		t.Fatalf("valid json: project=%q err=%v", projectID, err)
	}
	_, err = fcmclient.ValidateServiceAccountJSON(`not-json`)
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
}
