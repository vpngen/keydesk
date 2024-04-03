package keydesk

//func GetSubscription(s push.Service) middleware.Responder {
//	n, err := s.GetSubscription()
//	if err != nil {
//		if errors.Is(err, storage.SubscriptionNotFound) {
//			return operations.NewGetSubscriptionNotFound()
//		}
//		return operations.NewGetSubscriptionInternalServerError()
//	}
//
//	return operations.NewGetSubscriptionOK().WithPayload(&models.Subscription{
//		Endpoint: &n.Endpoint,
//		Keys: &models.SubscriptionKeys{
//			Auth:   n.Keys.Auth,
//			P256dh: n.Keys.P256dh,
//		},
//	})
//}
//
//func PostSubscription(s push.Service, sub webpush.Subscription) middleware.Responder {
//	if err := s.SaveSubscription(sub); err != nil {
//		return operations.NewPostUserInternalServerError()
//	}
//	return operations.NewPostSubscriptionOK()
//}
