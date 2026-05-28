import { describe, it, expect } from 'vitest';
import {
  isGoogleScoutAvailable,
  isGoogleScoutLocked,
} from '../../apps/web/src/lib/capabilities.js';

describe('isGoogleScoutAvailable', () => {
  it('returns true when snapshot is missing', () => {
    expect(isGoogleScoutAvailable(null)).toBe(true);
    expect(isGoogleScoutAvailable(undefined)).toBe(true);
  });

  it('returns true when core.scout.run.google capability is true', () => {
    const snapshot = { capabilities: { 'core.scout.run.google': true } };
    expect(isGoogleScoutAvailable(snapshot)).toBe(true);
  });

  it('returns false when core.scout.run.google capability is false', () => {
    const snapshot = { capabilities: { 'core.scout.run.google': false } };
    expect(isGoogleScoutAvailable(snapshot)).toBe(false);
  });
});

describe('isGoogleScoutLocked', () => {
  it('returns false when snapshot is missing', () => {
    expect(isGoogleScoutLocked(null)).toBe(false);
    expect(isGoogleScoutLocked(undefined)).toBe(false);
  });

  it('returns false when core.scout.run.google capability is true', () => {
    const snapshot = { capabilities: { 'core.scout.run.google': true } };
    expect(isGoogleScoutLocked(snapshot)).toBe(false);
  });

  it('returns true when core.scout.run.google capability is false with loaded capabilities', () => {
    const snapshot = { capabilities: { 'core.scout.run.google': false } };
    expect(isGoogleScoutLocked(snapshot)).toBe(true);
  });
});
