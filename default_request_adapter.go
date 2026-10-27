package bundle

import (
	"errors"
	nethttp "net/http"
	"sync"

	abs "github.com/microsoft/kiota-abstractions-go"
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	absser "github.com/microsoft/kiota-abstractions-go/serialization"
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

var (
	setupDefaultsOnce sync.Once
	setupDefaultsLock sync.Mutex
)

// ResetDefaults resets the default registration state and clears default serializer/deserializer registries.
// It is thread-safe and allows consumers and test suites to reset registrations between test runs or dynamic reconfigurations.
func ResetDefaults() {
	setupDefaultsLock.Lock()
	defer setupDefaultsLock.Unlock()

	setupDefaultsOnce = sync.Once{}
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories = make(map[string]absser.SerializationWriterFactory)
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	absser.DefaultParseNodeFactoryInstance.Lock()
	absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories = make(map[string]absser.ParseNodeFactory)
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
