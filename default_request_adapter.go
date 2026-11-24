package bundle

import (
	"context"
	"errors"
	nethttp "net/http"
	"reflect"
	"regexp"
	"strings"
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

var (
	// ErrNilRequestAdapter is returned when an operation is performed on a nil DefaultRequestAdapter.
	ErrNilRequestAdapter = errors.New("request adapter cannot be nil")
	// ErrUninitializedRequestAdapter is returned when an operation is performed on an uninitialized DefaultRequestAdapter.
	ErrUninitializedRequestAdapter = errors.New("request adapter is not initialized")
	// ErrNilAuthenticationProvider is returned when an authentication provider is nil.
	ErrNilAuthenticationProvider = errors.New("authenticationProvider cannot be nil")
	// ErrFailedToInitializeAdapter is returned when the underlying request adapter fails to initialize.
	ErrFailedToInitializeAdapter = errors.New("failed to initialize request adapter")
)

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
	if isNilValue(authenticationProvider) {
		return nil, ErrNilAuthenticationProvider
	}
	defaultWriterFactory, defaultParseFactory := ensureDefaults()
	if httpClient == nil {
		httpClient = khttp.GetDefaultClient()
	}
	if isNilValue(serializationWriterFactory) {
		backingStoreWriters := isWriterRegistryBackingStoreEnabled(defaultWriterFactory)
		if defaultWriterFactory == nil || needsLocalWriterRegistry(defaultWriterFactory) {
			serializationWriterFactory = &fallbackSerializationWriterFactory{
				primary:  defaultWriterFactory,
				fallback: newSafeDefaultSerializationWriterFactoryRegistry(backingStoreWriters),
			}
		} else {
			serializationWriterFactory = defaultWriterFactory
		}
	}
	if isNilValue(parseNodeFactory) {
		backingStoreParsers := isParseRegistryBackingStoreEnabled(defaultParseFactory)
		if defaultParseFactory == nil || needsLocalParseRegistry(defaultParseFactory) {
			parseNodeFactory = &fallbackParseNodeFactory{
				primary:  defaultParseFactory,
				fallback: newSafeDefaultParseNodeFactoryRegistry(backingStoreParsers),
			}
		} else {
			parseNodeFactory = defaultParseFactory
		}
	}
	defaultAdapter, err := khttp.NewNetHttpRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClientAndObservabilityOptions(authenticationProvider, parseNodeFactory, serializationWriterFactory, httpClient, observabilityOptions)
	if err != nil {
		return nil, err
	}
	if defaultAdapter == nil {
		return nil, ErrFailedToInitializeAdapter
	}
	result := &DefaultRequestAdapter{
		NetHttpRequestAdapter: *defaultAdapter,
	}

	return result, nil
}

// GetBaseUrl returns the base URL for the request adapter or empty string if the receiver is nil or uninitialized.
func (a *DefaultRequestAdapter) GetBaseUrl() string {
	if a == nil || a.isUninitialized() {
		return ""
	}
	return a.NetHttpRequestAdapter.GetBaseUrl()
}

// SetBaseUrl sets the base URL for the request adapter or performs a no-op if the receiver is nil or uninitialized.
func (a *DefaultRequestAdapter) SetBaseUrl(baseUrl string) {
	if a == nil || a.isUninitialized() {
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

// EnableBackingStore enables the backing store proxies for the SerializationWriters and ParseNodes in use or performs a no-op if the receiver is nil or uninitialized.
// When fallback composite factories are in use, it wraps their underlying primary and fallback delegates exactly once, preventing duplicate proxy wrapping.
func (a *DefaultRequestAdapter) EnableBackingStore(factory store.BackingStoreFactory) {
	if a == nil || a.isUninitialized() {
		return
	}
	if factory != nil {
		store.BackingStoreFactoryInstance = factory
	}

	writerFactory := a.GetSerializationWriterFactory()
	parseFactory := a.GetParseNodeFactory()

	fw, isFallbackWriter := writerFactory.(*fallbackSerializationWriterFactory)
	fp, isFallbackParse := parseFactory.(*fallbackParseNodeFactory)

	if isFallbackWriter && fw != nil {
		enableBackingStoreForWriterRegistry(fw.primary)
		enableBackingStoreForWriterRegistry(fw.fallback)
	}
	if isFallbackParse && fp != nil {
		enableBackingStoreForParseRegistry(fp.primary)
		enableBackingStoreForParseRegistry(fp.fallback)
	}

	if !isFallbackWriter && !isFallbackParse {
		a.NetHttpRequestAdapter.EnableBackingStore(factory)
	} else {
		if !isFallbackWriter {
			if reg, ok := writerFactory.(*absser.SerializationWriterFactoryRegistry); ok {
				enableBackingStoreForWriterRegistry(reg)
			}
		}
		if !isFallbackParse {
			if reg, ok := parseFactory.(*absser.ParseNodeFactoryRegistry); ok {
				enableBackingStoreForParseRegistry(reg)
			}
		}
	}
}

func enableBackingStoreForWriterRegistry(registry *absser.SerializationWriterFactoryRegistry) {
	if registry == nil {
		return
	}
	registry.Lock()
	defer registry.Unlock()
	if registry.ContentTypeAssociatedFactories == nil {
		return
	}
	for key, value := range registry.ContentTypeAssociatedFactories {
		if !isNilValue(value) {
			if _, ok := value.(*store.BackingStoreSerializationWriterProxyFactory); !ok {
				registry.ContentTypeAssociatedFactories[key] = store.NewBackingStoreSerializationWriterProxyFactory(value)
			}
		}
	}
}

func enableBackingStoreForParseRegistry(registry *absser.ParseNodeFactoryRegistry) {
	if registry == nil {
		return
	}
	registry.Lock()
	defer registry.Unlock()
	if registry.ContentTypeAssociatedFactories == nil {
		return
	}
	for key, value := range registry.ContentTypeAssociatedFactories {
		if !isNilValue(value) {
			if _, ok := value.(*store.BackingStoreParseNodeFactory); !ok {
				registry.ContentTypeAssociatedFactories[key] = store.NewBackingStoreParseNodeFactory(value)
			}
		}
	}
}

func (a *DefaultRequestAdapter) isUninitialized() bool {
	return isNilValue(a.GetSerializationWriterFactory()) || isNilValue(a.GetParseNodeFactory())
}

func (a *DefaultRequestAdapter) checkReceiver() error {
	if a == nil {
		return ErrNilRequestAdapter
	}
	if a.isUninitialized() {
		return ErrUninitializedRequestAdapter
	}
	return nil
}

// ConvertToNativeRequest converts the given RequestInformation into a native HTTP request or returns an error if the receiver is nil or uninitialized.
func (a *DefaultRequestAdapter) ConvertToNativeRequest(ctx context.Context, requestInfo *abs.RequestInformation) (any, error) {
	if err := a.checkReceiver(); err != nil {
		return nil, err
	}
	return a.NetHttpRequestAdapter.ConvertToNativeRequest(ctx, requestInfo)
}

// Send executes the HTTP request specified by the given RequestInformation and returns the deserialized response model or an error if the receiver is nil or uninitialized.
func (a *DefaultRequestAdapter) Send(ctx context.Context, requestInfo *abs.RequestInformation, constructor absser.ParsableFactory, errorMappings abs.ErrorMappings) (absser.Parsable, error) {
	if err := a.checkReceiver(); err != nil {
		return nil, err
	}
	return a.NetHttpRequestAdapter.Send(ctx, requestInfo, constructor, errorMappings)
}

// SendEnum executes the HTTP request specified by the given RequestInformation and returns the deserialized response model or an error if the receiver is nil or uninitialized.
func (a *DefaultRequestAdapter) SendEnum(ctx context.Context, requestInfo *abs.RequestInformation, parser absser.EnumFactory, errorMappings abs.ErrorMappings) (any, error) {
	if err := a.checkReceiver(); err != nil {
		return nil, err
	}
	return a.NetHttpRequestAdapter.SendEnum(ctx, requestInfo, parser, errorMappings)
}

// SendCollection executes the HTTP request specified by the given RequestInformation and returns the deserialized response model collection or an error if the receiver is nil or uninitialized.
func (a *DefaultRequestAdapter) SendCollection(ctx context.Context, requestInfo *abs.RequestInformation, constructor absser.ParsableFactory, errorMappings abs.ErrorMappings) ([]absser.Parsable, error) {
	if err := a.checkReceiver(); err != nil {
		return nil, err
	}
	return a.NetHttpRequestAdapter.SendCollection(ctx, requestInfo, constructor, errorMappings)
}

// SendEnumCollection executes the HTTP request specified by the given RequestInformation and returns the deserialized response model collection or an error if the receiver is nil or uninitialized.
func (a *DefaultRequestAdapter) SendEnumCollection(ctx context.Context, requestInfo *abs.RequestInformation, parser absser.EnumFactory, errorMappings abs.ErrorMappings) ([]any, error) {
	if err := a.checkReceiver(); err != nil {
		return nil, err
	}
	return a.NetHttpRequestAdapter.SendEnumCollection(ctx, requestInfo, parser, errorMappings)
}

// SendPrimitive executes the HTTP request specified by the given RequestInformation and returns the deserialized primitive response model or an error if the receiver is nil or uninitialized.
func (a *DefaultRequestAdapter) SendPrimitive(ctx context.Context, requestInfo *abs.RequestInformation, typeName string, errorMappings abs.ErrorMappings) (any, error) {
	if err := a.checkReceiver(); err != nil {
		return nil, err
	}
	return a.NetHttpRequestAdapter.SendPrimitive(ctx, requestInfo, typeName, errorMappings)
}

// SendPrimitiveCollection executes the HTTP request specified by the given RequestInformation and returns the deserialized primitive response model collection or an error if the receiver is nil or uninitialized.
func (a *DefaultRequestAdapter) SendPrimitiveCollection(ctx context.Context, requestInfo *abs.RequestInformation, typeName string, errorMappings abs.ErrorMappings) ([]any, error) {
	if err := a.checkReceiver(); err != nil {
		return nil, err
	}
	return a.NetHttpRequestAdapter.SendPrimitiveCollection(ctx, requestInfo, typeName, errorMappings)
}

// SendNoContent executes the HTTP request specified by the given RequestInformation with no return content or returns an error if the receiver is nil or uninitialized.
func (a *DefaultRequestAdapter) SendNoContent(ctx context.Context, requestInfo *abs.RequestInformation, errorMappings abs.ErrorMappings) error {
	if err := a.checkReceiver(); err != nil {
		return err
	}
	return a.NetHttpRequestAdapter.SendNoContent(ctx, requestInfo, errorMappings)
}

var (
	setupDefaultsLock   sync.Mutex
	defaultsInitialized bool
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

	defaultsInitialized = false
	store.BackingStoreFactoryInstance = store.NewInMemoryBackingStore

	prepareWriterRegistry(true)
	prepareParseRegistry(true)
}

var defaultSerializerContentTypes = [...]string{
	"application/json",
	"text/plain",
	"application/x-www-form-urlencoded",
	"multipart/form-data",
}

var defaultDeserializerContentTypes = [...]string{
	"application/json",
	"text/plain",
	"application/x-www-form-urlencoded",
}

func cleanNilEntries[F any](factories map[string]F) {
	for k, v := range factories {
		if isNilValue(v) {
			delete(factories, k)
		}
	}
}

func hasMissingSerializerDefaults(factories map[string]absser.SerializationWriterFactory) bool {
	for _, ct := range defaultSerializerContentTypes {
		if f, ok := factories[ct]; !ok || isNilValue(f) {
			return true
		}
	}
	return false
}

func hasMissingDeserializerDefaults(factories map[string]absser.ParseNodeFactory) bool {
	for _, ct := range defaultDeserializerContentTypes {
		if f, ok := factories[ct]; !ok || isNilValue(f) {
			return true
		}
	}
	return false
}

func prepareWriterRegistry(clearExisting bool) bool {
	writerReg := absser.DefaultSerializationWriterFactoryInstance
	return configureRegistry(
		writerReg,
		writerReg == nil,
		clearExisting,
		func() {
			absser.DefaultSerializationWriterFactoryInstance = absser.NewSerializationWriterFactoryRegistry()
		},
		func(r *absser.SerializationWriterFactoryRegistry) map[string]absser.SerializationWriterFactory {
			return r.ContentTypeAssociatedFactories
		},
		func(r *absser.SerializationWriterFactoryRegistry, m map[string]absser.SerializationWriterFactory) {
			r.ContentTypeAssociatedFactories = m
		},
		hasMissingSerializerDefaults,
	)
}

func prepareParseRegistry(clearExisting bool) bool {
	parseReg := absser.DefaultParseNodeFactoryInstance
	return configureRegistry(
		parseReg,
		parseReg == nil,
		clearExisting,
		func() { absser.DefaultParseNodeFactoryInstance = absser.NewParseNodeFactoryRegistry() },
		func(r *absser.ParseNodeFactoryRegistry) map[string]absser.ParseNodeFactory {
			return r.ContentTypeAssociatedFactories
		},
		func(r *absser.ParseNodeFactoryRegistry, m map[string]absser.ParseNodeFactory) {
			r.ContentTypeAssociatedFactories = m
		},
		hasMissingDeserializerDefaults,
	)
}

func configureRegistry[R sync.Locker, F any](
	reg R,
	isNilReg bool,
	clearExisting bool,
	initFn func(),
	getMap func(R) map[string]F,
	setMap func(R, map[string]F),
	hasMissing func(map[string]F) bool,
) bool {
	if isNilReg {
		initFn()
		return true
	}
	reg.Lock()
	defer reg.Unlock()

	factories := getMap(reg)
	if factories == nil {
		setMap(reg, make(map[string]F))
		return true
	}
	if clearExisting {
		for k := range factories {
			delete(factories, k)
		}
		return false
	}
	cleanNilEntries(factories)
	return len(factories) == 0 || hasMissing(factories)
}

// SetupDefaults registers the default serializers and deserializers in the global registries,
// and restores missing or nil default entries if registries have been partially cleared.
// It ensures default serializer and deserializer registry singletons are initialized if nil before registration.
// It synchronizes registration cycles via a package-level mutex.
//
// Note: Because the underlying registry lookups in kiota-abstractions-go (GetSerializationWriter and GetRootParseNode)
// read from their factory maps without locking, callers must ensure that registry configuration and healing via
// SetupDefaults occurs during a quiescent initialization phase (such as application startup or test suite setup)
// when no concurrent request serialization or response parsing is active.
func SetupDefaults() {
	setupDefaultsLock.Lock()
	defer setupDefaultsLock.Unlock()
	setupDefaults()
	defaultsInitialized = true
}

func isWriterRegistryBackingStoreEnabled(reg *absser.SerializationWriterFactoryRegistry) bool {
	if reg == nil {
		return false
	}
	reg.Lock()
	defer reg.Unlock()
	for _, f := range reg.ContentTypeAssociatedFactories {
		if p, ok := f.(*store.BackingStoreSerializationWriterProxyFactory); ok && p != nil {
			return true
		}
	}
	return false
}

func isParseRegistryBackingStoreEnabled(reg *absser.ParseNodeFactoryRegistry) bool {
	if reg == nil {
		return false
	}
	reg.Lock()
	defer reg.Unlock()
	for _, f := range reg.ContentTypeAssociatedFactories {
		if p, ok := f.(*store.BackingStoreParseNodeFactory); ok && p != nil {
			return true
		}
	}
	return false
}

func needsLocalWriterRegistry(reg *absser.SerializationWriterFactoryRegistry) bool {
	if reg == nil {
		return true
	}
	reg.Lock()
	defer reg.Unlock()
	if reg.ContentTypeAssociatedFactories == nil || len(reg.ContentTypeAssociatedFactories) == 0 {
		return true
	}
	for _, ct := range defaultSerializerContentTypes {
		if f, ok := reg.ContentTypeAssociatedFactories[ct]; !ok || isNilValue(f) {
			return true
		}
	}
	for _, f := range reg.ContentTypeAssociatedFactories {
		if isNilValue(f) {
			return true
		}
	}
	return false
}

func needsLocalParseRegistry(reg *absser.ParseNodeFactoryRegistry) bool {
	if reg == nil {
		return true
	}
	reg.Lock()
	defer reg.Unlock()
	if reg.ContentTypeAssociatedFactories == nil || len(reg.ContentTypeAssociatedFactories) == 0 {
		return true
	}
	for _, ct := range defaultDeserializerContentTypes {
		if f, ok := reg.ContentTypeAssociatedFactories[ct]; !ok || isNilValue(f) {
			return true
		}
	}
	for _, f := range reg.ContentTypeAssociatedFactories {
		if isNilValue(f) {
			return true
		}
	}
	return false
}

var contentTypeVendorCleanupPattern = regexp.MustCompile(`[^/]+\+`)

func hasValidFactory(reg *absser.SerializationWriterFactoryRegistry, contentType string) bool {
	if reg == nil {
		return false
	}
	trimmed := strings.TrimSpace(contentType)
	if trimmed == "" {
		return false
	}
	vendorSpecificContentType := strings.TrimSpace(strings.Split(trimmed, ";")[0])
	cleanedContentType := contentTypeVendorCleanupPattern.ReplaceAllString(vendorSpecificContentType, "")

	reg.Lock()
	defer reg.Unlock()
	if reg.ContentTypeAssociatedFactories == nil {
		return false
	}

	for _, ct := range []string{
		vendorSpecificContentType,
		strings.ToLower(vendorSpecificContentType),
		cleanedContentType,
		strings.ToLower(cleanedContentType),
	} {
		if fac, ok := reg.ContentTypeAssociatedFactories[ct]; ok {
			return !isNilValue(fac)
		}
	}
	return false
}

func hasValidParseFactory(reg *absser.ParseNodeFactoryRegistry, contentType string) bool {
	if reg == nil {
		return false
	}
	trimmed := strings.TrimSpace(contentType)
	if trimmed == "" {
		return false
	}
	vendorSpecificContentType := strings.TrimSpace(strings.Split(trimmed, ";")[0])
	cleanedContentType := contentTypeVendorCleanupPattern.ReplaceAllString(vendorSpecificContentType, "")

	reg.Lock()
	defer reg.Unlock()
	if reg.ContentTypeAssociatedFactories == nil {
		return false
	}

	for _, ct := range []string{
		vendorSpecificContentType,
		strings.ToLower(vendorSpecificContentType),
		cleanedContentType,
		strings.ToLower(cleanedContentType),
	} {
		if fac, ok := reg.ContentTypeAssociatedFactories[ct]; ok {
			return !isNilValue(fac)
		}
	}
	return false
}

type fallbackSerializationWriterFactory struct {
	primary  *absser.SerializationWriterFactoryRegistry
	fallback *absser.SerializationWriterFactoryRegistry
}

func (f *fallbackSerializationWriterFactory) GetValidContentType() (string, error) {
	return "", errors.New("the registry supports multiple content types. Get the registered factory instead")
}

func (f *fallbackSerializationWriterFactory) GetSerializationWriter(contentType string) (absser.SerializationWriter, error) {
	if f != nil && f.primary != nil && hasValidFactory(f.primary, contentType) {
		return f.primary.GetSerializationWriter(contentType)
	}
	if f != nil && f.fallback != nil {
		return f.fallback.GetSerializationWriter(contentType)
	}
	return nil, errors.New("content type " + contentType + " does not have a factory registered to be parsed")
}

func (f *fallbackSerializationWriterFactory) GetPrimary() *absser.SerializationWriterFactoryRegistry {
	if f == nil {
		return nil
	}
	return f.primary
}

func (f *fallbackSerializationWriterFactory) GetFallback() *absser.SerializationWriterFactoryRegistry {
	if f == nil {
		return nil
	}
	return f.fallback
}

type fallbackParseNodeFactory struct {
	primary  *absser.ParseNodeFactoryRegistry
	fallback *absser.ParseNodeFactoryRegistry
}

func (f *fallbackParseNodeFactory) GetValidContentType() (string, error) {
	return "", errors.New("the registry supports multiple content types. Get the registered factory instead")
}

func (f *fallbackParseNodeFactory) GetRootParseNode(contentType string, content []byte) (absser.ParseNode, error) {
	if f != nil && f.primary != nil && hasValidParseFactory(f.primary, contentType) {
		return f.primary.GetRootParseNode(contentType, content)
	}
	if f != nil && f.fallback != nil {
		return f.fallback.GetRootParseNode(contentType, content)
	}
	return nil, errors.New("content type " + contentType + " does not have a factory registered to be parsed")
}

func (f *fallbackParseNodeFactory) GetPrimary() *absser.ParseNodeFactoryRegistry {
	if f == nil {
		return nil
	}
	return f.primary
}

func (f *fallbackParseNodeFactory) GetFallback() *absser.ParseNodeFactoryRegistry {
	if f == nil {
		return nil
	}
	return f.fallback
}

func newSafeDefaultSerializationWriterFactoryRegistry(backingStore bool) *absser.SerializationWriterFactoryRegistry {
	reg := absser.NewSerializationWriterFactoryRegistry()
	var jsonF absser.SerializationWriterFactory = serjson.NewJsonSerializationWriterFactory()
	var textF absser.SerializationWriterFactory = sertext.NewTextSerializationWriterFactory()
	var formF absser.SerializationWriterFactory = serform.NewFormSerializationWriterFactory()
	var multipartF absser.SerializationWriterFactory = sermultipart.NewMultipartSerializationWriterFactory()
	if backingStore {
		jsonF = store.NewBackingStoreSerializationWriterProxyFactory(jsonF)
		textF = store.NewBackingStoreSerializationWriterProxyFactory(textF)
		formF = store.NewBackingStoreSerializationWriterProxyFactory(formF)
		multipartF = store.NewBackingStoreSerializationWriterProxyFactory(multipartF)
	}
	reg.ContentTypeAssociatedFactories["application/json"] = jsonF
	reg.ContentTypeAssociatedFactories["text/plain"] = textF
	reg.ContentTypeAssociatedFactories["application/x-www-form-urlencoded"] = formF
	reg.ContentTypeAssociatedFactories["multipart/form-data"] = multipartF
	return reg
}

func newSafeDefaultParseNodeFactoryRegistry(backingStore bool) *absser.ParseNodeFactoryRegistry {
	reg := absser.NewParseNodeFactoryRegistry()
	var jsonF absser.ParseNodeFactory = serjson.NewJsonParseNodeFactory()
	var textF absser.ParseNodeFactory = sertext.NewTextParseNodeFactory()
	var formF absser.ParseNodeFactory = serform.NewFormParseNodeFactory()
	if backingStore {
		jsonF = store.NewBackingStoreParseNodeFactory(jsonF)
		textF = store.NewBackingStoreParseNodeFactory(textF)
		formF = store.NewBackingStoreParseNodeFactory(formF)
	}
	reg.ContentTypeAssociatedFactories["application/json"] = jsonF
	reg.ContentTypeAssociatedFactories["text/plain"] = textF
	reg.ContentTypeAssociatedFactories["application/x-www-form-urlencoded"] = formF
	return reg
}

func ensureDefaults() (*absser.SerializationWriterFactoryRegistry, *absser.ParseNodeFactoryRegistry) {
	setupDefaultsLock.Lock()
	defer setupDefaultsLock.Unlock()

	writerReg := absser.DefaultSerializationWriterFactoryInstance
	parseReg := absser.DefaultParseNodeFactoryInstance

	recreatedWriters := false
	if writerReg == nil {
		writerReg = absser.NewSerializationWriterFactoryRegistry()
		absser.DefaultSerializationWriterFactoryInstance = writerReg
		registerDefaultSerializers(false)
		recreatedWriters = true
	}
	recreatedParsers := false
	if parseReg == nil {
		parseReg = absser.NewParseNodeFactoryRegistry()
		absser.DefaultParseNodeFactoryInstance = parseReg
		registerDefaultDeserializers(false)
		recreatedParsers = true
	}

	if !defaultsInitialized {
		if !recreatedWriters {
			writerReg.Lock()
			emptyWriters := writerReg.ContentTypeAssociatedFactories != nil && len(writerReg.ContentTypeAssociatedFactories) == 0
			writerReg.Unlock()
			if emptyWriters {
				registerDefaultSerializers(false)
			}
		}

		if !recreatedParsers {
			parseReg.Lock()
			emptyParsers := parseReg.ContentTypeAssociatedFactories != nil && len(parseReg.ContentTypeAssociatedFactories) == 0
			parseReg.Unlock()
			if emptyParsers {
				registerDefaultDeserializers(false)
			}
		}

		defaultsInitialized = true
	}

	return absser.DefaultSerializationWriterFactoryInstance, absser.DefaultParseNodeFactoryInstance
}

func registerDefaultSerializers(backingStore bool) {
	abs.RegisterDefaultSerializer(func() absser.SerializationWriterFactory {
		var f absser.SerializationWriterFactory = serjson.NewJsonSerializationWriterFactory()
		if backingStore {
			f = store.NewBackingStoreSerializationWriterProxyFactory(f)
		}
		return f
	})
	abs.RegisterDefaultSerializer(func() absser.SerializationWriterFactory {
		var f absser.SerializationWriterFactory = sertext.NewTextSerializationWriterFactory()
		if backingStore {
			f = store.NewBackingStoreSerializationWriterProxyFactory(f)
		}
		return f
	})
	abs.RegisterDefaultSerializer(func() absser.SerializationWriterFactory {
		var f absser.SerializationWriterFactory = serform.NewFormSerializationWriterFactory()
		if backingStore {
			f = store.NewBackingStoreSerializationWriterProxyFactory(f)
		}
		return f
	})
	abs.RegisterDefaultSerializer(func() absser.SerializationWriterFactory {
		var f absser.SerializationWriterFactory = sermultipart.NewMultipartSerializationWriterFactory()
		if backingStore {
			f = store.NewBackingStoreSerializationWriterProxyFactory(f)
		}
		return f
	})
}

func registerDefaultDeserializers(backingStore bool) {
	abs.RegisterDefaultDeserializer(func() absser.ParseNodeFactory {
		var f absser.ParseNodeFactory = serjson.NewJsonParseNodeFactory()
		if backingStore {
			f = store.NewBackingStoreParseNodeFactory(f)
		}
		return f
	})
	abs.RegisterDefaultDeserializer(func() absser.ParseNodeFactory {
		var f absser.ParseNodeFactory = sertext.NewTextParseNodeFactory()
		if backingStore {
			f = store.NewBackingStoreParseNodeFactory(f)
		}
		return f
	})
	abs.RegisterDefaultDeserializer(func() absser.ParseNodeFactory {
		var f absser.ParseNodeFactory = serform.NewFormParseNodeFactory()
		if backingStore {
			f = store.NewBackingStoreParseNodeFactory(f)
		}
		return f
	})
}

func isNilValue(val any) bool {
	if val == nil {
		return true
	}
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func setupDefaults() {
	backingStoreWriters := isWriterRegistryBackingStoreEnabled(absser.DefaultSerializationWriterFactoryInstance)
	backingStoreParsers := isParseRegistryBackingStoreEnabled(absser.DefaultParseNodeFactoryInstance)

	needsWriters := prepareWriterRegistry(false)
	needsParsers := prepareParseRegistry(false)

	if !defaultsInitialized || needsWriters {
		registerDefaultSerializers(backingStoreWriters)
	}
	if !defaultsInitialized || needsParsers {
		registerDefaultDeserializers(backingStoreParsers)
	}
}
