package usecase

type Dependencies struct {
	WebhookSender   WebhookSender
	ReportGenerator ReportGenerator
	TokenCleaner    TokenCleaner
}

type Usecase struct {
	webhookSender   WebhookSender
	reportGenerator ReportGenerator
	tokenCleaner    TokenCleaner
}

func New(dependencies Dependencies) *Usecase {
	return &Usecase{
		webhookSender:   dependencies.WebhookSender,
		reportGenerator: dependencies.ReportGenerator,
		tokenCleaner:    dependencies.TokenCleaner,
	}
}
