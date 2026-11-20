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
	serjson "github.com/microsoft/kiota-serialization-json-go"
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
	assert.ErrorIs(t, err, ErrNilAuthenticationProvider)
}

func TestAdapterThrowsErrorOnNilAuthProviderAcrossConstructors(t *testing.T) {
	_, err1 := NewDefaultRequestAdapterWithParseNodeFactory(nil, nil)
	assert.Error(t, err1)
	assert.ErrorIs(t, err1, ErrNilAuthenticationProvider)

	_, err2 := NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory(nil, nil, nil)
	assert.Error(t, err2)
	assert.ErrorIs(t, err2, ErrNilAuthenticationProvider)

	_, err3 := NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(nil, nil, nil, nil)
	assert.Error(t, err3)
	assert.ErrorIs(t, err3, ErrNilAuthenticationProvider)

	_, err4 := NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClientAndObservabilityOptions(nil, nil, nil, nil, khttp.ObservabilityOptions{})
	assert.Error(t, err4)
	assert.ErrorIs(t, err4, ErrNilAuthenticationProvider)
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
	assert.ErrorIs(t, err, ErrNilRequestAdapter)

	// Send methods on nil adapter must return an error and not panic
	resp, err := nilAdapter.Send(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
	assert.ErrorIs(t, err, ErrNilRequestAdapter)

	respColl, err := nilAdapter.SendCollection(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respColl)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
	assert.ErrorIs(t, err, ErrNilRequestAdapter)

	respEnum, err := nilAdapter.SendEnum(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respEnum)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
	assert.ErrorIs(t, err, ErrNilRequestAdapter)

	respEnumColl, err := nilAdapter.SendEnumCollection(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respEnumColl)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
	assert.ErrorIs(t, err, ErrNilRequestAdapter)

	respPrim, err := nilAdapter.SendPrimitive(context.Background(), reqInfo, "string", nil)
	assert.Error(t, err)
	assert.Nil(t, respPrim)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
	assert.ErrorIs(t, err, ErrNilRequestAdapter)

	respPrimColl, err := nilAdapter.SendPrimitiveCollection(context.Background(), reqInfo, "string", nil)
	assert.Error(t, err)
	assert.Nil(t, respPrimColl)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
	assert.ErrorIs(t, err, ErrNilRequestAdapter)

	err = nilAdapter.SendNoContent(context.Background(), reqInfo, nil)
	assert.Error(t, err)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
	assert.ErrorIs(t, err, ErrNilRequestAdapter)
}

func TestUninitializedDefaultRequestAdapterBaseUrl(t *testing.T) {
	uninitAdapter := &DefaultRequestAdapter{}

	// GetBaseUrl on uninitialized adapter returns empty string
	assert.Equal(t, "", uninitAdapter.GetBaseUrl())

	// SetBaseUrl on uninitialized adapter is a safe no-op
	assert.NotPanics(t, func() {
		uninitAdapter.SetBaseUrl("https://example.com")
	})

	// BaseUrl remains empty after no-op SetBaseUrl on uninitialized adapter
	assert.Equal(t, "", uninitAdapter.GetBaseUrl())
}

// TestConcurrentResetAndSetupDefaults validates that concurrent invocations of ResetDefaults and SetupDefaults
// are serialized via setupDefaultsLock and do not race with each other or corrupt initialization state.
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

func TestUninitializedDefaultRequestAdapterEnableBackingStore(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	uninitAdapter := &DefaultRequestAdapter{}

	customCalled := false
	customFactory := func() store.BackingStore {
		customCalled = true
		return store.NewInMemoryBackingStore()
	}

	// Calling EnableBackingStore on uninitialized adapter must be a safe no-op
	// and must not mutate store.BackingStoreFactoryInstance.
	assert.NotPanics(t, func() {
		uninitAdapter.EnableBackingStore(customFactory)
	})

	assert.Nil(t, uninitAdapter.GetParseNodeFactory())
	assert.Nil(t, uninitAdapter.GetSerializationWriterFactory())

	// store.BackingStoreFactoryInstance must remain default and customFactory must not be invoked
	bs := store.BackingStoreFactoryInstance()
	assert.NotNil(t, bs)
	assert.False(t, customCalled)
	_, isInMemory := bs.(*store.InMemoryBackingStore)
	assert.True(t, isInMemory)

	// Nil factory input on uninitialized adapter must also not panic
	assert.NotPanics(t, func() {
		uninitAdapter.EnableBackingStore(nil)
	})
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
	assert.ErrorIs(t, err, ErrNilRequestAdapter)

	// ConvertToNativeRequest on uninitialized adapter returns error and does not panic
	uninitAdapter := &DefaultRequestAdapter{}
	uninitConverted, err := uninitAdapter.ConvertToNativeRequest(context.Background(), reqInfo)
	assert.Error(t, err)
	assert.Nil(t, uninitConverted)
	assert.Equal(t, "request adapter is not initialized", err.Error())
	assert.ErrorIs(t, err, ErrUninitializedRequestAdapter)
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
	assert.ErrorIs(t, err, ErrNilRequestAdapter)

	respColl, err := nilAdapter.SendCollection(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respColl)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
	assert.ErrorIs(t, err, ErrNilRequestAdapter)

	respEnum, err := nilAdapter.SendEnum(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respEnum)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
	assert.ErrorIs(t, err, ErrNilRequestAdapter)

	respEnumColl, err := nilAdapter.SendEnumCollection(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respEnumColl)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
	assert.ErrorIs(t, err, ErrNilRequestAdapter)

	respPrim, err := nilAdapter.SendPrimitive(context.Background(), reqInfo, "string", nil)
	assert.Error(t, err)
	assert.Nil(t, respPrim)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
	assert.ErrorIs(t, err, ErrNilRequestAdapter)

	respPrimColl, err := nilAdapter.SendPrimitiveCollection(context.Background(), reqInfo, "string", nil)
	assert.Error(t, err)
	assert.Nil(t, respPrimColl)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
	assert.ErrorIs(t, err, ErrNilRequestAdapter)

	err = nilAdapter.SendNoContent(context.Background(), reqInfo, nil)
	assert.Error(t, err)
	assert.Equal(t, "request adapter cannot be nil", err.Error())
	assert.ErrorIs(t, err, ErrNilRequestAdapter)
}

func TestUninitializedDefaultRequestAdapterSendMethods(t *testing.T) {
	uninitAdapter := &DefaultRequestAdapter{}
	reqInfo := abs.NewRequestInformation()
	parsedUrl, err := url.Parse("https://graph.microsoft.com/v1.0/me")
	assert.NoError(t, err)
	reqInfo.SetUri(*parsedUrl)
	reqInfo.Method = abs.GET

	resp, err := uninitAdapter.Send(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "request adapter is not initialized", err.Error())
	assert.ErrorIs(t, err, ErrUninitializedRequestAdapter)

	respColl, err := uninitAdapter.SendCollection(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respColl)
	assert.Equal(t, "request adapter is not initialized", err.Error())
	assert.ErrorIs(t, err, ErrUninitializedRequestAdapter)

	respEnum, err := uninitAdapter.SendEnum(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respEnum)
	assert.Equal(t, "request adapter is not initialized", err.Error())
	assert.ErrorIs(t, err, ErrUninitializedRequestAdapter)

	respEnumColl, err := uninitAdapter.SendEnumCollection(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, respEnumColl)
	assert.Equal(t, "request adapter is not initialized", err.Error())
	assert.ErrorIs(t, err, ErrUninitializedRequestAdapter)

	respPrim, err := uninitAdapter.SendPrimitive(context.Background(), reqInfo, "string", nil)
	assert.Error(t, err)
	assert.Nil(t, respPrim)
	assert.Equal(t, "request adapter is not initialized", err.Error())
	assert.ErrorIs(t, err, ErrUninitializedRequestAdapter)

	respPrimColl, err := uninitAdapter.SendPrimitiveCollection(context.Background(), reqInfo, "string", nil)
	assert.Error(t, err)
	assert.Nil(t, respPrimColl)
	assert.Equal(t, "request adapter is not initialized", err.Error())
	assert.ErrorIs(t, err, ErrUninitializedRequestAdapter)

	err = uninitAdapter.SendNoContent(context.Background(), reqInfo, nil)
	assert.Error(t, err)
	assert.Equal(t, "request adapter is not initialized", err.Error())
	assert.ErrorIs(t, err, ErrUninitializedRequestAdapter)
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
			assert.ErrorIs(t, err, ErrNilAuthenticationProvider)
		})
	}
}

func TestUninitializedDefaultRequestAdapterWithTypedNilFactories(t *testing.T) {
	uninitAdapter := &DefaultRequestAdapter{}
	assert.True(t, uninitAdapter.isUninitialized())

	reqInfo := abs.NewRequestInformation()
	parsedUrl, err := url.Parse("https://graph.microsoft.com/v1.0/me")
	assert.NoError(t, err)
	reqInfo.SetUri(*parsedUrl)
	reqInfo.Method = abs.GET

	resp, err := uninitAdapter.Send(context.Background(), reqInfo, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrUninitializedRequestAdapter)

	err = uninitAdapter.SendNoContent(context.Background(), reqInfo, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUninitializedRequestAdapter)
}

func TestAdapterFallsBackToDefaultParseNodeFactoryOnTypedNil(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	authProvider := &absauth.AnonymousAuthenticationProvider{}
	var typedNilParseFactory *serjson.JsonParseNodeFactory

	adapter, err := NewDefaultRequestAdapterWithParseNodeFactory(authProvider, typedNilParseFactory)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.Equal(t, absser.DefaultParseNodeFactoryInstance, adapter.GetParseNodeFactory())

	rootNode, err := adapter.GetParseNodeFactory().GetRootParseNode("application/json", []byte(`{"key":"value"}`))
	assert.NoError(t, err)
	assert.NotNil(t, rootNode)
}

func TestAdapterFallsBackToDefaultSerializationWriterFactoryOnTypedNil(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	authProvider := &absauth.AnonymousAuthenticationProvider{}
	var typedNilWriterFactory *serjson.JsonSerializationWriterFactory

	adapter, err := NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory(authProvider, nil, typedNilWriterFactory)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.Equal(t, absser.DefaultSerializationWriterFactoryInstance, adapter.GetSerializationWriterFactory())

	writer, err := adapter.GetSerializationWriterFactory().GetSerializationWriter("application/json")
	assert.NoError(t, err)
	assert.NotNil(t, writer)
}

func TestSetupDefaultsWithNilWriterRegistryMap(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	absser.DefaultSerializationWriterFactoryInstance = &absser.SerializationWriterFactoryRegistry{}
	assert.NotPanics(t, func() {
		SetupDefaults()
	})
	assert.NotNil(t, absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories)

	writer, err := absser.DefaultSerializationWriterFactoryInstance.GetSerializationWriter("application/json")
	assert.NoError(t, err)
	assert.NotNil(t, writer)
}

func TestSetupDefaultsWithNilParseRegistryMap(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	absser.DefaultParseNodeFactoryInstance = &absser.ParseNodeFactoryRegistry{}
	assert.NotPanics(t, func() {
		SetupDefaults()
	})
	assert.NotNil(t, absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories)

	rootNode, err := absser.DefaultParseNodeFactoryInstance.GetRootParseNode("application/json", []byte(`{"status":"ok"}`))
	assert.NoError(t, err)
	assert.NotNil(t, rootNode)
}

func TestSetupDefaultsReinitializesNilMapWithoutResetDefaults(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	// Run initial SetupDefaults so defaultsInitialized is true.
	SetupDefaults()

	customWriter := absser.NewSerializationWriterFactoryRegistry()
	absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["application/x-custom"] = customWriter
	absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories = nil

	assert.NotPanics(t, func() {
		SetupDefaults()
	})

	// Unaffected writer registry keeps custom registrations.
	assert.Contains(t, absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories, "application/x-custom")
	// Nil parser map is re-initialized and re-registered with default deserializers.
	assert.NotNil(t, absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories)
	assert.Contains(t, absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories, "application/json")
}

func TestSetupDefaultsRepopulatesEmptyWriterRegistryMap(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	// Run initial SetupDefaults so defaultsInitialized is true.
	SetupDefaults()

	// Clear writer registry to an empty (non-nil) map without calling ResetDefaults.
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories = make(map[string]absser.SerializationWriterFactory)
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	assert.NotPanics(t, func() {
		SetupDefaults()
	})

	absser.DefaultSerializationWriterFactoryInstance.Lock()
	serializerMap := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories
	assert.NotEmpty(t, serializerMap)
	assert.Contains(t, serializerMap, "application/json")
	assert.Contains(t, serializerMap, "text/plain")
	assert.Contains(t, serializerMap, "application/x-www-form-urlencoded")
	assert.Contains(t, serializerMap, "multipart/form-data")
	absser.DefaultSerializationWriterFactoryInstance.Unlock()
}

func TestSetupDefaultsRepopulatesEmptyParseNodeRegistryMap(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	// Run initial SetupDefaults so defaultsInitialized is true.
	SetupDefaults()

	// Clear parse node registry to an empty (non-nil) map without calling ResetDefaults.
	absser.DefaultParseNodeFactoryInstance.Lock()
	absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories = make(map[string]absser.ParseNodeFactory)
	absser.DefaultParseNodeFactoryInstance.Unlock()

	assert.NotPanics(t, func() {
		SetupDefaults()
	})

	absser.DefaultParseNodeFactoryInstance.Lock()
	deserializerMap := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories
	assert.NotEmpty(t, deserializerMap)
	assert.Contains(t, deserializerMap, "application/json")
	assert.Contains(t, deserializerMap, "text/plain")
	assert.Contains(t, deserializerMap, "application/x-www-form-urlencoded")
	absser.DefaultParseNodeFactoryInstance.Unlock()
}

func TestSetupDefaultsRestoresMissingDefaultSerializersWhenPartiallyPopulated(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	// Initial setup
	SetupDefaults()

	// Retain only a custom serializer and remove default serializers
	customContentType := "application/x-custom-serializer"
	customWriter := absser.NewSerializationWriterFactoryRegistry()

	absser.DefaultSerializationWriterFactoryInstance.Lock()
	absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories = map[string]absser.SerializationWriterFactory{
		customContentType: customWriter,
	}
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	// Calling SetupDefaults without ResetDefaults must restore all missing defaults while preserving custom registrations
	SetupDefaults()

	absser.DefaultSerializationWriterFactoryInstance.Lock()
	serializerMap := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories
	assert.Contains(t, serializerMap, customContentType)
	assert.Contains(t, serializerMap, "application/json")
	assert.Contains(t, serializerMap, "text/plain")
	assert.Contains(t, serializerMap, "application/x-www-form-urlencoded")
	assert.Contains(t, serializerMap, "multipart/form-data")
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	// Verify one deleted default serializer is also restored
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	delete(absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories, "application/json")
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	SetupDefaults()

	absser.DefaultSerializationWriterFactoryInstance.Lock()
	assert.Contains(t, absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories, "application/json")
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	// Verify that overriding a default content type (e.g. application/json) with a custom factory
	// and removing a different default (e.g. text/plain) preserves the exact custom factory when healing.
	customJsonWriter := absser.NewSerializationWriterFactoryRegistry()
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["application/json"] = customJsonWriter
	delete(absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories, "text/plain")
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	SetupDefaults()

	absser.DefaultSerializationWriterFactoryInstance.Lock()
	assert.Same(t, customJsonWriter, absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["application/json"])
	assert.NotNil(t, absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["text/plain"])
	absser.DefaultSerializationWriterFactoryInstance.Unlock()
}

func TestSetupDefaultsRestoresMissingDefaultDeserializersWhenPartiallyPopulated(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	// Initial setup
	SetupDefaults()

	// Retain only a custom deserializer and remove default deserializers
	customContentType := "application/x-custom-deserializer"
	customParser := absser.NewParseNodeFactoryRegistry()

	absser.DefaultParseNodeFactoryInstance.Lock()
	absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories = map[string]absser.ParseNodeFactory{
		customContentType: customParser,
	}
	absser.DefaultParseNodeFactoryInstance.Unlock()

	// Calling SetupDefaults without ResetDefaults must restore all missing defaults while preserving custom registrations
	SetupDefaults()

	absser.DefaultParseNodeFactoryInstance.Lock()
	deserializerMap := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories
	assert.Contains(t, deserializerMap, customContentType)
	assert.Contains(t, deserializerMap, "application/json")
	assert.Contains(t, deserializerMap, "text/plain")
	assert.Contains(t, deserializerMap, "application/x-www-form-urlencoded")
	absser.DefaultParseNodeFactoryInstance.Unlock()

	// Verify one deleted default deserializer is also restored
	absser.DefaultParseNodeFactoryInstance.Lock()
	delete(absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories, "text/plain")
	absser.DefaultParseNodeFactoryInstance.Unlock()

	SetupDefaults()

	absser.DefaultParseNodeFactoryInstance.Lock()
	assert.Contains(t, absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories, "text/plain")
	absser.DefaultParseNodeFactoryInstance.Unlock()

	// Verify that overriding a default content type (e.g. application/json) with a custom factory
	// and removing a different default (e.g. text/plain) preserves the exact custom factory when healing.
	customJsonParser := absser.NewParseNodeFactoryRegistry()
	absser.DefaultParseNodeFactoryInstance.Lock()
	absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["application/json"] = customJsonParser
	delete(absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories, "text/plain")
	absser.DefaultParseNodeFactoryInstance.Unlock()

	SetupDefaults()

	absser.DefaultParseNodeFactoryInstance.Lock()
	assert.Same(t, customJsonParser, absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["application/json"])
	assert.NotNil(t, absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["text/plain"])
	absser.DefaultParseNodeFactoryInstance.Unlock()
}

func TestSetupDefaultsCleansNilEntriesAndRestoresDefaults(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	SetupDefaults()

	// Set nil and typed-nil factories for default content types while preserving a custom override on another default.
	customTextWriter := absser.NewSerializationWriterFactoryRegistry()
	customTextParser := absser.NewParseNodeFactoryRegistry()
	var typedNilWriter *serjson.JsonSerializationWriterFactory
	var typedNilParser *serjson.JsonParseNodeFactory

	absser.DefaultSerializationWriterFactoryInstance.Lock()
	absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["application/json"] = nil
	absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["application/x-www-form-urlencoded"] = typedNilWriter
	absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["text/plain"] = customTextWriter
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	absser.DefaultParseNodeFactoryInstance.Lock()
	absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["application/json"] = nil
	absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["application/x-www-form-urlencoded"] = typedNilParser
	absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["text/plain"] = customTextParser
	absser.DefaultParseNodeFactoryInstance.Unlock()

	// SetupDefaults must clean nil and typed-nil entries, restore valid defaults, and preserve custom overrides.
	assert.NotPanics(t, func() {
		SetupDefaults()
	})

	absser.DefaultSerializationWriterFactoryInstance.Lock()
	writerFactory := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["application/json"]
	formWriterFactory := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["application/x-www-form-urlencoded"]
	assert.False(t, isNilValue(writerFactory))
	assert.False(t, isNilValue(formWriterFactory))
	assert.Same(t, customTextWriter, absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["text/plain"])
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	absser.DefaultParseNodeFactoryInstance.Lock()
	parseFactory := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["application/json"]
	formParseFactory := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["application/x-www-form-urlencoded"]
	assert.False(t, isNilValue(parseFactory))
	assert.False(t, isNilValue(formParseFactory))
	assert.Same(t, customTextParser, absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["text/plain"])
	absser.DefaultParseNodeFactoryInstance.Unlock()

	writer, err := absser.DefaultSerializationWriterFactoryInstance.GetSerializationWriter("application/json")
	assert.NoError(t, err)
	assert.NotNil(t, writer)

	rootNode, err := absser.DefaultParseNodeFactoryInstance.GetRootParseNode("application/json", []byte(`{"status":"clean"}`))
	assert.NoError(t, err)
	assert.NotNil(t, rootNode)
}

func TestNewDefaultRequestAdapterDoesNotMutateActiveRegistriesDuringConcurrentLookups(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	authProvider := &absauth.AnonymousAuthenticationProvider{}
	existingAdapter, err := NewDefaultRequestAdapter(authProvider)
	assert.NoError(t, err)
	assert.NotNil(t, existingAdapter)

	// Introduce a nil entry and remove an unused default during a quiescent state before concurrent lookups begin.
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["application/x-nil-writer"] = nil
	delete(absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories, "text/plain")
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	absser.DefaultParseNodeFactoryInstance.Lock()
	absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["application/x-nil-parser"] = nil
	delete(absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories, "text/plain")
	absser.DefaultParseNodeFactoryInstance.Unlock()

	const workers = 12
	const loops = 50
	var wg sync.WaitGroup
	wg.Add(workers * 2)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < loops; j++ {
				w, wErr := existingAdapter.GetSerializationWriterFactory().GetSerializationWriter("application/json")
				assert.NoError(t, wErr)
				assert.NotNil(t, w)

				pn, pErr := existingAdapter.GetParseNodeFactory().GetRootParseNode("application/json", []byte(`{"ok":true}`))
				assert.NoError(t, pErr)
				assert.NotNil(t, pn)
			}
		}()

		go func() {
			defer wg.Done()
			for j := 0; j < loops; j++ {
				adapter, aErr := NewDefaultRequestAdapter(authProvider)
				assert.NoError(t, aErr)
				assert.NotNil(t, adapter)
			}
		}()
	}

	wg.Wait()

	// Creating adapters after initialization must not have mutated active registry maps.
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	_, hasNilWriter := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["application/x-nil-writer"]
	_, hasTextWriter := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["text/plain"]
	absser.DefaultSerializationWriterFactoryInstance.Unlock()
	assert.True(t, hasNilWriter, "adapter creation must not delete nil entries from active writer registry")
	assert.False(t, hasTextWriter, "adapter creation must not write missing defaults into active writer registry")

	absser.DefaultParseNodeFactoryInstance.Lock()
	_, hasNilParser := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["application/x-nil-parser"]
	_, hasTextParser := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["text/plain"]
	absser.DefaultParseNodeFactoryInstance.Unlock()
	assert.True(t, hasNilParser, "adapter creation must not delete nil entries from active parse registry")
	assert.False(t, hasTextParser, "adapter creation must not write missing defaults into active parse registry")

	// If one singleton is nilled, ensureDefaults recreates only that singleton without mutating the other active registry.
	absser.DefaultSerializationWriterFactoryInstance = nil
	adapterAfterNilWriter, err := NewDefaultRequestAdapter(authProvider)
	assert.NoError(t, err)
	assert.NotNil(t, adapterAfterNilWriter)

	absser.DefaultParseNodeFactoryInstance.Lock()
	_, stillHasNilParser := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["application/x-nil-parser"]
	_, stillHasTextParser := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["text/plain"]
	absser.DefaultParseNodeFactoryInstance.Unlock()
	assert.True(t, stillHasNilParser, "recreating nil writer singleton must not mutate active parse registry")
	assert.False(t, stillHasTextParser, "recreating nil writer singleton must not heal active parse registry")

	// Explicit quiescent SetupDefaults cleans nil entries and restores missing defaults.
	SetupDefaults()

	absser.DefaultParseNodeFactoryInstance.Lock()
	_, nilParserAfterHeal := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["application/x-nil-parser"]
	_, textParserAfterHeal := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["text/plain"]
	absser.DefaultParseNodeFactoryInstance.Unlock()
	assert.False(t, nilParserAfterHeal)
	assert.True(t, textParserAfterHeal)
}

func TestFirstNewDefaultRequestAdapterDoesNotMutatePrepopulatedRegistries(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	// Pre-populate shared registries as if another Kiota client had initialized them
	// before any bundle adapter was instantiated in this process.
	SetupDefaults()

	// Introduce an unused nil entry and remove an unused default in both registries.
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["application/x-nil-writer"] = nil
	delete(absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories, "text/plain")
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	absser.DefaultParseNodeFactoryInstance.Lock()
	absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["application/x-nil-parser"] = nil
	delete(absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories, "text/plain")
	absser.DefaultParseNodeFactoryInstance.Unlock()

	// Explicitly simulate defaultsInitialized == false so this is the very first bundle constructor call.
	setupDefaultsLock.Lock()
	defaultsInitialized = false
	setupDefaultsLock.Unlock()

	authProvider := &absauth.AnonymousAuthenticationProvider{}
	adapter, err := NewDefaultRequestAdapter(authProvider)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)

	// The first constructor call must NOT call setupDefaults() or mutate the active registries.
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	_, hasNilWriter := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["application/x-nil-writer"]
	_, hasTextWriter := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["text/plain"]
	absser.DefaultSerializationWriterFactoryInstance.Unlock()
	assert.True(t, hasNilWriter, "first adapter creation must not delete nil entries from prepopulated writer registry")
	assert.False(t, hasTextWriter, "first adapter creation must not write missing defaults into prepopulated writer registry")

	absser.DefaultParseNodeFactoryInstance.Lock()
	_, hasNilParser := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["application/x-nil-parser"]
	_, hasTextParser := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["text/plain"]
	absser.DefaultParseNodeFactoryInstance.Unlock()
	assert.True(t, hasNilParser, "first adapter creation must not delete nil entries from prepopulated parse registry")
	assert.False(t, hasTextParser, "first adapter creation must not write missing defaults into prepopulated parse registry")
}

func TestNewDefaultRequestAdapterWithEmptyOrNilRegistryMaps(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	SetupDefaults()

	authProvider := &absauth.AnonymousAuthenticationProvider{}

	// Case 1: Replace DefaultSerializationWriterFactoryInstance with an empty registry (len == 0).
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories = make(map[string]absser.SerializationWriterFactory)
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	// Replace DefaultParseNodeFactoryInstance with a registry having a nil map.
	absser.DefaultParseNodeFactoryInstance.Lock()
	absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories = nil
	absser.DefaultParseNodeFactoryInstance.Unlock()

	adapter, err := NewDefaultRequestAdapter(authProvider)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)

	// Adapter must have usable default factories, not empty ones.
	writer, wErr := adapter.GetSerializationWriterFactory().GetSerializationWriter("application/json")
	assert.NoError(t, wErr)
	assert.NotNil(t, writer)

	node, pErr := adapter.GetParseNodeFactory().GetRootParseNode("application/json", []byte(`{"status":"ok"}`))
	assert.NoError(t, pErr)
	assert.NotNil(t, node)

	// Case 2: Explicit fallback when singletons are replaced with fresh empty instances.
	absser.DefaultSerializationWriterFactoryInstance = absser.NewSerializationWriterFactoryRegistry()
	absser.DefaultParseNodeFactoryInstance = &absser.ParseNodeFactoryRegistry{}

	adapter2, err2 := NewDefaultRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory(authProvider, nil, nil)
	assert.NoError(t, err2)
	assert.NotNil(t, adapter2)

	writer2, wErr2 := adapter2.GetSerializationWriterFactory().GetSerializationWriter("application/json")
	assert.NoError(t, wErr2)
	assert.NotNil(t, writer2)

	node2, pErr2 := adapter2.GetParseNodeFactory().GetRootParseNode("application/json", []byte(`{"status":"ok"}`))
	assert.NoError(t, pErr2)
	assert.NotNil(t, node2)
}

func TestSetupDefaultsPreservesBackingStoreWrapping(t *testing.T) {
	ResetDefaults()
	t.Cleanup(ResetDefaults)

	SetupDefaults()

	authProvider := &absauth.AnonymousAuthenticationProvider{}
	adapter, err := NewDefaultRequestAdapter(authProvider)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)

	// Enable backing store on the adapter and shared registries.
	adapter.EnableBackingStore(nil)

	// Verify existing factories are wrapped with backing-store proxies.
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	jsonWriter := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["application/json"]
	textWriter := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["text/plain"]
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	_, writerProxy := jsonWriter.(*store.BackingStoreSerializationWriterProxyFactory)
	assert.True(t, writerProxy, "initial serializer should be wrapped with backing store proxy")

	// Delete application/json from both registries to simulate missing defaults after backing store was enabled.
	absser.DefaultSerializationWriterFactoryInstance.Lock()
	delete(absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories, "application/json")
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	absser.DefaultParseNodeFactoryInstance.Lock()
	delete(absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories, "application/json")
	absser.DefaultParseNodeFactoryInstance.Unlock()

	// Calling SetupDefaults to heal registries must restore application/json wrapped with backing store proxy.
	SetupDefaults()

	absser.DefaultSerializationWriterFactoryInstance.Lock()
	restoredWriter := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["application/json"]
	persistedTextWriter := absser.DefaultSerializationWriterFactoryInstance.ContentTypeAssociatedFactories["text/plain"]
	absser.DefaultSerializationWriterFactoryInstance.Unlock()

	_, restoredWriterProxy := restoredWriter.(*store.BackingStoreSerializationWriterProxyFactory)
	assert.True(t, restoredWriterProxy, "restored serializer must preserve backing-store proxy wrapping")
	assert.Same(t, textWriter, persistedTextWriter, "existing wrapped factory must not be double-wrapped")

	absser.DefaultParseNodeFactoryInstance.Lock()
	restoredParser := absser.DefaultParseNodeFactoryInstance.ContentTypeAssociatedFactories["application/json"]
	absser.DefaultParseNodeFactoryInstance.Unlock()

	_, restoredParserProxy := restoredParser.(*store.BackingStoreParseNodeFactory)
	assert.True(t, restoredParserProxy, "restored deserializer must preserve backing-store proxy wrapping")
}
