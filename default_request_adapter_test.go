package bundle

import (
	nethttp "net/http"
	"sync"
	"testing"

	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	absser "github.com/microsoft/kiota-abstractions-go/serialization"
	"github.com/stretchr/testify/assert"
)

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
}
