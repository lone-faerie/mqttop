package discovery

type Option string

// Options for origin
const (
	Name       Option = "name"
	SWVersion  Option = "sw"
	SupportURL Option = "url"
)

// Options for device
const (
	ConfigurationURL Option = "cu"
	Connections      Option = "cns"
	Identifiers      Option = "ids"
	Manufacturer     Option = "mf"
	Model            Option = "mdl"
	ModelID          Option = "mdl_id"
	HWVersion        Option = "hw"
	SuggestedArea    Option = "sa"
	SerialNumber     Option = "sn"
)

// Options for components
const (
	Availability              Option = "avty"
	AvailabilityMode          Option = "avty_mode"
	AvailabilityTopic         Option = "avty_t"
	AvailabilityTemplate      Option = "avty_tpl"
	CommandOffTemplate        Option = "cmd_off_tpl"
	CommandOnTemplate         Option = "cmd_on_tpl"
	CommandTopic              Option = "cmd_t"
	CommandTemplate           Option = "cmd_tpl"
	DeviceClass               Option = "dev_cla"
	DisplayPrecision          Option = "dsp_prc"
	EnabledByDefault          Option = "en"
	EntityCategory            Option = "ent_cat"
	ForceUpdate               Option = "frc_upd"
	Icon                      Option = "ic"
	JSONAttributes            Option = "json_attr"
	JSONAttributesTopic       Option = "json_attr_t"
	JSONAttributesTemplate    Option = "json_attr_tpl"
	Max                       Option = "max"
	MaxTemp                   Option = "max_temp"
	Min                       Option = "min"
	MinTemp                   Option = "min_temp"
	ObjectID                  Option = "obj_id"
	Options                   Option = "ops"
	Platform                  Option = "p"
	Payload                   Option = "pl"
	PayloadAvailable          Option = "pl_ avail"
	PayloadNotAvailable       Option = "pl_not_avail"
	Retain                    Option = "ret"
	StateClass                Option = "stat_cla"
	StateTopic                Option = "stat_t"
	StateTemplate             Option = "stat_tpl"
	StateValueTemplate        Option = "stat_val_tpl"
	SuggestedDisplayPrecision Option = "sug_dsp_prc"
	Topic                     Option = "t"
	TemperatureStateTopic     Option = "temp_stat_t"
	TemperatureStateTemplate  Option = "temp_stat_tpl"
	TemperatureUnit           Option = "temp_unit"
	UniqueID                  Option = "uniq_id"
	UnitOfMeasurement         Option = "unit_of_meas"
	ValueTemplate             Option = "val_tpl"
)
