package config

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestViperConfigGetString(t *testing.T) {
	v := viper.New()
	v.Set("name", "test")
	cfg := New(v)

	if got := cfg.GetString("name"); got != "test" {
		t.Errorf("GetString('name') = %q, want %q", got, "test")
	}
}

func TestViperConfigGetInt(t *testing.T) {
	v := viper.New()
	v.Set("port", 8080)
	cfg := New(v)

	if got := cfg.GetInt("port"); got != 8080 {
		t.Errorf("GetInt('port') = %d, want %d", got, 8080)
	}
}

func TestViperConfigGetBool(t *testing.T) {
	v := viper.New()
	v.Set("enabled", true)
	cfg := New(v)

	if got := cfg.GetBool("enabled"); !got {
		t.Error("GetBool('enabled') = false, want true")
	}
}

func TestViperConfigGetDuration(t *testing.T) {
	v := viper.New()
	v.Set("timeout", "5s")
	cfg := New(v)

	want := 5 * time.Second
	if got := cfg.GetDuration("timeout"); got != want {
		t.Errorf("GetDuration('timeout') = %v, want %v", got, want)
	}
}

func TestViperConfigIsSet(t *testing.T) {
	v := viper.New()
	v.Set("exists", true)
	cfg := New(v)

	if !cfg.IsSet("exists") {
		t.Error("IsSet('exists') = false, want true")
	}
	if cfg.IsSet("missing") {
		t.Error("IsSet('missing') = true, want false")
	}
}

func TestViperConfigSub(t *testing.T) {
	v := viper.New()
	v.Set("plugins.recon.enabled", true)
	v.Set("plugins.recon.interval", 30)
	cfg := New(v)

	sub := cfg.Sub("plugins.recon")
	if sub == nil {
		t.Fatal("Sub('plugins.recon') = nil")
	}
	if got := sub.GetBool("enabled"); !got {
		t.Error("sub.GetBool('enabled') = false, want true")
	}
	if got := sub.GetInt("interval"); got != 30 {
		t.Errorf("sub.GetInt('interval') = %d, want %d", got, 30)
	}
}

func TestViperConfigSubMissing(t *testing.T) {
	v := viper.New()
	cfg := New(v)

	sub := cfg.Sub("nonexistent")
	if sub == nil {
		t.Fatal("Sub('nonexistent') should return empty Config, not nil")
	}
	// Should return zero values without panic.
	if got := cfg.GetString("anything"); got != "" {
		t.Errorf("empty config GetString() = %q, want empty", got)
	}
	_ = sub
}

func TestViperConfigUnmarshal(t *testing.T) {
	v := viper.New()
	v.Set("host", "localhost")
	v.Set("port", 9090)
	cfg := New(v)

	var target struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	}
	if err := cfg.Unmarshal(&target); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if target.Host != "localhost" {
		t.Errorf("Host = %q, want %q", target.Host, "localhost")
	}
	if target.Port != 9090 {
		t.Errorf("Port = %d, want %d", target.Port, 9090)
	}
}

func TestNilViper(t *testing.T) {
	cfg := New(nil)
	// Should not panic and return zero values.
	if got := cfg.GetString("key"); got != "" {
		t.Errorf("nil viper GetString() = %q, want empty", got)
	}
}
