package bundle

import (
	"errors"
	nethttp "net/http"
	"sync"

	abs "github.com/microsoft/kiota-abstractions-go"
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	absser "github.com/microsoft/kiota-abstractions-go/serialization"
	"github.com/microsoft/kiota-abstractions-go/store"
	khttp "github.com/microsoft/kiota-http-go"
	serform "github.com/microsoft/kiota-serialization-form-go"
	serjson "github.com/microsoft/kiota-serialization-json-go"
	sermultipart "github.com/microsoft/kiota-serialization-multipart-go"
	sertext "github.com/microsoft/kiota-serialization-text-go"
)

// DefaultRequestAdapter is the core service used by GraphServiceClient to make requests to Microsoft Graph.
type DefaultRequestAdapter struct {
	khttp.NetHttpRequestAdapter
}

// NewDefaultRequestAdapter creates a new DefaultRequestAdapter with the given parameters
func NewDefaultRequestAdapter(authenticationProvider absauth.AuthenticationProvider) (*DefaultRequestAdapter, error) {
	return NewDefaultRequestAdapterWithParseNodeFactory(authenticationProvider, nil)
}

// NewDefaultRequestAdapterWithParseNodeFactory creates a new DefaultRequestAdapter with the given parameters
func NewDefaultRequestAdapterWithParseNodeFactory(authenticationProvider absauth.AuthenticationProvider, parseNodeFactory absser.ParseNodeFactory) (*DefaultRequestAdapter, error) {
	return NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory(authenticationProvider, parseNodeFactory, nil)
}

// NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory creates a new DefaultRequestAdapter with the given parameters
func NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory(authenticationProvider absauth.AuthenticationProvider, parseNodeFactory absser.ParseNodeFactory, serializationWriterFactory absser.SerializationWriterFactory) (*DefaultRequestAdapter, error) {
	return NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authenticationProvider, parseNodeFactory, serializationWriterFactory, nil)
}

// NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient creates a new DefaultRequestAdapter with the given parameters
func NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authenticationProvider absauth.AuthenticationProvider, parseNodeFactory absser.ParseNodeFactory, serializationWriterFactory absser.SerializationWriterFactory, httpClient *nethttp.Client) (*DefaultRequestAdapter, error) {
	return NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClientAndObservabilityOptions(authenticationProvider, parseNodeFactory, serializationWriterFactory, httpClient, khttp.ObservabilityOptions{})
}

// NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClientAndObservabilityOptions creates a new DefaultRequestAdapter with the given parameters including observability options
func NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClientAndObservabilityOptions(authenticationProvider absauth.AuthenticationProvider, parseNodeFactory absser.ParseNodeFactory, serializationWriterFactory absser.SerializationWriterFactory, httpClient *nethttp.Client, observabilityOptions khttp.ObservabilityOptions) (*DefaultRequestAdapter, error) {
	if authenticationProvider == nil {
		return nil, errors.New("authenticationProvider cannot be nil")
	}
	SetupDefaults()
	if httpClient == nil {
		httpClient = khttp.GetDefaultClient()
	}
	if serializationWriterFactory == nil {
		serializationWriterFactory = absser.DefaultSerializationWriterFactoryInstance
	}
	if parseNodeFactory == nil {
		parseNodeFactory = absser.DefaultParseNodeFactoryInstance
	}
	defaultAdapter, err := khttp.NewNetHttpRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClientAndObservabilityOptions(authenticationProvider, parseNodeFactory, serializationWriterFactory, httpClient, observabilityOptions)
	if err != nil {
		return nil, err
	}
	if defaultAdapter == nil {
		return nil, errors.New("failed to initialize request adapter")
	}
	result := &DefaultRequestAdapter{
		NetHttpRequestAdapter: *defaultAdapter,
	}

	return result, nil
}

// GetBaseUrl returns the base URL for the request adapter or empty string if the receiver is nil.
func (a *DefaultRequestAdapter) GetBaseUrl() string {
	if a == nil {
		return ""
	}
	return a.NetHttpRequestAdapter.GetBaseUrl()
}

// SetBaseUrl sets the base URL for the request adapter or performs a no-op if the receiver is nil.
func (a *DefaultRequestAdapter) SetBaseUrl(baseUrl string) {
	if a == nil {
		return
	}
	a.NetHttpRequestAdapter.SetBaseUrl(baseUrl)
}

// GetSerializationWriterFactory returns the serialization writer factory or nil if the receiver is nil.
func (a *DefaultRequestAdapter) GetSerializationWriterFactory() absser.SerializationWriterFactory {
	if a == nil {
		return nil
	}
	return a.NetHttpRequestAdapter.GetSerializationWriterFactory()
}

// EnableBackingStore enables the backing store proxies for the SerializationWriters and ParseNodes in use or performs a no-op if the receiver is nil.
func (a *DefaultRequestAdapter) EnableBackingStore(factory store.BackingStoreFactory) {
	if a == nil {
		return
	}
	a.NetHttpRequestAdapter.EnableBackingStore(factory)
}

var (
	setupDefaultsOnce sync.Once
	setupDefaultsLock sync.Mutex
)

// ResetDefaults resets the default registration state and clears default serializer/deserializer registries.
// It synchronizes reset and registration cycles against SetupDefaults via a package-level mutex.
//
// Note: Because the underlying registry lookups in kiota-abstractions-go (GetSerializationWriter and GetRootParseNode)
// read from their factory maps without locking, callers must ensure that all request adapters and registry consumers
// remain idle throughout reset and reinitialization. ResetDefaults is intended for test suites and initialization
// lifecycles between test runs or configuration phases, and must not be called concurrently with active request
// serialization or response parsing.
func ResetDefaults() {
	setupDefaultsLock.Lock()
	defer setupDefaultsLock.Unlock()

	setupDefaultsOnce = sync.Once{}
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	if absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories != nil {
		for k := range absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories {
			delete(absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories, k)
		}
	} else {
		absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories = make(map[string]absser.SerializationWriterFactory)
	}
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	absser.DefaultParseNodeFactoryInstance.Lock()
	if absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories != nil {
		for k := range absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories {
			delete(absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories, k)
		}
	} else {
		absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories = make(map[string]absser.ParseNodeFactory)
	}
	absser.DefaultParseNodeFactoryInstance.Unlock()
}

// SetupDefaults registers the default serializers and deserializers in the global registries.
// It is thread-safe and executes only once.
func SetupDefaults() {
	setupDefaultsLock.Lock()
	defer setupDefaultsLock.Unlock()
	setupDefaults()
}

func setupDefaults() {
	setupDefaultsOnce.Do(func() {
		abs.RegisterDefaultSerializer(func() absser.SerializationWriterFactory {
			return serjson.NewJsonSerializationWriterFactory()
		})
		abs.RegisterDefaultSerializer(func() absser.SerializationWriterFactory {
			return sertext.NewTextSerializationWriterFactory()
		})
		abs.RegisterDefaultSerializer(func() absser.SerializationWriterFactory {
			return serform.NewFormSerializationWriterFactory()
		})
		abs.RegisterDefaultSerializer(func() absser.SerializationWriterFactory {
			return sermultipart.NewMultipartSerializationWriterFactory()
		})
		abs.RegisterDefaultDeserializer(func() absser.ParseNodeFactory {
			return serjson.NewJsonParseNodeFactory()
		})
		abs.RegisterDefaultDeserializer(func() absser.ParseNodeFactory {
			return sertext.NewTextParseNodeFactory()
		})
		abs.RegisterDefaultDeserializer(func() absser.ParseNodeFactory {
			return serform.NewFormParseNodeFactory()
		})
	})
}
