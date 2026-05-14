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

func TestLoadFromEnv_DefaultListenAndReflectionVariants(t *testing.T) {
	t.Setenv(EnvGRPCListen, "")
	t.Setenv(EnvExpectedToken, "")
	t.Setenv(EnvGoogleApplicationCredentials, "")
	t.Setenv(EnvGrpcTLSCertFile, "")
	t.Setenv(EnvGrpcTLSKeyFile, "")
	t.Setenv(EnvGrpcMTLSClientCAFile, "")

	for _, raw := range []string{"", "0", "false", "FALSE", "no"} {
		t.Run("reflection_off_"+raw, func(t *testing.T) {
			t.Setenv(EnvEnableReflection, raw)
			c, err := LoadFromEnv()
			if err != nil {
				t.Fatal(err)
			}
			if c.EnableReflection {
				t.Fatalf("reflection should be off for %q", raw)
			}
			if c.GRPCListen != ":50053" {
				t.Fatalf("listen: %q", c.GRPCListen)
			}
		})
	}

	for _, raw := range []string{"1", "true", "TRUE", "yes", "on"} {
		t.Run("reflection_on_"+raw, func(t *testing.T) {
			t.Setenv(EnvEnableReflection, raw)
			c, err := LoadFromEnv()
			if err != nil {
				t.Fatal(err)
			}
			if !c.EnableReflection {
				t.Fatalf("reflection should be on for %q", raw)
			}
		})
	}
}

