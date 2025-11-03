package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	keyRotations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llmproxy_key_rotations_total",
		Help: "Total number of API key rotations",
	}, []string{"tenant", "provider", "success"})

	keyUsage = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llmproxy_key_usage_total",
		Help: "Total number of API key usages",
	}, []string{"tenant", "provider"})

	keyRevocations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llmproxy_key_revocations_total",
		Help: "Total number of API key revocations",
	}, []string{"tenant", "provider"})
)

type KeyManager struct {
	store     KeyStore
	encryptor *Encryptor
	rotator   *KeyRotator
	mu        sync.RWMutex
	auditLog  AuditLogger
}

type APIKey struct {
	ID          string
	Tenant      string
	Provider    string
	Key         string // encrypted
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	LastUsedAt  *time.Time
	UsageCount  int64
	Revoked     bool
	RotationDue *time.Time
}

type KeyStore interface {
	Save(key *APIKey) error
	Get(id string) (*APIKey, error)
	GetByTenantProvider(tenant, provider string) ([]*APIKey, error)
	Delete(id string) error
	List() ([]*APIKey, error)
}

type AuditLogger interface {
	LogKeyCreation(key *APIKey)
	LogKeyRotation(oldKey, newKey *APIKey)
	LogKeyRevocation(key *APIKey)
	LogKeyUsage(key *APIKey)
}

type Encryptor struct {
	key []byte
}

type KeyRotator struct {
	manager          *KeyManager
	rotationInterval time.Duration
	ticker           *time.Ticker
	stopCh           chan struct{}
}

type KeyManagerConfig struct {
	Store             KeyStore
	EncryptionKey     string
	RotationInterval  time.Duration
	AuditLog          AuditLogger
	AutoRotateEnabled bool
}

func NewKeyManager(config KeyManagerConfig) (*KeyManager, error) {
	if config.EncryptionKey == "" {
		return nil, fmt.Errorf("encryption key is required")
	}

	if config.RotationInterval == 0 {
		config.RotationInterval = 90 * 24 * time.Hour // 90 days default
	}

	encryptor, err := NewEncryptor(config.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	manager := &KeyManager{
		store:     config.Store,
		encryptor: encryptor,
		auditLog:  config.AuditLog,
	}

	if config.AutoRotateEnabled {
		manager.rotator = &KeyRotator{
			manager:          manager,
			rotationInterval: config.RotationInterval,
			stopCh:           make(chan struct{}),
		}
		manager.rotator.Start()
	}

	return manager, nil
}

func (km *KeyManager) CreateKey(tenant, provider, plainKey string, expiresAt *time.Time) (*APIKey, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	encryptedKey, err := km.encryptor.Encrypt(plainKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt key: %w", err)
	}

	keyID := generateKeyID(tenant, provider)

	rotationDue := time.Now().Add(km.rotator.rotationInterval)

	apiKey := &APIKey{
		ID:          keyID,
		Tenant:      tenant,
		Provider:    provider,
		Key:         encryptedKey,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		UsageCount:  0,
		Revoked:     false,
		RotationDue: &rotationDue,
	}

	if err := km.store.Save(apiKey); err != nil {
		return nil, fmt.Errorf("failed to save key: %w", err)
	}

	if km.auditLog != nil {
		km.auditLog.LogKeyCreation(apiKey)
	}

	logrus.WithFields(logrus.Fields{
		"tenant":   tenant,
		"provider": provider,
		"key_id":   keyID,
	}).Info("Created new API key")

	return apiKey, nil
}

func (km *KeyManager) GetKey(tenant, provider string) (string, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	keys, err := km.store.GetByTenantProvider(tenant, provider)
	if err != nil {
		return "", fmt.Errorf("failed to get keys: %w", err)
	}

	if len(keys) == 0 {
		return "", fmt.Errorf("no keys found for tenant %s, provider %s", tenant, provider)
	}

	var activeKey *APIKey
	for _, key := range keys {
		if key.Revoked {
			continue
		}
		if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
			continue
		}
		activeKey = key
		break
	}

	if activeKey == nil {
		return "", fmt.Errorf("no active keys found for tenant %s, provider %s", tenant, provider)
	}

	plainKey, err := km.encryptor.Decrypt(activeKey.Key)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt key: %w", err)
	}

	go km.updateKeyUsage(activeKey.ID)

	return plainKey, nil
}

func (km *KeyManager) RotateKey(keyID string, newPlainKey string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	oldKey, err := km.store.Get(keyID)
	if err != nil {
		keyRotations.WithLabelValues(oldKey.Tenant, oldKey.Provider, "false").Inc()
		return fmt.Errorf("failed to get key: %w", err)
	}

	newKey, err := km.CreateKey(oldKey.Tenant, oldKey.Provider, newPlainKey, oldKey.ExpiresAt)
	if err != nil {
		keyRotations.WithLabelValues(oldKey.Tenant, oldKey.Provider, "false").Inc()
		return fmt.Errorf("failed to create new key: %w", err)
	}

	oldKey.Revoked = true
	if err := km.store.Save(oldKey); err != nil {
		keyRotations.WithLabelValues(oldKey.Tenant, oldKey.Provider, "false").Inc()
		return fmt.Errorf("failed to revoke old key: %w", err)
	}

	if km.auditLog != nil {
		km.auditLog.LogKeyRotation(oldKey, newKey)
	}

	keyRotations.WithLabelValues(oldKey.Tenant, oldKey.Provider, "true").Inc()

	logrus.WithFields(logrus.Fields{
		"tenant":     oldKey.Tenant,
		"provider":   oldKey.Provider,
		"old_key_id": oldKey.ID,
		"new_key_id": newKey.ID,
	}).Info("Rotated API key")

	return nil
}

func (km *KeyManager) RevokeKey(keyID string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	key, err := km.store.Get(keyID)
	if err != nil {
		return fmt.Errorf("failed to get key: %w", err)
	}

	key.Revoked = true
	if err := km.store.Save(key); err != nil {
		return fmt.Errorf("failed to save revoked key: %w", err)
	}

	if km.auditLog != nil {
		km.auditLog.LogKeyRevocation(key)
	}

	keyRevocations.WithLabelValues(key.Tenant, key.Provider).Inc()

	logrus.WithFields(logrus.Fields{
		"tenant":   key.Tenant,
		"provider": key.Provider,
		"key_id":   keyID,
	}).Info("Revoked API key")

	return nil
}

func (km *KeyManager) updateKeyUsage(keyID string) {
	key, err := km.store.Get(keyID)
	if err != nil {
		logrus.WithError(err).Warn("Failed to get key for usage update")
		return
	}

	now := time.Now()
	key.LastUsedAt = &now
	key.UsageCount++

	if err := km.store.Save(key); err != nil {
		logrus.WithError(err).Warn("Failed to update key usage")
		return
	}

	keyUsage.WithLabelValues(key.Tenant, key.Provider).Inc()

	if km.auditLog != nil {
		km.auditLog.LogKeyUsage(key)
	}
}

func (km *KeyManager) Stop() {
	if km.rotator != nil {
		km.rotator.Stop()
	}
}

func NewEncryptor(keyString string) (*Encryptor, error) {
	hash := sha256.Sum256([]byte(keyString))
	return &Encryptor{key: hash[:]}, nil
}

func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := data[:nonceSize]
	ciphertextBytes := data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func (kr *KeyRotator) Start() {
	kr.ticker = time.NewTicker(24 * time.Hour) // Check daily

	go func() {
		for {
			select {
			case <-kr.ticker.C:
				kr.checkAndRotate()
			case <-kr.stopCh:
				return
			}
		}
	}()

	logrus.WithField("interval", kr.rotationInterval).Info("Key rotator started")
}

func (kr *KeyRotator) Stop() {
	if kr.ticker != nil {
		kr.ticker.Stop()
	}
	close(kr.stopCh)
	logrus.Info("Key rotator stopped")
}

func (kr *KeyRotator) checkAndRotate() {
	keys, err := kr.manager.store.List()
	if err != nil {
		logrus.WithError(err).Error("Failed to list keys for rotation check")
		return
	}

	now := time.Now()
	for _, key := range keys {
		if key.Revoked {
			continue
		}
		if key.RotationDue != nil && now.After(*key.RotationDue) {
			logrus.WithFields(logrus.Fields{
				"tenant":   key.Tenant,
				"provider": key.Provider,
				"key_id":   key.ID,
			}).Warn("API key rotation due - manual rotation required")
		}
	}
}

func generateKeyID(tenant, provider string) string {
	data := fmt.Sprintf("%s:%s:%d", tenant, provider, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return base64.URLEncoding.EncodeToString(hash[:16])
}

type InMemoryKeyStore struct {
	keys map[string]*APIKey
	mu   sync.RWMutex
}

func NewInMemoryKeyStore() *InMemoryKeyStore {
	return &InMemoryKeyStore{
		keys: make(map[string]*APIKey),
	}
}

func (s *InMemoryKeyStore) Save(key *APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyCopy := *key
	s.keys[key.ID] = &keyCopy
	return nil
}

func (s *InMemoryKeyStore) Get(id string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, exists := s.keys[id]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", id)
	}

	keyCopy := *key
	return &keyCopy, nil
}

func (s *InMemoryKeyStore) GetByTenantProvider(tenant, provider string) ([]*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []*APIKey
	for _, key := range s.keys {
		if key.Tenant == tenant && key.Provider == provider {
			keyCopy := *key
			keys = append(keys, &keyCopy)
		}
	}

	return keys, nil
}

func (s *InMemoryKeyStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.keys, id)
	return nil
}

func (s *InMemoryKeyStore) List() ([]*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]*APIKey, 0, len(s.keys))
	for _, key := range s.keys {
		keyCopy := *key
		keys = append(keys, &keyCopy)
	}

	return keys, nil
}

type NoOpAuditLogger struct{}

func (l *NoOpAuditLogger) LogKeyCreation(key *APIKey)            {}
func (l *NoOpAuditLogger) LogKeyRotation(oldKey, newKey *APIKey) {}
func (l *NoOpAuditLogger) LogKeyRevocation(key *APIKey)          {}
func (l *NoOpAuditLogger) LogKeyUsage(key *APIKey)               {}
