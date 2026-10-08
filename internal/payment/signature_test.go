package payment

import "testing"

func TestSignAndVerify(t *testing.T) {
	body := []byte(`{"event_id":"evt_1"}`)
	secret := "dev-webhook-secret"
	sig := Sign(body, secret)
	if !Verify(body, secret, sig) {
		t.Fatal("expected valid signature")
	}
	if Verify(body, secret, "bad") {
		t.Fatal("expected invalid signature")
	}
}
