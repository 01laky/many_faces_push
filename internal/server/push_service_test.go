package server

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"firebase.google.com/go/v4/messaging"
	pushv1 "github.com/01laky/many_faces_push/gen/manyfaces/push/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubFCM struct {
	batch *messaging.BatchResponse
	err   error
	calls int
	last  *messaging.MulticastMessage
}

func (s *stubFCM) SendEachForMulticast(_ context.Context, msg *messaging.MulticastMessage) (*messaging.BatchResponse, error) {
	s.calls++
	s.last = msg
	if s.err != nil {
		return nil, s.err
	}
	return s.batch, nil
}

func TestPushService_SendPush_nilFCM(t *testing.T) {
	svc := NewPushService(nil, slog.Default())
	_, err := svc.SendPush(context.Background(), &pushv1.SendPushRequest{RegistrationTokens: []string{"t"}, TitleLocKey: "k", BodyLocKey: "b"})
	if err == nil {
		t.Fatal("expected error")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("code: %v", err)
	}
}

func TestPushService_SendPush_nilRequest(t *testing.T) {
	svc := NewPushServiceWithResolver(&stubResolver{client: &stubFCM{}}, slog.Default())
	_, err := svc.SendPush(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code: %v", err)
	}
}

func TestPushService_SendPush_emptyTokens(t *testing.T) {
	svc := NewPushServiceWithResolver(&stubResolver{client: &stubFCM{}}, slog.Default())
	_, err := svc.SendPush(context.Background(), &pushv1.SendPushRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code: %v", err)
	}
}

func TestPushService_SendPush_successAndFailurePerToken(t *testing.T) {
	stub := &stubFCM{
		batch: &messaging.BatchResponse{
			Responses: []*messaging.SendResponse{
				{Success: true, MessageID: "m1"},
				{Success: false, Error: errors.New("token expired")},
			},
		},
	}
	svc := NewPushServiceWithResolver(&stubResolver{client: stub}, slog.Default())
	resp, err := svc.SendPush(context.Background(), &pushv1.SendPushRequest{
		RegistrationTokens: []string{"tok-a", "tok-b"},
		TitleLocKey:        "title",
		BodyLocKey:         "body",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if resp.Sent != 1 || resp.Failed != 1 || len(resp.Results) != 2 {
		t.Fatalf("counts: sent=%d failed=%d results=%d", resp.Sent, resp.Failed, len(resp.Results))
	}
	if resp.Results[0].OutcomeCode != "OK" || resp.Results[1].OutcomeCode != "UNKNOWN" {
		t.Fatalf("outcomes: %+v", resp.Results)
	}
	if resp.Results[0].TokenSha256Prefix == "" {
		t.Fatal("expected token hash prefix")
	}
}

func TestPushService_SendPush_chunksOver500Tokens(t *testing.T) {
	tokens := make([]string, 501)
	for i := range tokens {
		tokens[i] = "t"
	}
	stub := &stubFCM{
		batch: &messaging.BatchResponse{
			Responses: []*messaging.SendResponse{{Success: true}},
		},
	}
	svc := NewPushServiceWithResolver(&stubResolver{client: stub}, slog.Default())
	_, err := svc.SendPush(context.Background(), &pushv1.SendPushRequest{
		RegistrationTokens: tokens,
		TitleLocKey:        "k",
		BodyLocKey:         "b",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if stub.calls != 2 {
		t.Fatalf("expected 2 FCM batches, got %d", stub.calls)
	}
}

func TestPushService_SendPush_fcmTransportFailure(t *testing.T) {
	stub := &stubFCM{err: errors.New("network down")}
	svc := NewPushServiceWithResolver(&stubResolver{client: stub}, slog.Default())
	_, err := svc.SendPush(context.Background(), &pushv1.SendPushRequest{
		RegistrationTokens: []string{"tok"},
		TitleLocKey:        "k",
		BodyLocKey:         "b",
	})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("code: %v", err)
	}
}

func TestTokenSHA256Prefix(t *testing.T) {
	p := tokenSHA256Prefix("my-device-token")
	if len(p) != 8 {
		t.Fatalf("prefix len: %d (%q)", len(p), p)
	}
	if p != tokenSHA256Prefix("my-device-token") {
		t.Fatal("expected stable prefix")
	}
}

func TestClassifyFCMError_nil(t *testing.T) {
	if classifyFCMError(nil) != "OK" {
		t.Fatal("expected OK")
	}
	if classifyFCMError(errors.New("generic")) != "UNKNOWN" {
		t.Fatal("expected UNKNOWN")
	}
}

func TestRedactDetail(t *testing.T) {
	if redactDetail(nil) != "" {
		t.Fatal("expected empty")
	}
	msg := "quota exceeded"
	if redactDetail(errors.New(msg)) != msg {
		t.Fatalf("detail: %q", redactDetail(errors.New(msg)))
	}
}

func TestCloneRequestForTokens_copiesFields(t *testing.T) {
	src := &pushv1.SendPushRequest{
		RegistrationTokens: []string{"a", "b"},
		TitleLocKey:        "t",
		BodyLocKey:         "b",
		TitleLocArgs:       []string{"1"},
		Data:               map[string]string{"k": "v"},
		AndroidChannelId:   "ch",
		CollapseKey:        "c",
		TtlSeconds:         60,
	}
	cloned := cloneRequestForTokens(src, []string{"x"})
	if len(cloned.RegistrationTokens) != 1 || cloned.RegistrationTokens[0] != "x" {
		t.Fatalf("tokens: %+v", cloned.RegistrationTokens)
	}
	if cloned.TitleLocKey != "t" || cloned.Data["k"] != "v" || cloned.TtlSeconds != 60 {
		t.Fatalf("clone: %+v", cloned)
	}
	if len(src.RegistrationTokens) != 2 {
		t.Fatal("source slice must not be mutated")
	}
}
