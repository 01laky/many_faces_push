package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"

	"firebase.google.com/go/v4/messaging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pushv1 "github.com/01laky/many_faces_push/gen/manyfaces/push/v1"
	"github.com/01laky/many_faces_push/internal/msgutil"
)

// PushService implements manyfaces.push.v1.PushService — FCM dispatch only; no domain authorization.
type PushService struct {
	pushv1.UnimplementedPushServiceServer

	// fcm may be nil when GOOGLE_APPLICATION_CREDENTIALS is unset; SendPush then fails fast with FailedPrecondition.
	fcm *messaging.Client
	log *slog.Logger
}

// NewPushService constructs a gRPC service. Pass fcm=nil to run in "metadata + transport only" mode for stacks without credentials.
func NewPushService(fcm *messaging.Client, log *slog.Logger) *PushService {
	return &PushService{fcm: fcm, log: log}
}

// SendPush forwards localization keys and data to FCM using Firebase Admin (HTTP v1 under the hood).
func (s *PushService) SendPush(ctx context.Context, req *pushv1.SendPushRequest) (*pushv1.SendPushResponse, error) {
	if s.fcm == nil {
		return nil, status.Error(codes.FailedPrecondition, "FCM is not configured: set GOOGLE_APPLICATION_CREDENTIALS to a service account JSON path")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	if len(req.RegistrationTokens) == 0 {
		return nil, status.Error(codes.InvalidArgument, "registration_tokens is empty")
	}

	const chunk = 500
	resp := &pushv1.SendPushResponse{}
	for i := 0; i < len(req.RegistrationTokens); i += chunk {
		end := i + chunk
		if end > len(req.RegistrationTokens) {
			end = len(req.RegistrationTokens)
		}
		sub := cloneRequestForTokens(req, req.RegistrationTokens[i:end])
		msg, err := msgutil.BuildMulticastMessage(sub)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid multicast payload: %v", err)
		}

		batch, err := s.fcm.SendEachForMulticast(ctx, msg)
		if err != nil {
			// Total failure path (rare): still return gRPC error so backend can retry.
			s.log.Error("send each for multicast failed", "error", err, "token_count", len(msg.Tokens))
			return nil, status.Errorf(codes.Unavailable, "fcm send failed: %v", err)
		}

		for j, send := range batch.Responses {
			tok := msg.Tokens[j]
			prefix := tokenSHA256Prefix(tok)
			if send.Success {
				resp.Sent++
				resp.Results = append(resp.Results, &pushv1.PerTokenResult{
					TokenSha256Prefix: prefix,
					PermanentInvalid:  false,
					OutcomeCode:       "OK",
					Detail:            "",
				})
				continue
			}

			permanent := messaging.IsUnregistered(send.Error)
			code := classifyFCMError(send.Error)
			resp.Failed++
			resp.Results = append(resp.Results, &pushv1.PerTokenResult{
				TokenSha256Prefix: prefix,
				PermanentInvalid:  permanent,
				OutcomeCode:       code,
				Detail:            redactDetail(send.Error),
			})
			s.log.Warn("fcm token send failed",
				"token_prefix", prefix,
				"permanent_invalid", permanent,
				"code", code,
				"detail", redactDetail(send.Error))
		}
	}

	return resp, nil
}

func cloneRequestForTokens(src *pushv1.SendPushRequest, tokens []string) *pushv1.SendPushRequest {
	return &pushv1.SendPushRequest{
		RegistrationTokens: append([]string(nil), tokens...),
		TitleLocKey:        src.TitleLocKey,
		BodyLocKey:         src.BodyLocKey,
		TitleLocArgs:       append([]string(nil), src.TitleLocArgs...),
		BodyLocArgs:        append([]string(nil), src.BodyLocArgs...),
		Data:               src.Data,
		AndroidChannelId:   src.AndroidChannelId,
		CollapseKey:        src.CollapseKey,
		TtlSeconds:         src.TtlSeconds,
	}
}

func tokenSHA256Prefix(token string) string {
	sum := sha256.Sum256([]byte(token))
	h := hex.EncodeToString(sum[:])
	if len(h) < 8 {
		return h
	}
	return h[:8]
}

func classifyFCMError(err error) string {
	if err == nil {
		return "OK"
	}
	if messaging.IsUnregistered(err) {
		return "UNREGISTERED"
	}
	if messaging.IsSenderIDMismatch(err) {
		return "SENDER_ID_MISMATCH"
	}
	if messaging.IsInvalidArgument(err) {
		return "INVALID_ARGUMENT"
	}
	if messaging.IsQuotaExceeded(err) {
		return "QUOTA_EXCEEDED"
	}
	if messaging.IsUnavailable(err) {
		return "UNAVAILABLE"
	}
	if messaging.IsInternal(err) {
		return "INTERNAL"
	}
	return "UNKNOWN"
}

func redactDetail(err error) string {
	if err == nil {
		return ""
	}
	// Never echo raw tokens; FCM errors are already token-scoped strings from the SDK.
	return err.Error()
}
