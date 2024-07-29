package bundle

import (
	"testing"

	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	absser "github.com/microsoft/kiota-abstractions-go/serialization"
	"github.com/stretchr/testify/assert"
)

func TestAdapterThrowsErrorOnNillAuthProvider(t *testing.T) {
	_, err := NewDefaultRequestAdapter(nil)
	assert.Error(t, err) // adapter can't be nill
}

func TestAerializersAreRegisteredAsExpected(t *testing.T) {
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
