# Mqttop
Provide system metrics over MQTT

## Configuration
Any string config field may be set to an environment variable `$<variable>` or Docker secret `!secret <secret>`.

| Field | Type | Description |
| ----- | ---- | ----------- |
| `interval` | `string` | Default update interval for metrics |
| `mqtt` | `MQTTConfig` | MQTT configuration |
| `discovery` | `DiscoveryConfig` | Discovery configuration |
| `log` | `LogConfig` | Log configuration |
| `cpu` | `CPUConfig` | CPU metric configuration |
| `memory` | `MemoryConfig` | Memory metric configuration |
| `disks` | `DisksConfig` | Disks metric configuration |
| `net` | `NetConfig` | Network metric configuration |
| `battery` | `BatteryConfig` | Battery metric configuration |
| `dirs` | `[]DirConfig` | List of directory metric configurations |
| `gpu` | `GPUConfig` | GPU metric configuration |
