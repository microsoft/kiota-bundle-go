package bundle

import (
	"context"
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

// GetSerializationWriterFactory returns the serialization writer factory currently in use by the request adapter, or nil if the receiver is nil.
func (a *DefaultRequestAdapter) GetSerializationWriterFactory() absser.SerializationWriterFactory {
	if a == nil {
		return nil
	}
	return a.NetHttpRequestAdapter.GetSerializationWriterFactory()
}

// GetParseNodeFactory returns the parse node factory currently in use by the request adapter, or nil if the receiver is nil.
func (a *DefaultRequestAdapter) GetParseNodeFactory() absser.ParseNodeFactory {
	if a == nil {
		return nil
	}
	return a.NetHttpRequestAdapter.GetParseNodeFactory()
}

// EnableBackingStore enables the backing store proxies for the SerializationWriters and ParseNodes in use or performs a no-op if the receiver is nil.
func (a *DefaultRequestAdapter) EnableBackingStore(factory store.BackingStoreFactory) {
	if a == nil {
		return
	}
	a.NetHttpRequestAdapter.EnableBackingStore(factory)
}

// ConvertToNativeRequest converts the given RequestInformation into a native HTTP request or returns an error if the receiver is nil.
func (a *DefaultRequestAdapter) ConvertToNativeRequest(ctx context.Context, requestInfo *abs.RequestInformation) (any, error) {
	if a == nil {
		return nil, errors.New("request adapter cannot be nil")
	}
	return a.NetHttpRequestAdapter.ConvertToNativeRequest(ctx, requestInfo)
}

// Send executes the HTTP request specified by the given RequestInformation and returns the deserialized response model or an error if the receiver is nil.
func (a *DefaultRequestAdapter) Send(ctx context.Context, requestInfo *abs.RequestInformation, constructor absser.ParsableFactory, errorMappings abs.ErrorMappings) (absser.Parsable, error) {
	if a == nil {
		return nil, errors.New("request adapter cannot be nil")
	}
	return a.NetHttpRequestAdapter.Send(ctx, requestInfo, constructor, errorMappings)
}

// SendEnum executes the HTTP request specified by the given RequestInformation and returns the deserialized response model or an error if the receiver is nil.
func (a *DefaultRequestAdapter) SendEnum(ctx context.Context, requestInfo *abs.RequestInformation, parser absser.EnumFactory, errorMappings abs.ErrorMappings) (any, error) {
	if a == nil {
		return nil, errors.New("request adapter cannot be nil")
	}
	return a.NetHttpRequestAdapter.SendEnum(ctx, requestInfo, parser, errorMappings)
}

// SendCollection executes the HTTP request specified by the given RequestInformation and returns the deserialized response model collection or an error if the receiver is nil.
func (a *DefaultRequestAdapter) SendCollection(ctx context.Context, requestInfo *abs.RequestInformation, constructor absser.ParsableFactory, errorMappings abs.ErrorMappings) ([]absser.Parsable, error) {
	if a == nil {
		return nil, errors.New("request adapter cannot be nil")
	}
	return a.NetHttpRequestAdapter.SendCollection(ctx, requestInfo, constructor, errorMappings)
}

// SendEnumCollection executes the HTTP request specified by the given RequestInformation and returns the deserialized response model collection or an error if the receiver is nil.
func (a *DefaultRequestAdapter) SendEnumCollection(ctx context.Context, requestInfo *abs.RequestInformation, parser absser.EnumFactory, errorMappings abs.ErrorMappings) ([]any, error) {
	if a == nil {
		return nil, errors.New("request adapter cannot be nil")
	}
	return a.NetHttpRequestAdapter.SendEnumCollection(ctx, requestInfo, parser, errorMappings)
}

// SendPrimitive executes the HTTP request specified by the given RequestInformation and returns the deserialized primitive response model or an error if the receiver is nil.
func (a *DefaultRequestAdapter) SendPrimitive(ctx context.Context, requestInfo *abs.RequestInformation, typeName string, errorMappings abs.ErrorMappings) (any, error) {
	if a == nil {
		return nil, errors.New("request adapter cannot be nil")
	}
	return a.NetHttpRequestAdapter.SendPrimitive(ctx, requestInfo, typeName, errorMappings)
}

// SendPrimitiveCollection executes the HTTP request specified by the given RequestInformation and returns the deserialized primitive response model collection or an error if the receiver is nil.
func (a *DefaultRequestAdapter) SendPrimitiveCollection(ctx context.Context, requestInfo *abs.RequestInformation, typeName string, errorMappings abs.ErrorMappings) ([]any, error) {
	if a == nil {
		return nil, errors.New("request adapter cannot be nil")
	}
	return a.NetHttpRequestAdapter.SendPrimitiveCollection(ctx, requestInfo, typeName, errorMappings)
}

// SendNoContent executes the HTTP request specified by the given RequestInformation with no return content or returns an error if the receiver is nil.
func (a *DefaultRequestAdapter) SendNoContent(ctx context.Context, requestInfo *abs.RequestInformation, errorMappings abs.ErrorMappings) error {
	if a == nil {
		return errors.New("request adapter cannot be nil")
	}
	return a.NetHttpRequestAdapter.SendNoContent(ctx, requestInfo, errorMappings)
}

var (
	setupDefaultsOnce sync.Once
	setupDefaultsLock sync.Mutex
)

// ResetDefaults resets the default registration state, clears default serializer/deserializer registries,
// and restores BackingStoreFactoryInstance to NewInMemoryBackingStore.
// It also ensures default serializer and deserializer registry singletons are initialized if nil.
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
	store.BackingStoreFactoryInstance = store.NewInMemoryBackingStore

	writerReg := absser.DefaultSerializationWriterFactoryInstance
	clearOrInitRegistry(
		writerReg,
		writerReg == nil,
		func() { absser.DefaultSerializationWriterFactoryInstance = absser.NewSerializationWriterFactoryRegistry() },
		func(r *absser.SerializationWriterFactoryRegistry) map[string]absser.SerializationWriterFactory {
			return r.ContentTypeAssociatedFactories
		},
		func(r *absser.SerializationWriterFactoryRegistry, m map[string]absser.SerializationWriterFactory) {
			r.ContentTypeAssociatedFactories = m
		},
	)

	parseReg := absser.DefaultParseNodeFactoryInstance
	clearOrInitRegistry(
		parseReg,
		parseReg == nil,
		func() { absser.DefaultParseNodeFactoryInstance = absser.NewParseNodeFactoryRegistry() },
		func(r *absser.ParseNodeFactoryRegistry) map[string]absser.ParseNodeFactory {
			return r.ContentTypeAssociatedFactories
		},
		func(r *absser.ParseNodeFactoryRegistry, m map[string]absser.ParseNodeFactory) {
			r.ContentTypeAssociatedFactories = m
		},
	)
}

func clearOrInitRegistry[R sync.Locker, F any](
	reg R,
	isNil bool,
	initFn func(),
	getMap func(R) map[string]F,
	setMap func(R, map[string]F),
) {
	if isNil {
		initFn()
		return
	}
	reg.Lock()
	defer reg.Unlock()

	factories := getMap(reg)
	if factories != nil {
		for k := range factories {
			delete(factories, k)
		}
	} else {
		setMap(reg, make(map[string]F))
	}
}

// SetupDefaults registers the default serializers and deserializers in the global registries.
// It ensures default serializer and deserializer registry singletons are initialized if nil before registration.
// It is thread-safe and executes only once.
func SetupDefaults() {
	setupDefaultsLock.Lock()
	defer setupDefaultsLock.Unlock()
	setupDefaults()
}

func registerDefaultSerializers() {
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
}

func registerDefaultDeserializers() {
	abs.RegisterDefaultDeserializer(func() absser.ParseNodeFactory {
		return serjson.NewJsonParseNodeFactory()
	})
	abs.RegisterDefaultDeserializer(func() absser.ParseNodeFactory {
		return sertext.NewTextParseNodeFactory()
	})
	abs.RegisterDefaultDeserializer(func() absser.ParseNodeFactory {
		return serform.NewFormParseNodeFactory()
	})
}

func setupDefaults() {
	recreatedWriters := false
	if absser.DefaultSerializationWriterFactoryInstance == nil {
		absser.DefaultSerializationWriterFactoryInstance = absser.NewSerializationWriterFactoryRegistry()
		recreatedWriters = true
	}

	recreatedParsers := false
	if absser.DefaultParseNodeFactoryInstance == nil {
		absser.DefaultParseNodeFactoryInstance = absser.NewParseNodeFactoryRegistry()
		recreatedParsers = true
	}

	setupDefaultsOnce.Do(func() {
		registerDefaultSerializers()
		registerDefaultDeserializers()
		recreatedWriters = false
		recreatedParsers = false
	})

	if recreatedWriters {
		registerDefaultSerializers()
	}
	if recreatedParsers {
		registerDefaultDeserializers()
	}
}
