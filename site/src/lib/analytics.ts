const API_KEY = '7edb71bd4028950875d4315efb53c2da';

let initialized = false;

interface AmplitudeWindow {
  amplitude: {
    add: (plugin: unknown) => void;
    init: (key: string, opts: Record<string, unknown>) => void;
  };
  sessionReplay: { plugin: (opts: { sampleRate: number }) => unknown };
}

// Amplitude browser snippet: the CDN script bundle exposes window.amplitude
// and window.sessionReplay; loaded dynamically so the webdriver guard keeps
// the prerender from firing events.
export function initAmplitude(): void {
  if (initialized) return;
  if (typeof window === 'undefined') return;
  if (navigator.webdriver) return;
  initialized = true;
  const s = document.createElement('script');
  s.src = `https://cdn.amplitude.com/script/${API_KEY}.js`;
  s.async = true;
  s.onload = () => {
    const w = window as unknown as AmplitudeWindow;
    w.amplitude.add(w.sessionReplay.plugin({ sampleRate: 1 }));
    w.amplitude.init(API_KEY, { fetchRemoteConfig: true, autocapture: false });
  };
  document.head.appendChild(s);
}
