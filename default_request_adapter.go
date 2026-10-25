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
	parseNodeFactory absser.ParseNodeFactory
	httpClient       *nethttp.Client
}

// NewDefaultRequestAdapter creates a new DefaultRequestAdapter with the given parameters
func NewDefaultRequestAdapter(authenticationProvider absauth.AuthenticationProvider) (*DefaultRequestAdapter, error) {
	return NewDefaultRequestAdapterWithParseNodeFactory(authenticationProvider, nil)
}

// NewDefaultRequestAdapterWithParseNodeFactory creates a new DefaultRequestAdapter with the given parameters
func NewDefaultRequestAdapterWithParseNodeFactory(authenticationProvider absauth.AuthenticationProvider, parseNodeFactory absser.ParseNodeFactory) (*DefaultRequestAdapter, error) {
	return NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory(authenticationProvider, parseNodeFactory, nil)
}

// NewDefaultRequestAdapterWithSerializationWriterFactory creates a new DefaultRequestAdapter with the given parameters
func NewDefaultRequestAdapterWithSerializationWriterFactory(authenticationProvider absauth.AuthenticationProvider, serializationWriterFactory absser.SerializationWriterFactory) (*DefaultRequestAdapter, error) {
	return NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authenticationProvider, nil, serializationWriterFactory, nil)
}

// NewDefaultRequestAdapterWithHttpClient creates a new DefaultRequestAdapter with the given authentication provider and custom HTTP client
func NewDefaultRequestAdapterWithHttpClient(authenticationProvider absauth.AuthenticationProvider, httpClient *nethttp.Client) (*DefaultRequestAdapter, error) {
	return NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authenticationProvider, nil, nil, httpClient)
}

// NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory creates a new DefaultRequestAdapter with the given parameters
func NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory(authenticationProvider absauth.AuthenticationProvider, parseNodeFactory absser.ParseNodeFactory, serializationWriterFactory absser.SerializationWriterFactory) (*DefaultRequestAdapter, error) {
	return NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authenticationProvider, parseNodeFactory, serializationWriterFactory, nil)
}

// NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient creates a new DefaultRequestAdapter with the given parameters
func NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authenticationProvider absauth.AuthenticationProvider, parseNodeFactory absser.ParseNodeFactory, serializationWriterFactory absser.SerializationWriterFactory, httpClient *nethttp.Client) (*DefaultRequestAdapter, error) {
	if authenticationProvider == nil {
		return nil, errors.New("authenticationProvider cannot be nil")
	}
	setupDefaults()
	if httpClient == nil {
		httpClient = khttp.GetDefaultClient()
	}
	if serializationWriterFactory == nil {
		serializationWriterFactory = absser.DefaultSerializationWriterFactoryInstance
	}
	if parseNodeFactory == nil {
		parseNodeFactory = absser.DefaultParseNodeFactoryInstance
	}
	defaultAdapter, err := khttp.NewNetHttpRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authenticationProvider, parseNodeFactory, serializationWriterFactory, httpClient)
	if err != nil {
		return nil, err
	}
	if defaultAdapter == nil {
		return nil, errors.New("failed to initialize request adapter")
	}
	result := &DefaultRequestAdapter{
		NetHttpRequestAdapter: *defaultAdapter,
		parseNodeFactory:      parseNodeFactory,
		httpClient:            httpClient,
	}

	return result, nil
}

// GetParseNodeFactory returns the parse node factory used by the request adapter.
func (a *DefaultRequestAdapter) GetParseNodeFactory() absser.ParseNodeFactory {
	if a == nil {
		return nil
	}
	return a.parseNodeFactory
}

// GetHttpClient returns the HTTP client used by the request adapter.
func (a *DefaultRequestAdapter) GetHttpClient() *nethttp.Client {
	if a == nil {
		return nil
	}
	return a.httpClient
}

// GetSerializationWriterFactory returns the serialization writer factory currently in use.
func (a *DefaultRequestAdapter) GetSerializationWriterFactory() absser.SerializationWriterFactory {
	if a == nil {
		return nil
	}
	return a.NetHttpRequestAdapter.GetSerializationWriterFactory()
}

// GetBaseUrl returns the base url for every request.
func (a *DefaultRequestAdapter) GetBaseUrl() string {
	if a == nil {
		return ""
	}
	return a.NetHttpRequestAdapter.GetBaseUrl()
}

// SetBaseUrl sets the base url for every request.
func (a *DefaultRequestAdapter) SetBaseUrl(baseUrl string) {
	if a == nil {
		return
	}
	a.NetHttpRequestAdapter.SetBaseUrl(baseUrl)
}

// EnableBackingStore enables the backing store proxies for the SerializationWriters and ParseNodes in use.
func (a *DefaultRequestAdapter) EnableBackingStore(factory store.BackingStoreFactory) {
	if a == nil {
		return
	}
	a.NetHttpRequestAdapter.EnableBackingStore(factory)
	if a.parseNodeFactory != nil {
		a.parseNodeFactory = abs.EnableBackingStoreForParseNodeFactory(a.parseNodeFactory)
	}
}

var setupDefaultsOnce sync.Once

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
