package bundle

import (
	"context"
	"io"
	nethttp "net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	abs "github.com/microsoft/kiota-abstractions-go"
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	absser "github.com/microsoft/kiota-abstractions-go/serialization"
	"github.com/microsoft/kiota-abstractions-go/store"
	khttp "github.com/microsoft/kiota-http-go"
	"github.com/stretchr/testify/assert"
)

type recordingTransport struct {
	req *nethttp.Request
}

func (t *recordingTransport) RoundTrip(req *nethttp.Request) (*nethttp.Response, error) {
	t.req = req
	return &nethttp.Response{
		StatusCode: nethttp.StatusOK,
		Header:     make(nethttp.Header),
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

func TestAdapterThrowsErrorOnNillAuthProvider(t *testing.T) {
	_, err := NewDefaultRequestAdapter(nil)
	assert.Error(t, err) // adapter can't be nill
}

func TestAdapterThrowsErrorOnNilAuthProviderAcrossConstructors(t *testing.T) {
	_, err1 := NewDefaultRequestAdapterWithParseNodeFactory(nil, nil)
	assert.Error(t, err1)

	_, err2 := NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory(nil, nil, nil)
	assert.Error(t, err2)

	_, err3 := NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(nil, nil, nil, nil)
	assert.Error(t, err3)

	_, err4 := NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClientAndObservabilityOptions(nil, nil, nil, nil, khttp.ObservabilityOptions{})
	assert.Error(t, err4)
}

func TestAerializersAreRegisteredAsExpected(t *testing.T) {
	TestSerializersAreRegisteredAsExpected(t)
}

func TestSerializersAreRegisteredAsExpected(t *testing.T) {
	authProvider := &absauth.AnonymousAuthenticationProvider{}
	_, err := NewDefaultRequestAdapter(authProvider)
	assert.NoError(t, err) // properly created adapter

	// validate
	serializerCount := len(absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories)
	deserializerCount := len(absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories)

	assert.Equal(t, 4, serializerCount)   // four serializers present
	assert.Equal(t, 3, deserializerCount) // three deserializers present

	serializerMap := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories
	deserializerMap := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories

	assert.Contains(t, serializerMap, "application/json")
	assert.Contains(t, deserializerMap, "application/json") // Serializer and deserializer present for application/json

	assert.Contains(t, serializerMap, "text/plain")
	assert.Contains(t, deserializerMap, "text/plain") // Serializer and deserializer present for text/plain

	assert.Contains(t, serializerMap, "application/x-www-form-urlencoded")
	assert.Contains(t, deserializerMap, "application/x-www-form-urlencoded") // Serializer and deserializer present for application/x-www-form-urlencoded

	assert.Contains(t, serializerMap, "multipart/form-data") // Serializer present for multipart/form-data
}

func TestConcurrentAdapterCreation(t *testing.T) {
	// Reset registration state so workers exercise concurrent first-time registration
	setupDefaultsOnce = sync.Once{}
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories = make(map[string]absser.SerializationWriterFactory)
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	absser.DefaultParseNodeFactoryInstance.Lock()
	absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories = make(map[string]absser.ParseNodeFactory)
	absser.DefaultParseNodeFactoryInstance.Unlock()

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			authProvider := &absauth.AnonymousAuthenticationProvider{}
			adapter, err := NewDefaultRequestAdapter(authProvider)
			assert.NoError(t, err)
			assert.NotNil(t, adapter)
		}()
	}

	close(start)
	wg.Wait()

	// Verify all expected default serializers and deserializers are registered
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	serializerCount := len(absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories)
	absser.DefaultSerializationWriterFactoryInstance.Unlock()
	assert.Equal(t, 4, serializerCount)

	absser.DefaultParseNodeFactoryInstance.Lock()
	deserializerCount := len(absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories)
	absser.DefaultParseNodeFactoryInstance.Unlock()
	assert.Equal(t, 3, deserializerCount)
}

func TestNewDefaultRequestAdapterWithCustomFactories(t *testing.T) {
	authProvider := &absauth.AnonymousAuthenticationProvider{}
	customClient := &nethttp.Client{}

	adapter1, err1 := NewDefaultRequestAdapterWithParseNodeFactory(authProvider, absser.DefaultParseNodeFactoryInstance)
	assert.NoError(t, err1)
	assert.NotNil(t, adapter1)

	adapter2, err2 := NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory(authProvider, absser.DefaultParseNodeFactoryInstance, absser.DefaultSerializationWriterFactoryInstance)
	assert.NoError(t, err2)
	assert.NotNil(t, adapter2)

	adapter3, err3 := NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authProvider, absser.DefaultParseNodeFactoryInstance, absser.DefaultSerializationWriterFactoryInstance, customClient)
	assert.NoError(t, err3)
	assert.NotNil(t, adapter3)

	recorder := &recordingTransport{}
	clientWithTransport := &nethttp.Client{Transport: recorder}
	obsOptions := khttp.ObservabilityOptions{
		IncludeEUIIAttributes: true,
	}
	adapter4, err4 := NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClientAndObservabilityOptions(
		authProvider,
		absser.DefaultParseNodeFactoryInstance,
		absser.DefaultSerializationWriterFactoryInstance,
		clientWithTransport,
		obsOptions,
	)
	assert.NoError(t, err4)
	assert.NotNil(t, adapter4)

	reqInfo := abs.NewRequestInformation()
	parsedUrl, err := url.Parse("https://example.com")
	assert.NoError(t, err)
	reqInfo.SetUri(*parsedUrl)
	reqInfo.Method = abs.GET

	err = adapter4.SendNoContent(context.Background(), reqInfo, nil)
	assert.NoError(t, err)
	assert.NotNil(t, recorder.req)

	capturedOptions := khttp.GetObservabilityOptionsFromRequest(recorder.req)
	assert.NotNil(t, capturedOptions)
	assert.True(t, capturedOptions.GetIncludeEUIIAttributes())
}

func TestSetupDefaultsExported(t *testing.T) {
	// Reset registration state to ensure standalone setup is exercised
	setupDefaultsOnce = sync.Once{}
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories = make(map[string]absser.SerializationWriterFactory)
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	absser.DefaultParseNodeFactoryInstance.Lock()
	absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories = make(map[string]absser.ParseNodeFactory)
	absser.DefaultParseNodeFactoryInstance.Unlock()

	SetupDefaults()

	absser.DefaultSerializationWriterFactoryInstance.Lock()
	serializerCount := len(absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories)
	absser.DefaultSerializationWriterFactoryInstance.Unlock()
	assert.Equal(t, 4, serializerCount)

	absser.DefaultParseNodeFactoryInstance.Lock()
	deserializerCount := len(absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories)
	absser.DefaultParseNodeFactoryInstance.Unlock()
	assert.Equal(t, 3, deserializerCount)
}

func TestDefaultRequestAdapterAccessorsAndNilGuards(t *testing.T) {
	authProvider := &absauth.AnonymousAuthenticationProvider{}
	adapter, err := NewDefaultRequestAdapter(authProvider)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)

	// Accessors on valid adapter
	assert.NotNil(t, adapter.GetHttpClient())
	assert.NotNil(t, adapter.GetParseNodeFactory())
	assert.NotNil(t, adapter.GetSerializationWriterFactory())
	assert.Equal(t, "", adapter.GetBaseUrl())

	adapter.SetBaseUrl("https://graph.microsoft.com/v1.0")
	assert.Equal(t, "https://graph.microsoft.com/v1.0", adapter.GetBaseUrl())

	// Accessors on nil adapter should not panic
	var nilAdapter *DefaultRequestAdapter
	assert.Nil(t, nilAdapter.GetHttpClient())
	assert.Nil(t, nilAdapter.GetParseNodeFactory())
	assert.Nil(t, nilAdapter.GetSerializationWriterFactory())
	assert.Equal(t, "", nilAdapter.GetBaseUrl())
	nilAdapter.SetBaseUrl("https://example.com") // no-op, shouldn't panic
}

type mockCustomParseNodeFactory struct{}

func (m *mockCustomParseNodeFactory) GetValidContentType() (string, error) {
	return "application/custom", nil
}

func (m *mockCustomParseNodeFactory) GetRootParseNode(contentType string, content []byte) (absser.ParseNode, error) {
	return nil, nil
}

func TestEnableBackingStoreSynchronizesParseNodeFactory(t *testing.T) {
	authProvider := &absauth.AnonymousAuthenticationProvider{}
	customFactory := &mockCustomParseNodeFactory{}

	adapter, err := NewDefaultRequestAdapterWithParseNodeFactory(authProvider, customFactory)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.Equal(t, customFactory, adapter.GetParseNodeFactory())

	// Interface implementation verification
	assert.Implements(t, (*abs.RequestAdapter)(nil), adapter)

	// Enable backing store
	adapter.EnableBackingStore(nil)

	// After enabling backing store, parseNodeFactory should be wrapped with BackingStoreParseNodeFactory
	backingStoreFactory, ok := adapter.GetParseNodeFactory().(*store.BackingStoreParseNodeFactory)
	assert.True(t, ok, "parseNodeFactory should be wrapped with BackingStoreParseNodeFactory")
	assert.NotNil(t, backingStoreFactory)

	// Nil adapter check
	var nilAdapter *DefaultRequestAdapter
	nilAdapter.EnableBackingStore(nil) // should not panic
}
