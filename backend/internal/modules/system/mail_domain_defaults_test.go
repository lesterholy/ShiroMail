package system

import "testing"

func TestDeriveMailTargetsFromAppBaseURL(t *testing.T) {
	tests := []struct {
		name       string
		appBaseURL string
		wantHost   string
		wantDKIM   string
		wantOK     bool
	}{
		{
			name:       "subdomain collapses to root domain",
			appBaseURL: "https://shiromail.galiais.com",
			wantHost:   "smtp.galiais.com",
			wantDKIM:   "shiro._domainkey.galiais.com",
			wantOK:     true,
		},
		{
			name:       "localhost ignored",
			appBaseURL: "http://localhost:5173",
			wantOK:     false,
		},
		{
			name:       "co uk keeps registrable domain",
			appBaseURL: "https://panel.mail.example.co.uk",
			wantHost:   "smtp.example.co.uk",
			wantDKIM:   "shiro._domainkey.example.co.uk",
			wantOK:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHost, gotDKIM, gotOK := deriveMailTargetsFromAppBaseURL(tt.appBaseURL)
			if gotOK != tt.wantOK {
				t.Fatalf("deriveMailTargetsFromAppBaseURL() ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotHost != tt.wantHost {
				t.Fatalf("deriveMailTargetsFromAppBaseURL() host = %q, want %q", gotHost, tt.wantHost)
			}
			if gotDKIM != tt.wantDKIM {
				t.Fatalf("deriveMailTargetsFromAppBaseURL() dkim = %q, want %q", gotDKIM, tt.wantDKIM)
			}
		})
	}
}
