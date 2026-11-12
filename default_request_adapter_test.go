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

var _ abs.RequestAdapter = (*DefaultRequestAdapter)(nil)

func TestDefaultRequestAdapterHonoursInterface(t *testing.T) {
	instance, err := NewDefaultRequestAdapter(&absauth.AnonymousAuthenticationProvider{})
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	assert.Implements(t, (*abs.RequestAdapter)(nil), instance)

	var nilAdapter *DefaultRequestAdapter
	assert.Implements(t, (*abs.RequestAdapter)(nil), nilAdapter)
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
	ResetDefaults()

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
	ResetDefaults()

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

func TestResetDefaults(t *testing.T) {
	// Override backing store factory to verify restoration on ResetDefaults
	customCalled := false
	store.BackingStoreFactoryInstance = func() store.BackingStore {
		customCalled = true
		return store.NewInMemoryBackingStore()
	}
	_ = store.BackingStoreFactoryInstance()
	assert.True(t, customCalled)

	// Ensure defaults are registered
	SetupDefaults()

	absser.DefaultSerializationWriterFactoryInstance.Lock()
	serializerCount := len(absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories)
	absser.DefaultSerializationWriterFactoryInstance.Unlock()
	assert.Equal(t, 4, serializerCount)

	absser.DefaultParseNodeFactoryInstance.Lock()
	deserializerCount := len(absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories)
	absser.DefaultParseNodeFactoryInstance.Unlock()
	assert.Equal(t, 3, deserializerCount)

	// Reset registrations
	ResetDefaults()

	absser.DefaultSerializationWriterFactoryInstance.Lock()
	serializerCount = len(absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories)
	absser.DefaultSerializationWriterFactoryInstance.Unlock()
	assert.Equal(t, 0, serializerCount)

	absser.DefaultParseNodeFactoryInstance.Lock()
	deserializerCount = len(absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories)
	absser.DefaultParseNodeFactoryInstance.Unlock()
	assert.Equal(t, 0, deserializerCount)

	// Verify BackingStoreFactoryInstance was restored to default InMemoryBackingStore
	customCalled = false
	bs := store.BackingStoreFactoryInstance()
	assert.NotNil(t, bs)
	assert.False(t, customCalled)
	_, isInMemory := bs.(*store.InMemoryBackingStore)
	assert.True(t, isInMemory)

	// Calling SetupDefaults again re-populates the registries
	SetupDefaults()

	absser.DefaultSerializationWriterFactoryInstance.Lock()
	serializerCount = len(absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories)
	absser.DefaultSerializationWriterFactoryInstance.Unlock()
	assert.Equal(t, 4, serializerCount)

	absser.DefaultParseNodeFactoryInstance.Lock()
	deserializerCount = len(absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories)
	absser.DefaultParseNodeFactoryInstance.Unlock()
	assert.Equal(t, 3, deserializerCount)
}

func TestResetDefaultsRestoresBackingStoreFactoryInstance(t *testing.T) {
	customStoreCalled := false
	customFactory := func() store.BackingStore {
		customStoreCalled = true
		return store.NewInMemoryBackingStore()
	}
	store.BackingStoreFactoryInstance = customFactory
	_ = store.BackingStoreFactoryInstance()
	assert.True(t, customStoreCalled)

	// Calling ResetDefaults must restore default InMemoryBackingStore factory
	ResetDefaults()

	assert.NotNil(t, store.BackingStoreFactoryInstance)
	bs := store.BackingStoreFactoryInstance()
	assert.NotNil(t, bs)
	_, isInMemory := bs.(*store.InMemoryBackingStore)
	assert.True(t, isInMemory)

	// Verify that customFactory is no longer invoked
	customStoreCalled = false
	_ = store.BackingStoreFactoryInstance()
	assert.False(t, customStoreCalled)
}

func TestResetDefaultsWithNilFactoryInstances(t *testing.T) {
	defer func() {
		ResetDefaults()
		SetupDefaults()
	}()

	// Simulate environments or test fixtures where factory singletons were cleared or nil
	absser.DefaultSerializationWriterFactoryInstance = nil
	absser.DefaultParseNodeFactoryInstance = nil

	// ResetDefaults must not panic and must safely reinitialize both registries
	ResetDefaults()

	assert.NotNil(t, absser.DefaultSerializationWriterFactoryInstance)
	assert.NotNil(t, absser.DefaultParseNodeFactoryInstance)

	absser.DefaultSerializationWriterFactoryInstance.Lock()
	serializerCount := len(absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories)
	absser.DefaultSerializationWriterFactoryInstance.Unlock()
	assert.Equal(t, 0, serializerCount)

	absser.DefaultParseNodeFactoryInstance.Lock()
	deserializerCount := len(absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories)
	absser.DefaultParseNodeFactoryInstance.Unlock()
	assert.Equal(t, 0, deserializerCount)

	// Verify one nil and one populated
	absser.DefaultSerializationWriterFactoryInstance = nil
	ResetDefaults()
	assert.NotNil(t, absser.DefaultSerializationWriterFactoryInstance)
	assert.NotNil(t, absser.DefaultParseNodeFactoryInstance)

	absser.DefaultParseNodeFactoryInstance = nil
	ResetDefaults()
	assert.NotNil(t, absser.DefaultSerializationWriterFactoryInstance)
	assert.NotNil(t, absser.DefaultParseNodeFactoryInstance)
}

func TestSetupDefaultsWithNilFactoryInstances(t *testing.T) {
	defer func() {
		ResetDefaults()
		SetupDefaults()
	}()

	// Reset state and clear factory registry singletons
	ResetDefaults()
	absser.DefaultSerializationWriterFactoryInstance = nil
	absser.DefaultParseNodeFactoryInstance = nil

	// SetupDefaults must safely reinitialize registries and register default serializers and deserializers
	SetupDefaults()

	assert.NotNil(t, absser.DefaultSerializationWriterFactoryInstance)
	assert.NotNil(t, absser.DefaultParseNodeFactoryInstance)

	absser.DefaultSerializationWriterFactoryInstance.Lock()
	serializerMap := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories
	assert.NotEmpty(t, serializerMap)
	assert.Contains(t, serializerMap, "application/json")
	assert.Contains(t, serializerMap, "text/plain")
	assert.Contains(t, serializerMap, "application/x-www-form-urlencoded")
	assert.Contains(t, serializerMap, "multipart/form-data")
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	absser.DefaultParseNodeFactoryInstance.Lock()
	deserializerMap := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories
	assert.NotEmpty(t, deserializerMap)
	assert.Contains(t, deserializerMap, "application/json")
	assert.Contains(t, deserializerMap, "text/plain")
	assert.Contains(t, deserializerMap, "application/x-www-form-urlencoded")
	absser.DefaultParseNodeFactoryInstance.Unlock()

	// Adapter creation must succeed using the newly initialized default registries
	authProvider := &absauth.AnonymousAuthenticationProvider{}
	adapter, err := NewDefaultRequestAdapter(authProvider)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.Equal(t, absser.DefaultSerializationWriterFactoryInstance, adapter.GetSerializationWriterFactory())
	assert.Equal(t, absser.DefaultParseNodeFactoryInstance, adapter.GetParseNodeFactory())

	// Test partial nil state for serialization factory
	ResetDefaults()
	absser.DefaultSerializationWriterFactoryInstance = nil
	SetupDefaults()
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	assert.NotEmpty(t, absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories)
	assert.Contains(t, absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories, "application/json")
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	// Test partial nil state for parse node factory
	ResetDefaults()
	absser.DefaultParseNodeFactoryInstance = nil
	SetupDefaults()
	absser.DefaultParseNodeFactoryInstance.Lock()
	assert.NotEmpty(t, absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories)
	assert.Contains(t, absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories, "application/json")
	absser.DefaultParseNodeFactoryInstance.Unlock()
}

func TestSetupDefaultsRecreatesAndReregistersClearedSingletonsWithoutResetDefaults(t *testing.T) {
	defer func() {
		ResetDefaults()
		SetupDefaults()
	}()

	// 1. Initial setup defaults
	ResetDefaults()
	SetupDefaults()

	// Add custom factory to unaffected registry to verify it is preserved
	customContentType := "application/x-custom-test"
	absser.DefaultParseNodeFactoryInstance.Lock()
	absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories[customContentType] = absser.NewParseNodeFactoryRegistry()
	absser.DefaultParseNodeFactoryInstance.Unlock()

	// 2. Clear serialization writer singleton WITHOUT calling ResetDefaults
	absser.DefaultSerializationWriterFactoryInstance = nil

	// Call SetupDefaults again
	SetupDefaults()

	// Verify writer registry was recreated AND re-registered with defaults
	assert.NotNil(t, absser.DefaultSerializationWriterFactoryInstance)
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	serializerMap := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories
	assert.NotEmpty(t, serializerMap)
	assert.Contains(t, serializerMap, "application/json")
	assert.Contains(t, serializerMap, "text/plain")
	assert.Contains(t, serializerMap, "application/x-www-form-urlencoded")
	assert.Contains(t, serializerMap, "multipart/form-data")
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	// Verify unaffected parse node registry was preserved (including custom factory)
	absser.DefaultParseNodeFactoryInstance.Lock()
	deserializerMap := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories
	assert.NotEmpty(t, deserializerMap)
	assert.Contains(t, deserializerMap, "application/json")
	assert.Contains(t, deserializerMap, customContentType)
	absser.DefaultParseNodeFactoryInstance.Unlock()

	// 3. Clear parse node factory singleton WITHOUT calling ResetDefaults
	absser.DefaultParseNodeFactoryInstance = nil

	// Call SetupDefaults again
	SetupDefaults()

	// Verify parse node registry was recreated AND re-registered with defaults
	assert.NotNil(t, absser.DefaultParseNodeFactoryInstance)
	absser.DefaultParseNodeFactoryInstance.Lock()
	deserializerMap2 := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories
	assert.NotEmpty(t, deserializerMap2)
	assert.Contains(t, deserializerMap2, "application/json")
	assert.Contains(t, deserializerMap2, "text/plain")
	assert.Contains(t, deserializerMap2, "application/x-www-form-urlencoded")
	absser.DefaultParseNodeFactoryInstance.Unlock()

	// Verify unaffected writer registry remains intact
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	assert.NotEmpty(t, absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories)
	assert.Contains(t, absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories, "application/json")
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	// Adapter creation must succeed and contain default serializers & deserializers
	authProvider := &absauth.AnonymousAuthenticationProvider{}
	adapter, err := NewDefaultRequestAdapter(authProvider)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.Equal(t, absser.DefaultSerializationWriterFactoryInstance, adapter.GetSerializationWriterFactory())
	assert.Equal(t, absser.DefaultParseNodeFactoryInstance, adapter.GetParseNodeFactory())
}

func TestDefaultRequestAdapterAccessorsAndNilSafety(t *testing.T) {
	authProvider := &absauth.AnonymousAuthenticationProvider{}
	adapter, err := NewDefaultRequestAdapter(authProvider)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)

	// Accessors on initialized adapter
	assert.NotNil(t, adapter.GetSerializationWriterFactory())
	assert.NotNil(t, adapter.GetParseNodeFactory())
	assert.Equal(t, "", adapter.GetBaseUrl())

	adapter.SetBaseUrl("https://graph.microsoft.com/v1.0")
	assert.Equal(t, "https://graph.microsoft.com/v1.0", adapter.GetBaseUrl())

	// Accessors on nil adapter must be safe and not panic
	var nilAdapter *DefaultRequestAdapter
	assert.Nil(t, nilAdapter.GetSerializationWriterFactory())
	assert.Nil(t, nilAdapter.GetParseNodeFactory())
	assert.Equal(t, "", nilAdapter.GetBaseUrl())
	nilAdapter.SetBaseUrl("https://example.com") // no-op, must not panic
	nilAdapter.EnableBackingStore(nil)           // no-op, must not panic

	reqInfo := abs.NewRequestInformation()
	converted, err := nilAdapter.ConvertToNativeRequest(context.Background(), reqInfo)
	assert.Error(t, err)
	assert.Nil(t, converted)
	assert.Equal(t, "request adapter cannot be nil", err.Error())

	// Send methods on nil adapter must return an error and not panic
	resp, err := nilAdapter.Send(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "request adapter cannot be nil", err.Error())

	respColl, err := nilAdapter.SendCollection(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respColl)
	assert.Equal(t, "request adapter cannot be nil", err.Error())

	respEnum, err := nilAdapter.SendEnum(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respEnum)
	assert.Equal(t, "request adapter cannot be nil", err.Error())

	respEnumColl, err := nilAdapter.SendEnumCollection(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respEnumColl)
	assert.Equal(t, "request adapter cannot be nil", err.Error())

	respPrim, err := nilAdapter.SendPrimitive(context.Background(), reqInfo, "string", nil)
	assert.Error(t, err)
	assert.Nil(t, respPrim)
	assert.Equal(t, "request adapter cannot be nil", err.Error())

	respPrimColl, err := nilAdapter.SendPrimitiveCollection(context.Background(), reqInfo, "string", nil)
	assert.Error(t, err)
	assert.Nil(t, respPrimColl)
	assert.Equal(t, "request adapter cannot be nil", err.Error())

	err = nilAdapter.SendNoContent(context.Background(), reqInfo, nil)
	assert.Error(t, err)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
}

// TestConcurrentResetAndSetupDefaults validates that concurrent invocations of ResetDefaults and SetupDefaults
// are serialized via setupDefaultsLock and do not race with each other or corrupt sync.Once state.
func TestConcurrentResetAndSetupDefaults(t *testing.T) {
	const iterations = 15
	var wg sync.WaitGroup
	wg.Add(iterations * 2)

	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			ResetDefaults()
		}()
		go func() {
			defer wg.Done()
			SetupDefaults()
		}()
	}

	wg.Wait()

	// Ensure final deterministic setup succeeds cleanly
	ResetDefaults()
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

func TestDefaultRequestAdapterEnableBackingStore(t *testing.T) {
	authProvider := &absauth.AnonymousAuthenticationProvider{}
	adapter, err := NewDefaultRequestAdapter(authProvider)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)

	// Enable backing store on initialized adapter with nil factory
	adapter.EnableBackingStore(nil)
	assert.NotNil(t, adapter.GetParseNodeFactory())

	// Enable backing store with a custom factory
	customStoreCalled := false
	customFactory := func() store.BackingStore {
		customStoreCalled = true
		return store.NewInMemoryBackingStore()
	}
	adapter.EnableBackingStore(customFactory)
	assert.NotNil(t, store.BackingStoreFactoryInstance)
	assert.NotNil(t, adapter.GetParseNodeFactory())
	_ = store.BackingStoreFactoryInstance()
	assert.True(t, customStoreCalled)

	// EnableBackingStore on nil adapter must be safe and not panic
	var nilAdapter *DefaultRequestAdapter
	nilAdapter.EnableBackingStore(nil)           // no-op, must not panic
	nilAdapter.EnableBackingStore(customFactory) // no-op, must not panic
	assert.Nil(t, nilAdapter.GetParseNodeFactory())

	// Reset defaults cleans up registrations and backing store factory singleton
	ResetDefaults()
	bs := store.BackingStoreFactoryInstance()
	assert.NotNil(t, bs)
	_, isInMemory := bs.(*store.InMemoryBackingStore)
	assert.True(t, isInMemory)
}

func TestDefaultRequestAdapterConvertToNativeRequest(t *testing.T) {
	authProvider := &absauth.AnonymousAuthenticationProvider{}
	adapter, err := NewDefaultRequestAdapter(authProvider)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)

	// ConvertToNativeRequest on initialized adapter
	reqInfo := abs.NewRequestInformation()
	parsedUrl, err := url.Parse("https://graph.microsoft.com/v1.0/users")
	assert.NoError(t, err)
	reqInfo.SetUri(*parsedUrl)
	reqInfo.Method = abs.GET

	nativeReq, err := adapter.ConvertToNativeRequest(context.Background(), reqInfo)
	assert.NoError(t, err)
	assert.NotNil(t, nativeReq)
	httpReq, ok := nativeReq.(*nethttp.Request)
	assert.True(t, ok)
	assert.Equal(t, "GET", httpReq.Method)
	assert.Equal(t, "https://graph.microsoft.com/v1.0/users", httpReq.URL.String())

	// ConvertToNativeRequest on nil adapter returns error and does not panic
	var nilAdapter *DefaultRequestAdapter
	converted, err := nilAdapter.ConvertToNativeRequest(context.Background(), reqInfo)
	assert.Error(t, err)
	assert.Nil(t, converted)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
}

func TestDefaultRequestAdapterSendMethods(t *testing.T) {
	recorder := &recordingTransport{}
	clientWithTransport := &nethttp.Client{Transport: recorder}
	authProvider := &absauth.AnonymousAuthenticationProvider{}
	adapter, err := NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(
		authProvider,
		nil,
		nil,
		clientWithTransport,
	)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)

	reqInfo := abs.NewRequestInformation()
	parsedUrl, err := url.Parse("https://graph.microsoft.com/v1.0/me")
	assert.NoError(t, err)
	reqInfo.SetUri(*parsedUrl)
	reqInfo.Method = abs.GET

	// Initialized adapter delegates SendNoContent to underlying NetHttpRequestAdapter
	err = adapter.SendNoContent(context.Background(), reqInfo, nil)
	assert.NoError(t, err)
	assert.NotNil(t, recorder.req)
	assert.Equal(t, "GET", recorder.req.Method)

	// Initialized adapter delegates nil requestInfo error handling to NetHttpRequestAdapter
	err = adapter.SendNoContent(context.Background(), nil, nil)
	assert.Error(t, err)
	assert.Equal(t, "requestInfo cannot be nil", err.Error())

	// All Send methods on nil adapter return error and do not panic
	var nilAdapter *DefaultRequestAdapter

	resp, err := nilAdapter.Send(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "request adapter cannot be nil", err.Error())

	respColl, err := nilAdapter.SendCollection(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respColl)
	assert.Equal(t, "request adapter cannot be nil", err.Error())

	respEnum, err := nilAdapter.SendEnum(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respEnum)
	assert.Equal(t, "request adapter cannot be nil", err.Error())

	respEnumColl, err := nilAdapter.SendEnumCollection(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respEnumColl)
	assert.Equal(t, "request adapter cannot be nil", err.Error())

	respPrim, err := nilAdapter.SendPrimitive(context.Background(), reqInfo, "string", nil)
	assert.Error(t, err)
	assert.Nil(t, respPrim)
	assert.Equal(t, "request adapter cannot be nil", err.Error())

	respPrimColl, err := nilAdapter.SendPrimitiveCollection(context.Background(), reqInfo, "string", nil)
	assert.Error(t, err)
	assert.Nil(t, respPrimColl)
	assert.Equal(t, "request adapter cannot be nil", err.Error())

	err = nilAdapter.SendNoContent(context.Background(), reqInfo, nil)
	assert.Error(t, err)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
}

func TestDefaultRequestAdapterGetParseNodeFactory(t *testing.T) {
	authProvider := &absauth.AnonymousAuthenticationProvider{}

	// Default initialization uses absser.DefaultParseNodeFactoryInstance
	adapter, err := NewDefaultRequestAdapter(authProvider)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.NotNil(t, adapter.GetParseNodeFactory())
	assert.Equal(t, absser.DefaultParseNodeFactoryInstance, adapter.GetParseNodeFactory())

	// Custom parse node factory is preserved and returned
	customFactory := absser.NewParseNodeFactoryRegistry()
	customAdapter, err := NewDefaultRequestAdapterWithParseNodeFactory(authProvider, customFactory)
	assert.NoError(t, err)
	assert.NotNil(t, customAdapter)
	assert.Equal(t, customFactory, customAdapter.GetParseNodeFactory())

	// Nil adapter returns nil safely without panicking
	var nilAdapter *DefaultRequestAdapter
	assert.Nil(t, nilAdapter.GetParseNodeFactory())
}

// TestConcurrentFactoryAccessors validates that concurrent invocations of GetParseNodeFactory
// and GetSerializationWriterFactory on both initialized and nil adapters are safe and do not panic.
func TestConcurrentFactoryAccessors(t *testing.T) {
	authProvider := &absauth.AnonymousAuthenticationProvider{}
	adapter, err := NewDefaultRequestAdapter(authProvider)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)

	var nilAdapter *DefaultRequestAdapter

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				pnf := adapter.GetParseNodeFactory()
				assert.NotNil(t, pnf)
				swf := adapter.GetSerializationWriterFactory()
				assert.NotNil(t, swf)
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				assert.Nil(t, nilAdapter.GetParseNodeFactory())
				assert.Nil(t, nilAdapter.GetSerializationWriterFactory())
			}
		}()
	}

	wg.Wait()
}

func TestAdapterThrowsErrorOnTypedNilAuthProvider(t *testing.T) {
	var typedNilAuth *absauth.ApiKeyAuthenticationProvider

	testCases := []struct {
		name      string
		construct func() (*DefaultRequestAdapter, error)
	}{
		{
			name: "NewDefaultRequestAdapter",
			construct: func() (*DefaultRequestAdapter, error) {
				return NewDefaultRequestAdapter(typedNilAuth)
			},
		},
		{
			name: "NewDefaultRequestAdapterWithParseNodeFactory",
			construct: func() (*DefaultRequestAdapter, error) {
				return NewDefaultRequestAdapterWithParseNodeFactory(typedNilAuth, nil)
			},
		},
		{
			name: "NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory",
			construct: func() (*DefaultRequestAdapter, error) {
				return NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory(typedNilAuth, nil, nil)
			},
		},
		{
			name: "NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient",
			construct: func() (*DefaultRequestAdapter, error) {
				return NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(typedNilAuth, nil, nil, nil)
			},
		},
		{
			name: "NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClientAndObservabilityOptions",
			construct: func() (*DefaultRequestAdapter, error) {
				return NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClientAndObservabilityOptions(typedNilAuth, nil, nil, nil, khttp.ObservabilityOptions{})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adapter, err := tc.construct()
			assert.Error(t, err)
			assert.Nil(t, adapter)
		})
	}
}
