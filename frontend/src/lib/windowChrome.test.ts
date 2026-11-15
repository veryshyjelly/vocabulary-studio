import { afterEach, describe, expect, it, vi } from "vitest";
import { System } from "@wailsio/runtime";
import { usesMacWindowChrome } from "./windowChrome";

vi.mock("@wailsio/runtime", () => ({ System: { IsMac: vi.fn() } }));

const originalWebKit = Object.getOwnPropertyDescriptor(window, "webkit");
afterEach(() => {
  if (originalWebKit) Object.defineProperty(window, "webkit", originalWebKit);
  else Reflect.deleteProperty(window, "webkit");
  vi.resetAllMocks();
});

describe("native window chrome", () => {
  it("reserves controls only in the native macOS WebView", () => {
    vi.mocked(System.IsMac).mockReturnValue(true);
    Object.defineProperty(window, "webkit", { configurable: true, value: { messageHandlers: { external: { postMessage: vi.fn() } } } });
    expect(usesMacWindowChrome()).toBe(true);
  });

  it("leaves macOS server/browser previews without native padding", () => {
    vi.mocked(System.IsMac).mockReturnValue(true);
    Object.defineProperty(window, "webkit", { configurable: true, value: undefined });
    expect(usesMacWindowChrome()).toBe(false);
  });

  it("leaves other native platforms unchanged", () => {
    vi.mocked(System.IsMac).mockReturnValue(false);
    Object.defineProperty(window, "webkit", { configurable: true, value: { messageHandlers: { external: { postMessage: vi.fn() } } } });
    expect(usesMacWindowChrome()).toBe(false);
  });

  it("does not mistake an unrelated WebKit API for the native bridge", () => {
    vi.mocked(System.IsMac).mockReturnValue(true);
    Object.defineProperty(window, "webkit", { configurable: true, value: { messageHandlers: {} } });
    expect(usesMacWindowChrome()).toBe(false);
  });
});
