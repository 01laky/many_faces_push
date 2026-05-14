package config

import "testing"

func TestConfig_Validate_tlsRequiresBothCertAndKey(t *testing.T) {
	t.Parallel()
	c := &Config{GrpcTLSCertFile: "/a.crt", GrpcTLSKeyFile: ""}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error")
	}
	c = &Config{GrpcTLSCertFile: "", GrpcTLSKeyFile: "/a.key"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestConfig_Validate_okWhenTlsUnsetOrBothSet(t *testing.T) {
	t.Parallel()
	c := &Config{}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c = &Config{GrpcTLSCertFile: "/a.crt", GrpcTLSKeyFile: "/a.key"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}
