// Package di provides dependency injection providers for Knowledge Base services.
// Providers are lazy factories that create instances only when first requested.
//
// DESIGN PRINCIPLE: "Freeze interfaces. Fluidly replace implementations."
// New data sources and extractors are added here without touching existing code.
package di

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// =============================================================================
// DATA SOURCE INTERFACES
// =============================================================================

// DataSource is the base interface for all external data sources
type DataSource interface {
	// Name returns the source identifier
	Name() string

	// HealthCheck verifies the source is available
	HealthCheck(ctx context.Context) error

	// Close releases any resources
	Close() error
}

// RxNavClient provides access to NLM RxNav API
type RxNavClient interface {
	DataSource
	GetRxCUIByName(ctx context.Context, drugName string) (string, error)
	GetNDCs(ctx context.Context, rxcui string) ([]string, error)
	GetInteractions(ctx context.Context, rxcui string) ([]DrugInteraction, error)
	GetProperties(ctx context.Context, rxcui string) (*DrugProperties, error)
}

// DailyMedClient provides access to FDA DailyMed SPL documents
type DailyMedClient interface {
	DataSource
	GetSPLByNDC(ctx context.Context, ndc string) (*SPLDocument, error)
	GetSPLBySetID(ctx context.Context, setID string) (*SPLDocument, error)
	GetSPLSection(ctx context.Context, setID, sectionCode string) (string, error)
	ListSPLsForDrug(ctx context.Context, drugName string) ([]SPLMetadata, error)
}

// DrugBankClient provides access to DrugBank API
type DrugBankClient interface {
	DataSource
	GetDrug(ctx context.Context, drugbankID string) (*DrugBankDrug, error)
	GetInteractions(ctx context.Context, drugbankID string) ([]DrugBankInteraction, error)
	SearchDrugs(ctx context.Context, query string) ([]DrugBankDrug, error)
}

// MEDRTClient provides access to NCI MED-RT API
type MEDRTClient interface {
	DataSource
	GetContraindications(ctx context.Context, rxcui string) ([]Contraindication, error)
	GetMayTreat(ctx context.Context, rxcui string) ([]Indication, error)
	GetPhysiologicEffects(ctx context.Context, rxcui string) ([]PhysiologicEffect, error)
}

// CMSDataLoader provides access to CMS public datasets
type CMSDataLoader interface {
	DataSource
	LoadFormulary(ctx context.Context, year int) ([]FormularyEntry, error)
	LoadPricing(ctx context.Context, ndc string) (*DrugPricing, error)
	LoadUtilization(ctx context.Context, ndc string) (*UtilizationData, error)
}

// LLMClient provides access to LLM services for extraction
type LLMClient interface {
	DataSource
	Extract(ctx context.Context, prompt string, content string) (string, error)
	ExtractStructured(ctx context.Context, prompt string, content string, schema interface{}) error
	GetTokenCount(ctx context.Context, text string) (int, error)
}

// =============================================================================
// DATA STRUCTURES (simplified for provider demonstration)
// =============================================================================

type DrugInteraction struct {
	DrugRxCUI     string `json:"drugRxcui"`
	DrugName      string `json:"drugName"`
	Severity      string `json:"severity"`
	Description   string `json:"description"`
}

type DrugProperties struct {
	RxCUI        string   `json:"rxcui"`
	Name         string   `json:"name"`
	GenericName  string   `json:"genericName"`
	BrandNames   []string `json:"brandNames"`
	DrugClass    string   `json:"drugClass"`
	Ingredients  []string `json:"ingredients"`
}

type SPLDocument struct {
	SetID         string            `json:"setId"`
	Version       string            `json:"version"`
	EffectiveTime time.Time         `json:"effectiveTime"`
	Sections      map[string]string `json:"sections"`
}

type SPLMetadata struct {
	SetID         string    `json:"setId"`
	Title         string    `json:"title"`
	EffectiveTime time.Time `json:"effectiveTime"`
}

type DrugBankDrug struct {
	DrugBankID  string   `json:"drugbankId"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Categories  []string `json:"categories"`
}

type DrugBankInteraction struct {
	DrugBankID  string `json:"drugbankId"`
	DrugName    string `json:"drugName"`
	Description string `json:"description"`
}

type Contraindication struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Indication struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type PhysiologicEffect struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type FormularyEntry struct {
	NDC            string  `json:"ndc"`
	DrugName       string  `json:"drugName"`
	TierLevel      int     `json:"tierLevel"`
	PriorAuth      bool    `json:"priorAuth"`
	StepTherapy    bool    `json:"stepTherapy"`
	QuantityLimit  float64 `json:"quantityLimit"`
}

type DrugPricing struct {
	NDC           string    `json:"ndc"`
	AWP           float64   `json:"awp"`
	WAC           float64   `json:"wac"`
	EffectiveDate time.Time `json:"effectiveDate"`
}

type UtilizationData struct {
	NDC              string  `json:"ndc"`
	TotalClaims      int64   `json:"totalClaims"`
	TotalCost        float64 `json:"totalCost"`
	UniquePatients   int64   `json:"uniquePatients"`
}

// =============================================================================
// PROVIDER FACTORY FUNCTIONS
// =============================================================================

// DataSourcesModule provides all data source dependencies
type DataSourcesModule struct {
	Config DataSourcesConfig
	Log    *logrus.Entry
}

// DataSourcesConfig contains configuration for all data sources
type DataSourcesConfig struct {
	// RxNav
	RxNavBaseURL    string
	RxNavTimeout    time.Duration

	// DailyMed
	DailyMedBaseURL string
	DailyMedTimeout time.Duration

	// DrugBank
	DrugBankAPIKey  string
	DrugBankBaseURL string
	DrugBankTimeout time.Duration

	// MED-RT
	MEDRTBaseURL string
	MEDRTTimeout time.Duration

	// CMS
	CMSDataDir     string
	CMSCacheEnabled bool

	// LLM
	LLMProvider    string // "openai", "anthropic", "azure"
	LLMAPIKey      string
	LLMModel       string
	LLMMaxTokens   int
	LLMTimeout     time.Duration

	// HTTP
	HTTPMaxConns   int
	HTTPTimeout    time.Duration
}

func (m *DataSourcesModule) Name() string {
	return "datasources"
}

func (m *DataSourcesModule) Dependencies() []Module {
	return nil // Base module, no dependencies
}

func (m *DataSourcesModule) Register(c *Container) error {
	// Register HTTP client (shared dependency)
	if err := c.RegisterFunc("http-client", TypeOf[*http.Client](), Singleton, m.provideHTTPClient); err != nil {
		return err
	}

	// Register RxNav client
	if err := c.RegisterFunc("rxnav-client", InterfaceType[RxNavClient](), Singleton, m.provideRxNavClient); err != nil {
		return err
	}

	// Register DailyMed client
	if err := c.RegisterFunc("dailymed-client", InterfaceType[DailyMedClient](), Singleton, m.provideDailyMedClient); err != nil {
		return err
	}

	// Register DrugBank client
	if err := c.RegisterFunc("drugbank-client", InterfaceType[DrugBankClient](), Singleton, m.provideDrugBankClient); err != nil {
		return err
	}

	// Register MED-RT client
	if err := c.RegisterFunc("medrt-client", InterfaceType[MEDRTClient](), Singleton, m.provideMEDRTClient); err != nil {
		return err
	}

	// Register CMS data loader
	if err := c.RegisterFunc("cms-loader", InterfaceType[CMSDataLoader](), Singleton, m.provideCMSLoader); err != nil {
		return err
	}

	// Register LLM client
	if err := c.RegisterFunc("llm-client", InterfaceType[LLMClient](), Singleton, m.provideLLMClient); err != nil {
		return err
	}

	return nil
}

// =============================================================================
// PROVIDER IMPLEMENTATIONS
// =============================================================================

func (m *DataSourcesModule) provideHTTPClient(ctx context.Context, c *Container) (interface{}, error) {
	return &http.Client{
		Timeout: m.Config.HTTPTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        m.Config.HTTPMaxConns,
			MaxIdleConnsPerHost: m.Config.HTTPMaxConns / 4,
			IdleConnTimeout:     90 * time.Second,
		},
	}, nil
}

func (m *DataSourcesModule) provideRxNavClient(ctx context.Context, c *Container) (interface{}, error) {
	httpClient, err := ResolveAs[*http.Client](ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve HTTP client: %w", err)
	}

	return NewRxNavClient(RxNavConfig{
		BaseURL:    m.Config.RxNavBaseURL,
		Timeout:    m.Config.RxNavTimeout,
		HTTPClient: httpClient,
		Log:        m.Log.WithField("datasource", "rxnav"),
	}), nil
}

func (m *DataSourcesModule) provideDailyMedClient(ctx context.Context, c *Container) (interface{}, error) {
	httpClient, err := ResolveAs[*http.Client](ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve HTTP client: %w", err)
	}

	return NewDailyMedClient(DailyMedConfig{
		BaseURL:    m.Config.DailyMedBaseURL,
		Timeout:    m.Config.DailyMedTimeout,
		HTTPClient: httpClient,
		Log:        m.Log.WithField("datasource", "dailymed"),
	}), nil
}

func (m *DataSourcesModule) provideDrugBankClient(ctx context.Context, c *Container) (interface{}, error) {
	httpClient, err := ResolveAs[*http.Client](ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve HTTP client: %w", err)
	}

	return NewDrugBankClient(DrugBankConfig{
		BaseURL:    m.Config.DrugBankBaseURL,
		APIKey:     m.Config.DrugBankAPIKey,
		Timeout:    m.Config.DrugBankTimeout,
		HTTPClient: httpClient,
		Log:        m.Log.WithField("datasource", "drugbank"),
	}), nil
}

func (m *DataSourcesModule) provideMEDRTClient(ctx context.Context, c *Container) (interface{}, error) {
	httpClient, err := ResolveAs[*http.Client](ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve HTTP client: %w", err)
	}

	return NewMEDRTClient(MEDRTConfig{
		BaseURL:    m.Config.MEDRTBaseURL,
		Timeout:    m.Config.MEDRTTimeout,
		HTTPClient: httpClient,
		Log:        m.Log.WithField("datasource", "medrt"),
	}), nil
}

func (m *DataSourcesModule) provideCMSLoader(ctx context.Context, c *Container) (interface{}, error) {
	return NewCMSDataLoader(CMSConfig{
		DataDir:      m.Config.CMSDataDir,
		CacheEnabled: m.Config.CMSCacheEnabled,
		Log:          m.Log.WithField("datasource", "cms"),
	}), nil
}

func (m *DataSourcesModule) provideLLMClient(ctx context.Context, c *Container) (interface{}, error) {
	return NewLLMClient(LLMConfig{
		Provider:  m.Config.LLMProvider,
		APIKey:    m.Config.LLMAPIKey,
		Model:     m.Config.LLMModel,
		MaxTokens: m.Config.LLMMaxTokens,
		Timeout:   m.Config.LLMTimeout,
		Log:       m.Log.WithField("datasource", "llm"),
	}), nil
}

// =============================================================================
// CLIENT IMPLEMENTATIONS (Stubs - actual implementations in datasources/*)
// =============================================================================

// RxNavConfig for RxNav client
type RxNavConfig struct {
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
	Log        *logrus.Entry
}

// NewRxNavClient creates a new RxNav client
func NewRxNavClient(cfg RxNavConfig) RxNavClient {
	return &rxNavClientImpl{cfg: cfg}
}

type rxNavClientImpl struct {
	cfg RxNavConfig
}

func (c *rxNavClientImpl) Name() string                       { return "rxnav" }
func (c *rxNavClientImpl) Close() error                       { return nil }
func (c *rxNavClientImpl) HealthCheck(ctx context.Context) error { return nil }
func (c *rxNavClientImpl) GetRxCUIByName(ctx context.Context, drugName string) (string, error) {
	return "", fmt.Errorf("not implemented - see datasources/rxnav/client.go")
}
func (c *rxNavClientImpl) GetNDCs(ctx context.Context, rxcui string) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *rxNavClientImpl) GetInteractions(ctx context.Context, rxcui string) ([]DrugInteraction, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *rxNavClientImpl) GetProperties(ctx context.Context, rxcui string) (*DrugProperties, error) {
	return nil, fmt.Errorf("not implemented")
}

// DailyMedConfig for DailyMed client
type DailyMedConfig struct {
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
	Log        *logrus.Entry
}

// NewDailyMedClient creates a new DailyMed client
func NewDailyMedClient(cfg DailyMedConfig) DailyMedClient {
	return &dailyMedClientImpl{cfg: cfg}
}

type dailyMedClientImpl struct {
	cfg DailyMedConfig
}

func (c *dailyMedClientImpl) Name() string                       { return "dailymed" }
func (c *dailyMedClientImpl) Close() error                       { return nil }
func (c *dailyMedClientImpl) HealthCheck(ctx context.Context) error { return nil }
func (c *dailyMedClientImpl) GetSPLByNDC(ctx context.Context, ndc string) (*SPLDocument, error) {
	return nil, fmt.Errorf("not implemented - see datasources/dailymed/client.go")
}
func (c *dailyMedClientImpl) GetSPLBySetID(ctx context.Context, setID string) (*SPLDocument, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *dailyMedClientImpl) GetSPLSection(ctx context.Context, setID, sectionCode string) (string, error) {
	return "", fmt.Errorf("not implemented")
}
func (c *dailyMedClientImpl) ListSPLsForDrug(ctx context.Context, drugName string) ([]SPLMetadata, error) {
	return nil, fmt.Errorf("not implemented")
}

// DrugBankConfig for DrugBank client
type DrugBankConfig struct {
	BaseURL    string
	APIKey     string
	Timeout    time.Duration
	HTTPClient *http.Client
	Log        *logrus.Entry
}

// NewDrugBankClient creates a new DrugBank client
func NewDrugBankClient(cfg DrugBankConfig) DrugBankClient {
	return &drugBankClientImpl{cfg: cfg}
}

type drugBankClientImpl struct {
	cfg DrugBankConfig
}

func (c *drugBankClientImpl) Name() string                       { return "drugbank" }
func (c *drugBankClientImpl) Close() error                       { return nil }
func (c *drugBankClientImpl) HealthCheck(ctx context.Context) error { return nil }
func (c *drugBankClientImpl) GetDrug(ctx context.Context, drugbankID string) (*DrugBankDrug, error) {
	return nil, fmt.Errorf("not implemented - see datasources/drugbank/client.go")
}
func (c *drugBankClientImpl) GetInteractions(ctx context.Context, drugbankID string) ([]DrugBankInteraction, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *drugBankClientImpl) SearchDrugs(ctx context.Context, query string) ([]DrugBankDrug, error) {
	return nil, fmt.Errorf("not implemented")
}

// MEDRTConfig for MED-RT client
type MEDRTConfig struct {
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
	Log        *logrus.Entry
}

// NewMEDRTClient creates a new MED-RT client
func NewMEDRTClient(cfg MEDRTConfig) MEDRTClient {
	return &medrtClientImpl{cfg: cfg}
}

type medrtClientImpl struct {
	cfg MEDRTConfig
}

func (c *medrtClientImpl) Name() string                       { return "medrt" }
func (c *medrtClientImpl) Close() error                       { return nil }
func (c *medrtClientImpl) HealthCheck(ctx context.Context) error { return nil }
func (c *medrtClientImpl) GetContraindications(ctx context.Context, rxcui string) ([]Contraindication, error) {
	return nil, fmt.Errorf("not implemented - see datasources/medrt/client.go")
}
func (c *medrtClientImpl) GetMayTreat(ctx context.Context, rxcui string) ([]Indication, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *medrtClientImpl) GetPhysiologicEffects(ctx context.Context, rxcui string) ([]PhysiologicEffect, error) {
	return nil, fmt.Errorf("not implemented")
}

// CMSConfig for CMS data loader
type CMSConfig struct {
	DataDir      string
	CacheEnabled bool
	Log          *logrus.Entry
}

// NewCMSDataLoader creates a new CMS data loader
func NewCMSDataLoader(cfg CMSConfig) CMSDataLoader {
	return &cmsDataLoaderImpl{cfg: cfg}
}

type cmsDataLoaderImpl struct {
	cfg CMSConfig
}

func (c *cmsDataLoaderImpl) Name() string                       { return "cms" }
func (c *cmsDataLoaderImpl) Close() error                       { return nil }
func (c *cmsDataLoaderImpl) HealthCheck(ctx context.Context) error { return nil }
func (c *cmsDataLoaderImpl) LoadFormulary(ctx context.Context, year int) ([]FormularyEntry, error) {
	return nil, fmt.Errorf("not implemented - see datasources/cms/loader.go")
}
func (c *cmsDataLoaderImpl) LoadPricing(ctx context.Context, ndc string) (*DrugPricing, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *cmsDataLoaderImpl) LoadUtilization(ctx context.Context, ndc string) (*UtilizationData, error) {
	return nil, fmt.Errorf("not implemented")
}

// LLMConfig for LLM client
type LLMConfig struct {
	Provider  string
	APIKey    string
	Model     string
	MaxTokens int
	Timeout   time.Duration
	Log       *logrus.Entry
}

// NewLLMClient creates a new LLM client
func NewLLMClient(cfg LLMConfig) LLMClient {
	return &llmClientImpl{cfg: cfg}
}

type llmClientImpl struct {
	cfg LLMConfig
}

func (c *llmClientImpl) Name() string                       { return "llm-" + c.cfg.Provider }
func (c *llmClientImpl) Close() error                       { return nil }
func (c *llmClientImpl) HealthCheck(ctx context.Context) error { return nil }
func (c *llmClientImpl) Extract(ctx context.Context, prompt string, content string) (string, error) {
	return "", fmt.Errorf("not implemented - see datasources/llm/client.go")
}
func (c *llmClientImpl) ExtractStructured(ctx context.Context, prompt string, content string, schema interface{}) error {
	return fmt.Errorf("not implemented")
}
func (c *llmClientImpl) GetTokenCount(ctx context.Context, text string) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

// =============================================================================
// DATABASE PROVIDERS
// =============================================================================

// DatabaseModule provides database connections
type DatabaseModule struct {
	FactStoreConfig DatabaseConfig
	Log             *logrus.Entry
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	MaxConns int
	MinConns int
}

func (m *DatabaseModule) Name() string {
	return "database"
}

func (m *DatabaseModule) Dependencies() []Module {
	return nil
}

func (m *DatabaseModule) Register(c *Container) error {
	return c.RegisterFunc("factstore-db", TypeOf[*sql.DB](), Singleton, m.provideFactStoreDB)
}

func (m *DatabaseModule) provideFactStoreDB(ctx context.Context, c *Container) (interface{}, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		m.FactStoreConfig.Host,
		m.FactStoreConfig.Port,
		m.FactStoreConfig.User,
		m.FactStoreConfig.Password,
		m.FactStoreConfig.Database,
		m.FactStoreConfig.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(m.FactStoreConfig.MaxConns)
	db.SetMaxIdleConns(m.FactStoreConfig.MinConns)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	m.Log.Info("Connected to Fact Store database")
	return db, nil
}

// =============================================================================
// CONVENIENCE FUNCTIONS
// =============================================================================

// DefaultDataSourcesConfig returns default configuration
func DefaultDataSourcesConfig() DataSourcesConfig {
	return DataSourcesConfig{
		// RxNav (NLM - free, no key required)
		RxNavBaseURL: "https://rxnav.nlm.nih.gov/REST",
		RxNavTimeout: 30 * time.Second,

		// DailyMed (FDA - free, no key required)
		DailyMedBaseURL: "https://dailymed.nlm.nih.gov/dailymed/services/v2",
		DailyMedTimeout: 30 * time.Second,

		// DrugBank (requires API key)
		DrugBankBaseURL: "https://api.drugbank.com/v1",
		DrugBankTimeout: 30 * time.Second,

		// MED-RT (NCI - free, no key required)
		MEDRTBaseURL: "https://rxnav.nlm.nih.gov/REST/Ndfrt",
		MEDRTTimeout: 30 * time.Second,

		// CMS
		CMSDataDir:      "/data/cms",
		CMSCacheEnabled: true,

		// LLM
		LLMProvider:  "openai",
		LLMModel:     "gpt-4",
		LLMMaxTokens: 4000,
		LLMTimeout:   120 * time.Second,

		// HTTP
		HTTPMaxConns: 100,
		HTTPTimeout:  60 * time.Second,
	}
}

// DefaultDatabaseConfig returns default Fact Store database configuration
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:     "localhost",
		Port:     5434, // Fact Store dedicated port
		User:     "factstore",
		Password: "factstore",
		Database: "canonical_factstore",
		SSLMode:  "disable",
		MaxConns: 25,
		MinConns: 5,
	}
}
