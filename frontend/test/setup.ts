import '@testing-library/jest-dom';
import { afterEach } from 'bun:test';
import { cleanup } from '@testing-library/react';

afterEach(() => cleanup());

// Fill the DOM gaps Mantine depends on that happy-dom doesn't implement.
if (typeof window !== 'undefined') {
  if (!window.matchMedia) {
    window.matchMedia = ((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener() {},
      removeListener() {},
      addEventListener() {},
      removeEventListener() {},
      dispatchEvent() {
        return false;
      },
    })) as unknown as typeof window.matchMedia;
  }
  if (!('ResizeObserver' in window)) {
    class ResizeObserverStub {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
    (window as unknown as { ResizeObserver: typeof ResizeObserverStub }).ResizeObserver = ResizeObserverStub;
  }
  // Mantine v9's autosize Textarea listens on document.fonts; happy-dom has no
  // FontFaceSet, so stub it.
  const doc = document as unknown as { fonts?: { addEventListener?: unknown } };
  if (!doc.fonts || typeof doc.fonts.addEventListener !== 'function') {
    Object.defineProperty(document, 'fonts', {
      configurable: true,
      value: { addEventListener() {}, removeEventListener() {}, ready: Promise.resolve() },
    });
  }
}
