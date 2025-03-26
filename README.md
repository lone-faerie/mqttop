# Mqttop
Provide system metrics over MQTT

## Installation
There are two provided docker images, one with GPU support and one without. To monitor the host metrics, mount the root directory and set the environment variable `$MQTTOP_ROOTFS_PATH` to the mount point in the container, and to monitor the host network metrics, set `network_mode` to `host`. In order for GPU support to work, you must have the [NVIDIA Container Toolkit](https://github.com/NVIDIA/nvidia-container-toolkit) installed.

### docker-compose.yml
```yaml
services:
  mqttop:
    image: ghcr.io/lone-faerie/mqttop:latest
    environment:
      - MQTTOP_ROOTFS_PATH=/host
    volumes:
      - "./config.yml:/config/config.yml"
      - "/:/host:ro"
    network_mode: host
```

### docker-compose.yml - GPU Support
```yaml
services:
  mqttop:
    image: ghcr.io/lone-faerie/mqttop:gpu
    environment:
      - MQTTOP_ROOTFS_PATH=/host
    volumes:
      - "./config.yml:/config/config.yml"
      - "/:/host:ro"
    network_mode: host
    runtime: nvidia
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: all
              capabilities: [gpu]
```

## Configuration
Configuration files are stored in yaml format. Configs can be broken up into multiple files and may be passed as either a list of files or directories. The path to config files is either the path(s) passed as arguments, the value of `$MQTTOP_CONFIG_PATH`, `$XDG_CONFIG_HOME/mqttop.yaml`, or `$HOME/.config/mqttop.yaml`. The default path for config files in the docker container is `/config/config.yml`.

Durations are parsed using Go's [time.ParseDuration](https://pkg.go.dev/time#ParseDuration) and any strings may be set to an environment variable `$<variable>` or Docker secret `!secret <secret>`.

| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `interval` | duration | 2s | Default update interval for metrics |
| `mqtt` | [MQTTConfig](#mqtt-configuration) | | MQTT configuration |
| `discovery` | [DiscoveryConfig](#discovery-configuration) | | Discovery configuration |
| `log` | [LogConfig](#log-configuration) | | Log configuration |
| `cpu` | [CPUConfig](#cpu-configuration) | | CPU metric configuration |
| `memory` | [MemoryConfig](#memory-configuration) | | Memory metric configuration |
| `disks` | [DisksConfig](#disks-configuration) | | Disks metric configuration |
| `net` | [NetConfig](#network-configuration) | | Network metric configuration |
| `battery` | [BatteryConfig](#battery-configuration) | | Battery metric configuration |
| `dirs` | list [DirConfig](#directory-configuration) | | List of directory metric configurations |
| `gpu` | [GPUConfig](#gpu-configuration) | | GPU metric configuration |

### MQTT Configuration
| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `broker` | string | "$MQTTOP_BROKER_ADDRESS" | Address of the MQTT broker |
| `client_id` | string | | Client ID used when connecting to the broker |
| `username` | string | "$MQTTOP_BROKER_USERNAME" | Username used to connect to the broker |
| `password` | string | "$MQTTOP_BROKER_PASSWORD" | Password used to connect to the broker |
| `keep_alive` | duration | 30s | Amount of time to wait before sending a PING to the broker |
| `cert_file` | string | | Path to the cert file for SSL, disabled if blank |
| `key_file` | string | | Path to the key file for SSL, disabled if blank |
| `reconnect_interval` | duration | 10m | Maximum time to wait before attempting to reconnect |
| `connect_timeout` | duration | 30s | Amount of time to wait when connecting before timeout |
| `ping_timeout` | duration | 10s | Amount of time to wait after sending a PING before deciding to timeout |
| `write_timeout` | duration | 0 | Amount of time to wait after publishing before deciding to timeout, 0 means never timeout |
| `birth_lwt_enabled` | bool | true | Enable/disable birth and LWT message |
| `birth_lwt_topic` | string | "mqttop/bridge/status" | Topic to publish birth and LWT message to |
| `log_level` | level | DISABLED | Log level to provide to the MQTT client |

See https://pkg.go.dev/github.com/eclipse/paho.mqtt.golang#ClientOptions

### Discovery Configuration
| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `enabled` | bool | true | Enabled/disable MQTT discovery |
| `prefix` | string | "homeassistant" | Prefix of discovery topic |
| `device_name` | string | | Name of device used for discovery, if blank or "hostname" will use device hostname, if "username" will use MQTT username |
| `node_id` | string | | Optional node ID to use for discovery |
| `availability` | string | | Topic to publish availability to, if blank will use MQTT `birth_lwt_topic` |
| `retained` | bool | true | Retain discovery payload at the broker |
| `qos` | int | QoS of discovery payload |
| `wait_topic` | string | | Topic to wait for payload on before publishing discovery, if blank will not wait |
| `wait_payload` | string | | Payload to wait for from `wait_topic` before publishing discovery, if blank will wait for any payload |

See https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery

### Log Configuration
| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `level` | level | INFO | Log level to use |
| `output` | string | | Where to output logs, one of stderr, stdout, or path to a file, if blank will default to stderr |
| `format` | string | | Format of log messages, either blank or json |

### CPU Configuration
| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `enabled` | bool | true | Enable/disable the metric |
| `interval` | duration | | Update interval of the metric, if 0 will be top-level `interval`
| `topic` | string | "mqttop/metric/cpu" | Topic to publish updates to |
| `name` | string | | Custom name to use for the CPU |
| `name_template` | string | | Template to use for the CPU name, will override `name` |
| `selection_mode` | string | `auto` | Mode used to select overall CPU temperature and frequency, one of `auto`, `first`, `average`, `max`, `min`, `random` |

### Memory Configuration
| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `enabled` | bool | true | Enable/disable the metric |
| `interval` | duration | | Update interval of the metric, if 0 will be top-level `interval` |
| `topic` | string | "mqttop/metric/memory" | Topic to publish updates to |
| `size_unit` | string | | Size unit to use for memory size, if blank, will be automatically determined |
| `include_swap` | bool | true | Include swap in the metrics |

### Disks Configuration
| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `enabled` | bool | true | Enable/disable the metric |
| `interval` | duration | | Update interval of the metric, if 0 will be top-level `interval` |
| `topic` | string | "mqttop/metric/disks" | Topic to publish updates to |
| `use_fstab` | bool | true | Use /etc/fstab to find disks |
| `rescan` | bool or duration | | Interval to rescan for disks, if true will use update interval, else the given interval |
| `show_io` | bool | true | Include disk IO in metrics |
| `disk` | list [DiskConfig](#disk-configuration) | | List of individual disk configurations |

### Disk Configuration
| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `enabled` | bool | true | Enable/disable the metric |
| `interval` | duration | | Update interval of the metric, if 0 will be top-level `interval` |
| `exclude` | bool | false | Exclude the disk from metrics |
| `name` | string | | Custom name to use for the disk |
| `name_template` | string | | Template to use for the disk name, will override `name` |
| `mount_point` | string | | Path to mount point of the disk |
| `size_unit` | string | | Size unit to use for disk size, if blank, will be automatically determined |
| `show_io` | bool | true | Include disk IO in metrics |

### Network Configuration
| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `enabled` | bool | true | Enable/disable the metric |
| `interval` | duration | | Update interval of the metric, if 0 will be top-level `interval` |
| `topic` | string | "mqttop/metric/net" | Topic to publish updates to |
| `only_physical` | bool | false | Only include physical network interfaces |
| `only_running` | bool | false | Only include running network interfaces |
| `include_bridge` | bool | false | Include bridge interfaces |
| `rescan` | bool or duration | | Interval to rescan for interfaces, if true will use update interval, else the given interval |
| `rate_unit` | string | | Rate unit to use for network throughput, if blank, will be automatically determined |
| `include` | list [NetIfaceConfig](#network-interface-config), list string | | List of network interface configurations to explicitly include, if string will be name of interface |
| `exclude` | list string | | List of network interfaces to explicitly exclude |

### Network Interface Configuration
| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `name` | string | | Name to use for representing the interface |
| `name_template` | string | | Template to use for the interface name, will override `name` |
| `interface` | string | | Name of the interface on the system |
| `rate_unit` | string | | Rate unit to use for network throughput, if blank, will use network config `rate_unit` |

### Battery Configuration
| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `enabled` | bool | true | Enable/disable the metric |
| `interval` | duration | | Update interval of the metric, if 0 will be top-level `interval` |
| `topic` | string | "mqttop/metric/battery" | Topic to publish updates to |
| `time_format` | string | | Format used to represent time remaining |

### Directory Configuration
| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `enabled` | bool | true | Enable/disable the metric |
| `interval` | duration | | Update interval of the metric, if 0 will be top-level `interval` |
| `topic` | string | "mqttop/metric/dir/<dir path>" | Topic to publish updates to |
| `name` | string | | Custom name to use for the directory |
| `name_template` | string | | Template to use for the directory name, will override `name` |
| `path` | string | | Path to the directory |
| `size_unit` | string | | Size unit to use for directory size, if blank, will be automatically determined |
| `watch` | bool | false | Watch the directory for changes instead of polling every update interval |
| `depth` | int | -1 | Maximum depth to recursively watch the directory, if < 0, will watch the entire depth |

### GPU Configuration
| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `enabled` | bool | true | Enable/disable the metric |
| `interval` | duration | | Update interval of the metric, if 0 will be top-level `interval` |
| `topic` | string | "mqttop/metric/gpu" | Topic to publish updates to |
| `name` | string | | Custom name to use for the directory |
| `name_template` | string | | Template to use for the directory name, will override `name` |
| `platform` | string | | Platform of GPU to use, currently only supports nvidia |
| `index` | int | 0 | Index of GPU to use |
| `size_unit` | string | | Size unit to use for memory size, if blank, will be automatically determined |
| `include_procs` | bool | false | Include GPU usage of processes |
