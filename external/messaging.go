package external

//go:generate mockgen -source=messaging.go -destination=../mocks/external/messaging.go
// type Sender interface {
// 	Send(message models.Message) error
// }

// type EmailSender struct {
// }

// func (e EmailSender) Send(message models.Message) error {
// 	// *** Send email logic ***
// 	return nil
// }
