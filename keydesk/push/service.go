package push

import (
	"encoding/json"
	"fmt"
	"github.com/SherClockHolmes/webpush-go"
	"github.com/vpngen/keydesk/keydesk/storage"
	"net/http"
)

type (
	Service struct {
		db *storage.BrigadeStorage
		vapidPriv,
		vapidPub string
	}

	NotificationOptions struct {
		Title   string  `json:"title"`
		Options Options `json:"options,omitempty"`
	}

	Action struct {
		Action string `json:"action"`
		Icon   string `json:"icon,omitempty"`
		Title  string `json:"title"`
	}

	Options struct {
		Body               string   `json:"body"`
		Actions            []Action `json:"actions,omitempty"`
		Badge              string   `json:"badge,omitempty"`
		Data               any      `json:"data,omitempty"`
		Dir                string   `json:"dir,omitempty"`
		Icon               string   `json:"icon,omitempty"`
		Image              string   `json:"image,omitempty"`
		Lang               string   `json:"lang,omitempty"`
		Renotify           bool     `json:"renotify,omitempty"`
		RequireInteraction bool     `json:"requireInteraction,omitempty"`
		Silent             bool     `json:"silent,omitempty"`
		Tag                string   `json:"tag,omitempty"`
		Timestamp          int64    `json:"timestamp,omitempty"`
		Vibrate            []int64  `json:"vibrate,omitempty"`
	}
)

func New(db *storage.BrigadeStorage, priv, pub string) Service {
	return Service{
		db:        db,
		vapidPriv: priv,
		vapidPub:  pub,
	}
}

func (s Service) SaveSubscription(sub webpush.Subscription) error {
	return s.db.SaveSubscription(sub)
}

func (s Service) GetSubscription() (webpush.Subscription, error) {
	return s.db.GetSubscription()
}

func (s Service) Push(notification NotificationOptions, sub webpush.Subscription, options webpush.Options) error {
	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	resp, err := webpush.SendNotification(data, &sub, &options)
	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("send notification code %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	return nil
}

//func (s Service) SendPushHandler(params operations.SendPushParams) middleware.Responder {
//	n := params.Body.Notification
//	nOpts := n.Options
//	opts := params.Body.Options
//
//	var (
//		vapidPub, vapidPriv string
//	)
//
//	if opts != nil && opts.PublicKey != "" && opts.PrivateKey != "" {
//		vapidPriv = opts.PrivateKey
//		vapidPub = opts.PublicKey
//	} else {
//		vapidPriv = s.vapidPriv
//		vapidPub = s.vapidPub
//	}
//
//	var actions []Action
//	for _, action := range nOpts.Actions {
//		actions = append(actions, Action{
//			Action: action.Action,
//			Icon:   action.Icon,
//			Title:  action.Title,
//		})
//	}
//
//	if err := s.Push(
//		NotificationOptions{
//			Title: swag.StringValue(n.Title),
//			Options: Options{
//				Body:               swag.StringValue(nOpts.Body),
//				Actions:            actions,
//				Badge:              nOpts.Badge,
//				Data:               nOpts.Data,
//				Dir:                nOpts.Dir,
//				Icon:               nOpts.Icon,
//				Image:              nOpts.Image,
//				Lang:               nOpts.Lang,
//				Renotify:           nOpts.Renotify,
//				RequireInteraction: nOpts.RequireInteraction,
//				Silent:             nOpts.Silent,
//				Tag:                nOpts.Tag,
//				Timestamp:          nOpts.Timestamp,
//				Vibrate:            nOpts.Vibrate,
//			},
//		},
//		webpush.Subscription{
//			Endpoint: swag.StringValue(params.Body.Subscription.Endpoint),
//			Keys: webpush.Keys{
//				P256dh: params.Body.Subscription.Keys.P256dh,
//				Auth:   params.Body.Subscription.Keys.Auth,
//			},
//		},
//		webpush.Options{
//			Subscriber:      opts.Subscriber,
//			Topic:           opts.Topic,
//			Urgency:         webpush.Urgency(opts.Urgency),
//			VAPIDPrivateKey: vapidPriv,
//			VAPIDPublicKey:  vapidPub,
//		}); err != nil {
//		log.Printf("send push: %s\n", err)
//		return operations.NewSendPushInternalServerError()
//	}
//
//	return operations.NewSendPushOK()
//}
