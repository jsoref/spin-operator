package runtimeconfig

import (
	"fmt"

	spinv1 "github.com/spinkube/spin-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type VaultVariablesProviderOptions struct {
	URL    string `toml:"url,omitempty"`
	Token  string `toml:"token,omitempty"`
	Mount  string `toml:"mount,omitempty"`
	Prefix string `toml:"prefix,omitempty"`
}

type EnvVariablesProviderOptions struct {
	Prefix     string `toml:"prefix,omitempty"`
	DotEnvPath string `toml:"dotenv_path,omitempty"`
}

type VariablesProvider struct {
	Type string `toml:"type,omitempty"`
	VaultVariablesProviderOptions
	EnvVariablesProviderOptions
}

type KeyValueStoreOptions map[string]string

type SQLiteDatabaseOptions map[string]string

type Spin struct {
	Variables []VariablesProvider `toml:"config_provider,omitempty"`

	KeyValueStores map[string]KeyValueStoreOptions `toml:"key_value_store,omitempty"`

	SQLiteDatabases map[string]SQLiteDatabaseOptions `toml:"sqlite_database,omitempty"`

	LLMCompute map[string]string `toml:"llm_compute,omitempty"`
}

func (s *Spin) AddKeyValueStore(
	name, storeType, namespace string,
	secrets map[types.NamespacedName]*corev1.Secret,
	configMaps map[types.NamespacedName]*corev1.ConfigMap,
	opts []spinv1.RuntimeConfigOption) error {
	if s.KeyValueStores == nil {
		s.KeyValueStores = make(map[string]KeyValueStoreOptions)
	}

	if _, ok := s.KeyValueStores[name]; ok {
		return fmt.Errorf("duplicate definition for key value store with name: %s", name)
	}

	options, err := renderOptionsIntoMap(storeType, namespace, opts, secrets, configMaps)
	if err != nil {
		return err
	}

	s.KeyValueStores[name] = options
	return nil
}

func (s *Spin) AddSQLiteDatabase(
	name, storeType, namespace string,
	secrets map[types.NamespacedName]*corev1.Secret,
	configMaps map[types.NamespacedName]*corev1.ConfigMap,
	opts []spinv1.RuntimeConfigOption) error {
	if s.SQLiteDatabases == nil {
		s.SQLiteDatabases = make(map[string]SQLiteDatabaseOptions)
	}
	if _, ok := s.SQLiteDatabases[name]; ok {
		return fmt.Errorf("duplicate definition for sqlite database with name: %s", name)
	}

	options, err := renderOptionsIntoMap(storeType, namespace, opts, secrets, configMaps)
	if err != nil {
		return err
	}

	s.SQLiteDatabases[name] = SQLiteDatabaseOptions(options)
	return nil
}

func (s *Spin) AddLLMCompute(computeType, namespace string,
	secrets map[types.NamespacedName]*corev1.Secret,
	configMaps map[types.NamespacedName]*corev1.ConfigMap,
	opts []spinv1.RuntimeConfigOption) error {
	computeOpts, err := renderOptionsIntoMap(computeType, namespace, opts, secrets, configMaps)
	if err != nil {
		return err
	}

	s.LLMCompute = computeOpts
	return nil
}

func renderOptionsIntoMap(typeOpt, namespace string,
	opts []spinv1.RuntimeConfigOption,
	secrets map[types.NamespacedName]*corev1.Secret, configMaps map[types.NamespacedName]*corev1.ConfigMap) (map[string]string, error) {
	options := map[string]string{
		"type": typeOpt,
	}

	for _, opt := range opts {
		var value string
		if opt.Value != "" {
			value = opt.Value
		} else if cmKeyRef := opt.ValueFrom.ConfigMapKeyRef; cmKeyRef != nil {
			cm, ok := configMaps[types.NamespacedName{Name: cmKeyRef.Name, Namespace: namespace}]
			if !ok {
				// This error shouldn't happen - we validate dependencies ahead of time, add this as a fallback error
				return nil, fmt.Errorf("unmet dependency while building config: configmap (%s/%s) not found", namespace, cmKeyRef.Name)
			}

			value = cm.Data[cmKeyRef.Key]
		} else if secKeyRef := opt.ValueFrom.SecretKeyRef; secKeyRef != nil {
			sec, ok := secrets[types.NamespacedName{Name: secKeyRef.Name, Namespace: namespace}]
			if !ok {
				// This error shouldn't happen - we validate dependencies ahead of time, add this as a fallback error
				return nil, fmt.Errorf("unmet dependency while building config: secret (%s/%s) not found", namespace, secKeyRef.Name)
			}

			value = string(sec.Data[secKeyRef.Key])
		}

		options[opt.Name] = value
	}

	return options, nil
}
