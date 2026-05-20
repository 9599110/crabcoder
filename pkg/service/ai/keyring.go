package ai

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

type KeyStore interface {
	Get(service, key string) (string, error)
	Set(service, key, value string) error
	Delete(service, key string) error
}

type envKeyStore struct{}

func NewEnvKeyStore() KeyStore {
	return &envKeyStore{}
}

func (s *envKeyStore) Get(service, key string) (string, error) {
	envKey := keyToEnv(service, key)
	return os.Getenv(envKey), nil
}

func (s *envKeyStore) Set(service, key, value string) error {
	return fmt.Errorf("环境变量存储不支持写入，请手动设置: export %s=<value>", keyToEnv(service, key))
}

func (s *envKeyStore) Delete(service, key string) error {
	return fmt.Errorf("环境变量存储不支持删除")
}

type platformKeyStore struct {
	fallback KeyStore
}

func NewPlatformKeyStore() KeyStore {
	return &platformKeyStore{fallback: NewEnvKeyStore()}
}

func (s *platformKeyStore) Get(service, key string) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return s.keychainGet(service, key)
	case "linux":
		return s.secretServiceGet(service, key)
	default:
		return s.fallback.Get(service, key)
	}
}

func (s *platformKeyStore) Set(service, key, value string) error {
	switch runtime.GOOS {
	case "darwin":
		return s.keychainSet(service, key, value)
	case "linux":
		return s.secretServiceSet(service, key, value)
	default:
		return s.fallback.Set(service, key, value)
	}
}

func (s *platformKeyStore) Delete(service, key string) error {
	switch runtime.GOOS {
	case "darwin":
		return s.keychainDelete(service, key)
	case "linux":
		return s.secretServiceDelete(service, key)
	default:
		return s.fallback.Delete(service, key)
	}
}

// macOS Keychain
func (s *platformKeyStore) keychainGet(service, key string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", service, "-a", key, "-w")
	out, err := cmd.Output()
	if err != nil {
		return s.fallback.Get(service, key)
	}
	return string(out), nil
}

func (s *platformKeyStore) keychainSet(service, key, value string) error {
	cmd := exec.Command("security", "add-generic-password",
		"-s", service, "-a", key, "-w", value, "-U")
	return cmd.Run()
}

func (s *platformKeyStore) keychainDelete(service, key string) error {
	cmd := exec.Command("security", "delete-generic-password",
		"-s", service, "-a", key)
	return cmd.Run()
}

// Linux Secret Service (via secret-tool)
func (s *platformKeyStore) secretServiceGet(service, key string) (string, error) {
	cmd := exec.Command("secret-tool", "lookup", "service", service, "key", key)
	out, err := cmd.Output()
	if err != nil {
		return s.fallback.Get(service, key)
	}
	return string(out), nil
}

func (s *platformKeyStore) secretServiceSet(service, key, value string) error {
	cmd := exec.Command("secret-tool", "store",
		"--label", fmt.Sprintf("CrabCoder %s", service),
		"service", service, "key", key)
	cmd.Stdin = nil
	return cmd.Run()
}

func (s *platformKeyStore) secretServiceDelete(service, key string) error {
	cmd := exec.Command("secret-tool", "clear", "service", service, "key", key)
	return cmd.Run()
}

func keyToEnv(service, key string) string {
	return fmt.Sprintf("%s_%s_API_KEY", service, key)
}

func ResolveAPIKey(provider string) string {
	store := NewPlatformKeyStore()
	key, err := store.Get("crabcoder", provider)
	if err == nil && key != "" {
		return key
	}
	return resolveAPIKey(provider)
}
