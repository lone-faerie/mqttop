package discovery

import (
	"context"
	"slices"

	"github.com/eclipse/paho.mqtt.golang"
)

func (d *Discovery) removeComponents(ctx context.Context, c mqtt.Client, components ...string) error {
	payload := []byte{}
	for name, cmp := range d.Components {
		if len(components) > 0 && !slices.Contains(components, name) {
			continue
		}
		platform := cmp[Platform].(string)
		topic, err := d.Topic(d.cfg.Prefix, platform, d.NodeID, name)
		if err != nil {
			return err
		}
		t := c.Publish(topic, d.cfg.QoS, d.cfg.Retained, payload)
		select {
		case <-ctx.Done():
			return nil
		case <-t.Done():
		}
		if err := t.Error(); err != nil {
			return err
		}
	}
	return nil
}

func (d *Discovery) removeDevice(ctx context.Context, c mqtt.Client) error {
	return d.removeDeviceNode(ctx, c, d.NodeID)
}

func (d *Discovery) removeDeviceNode(ctx context.Context, c mqtt.Client, nodeID string) error {
	topic, err := d.Topic(d.cfg.Prefix, "device", nodeID, d.ObjectID)
	if err != nil {
		return err
	}
	t := c.Publish(topic, d.cfg.QoS, d.cfg.Retained, []byte{})
	select {
	case <-ctx.Done():
		return nil
	case <-t.Done():
	}
	return t.Error()
}

var migratePayload = []byte("{\"migrate_discovery\": true}")

// Migrate publishes `{"migrate_discovery": true}` to each component's
// discovery topic. This is the first step required for migrating
// component discoveries to a device discovery.
func (d *Discovery) Migrate(ctx context.Context, c mqtt.Client) error {
	return d.migrate(ctx, c, d.NodeID)
}

func (d *Discovery) migrate(ctx context.Context, c mqtt.Client, nodeID string) error {
	for name, cmp := range d.Components {
		platform := cmp[Platform].(string)
		topic, err := d.Topic(d.cfg.Prefix, platform, nodeID, name)
		if err != nil {
			return err
		}
		t := c.Publish(topic, d.cfg.QoS, d.cfg.Retained, migratePayload)
		select {
		case <-ctx.Done():
			return nil
		case <-t.Done():
		}
		if err := t.Error(); err != nil {
			return err
		}
	}
	return nil
}

// Rollback publishes `{"migrate_discovery": true}` to the device discovery topic.
// This is the first step required for rolling back a device discovery to individual
// component discoveries.
func (d *Discovery) Rollback(ctx context.Context, c mqtt.Client) error {
	return d.rollback(ctx, c, d.NodeID)
}

func (d *Discovery) rollback(ctx context.Context, c mqtt.Client, nodeID string) error {
	topic, err := d.Topic(d.cfg.Prefix, "device", nodeID, d.ObjectID)
	if err != nil {
		return err
	}
	t := c.Publish(topic, d.cfg.QoS, d.cfg.Retained, migratePayload)
	select {
	case <-ctx.Done():
		return nil
	case <-t.Done():
	}
	return t.Error()
}
