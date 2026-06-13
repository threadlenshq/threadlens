import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { initTelemetry, isTelemetryEnabled, recordEvent, updateTelemetryStatus, getInstanceId } from './telemetry.js';

describe('telemetry client', () => {
  let sendBeaconMock;

  beforeEach(() => {
    sendBeaconMock = vi.fn(() => true);
    Object.defineProperty(globalThis, 'navigator', {
      value: { sendBeacon: sendBeaconMock },
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    initTelemetry(null);
  });

  it('isTelemetryEnabled returns false when not initialized', () => {
    expect(isTelemetryEnabled()).toBe(false);
  });

  it('isTelemetryEnabled returns false when env_opt_in is disabled', () => {
    initTelemetry({
      env_opt_in: 'disabled',
      ui_consent: 'granted',
      instance_id: 'test-uuid',
    });
    expect(isTelemetryEnabled()).toBe(false);
  });

  it('isTelemetryEnabled returns false when ui_consent is declined in consent mode', () => {
    initTelemetry({
      env_opt_in: 'consent',
      ui_consent: 'declined',
      instance_id: 'test-uuid',
    });
    expect(isTelemetryEnabled()).toBe(false);
  });

  it('isTelemetryEnabled returns false when instance_id is empty', () => {
    initTelemetry({
      env_opt_in: 'enabled',
      ui_consent: 'granted',
      instance_id: '',
    });
    expect(isTelemetryEnabled()).toBe(false);
  });

  it('isTelemetryEnabled returns true when mode is enabled', () => {
    initTelemetry({
      env_opt_in: 'enabled',
      ui_consent: 'declined',
      instance_id: 'test-uuid',
      scout_version: '0.7.2',
      deployment_type: 'docker',
      os_platform: 'linux',
    });
    expect(isTelemetryEnabled()).toBe(true);
  });

  it('isTelemetryEnabled returns true when mode is consent and consent is granted', () => {
    initTelemetry({
      env_opt_in: 'consent',
      ui_consent: 'granted',
      instance_id: 'test-uuid',
      scout_version: '0.7.2',
      deployment_type: 'docker',
      os_platform: 'linux',
    });
    expect(isTelemetryEnabled()).toBe(true);
  });

  it('recordEvent does nothing when telemetry is disabled', () => {
    initTelemetry({ env_opt_in: 'disabled', ui_consent: 'granted', instance_id: 'uuid' });
    recordEvent('feature_used:scout_run');
    expect(sendBeaconMock).not.toHaveBeenCalled();
  });

  it('recordEvent calls sendBeacon with correct payload', () => {
    initTelemetry({
      env_opt_in: 'enabled',
      ui_consent: 'granted',
      instance_id: 'test-uuid-123',
      scout_version: '0.7.2',
      deployment_type: 'docker',
      os_platform: 'linux',
    });
    recordEvent('feature_used:scout_run');
    expect(sendBeaconMock).toHaveBeenCalledTimes(1);
    const [url, blob] = sendBeaconMock.mock.calls[0];
    expect(url).toBe('https://telemetry.threadlens.dev/v1/events');
    expect(blob).toBeInstanceOf(Blob);
  });

  it('updateTelemetryStatus updates consent state in consent mode', () => {
    initTelemetry({
      env_opt_in: 'consent',
      ui_consent: 'unset',
      instance_id: 'test-uuid',
    });
    expect(isTelemetryEnabled()).toBe(false);
    updateTelemetryStatus({ ui_consent: 'granted' });
    expect(isTelemetryEnabled()).toBe(true);
  });

  it('getInstanceId returns the cached instance_id', () => {
    initTelemetry({ instance_id: 'my-uuid' });
    expect(getInstanceId()).toBe('my-uuid');
  });

  it('getInstanceId returns empty string when not initialized', () => {
    expect(getInstanceId()).toBe('');
  });
});
