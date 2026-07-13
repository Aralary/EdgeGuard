package rabbitmq

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestDeliveryAttempt(t *testing.T) {
	testCases := []struct {
		name    string
		headers amqp.Table
		want    int
	}{
		{name: "first delivery", headers: nil, want: 1},
		{name: "second delivery", headers: amqp.Table{"x-delivery-count": int64(1)}, want: 2},
		{name: "int32 header", headers: amqp.Table{"x-delivery-count": int32(3)}, want: 4},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := deliveryAttempt(testCase.headers); got != testCase.want {
				t.Fatalf("deliveryAttempt() = %d, want %d", got, testCase.want)
			}
		})
	}
}
