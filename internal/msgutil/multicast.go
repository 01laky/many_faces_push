// Package msgutil maps our protobuf contract onto Firebase Admin multicast messages (no network I/O).
package msgutil

import (
	"fmt"
	"time"

	"firebase.google.com/go/v4/messaging"
	pushv1 "github.com/01laky/many_faces_push/gen/manyfaces/push/v1"
)

const maxTokensPerMulticast = 500 // FCM hard limit enforced by the Admin SDK.

// BuildMulticastMessage converts SendPushRequest fields into a Firebase MulticastMessage (shared Android + iOS localization).
func BuildMulticastMessage(req *pushv1.SendPushRequest) (*messaging.MulticastMessage, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	if len(req.RegistrationTokens) == 0 {
		return nil, fmt.Errorf("no registration tokens")
	}
	if len(req.RegistrationTokens) > maxTokensPerMulticast {
		return nil, fmt.Errorf("too many tokens: %d (max %d)", len(req.RegistrationTokens), maxTokensPerMulticast)
	}

	data := map[string]string{}
	for k, v := range req.Data {
		data[k] = v
	}

	android := &messaging.AndroidConfig{
		CollapseKey: req.CollapseKey,
		Notification: &messaging.AndroidNotification{
			TitleLocKey:  req.TitleLocKey,
			BodyLocKey:   req.BodyLocKey,
			TitleLocArgs: append([]string(nil), req.TitleLocArgs...),
			BodyLocArgs:  append([]string(nil), req.BodyLocArgs...),
			ChannelID:    req.AndroidChannelId,
		},
	}
	if req.TtlSeconds > 0 {
		d := time.Duration(req.TtlSeconds) * time.Second
		android.TTL = &d
	}

	apns := &messaging.APNSConfig{
		Payload: &messaging.APNSPayload{
			Aps: &messaging.Aps{
				Alert: &messaging.ApsAlert{
					TitleLocKey:  req.TitleLocKey,
					TitleLocArgs: append([]string(nil), req.TitleLocArgs...),
					LocKey:       req.BodyLocKey,
					LocArgs:      append([]string(nil), req.BodyLocArgs...),
				},
			},
		},
	}

	return &messaging.MulticastMessage{
		Tokens:  append([]string(nil), req.RegistrationTokens...),
		Data:    data,
		Android: android,
		APNS:    apns,
	}, nil
}
