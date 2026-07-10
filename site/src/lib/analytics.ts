import * as amplitude from '@amplitude/unified';

// Set the project's Amplitude API key to enable analytics; empty = disabled.
const API_KEY = '';

let initialized = false;

export function initAmplitude(): void {
  if (initialized) return;
  if (!API_KEY) return;
  if (typeof window === 'undefined') return;
  if (navigator.webdriver) return;
  initialized = true;
  amplitude.initAll(API_KEY, {
    analytics: { autocapture: true },
    sessionReplay: { sampleRate: 1 },
  });
}
