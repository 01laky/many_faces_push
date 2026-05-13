package msgutil

import (
	"testing"
	"time"

	pushv1 "github.com/01laky/many_faces_push/gen/manyfaces/push/v1"
)

func TestBuildMulticastMessage_LocalizationAndData(t *testing.T) {
	req := &pushv1.SendPushRequest{
		RegistrationTokens: []string{"tok-a"},
		TitleLocKey:        "push_title_smoke",
		BodyLocKey:         "push_body_smoke",
		TitleLocArgs:       []string{"1"},
		BodyLocArgs:        []string{"world"},
		Data:               map[string]string{"route": "home"},
		AndroidChannelId:   "default",
		CollapseKey:        "c1",
		TtlSeconds:         120,
	}
	msg, err := BuildMulticastMessage(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.Tokens) != 1 || msg.Tokens[0] != "tok-a" {
		t.Fatalf("tokens: %+v", msg.Tokens)
	}
	if msg.Data["route"] != "home" {
		t.Fatalf("data: %+v", msg.Data)
	}
	if msg.Android.Notification.TitleLocKey != "push_title_smoke" || msg.Android.Notification.BodyLocKey != "push_body_smoke" {
		t.Fatalf("android loc: %+v", msg.Android.Notification)
	}
	if msg.APNS.Payload.Aps.Alert.TitleLocKey != "push_title_smoke" || msg.APNS.Payload.Aps.Alert.LocKey != "push_body_smoke" {
		t.Fatalf("apns alert: %+v", msg.APNS.Payload.Aps.Alert)
	}
	if msg.Android.CollapseKey != "c1" {
		t.Fatalf("collapse: %q", msg.Android.CollapseKey)
	}
	if msg.Android.TTL == nil || *msg.Android.TTL != 120*time.Second {
		t.Fatalf("ttl: %+v", msg.Android.TTL)
	}
}

func TestBuildMulticastMessage_TokenLimit(t *testing.T) {
	tokens := make([]string, maxTokensPerMulticast+1)
	for i := range tokens {
		tokens[i] = "x"
	}
	_, err := BuildMulticastMessage(&pushv1.SendPushRequest{RegistrationTokens: tokens})
	if err == nil {
		t.Fatal("expected error for too many tokens")
	}
}
